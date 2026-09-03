package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"lotustab/internal/core/driver"
	"lotustab/internal/drivers/sqlutil"
)

func (c *conn) DescribeTable(ctx context.Context, ref driver.TableRef) (*driver.TableStructure, error) {
	ref = driver.ClampTableRef(ref)
	if ref.Name == "" {
		return nil, fmt.Errorf("table name is required")
	}
	dbName := clickhouseDBName(ref.Database, ref.Schema)

	var out *driver.TableStructure
	err := c.withSession(ctx, func(session *sql.Conn) error {
		cols, err := clickhouseDescribeColumns(ctx, session, dbName, ref.Name)
		if err != nil {
			return err
		}
		out = &driver.TableStructure{Columns: cols, CanReorder: true}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *conn) AlterTable(ctx context.Context, req driver.AlterTableRequest) error {
	req = driver.ClampAlterTable(req)
	if req.Name == "" {
		return fmt.Errorf("table name is required")
	}
	if len(req.Ops) == 0 {
		return fmt.Errorf("no schema changes")
	}
	dbName := clickhouseDBName(req.Database, req.Schema)
	qualified := clickhouseQualified(dbName, req.Name)
	stmts, err := clickhouseAlterSQL(qualified, req.Ops)
	if err != nil {
		return err
	}
	return c.withSession(ctx, func(session *sql.Conn) error {
		for _, stmt := range stmts {
			if _, err := session.ExecContext(ctx, stmt); err != nil {
				return err
			}
		}
		return nil
	})
}

func clickhouseDescribeColumns(ctx context.Context, q sqlutil.Querier, database, name string) ([]driver.ColumnInfo, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if database == "" {
		rows, err = q.QueryContext(ctx, `
			SELECT name, type, default_kind, default_expression, comment, position, is_in_primary_key
			FROM system.columns
			WHERE database = currentDatabase()
			  AND table = ?
			ORDER BY position`, name)
	} else {
		rows, err = q.QueryContext(ctx, `
			SELECT name, type, default_kind, default_expression, comment, position, is_in_primary_key
			FROM system.columns
			WHERE database = ?
			  AND table = ?
			ORDER BY position`, database, name)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]driver.ColumnInfo, 0)
	for rows.Next() {
		var (
			colName, colType, defaultKind, defExpr, comment string
			ordinal                                         uint64
			inPK                                            uint8
		)
		if err := rows.Scan(&colName, &colType, &defaultKind, &defExpr, &comment, &ordinal, &inPK); err != nil {
			return nil, err
		}
		info := driver.ColumnInfo{
			Name:         colName,
			Type:         colType,
			Nullable:     clickhouseNullable(colType),
			IsPrimaryKey: inPK == 1,
			Extra:        clickhouseExtra(defaultKind),
			Comment:      comment,
			Ordinal:      int(ordinal),
		}
		if defExpr != "" {
			info.Default = &defExpr
		}
		out = append(out, info)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func clickhouseNullable(colType string) bool {
	return strings.Contains(strings.ToLower(colType), "nullable(")
}

func clickhouseExtra(defaultKind string) string {
	kind := strings.TrimSpace(defaultKind)
	if kind == "" || strings.EqualFold(kind, "DEFAULT") {
		return ""
	}
	return kind
}

func clickhouseAlterSQL(qualified string, ops []driver.ColumnOp) ([]string, error) {
	if len(ops) == 0 {
		return nil, fmt.Errorf("no schema changes")
	}
	out := make([]string, 0)
	for i, op := range ops {
		stmts, err := clickhouseAlterStmts(qualified, op)
		if err != nil {
			return nil, fmt.Errorf("op %d: %w", i, err)
		}
		out = append(out, stmts...)
	}
	return out, nil
}

func clickhouseAlterStmts(qualified string, op driver.ColumnOp) ([]string, error) {
	q := sqlutil.QuoteClickHouse
	switch op.Kind {
	case driver.ColumnOpDrop:
		if op.Name == "" {
			return nil, fmt.Errorf("drop requires a column name")
		}
		return []string{"ALTER TABLE " + qualified + " DROP COLUMN " + q(op.Name)}, nil
	case driver.ColumnOpAdd:
		if op.Column == nil {
			return nil, fmt.Errorf("add requires a column definition")
		}
		def, err := clickhouseColumnDef(*op.Column)
		if err != nil {
			return nil, err
		}
		return []string{"ALTER TABLE " + qualified + " ADD COLUMN " + def + clickhousePosition(op.After)}, nil
	case driver.ColumnOpModify:
		if op.Column == nil {
			return nil, fmt.Errorf("modify requires a column definition")
		}
		oldName := op.Name
		if oldName == "" {
			oldName = op.Column.Name
		}
		if oldName == "" {
			return nil, fmt.Errorf("modify requires a column name")
		}
		return clickhouseModifyStmts(qualified, oldName, *op.Column, op.After)
	case driver.ColumnOpReorder:
		if op.Column == nil {
			return nil, fmt.Errorf("reorder requires a column definition")
		}
		name := op.Name
		if name == "" {
			name = op.Column.Name
		}
		if name == "" {
			return nil, fmt.Errorf("reorder requires a column name")
		}
		if op.Column.Type == "" {
			return nil, fmt.Errorf("reorder requires a column type")
		}
		return []string{
			"ALTER TABLE " + qualified + " MODIFY COLUMN " + q(name) + " " + op.Column.Type + clickhousePosition(op.After),
		}, nil
	default:
		return nil, fmt.Errorf("unknown alter op %q", op.Kind)
	}
}

func clickhouseModifyStmts(qualified, oldName string, col driver.ColumnInfo, after *string) ([]string, error) {
	q := sqlutil.QuoteClickHouse
	current := oldName
	stmts := make([]string, 0, 2)
	newName := col.Name
	if newName == "" {
		newName = oldName
	}
	if newName != oldName {
		stmts = append(stmts, "ALTER TABLE "+qualified+" RENAME COLUMN "+q(oldName)+" TO "+q(newName))
		current = newName
	}
	def := driver.ColumnInfo{
		Name:    current,
		Type:    col.Type,
		Default: col.Default,
		Comment: col.Comment,
	}
	colDef, err := clickhouseColumnDef(def)
	if err != nil {
		return nil, err
	}
	stmts = append(stmts, "ALTER TABLE "+qualified+" MODIFY COLUMN "+colDef+clickhousePosition(after))
	return stmts, nil
}

func clickhouseColumnDef(col driver.ColumnInfo) (string, error) {
	if col.Name == "" {
		return "", fmt.Errorf("column name is required")
	}
	if col.Type == "" {
		return "", fmt.Errorf("column type is required")
	}
	var b strings.Builder
	b.WriteString(sqlutil.QuoteClickHouse(col.Name))
	b.WriteByte(' ')
	b.WriteString(col.Type)
	if col.Default != nil {
		b.WriteString(" DEFAULT ")
		b.WriteString(sqlutil.FormatDefault(*col.Default))
	}
	if col.Comment != "" {
		b.WriteString(" COMMENT ")
		b.WriteString(sqlutil.QuoteString(col.Comment))
	}
	return b.String(), nil
}

func clickhousePosition(after *string) string {
	if after == nil {
		return ""
	}
	if *after == "" {
		return " FIRST"
	}
	return " AFTER " + sqlutil.QuoteClickHouse(*after)
}
