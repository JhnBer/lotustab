package postgres

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"lotustab/internal/core/driver"
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
	return &conn{pool: pool}, nil
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
	pool *pgxpool.Pool
}

func (c *conn) Ping(ctx context.Context) error {
	return c.pool.Ping(ctx)
}

func (c *conn) Query(ctx context.Context, sql string) (*driver.QueryResult, error) {
	const maxRows = 5000
	start := time.Now()
	rows, err := c.pool.Query(ctx, sql)
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

func (c *conn) Close() error {
	c.pool.Close()
	return nil
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
