package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"

	"lotustab/internal/core/driver"
	"lotustab/internal/drivers/sqlutil"
)

const (
	id          = "mysql"
	defaultHost = "localhost"
	defaultPort = "3306"
)

// Driver implements driver.Driver for MySQL / MariaDB.
type Driver struct{}

// New returns a MySQL driver.
func New() *Driver {
	return &Driver{}
}

func (d *Driver) ID() string { return id }

func (d *Driver) DisplayName() string { return "MySQL" }

func (d *Driver) ConnectionSpec() []driver.Field {
	return []driver.Field{
		{Key: "host", Label: "Host", Type: driver.FieldText, Required: true, Default: defaultHost, Placeholder: "localhost", Group: "host_port"},
		{Key: "port", Label: "Port", Type: driver.FieldNumber, Required: true, Default: defaultPort, Placeholder: defaultPort, Group: "host_port"},
		{Key: "user", Label: "User", Type: driver.FieldText, Required: true, Placeholder: "root", Group: "auth"},
		{Key: "password", Label: "Password", Type: driver.FieldPassword, Secret: true, Placeholder: "stored in keychain", Group: "auth"},
		{Key: "database", Label: "Database", Type: driver.FieldText, Placeholder: "optional", Fetchable: true},
		{
			Key:     "tls",
			Label:   "TLS",
			Type:    driver.FieldSelect,
			Default: "false",
			Options: []driver.Option{
				{Label: "disabled", Value: "false"},
				{Label: "enabled", Value: "true"},
				{Label: "skip-verify", Value: "skip-verify"},
			},
		},
	}
}

func (d *Driver) Open(ctx context.Context, cfg driver.ConnConfig) (driver.Conn, error) {
	mc, err := mysqlConfig(cfg)
	if err != nil {
		return nil, err
	}
	connector, err := mysqldriver.NewConnector(mc)
	if err != nil {
		return nil, fmt.Errorf("mysql open: %w", err)
	}
	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(4)
	db.SetConnMaxLifetime(30 * time.Minute)

	connectCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := db.PingContext(connectCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("mysql ping: %w", err)
	}
	return &conn{db: db}, nil
}

func mysqlConfig(cfg driver.ConnConfig) (*mysqldriver.Config, error) {
	port := cfg.Param("port", defaultPort)
	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		return nil, fmt.Errorf("invalid port: %s", port)
	}
	mc := mysqldriver.NewConfig()
	mc.User = cfg.Param("user", "")
	mc.Passwd = cfg.Password
	mc.Net = "tcp"
	mc.Addr = net.JoinHostPort(cfg.Param("host", defaultHost), port)
	mc.DBName = cfg.Param("database", "")
	mc.Params = map[string]string{}
	tls := cfg.Param("tls", "false")
	if tls != "" && tls != "false" {
		mc.TLSConfig = tls
	}
	mc.ParseTime = true
	mc.Timeout = 15 * time.Second
	mc.ReadTimeout = 60 * time.Second
	mc.WriteTimeout = 30 * time.Second
	if cfg.Dial != nil {
		mc.DialFunc = cfg.Dial
	}
	return mc, nil
}

type conn struct {
	db *sql.DB
}

func (c *conn) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

func (c *conn) Query(ctx context.Context, sqlText string, limit, offset int) (*driver.QueryResult, error) {
	page := driver.PrepareQueryPage(sqlText, limit, offset)
	result, err := sqlutil.Query(ctx, c.db, page.SQL, page.MaxRows)
	if err != nil {
		return nil, err
	}
	return driver.FinishQueryPage(result, page), nil
}

func (c *conn) Databases(ctx context.Context) ([]string, error) {
	return sqlutil.ListStrings(ctx, c.db, "SHOW DATABASES")
}

func (c *conn) ListTables(ctx context.Context, database string) ([]driver.TableInfo, error) {
	database = strings.TrimSpace(database)
	var (
		rows *sql.Rows
		err  error
	)
	if database == "" {
		rows, err = c.db.QueryContext(ctx, `
			SELECT table_schema, table_name, table_type
			FROM information_schema.tables
			WHERE table_schema = DATABASE()
			ORDER BY table_name`)
	} else {
		rows, err = c.db.QueryContext(ctx, `
			SELECT table_schema, table_name, table_type
			FROM information_schema.tables
			WHERE table_schema = ?
			ORDER BY table_name`, database)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]driver.TableInfo, 0)
	for rows.Next() {
		var schema, name, tableType string
		if err := rows.Scan(&schema, &name, &tableType); err != nil {
			return nil, err
		}
		out = append(out, driver.TableInfo{
			Schema: schema,
			Name:   name,
			Type:   mysqlObjectType(tableType),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *conn) BrowseTable(ctx context.Context, req driver.TableBrowseRequest) (*driver.TableBrowseResult, error) {
	req = driver.ClampTableBrowse(req)
	if req.Name == "" {
		return nil, fmt.Errorf("table name is required")
	}
	dbName := req.Database
	if dbName == "" {
		dbName = req.Schema
	}
	qualified := mysqlQualified(dbName, req.Name)
	where, args, err := sqlutil.WhereClause(sqlutil.QuoteMySQL, sqlutil.Question, req.Filters)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	total, err := sqlutil.Count(ctx, c.db, "SELECT COUNT(*) FROM "+qualified+where, args...)
	if err != nil {
		return nil, err
	}

	selectSQL := "SELECT * FROM " + qualified + where + sqlutil.OrderClause(sqlutil.QuoteMySQL, req.OrderBy, req.OrderDir) + " LIMIT ? OFFSET ?"
	selectArgs := append(append([]any{}, args...), req.Limit, req.Offset)
	result, err := sqlutil.QueryContext(ctx, c.db, req.Limit, selectSQL, selectArgs...)
	if err != nil {
		return nil, err
	}
	out := driver.BrowseFromQuery(result, total)
	out.DurationMs = time.Since(start).Milliseconds()
	pk, err := mysqlPrimaryKey(ctx, c.db, dbName, req.Name)
	if err != nil {
		return nil, err
	}
	out.PrimaryKey = pk
	return out, nil
}

func (c *conn) ApplyEdits(ctx context.Context, req driver.ApplyEditsRequest) error {
	req = driver.ClampApplyEdits(req)
	if req.Name == "" {
		return fmt.Errorf("table name is required")
	}
	dbName := req.Database
	if dbName == "" {
		dbName = req.Schema
	}
	pk, err := mysqlPrimaryKey(ctx, c.db, dbName, req.Name)
	if err != nil {
		return err
	}
	if len(pk) == 0 {
		return fmt.Errorf("table has no primary key")
	}

	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	qualified := mysqlQualified(dbName, req.Name)
	err = sqlutil.ApplyRowEdits(ctx, func(ctx context.Context, q string, args ...any) (int64, error) {
		res, execErr := tx.ExecContext(ctx, q, args...)
		if execErr != nil {
			return 0, execErr
		}
		return res.RowsAffected()
	}, sqlutil.QuoteMySQL, sqlutil.Question, qualified, pk, req.Rows)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func mysqlPrimaryKey(ctx context.Context, db *sql.DB, database, name string) ([]string, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if database == "" {
		rows, err = db.QueryContext(ctx, `
			SELECT COLUMN_NAME
			FROM information_schema.KEY_COLUMN_USAGE
			WHERE TABLE_SCHEMA = DATABASE()
			  AND TABLE_NAME = ?
			  AND CONSTRAINT_NAME = 'PRIMARY'
			ORDER BY ORDINAL_POSITION`, name)
	} else {
		rows, err = db.QueryContext(ctx, `
			SELECT COLUMN_NAME
			FROM information_schema.KEY_COLUMN_USAGE
			WHERE TABLE_SCHEMA = ?
			  AND TABLE_NAME = ?
			  AND CONSTRAINT_NAME = 'PRIMARY'
			ORDER BY ORDINAL_POSITION`, database, name)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]string, 0)
	for rows.Next() {
		var col string
		if err := rows.Scan(&col); err != nil {
			return nil, err
		}
		out = append(out, col)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func mysqlQualified(database, name string) string {
	if database == "" {
		return sqlutil.QuoteMySQL(name)
	}
	return sqlutil.QuoteMySQL(database) + "." + sqlutil.QuoteMySQL(name)
}

func mysqlObjectType(tableType string) string {
	switch strings.ToUpper(strings.TrimSpace(tableType)) {
	case "VIEW", "SYSTEM VIEW":
		return driver.ObjectView
	default:
		return driver.ObjectTable
	}
}

func (c *conn) Close() error {
	return c.db.Close()
}
