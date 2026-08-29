package postgres

import (
	"context"
	"fmt"
	"maps"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"lotustab/internal/core/driver"
	"lotustab/internal/drivers/sqlutil"
)

const (
	id          = "postgres"
	defaultHost = "localhost"
	defaultPort = "5432"
)

// Driver implements driver.Driver for PostgreSQL.
type Driver struct{}

// New returns a PostgreSQL driver.
func New() *Driver {
	return &Driver{}
}

func (d *Driver) ID() string { return id }

func (d *Driver) DisplayName() string { return "PostgreSQL" }

func (d *Driver) ConnectionSpec() []driver.Field {
	return []driver.Field{
		{Key: "host", Label: "Host", Type: driver.FieldText, Required: true, Default: defaultHost, Placeholder: "localhost", Group: "host_port"},
		{Key: "port", Label: "Port", Type: driver.FieldNumber, Required: true, Default: defaultPort, Placeholder: defaultPort, Group: "host_port"},
		{Key: "user", Label: "User", Type: driver.FieldText, Required: true, Placeholder: "postgres", Group: "auth"},
		{Key: "password", Label: "Password", Type: driver.FieldPassword, Secret: true, Placeholder: "stored in keychain", Group: "auth"},
		{Key: "database", Label: "Database", Type: driver.FieldText, Placeholder: "postgres", Fetchable: true},
		{
			Key:     "sslmode",
			Label:   "SSL mode",
			Type:    driver.FieldSelect,
			Default: "prefer",
			Options: []driver.Option{
				{Label: "disable", Value: "disable"},
				{Label: "prefer", Value: "prefer"},
				{Label: "require", Value: "require"},
				{Label: "verify-full", Value: "verify-full"},
			},
		},
	}
}

func (d *Driver) Open(ctx context.Context, cfg driver.ConnConfig) (driver.Conn, error) {
	pool, err := openPool(ctx, cfg)
	if err != nil {
		return nil, err
	}
	database := cfg.Param("database", "")
	if err := pool.QueryRow(ctx, "SELECT current_database()").Scan(&database); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres current database: %w", err)
	}
	params := maps.Clone(cfg.Params)
	if params == nil {
		params = map[string]string{}
	}
	return &conn{
		pool:     pool,
		cfg:      driver.ConnConfig{Params: params, Password: cfg.Password},
		database: database,
	}, nil
}

func openPool(ctx context.Context, cfg driver.ConnConfig) (*pgxpool.Pool, error) {
	dsn, err := dsnFrom(cfg)
	if err != nil {
		return nil, err
	}
	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres config: %w", err)
	}
	poolCfg.MaxConns = 4

	connectCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("postgres open: %w", err)
	}
	if err := pool.Ping(connectCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	return pool, nil
}

func dsnFrom(cfg driver.ConnConfig) (string, error) {
	host := cfg.Param("host", defaultHost)
	port := cfg.Param("port", defaultPort)
	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		return "", fmt.Errorf("invalid port: %s", port)
	}
	u := url.URL{
		Scheme: "postgres",
		Host:   net.JoinHostPort(host, port),
		Path:   "/" + cfg.Param("database", ""),
	}
	user := cfg.Param("user", "")
	if user != "" {
		u.User = url.UserPassword(user, cfg.Password)
	}
	q := url.Values{}
	q.Set("sslmode", cfg.Param("sslmode", "prefer"))
	u.RawQuery = q.Encode()
	return u.String(), nil
}

type conn struct {
	pool     *pgxpool.Pool
	cfg      driver.ConnConfig
	database string
}

func (c *conn) Ping(ctx context.Context) error {
	return c.pool.Ping(ctx)
}

func (c *conn) Query(ctx context.Context, sql string, limit, offset int) (*driver.QueryResult, error) {
	page := driver.PrepareQueryPage(sql, limit, offset)
	result, err := queryPool(ctx, c.pool, page.MaxRows, page.SQL)
	if err != nil {
		return nil, err
	}
	return driver.FinishQueryPage(result, page), nil
}

func (c *conn) Databases(ctx context.Context) ([]string, error) {
	rows, err := c.pool.Query(ctx, `SELECT datname FROM pg_database WHERE datallowconn ORDER BY 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []string{}
	}
	return out, nil
}

func (c *conn) ListTables(ctx context.Context, database string) ([]driver.TableInfo, error) {
	pool, cleanup, err := c.poolFor(ctx, database)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	rows, err := pool.Query(ctx, `
		SELECT table_schema, table_name, table_type
		FROM information_schema.tables
		WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
		ORDER BY table_schema, table_name`)
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
		kind := driver.ObjectTable
		if strings.EqualFold(tableType, "VIEW") {
			kind = driver.ObjectView
		}
		out = append(out, driver.TableInfo{Schema: schema, Name: name, Type: kind})
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
	pool, cleanup, err := c.poolFor(ctx, req.Database)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	qualified := postgresQualified(req.Schema, req.Name)
	start := time.Now()

	var total int64
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM "+qualified).Scan(&total); err != nil {
		return nil, err
	}

	selectSQL := "SELECT * FROM " + qualified + sqlutil.OrderClause(sqlutil.QuotePostgres, req.OrderBy, req.OrderDir) + " LIMIT $1 OFFSET $2"
	result, err := queryPool(ctx, pool, req.Limit, selectSQL, req.Limit, req.Offset)
	if err != nil {
		return nil, err
	}
	out := driver.BrowseFromQuery(result, total)
	out.DurationMs = time.Since(start).Milliseconds()
	return out, nil
}

func (c *conn) Close() error {
	c.pool.Close()
	return nil
}

func (c *conn) poolFor(ctx context.Context, database string) (*pgxpool.Pool, func(), error) {
	database = strings.TrimSpace(database)
	if database == "" || database == c.database {
		return c.pool, func() {}, nil
	}
	params := maps.Clone(c.cfg.Params)
	if params == nil {
		params = map[string]string{}
	}
	params["database"] = database
	pool, err := openPool(ctx, driver.ConnConfig{Params: params, Password: c.cfg.Password})
	if err != nil {
		return nil, nil, err
	}
	return pool, pool.Close, nil
}

func postgresQualified(schema, name string) string {
	if schema == "" {
		return sqlutil.QuotePostgres(name)
	}
	return sqlutil.QuotePostgres(schema) + "." + sqlutil.QuotePostgres(name)
}

func queryPool(ctx context.Context, pool *pgxpool.Pool, maxRows int, sqlText string, args ...any) (*driver.QueryResult, error) {
	start := time.Now()
	rows, err := pool.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fields := rows.FieldDescriptions()
	columns := make([]driver.Column, len(fields))
	for i, f := range fields {
		columns[i] = driver.Column{
			Name: f.Name,
			Type: typeName(f.DataTypeOID),
		}
	}

	out := make([][]any, 0)
	truncated := false
	for rows.Next() {
		if len(out) >= maxRows {
			truncated = true
			break
		}
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}
		row := make([]any, len(values))
		for i, v := range values {
			row[i] = driver.NormalizeValue(v)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	tag := rows.CommandTag()
	affected := tag.RowsAffected()
	if affected == 0 && len(out) > 0 {
		affected = int64(len(out))
	}

	return &driver.QueryResult{
		Columns:      columns,
		Rows:         out,
		RowsAffected: affected,
		DurationMs:   time.Since(start).Milliseconds(),
		Truncated:    truncated,
	}, nil
}

func typeName(oid uint32) string {
	switch oid {
	case 16:
		return "bool"
	case 20:
		return "int8"
	case 21:
		return "int2"
	case 23:
		return "int4"
	case 25:
		return "text"
	case 114:
		return "json"
	case 700:
		return "float4"
	case 701:
		return "float8"
	case 1043:
		return "varchar"
	case 1082:
		return "date"
	case 1114:
		return "timestamp"
	case 1184:
		return "timestamptz"
	case 1700:
		return "numeric"
	case 2950:
		return "uuid"
	case 3802:
		return "jsonb"
	default:
		return fmt.Sprintf("oid:%d", oid)
	}
}
