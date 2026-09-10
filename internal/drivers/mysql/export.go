package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"lotustab/internal/core/driver"
	"lotustab/internal/drivers/sqlutil"
)

func (c *conn) StreamTable(ctx context.Context, req driver.StreamTableRequest, fn func(cols []driver.Column, rows [][]any) error) error {
	req = driver.ClampStreamTable(req)
	if req.Name == "" {
		return fmt.Errorf("table name is required")
	}
	if fn == nil {
		return fmt.Errorf("stream callback is required")
	}
	dbName := req.Database
	if dbName == "" {
		dbName = req.Schema
	}
	qualified := mysqlQualified(dbName, req.Name)
	query := sqlutil.SelectStar(qualified, req.WhereSQL)
	log.Printf("export stream: %s", query)

	return c.withSession(ctx, func(session *sql.Conn) error {
		if _, err := session.ExecContext(ctx, "START TRANSACTION WITH CONSISTENT SNAPSHOT"); err != nil {
			return err
		}
		committed := false
		defer func() {
			if !committed {
				_, _ = session.ExecContext(context.Background(), "ROLLBACK")
			}
		}()
		if err := sqlutil.StreamQuery(ctx, session, query, nil, fn); err != nil {
			return err
		}
		if _, err := session.ExecContext(ctx, "COMMIT"); err != nil {
			return err
		}
		committed = true
		return nil
	})
}

func (c *conn) TableDDL(ctx context.Context, ref driver.TableRef) (string, error) {
	ref = driver.ClampTableRef(ref)
	if ref.Name == "" {
		return "", fmt.Errorf("table name is required")
	}
	dbName := ref.Database
	if dbName == "" {
		dbName = ref.Schema
	}
	qualified := mysqlQualified(dbName, ref.Name)
	var ddl string
	err := c.withSession(ctx, func(session *sql.Conn) error {
		var table string
		return session.QueryRowContext(ctx, "SHOW CREATE TABLE "+qualified).Scan(&table, &ddl)
	})
	if err != nil {
		return "", err
	}
	if ddl == "" {
		return "", fmt.Errorf("empty CREATE TABLE")
	}
	if ddl[len(ddl)-1] != ';' {
		ddl += ";"
	}
	return ddl, nil
}
