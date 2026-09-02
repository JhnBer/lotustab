package mysql

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
	dbName := ref.Database
	if dbName == "" {
		dbName = ref.Schema
	}

	var out *driver.TableStructure
	err := c.withSession(ctx, func(session *sql.Conn) error {
		cols, err := mysqlDescribeColumns(ctx, session, dbName, ref.Name)
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
	dbName := req.Database
	if dbName == "" {
		dbName = req.Schema
	}
	qualified := mysqlQualified(dbName, req.Name)
	sqlText, err := mysqlAlterSQL(qualified, req.Ops)
	if err != nil {
		return err
	}
	return c.withSession(ctx, func(session *sql.Conn) error {
		_, err := session.ExecContext(ctx, sqlText)
		return err
	})
}

func mysqlDescribeColumns(ctx context.Context, q sqlutil.Querier, database, name string) ([]driver.ColumnInfo, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if database == "" {
		rows, err = q.QueryContext(ctx, `
			SELECT COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE, COLUMN_DEFAULT, EXTRA, COLUMN_KEY, ORDINAL_POSITION, COLUMN_COMMENT
			FROM information_schema.COLUMNS
			WHERE TABLE_SCHEMA = DATABASE()
			  AND TABLE_NAME = ?
			ORDER BY ORDINAL_POSITION`, name)
	} else {
		rows, err = q.QueryContext(ctx, `
			SELECT COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE, COLUMN_DEFAULT, EXTRA, COLUMN_KEY, ORDINAL_POSITION, COLUMN_COMMENT
			FROM information_schema.COLUMNS
			WHERE TABLE_SCHEMA = ?
			  AND TABLE_NAME = ?
			ORDER BY ORDINAL_POSITION`, database, name)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]driver.ColumnInfo, 0)
	for rows.Next() {
		var (
			colName, colType, nullable, extra, colKey, comment string
			ordinal                                            int
			def                                                sql.NullString
		)
		if err := rows.Scan(&colName, &colType, &nullable, &def, &extra, &colKey, &ordinal, &comment); err != nil {
			return nil, err
		}
		info := driver.ColumnInfo{
			Name:          colName,
			Type:          colType,
			Nullable:      strings.EqualFold(nullable, "YES"),
			AutoIncrement: strings.Contains(strings.ToLower(extra), "auto_increment"),
			IsPrimaryKey:  strings.EqualFold(colKey, "PRI"),
			Extra:         extra,
			Comment:       comment,
			Ordinal:       ordinal,
		}
		if def.Valid {
			info.Default = &def.String
		}
		out = append(out, info)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func mysqlAlterSQL(qualified string, ops []driver.ColumnOp) (string, error) {
	if len(ops) == 0 {
		return "", fmt.Errorf("no schema changes")
	}
	clauses := make([]string, 0, len(ops))
	for i, op := range ops {
		clause, err := mysqlAlterClause(op)
		if err != nil {
			return "", fmt.Errorf("op %d: %w", i, err)
		}
		clauses = append(clauses, clause)
	}
	return "ALTER TABLE " + qualified + " " + strings.Join(clauses, ", "), nil
}

func mysqlAlterClause(op driver.ColumnOp) (string, error) {
	switch op.Kind {
	case driver.ColumnOpDrop:
		if op.Name == "" {
			return "", fmt.Errorf("drop requires a column name")
		}
		return "DROP COLUMN " + sqlutil.QuoteMySQL(op.Name), nil
	case driver.ColumnOpAdd:
		if op.Column == nil {
			return "", fmt.Errorf("add requires a column definition")
		}
		def, err := mysqlColumnDef(*op.Column)
		if err != nil {
			return "", err
		}
		return "ADD COLUMN " + def + mysqlPosition(op.After), nil
	case driver.ColumnOpModify, driver.ColumnOpReorder:
		if op.Column == nil {
			return "", fmt.Errorf("%s requires a column definition", op.Kind)
		}
		oldName := op.Name
		if oldName == "" {
			oldName = op.Column.Name
		}
		if oldName == "" {
			return "", fmt.Errorf("%s requires a column name", op.Kind)
		}
		def, err := mysqlColumnDef(*op.Column)
		if err != nil {
			return "", err
		}
		return "CHANGE COLUMN " + sqlutil.QuoteMySQL(oldName) + " " + def + mysqlPosition(op.After), nil
	default:
		return "", fmt.Errorf("unknown alter op %q", op.Kind)
	}
}

func mysqlColumnDef(col driver.ColumnInfo) (string, error) {
	if col.Name == "" {
		return "", fmt.Errorf("column name is required")
	}
	if col.Type == "" {
		return "", fmt.Errorf("column type is required")
	}
	var b strings.Builder
	b.WriteString(sqlutil.QuoteMySQL(col.Name))
	b.WriteByte(' ')
	b.WriteString(col.Type)
	if col.Nullable {
		b.WriteString(" NULL")
	} else {
		b.WriteString(" NOT NULL")
	}
	if col.Default != nil {
		b.WriteString(" DEFAULT ")
		b.WriteString(sqlutil.FormatDefault(*col.Default))
	}
	if col.AutoIncrement {
		b.WriteString(" AUTO_INCREMENT")
	}
	if col.Comment != "" {
		b.WriteString(" COMMENT ")
		b.WriteString(sqlutil.QuoteString(col.Comment))
	}
	return b.String(), nil
}

func mysqlPosition(after *string) string {
	if after == nil {
		return ""
	}
	if *after == "" {
		return " FIRST"
	}
	return " AFTER " + sqlutil.QuoteMySQL(*after)
}
