package clickhouse

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
	dbName := clickhouseDBName(req.Database, req.Schema)
	qualified := clickhouseQualified(dbName, req.Name)
	query := sqlutil.SelectStar(qualified, req.WhereSQL)
	log.Printf("export stream: %s", query)

	return c.withSession(ctx, func(session *sql.Conn) error {
		return sqlutil.StreamQuery(ctx, session, query, nil, fn)
	})
}

func (c *conn) TableDDL(ctx context.Context, ref driver.TableRef) (string, error) {
	ref = driver.ClampTableRef(ref)
	if ref.Name == "" {
		return "", fmt.Errorf("table name is required")
	}
	dbName := clickhouseDBName(ref.Database, ref.Schema)
	qualified := clickhouseQualified(dbName, ref.Name)
	var ddl string
	err := c.withSession(ctx, func(session *sql.Conn) error {
		return session.QueryRowContext(ctx, "SHOW CREATE TABLE "+qualified).Scan(&ddl)
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
