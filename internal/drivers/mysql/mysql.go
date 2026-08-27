package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strconv"
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
		{Key: "host", Label: "Host", Type: driver.FieldText, Required: true, Default: defaultHost, Placeholder: "localhost"},
		{Key: "port", Label: "Port", Type: driver.FieldNumber, Required: true, Default: defaultPort, Placeholder: defaultPort},
		{Key: "user", Label: "User", Type: driver.FieldText, Required: true, Placeholder: "root"},
		{Key: "password", Label: "Password", Type: driver.FieldPassword, Secret: true, Placeholder: "stored in keychain"},
		{Key: "database", Label: "Database", Type: driver.FieldText, Placeholder: "optional"},
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
	dsn, err := dsnFrom(cfg)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("mysql open: %w", err)
	}
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

func dsnFrom(cfg driver.ConnConfig) (string, error) {
	port := cfg.Param("port", defaultPort)
	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		return "", fmt.Errorf("invalid port: %s", port)
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
	return mc.FormatDSN(), nil
}

type conn struct {
	db *sql.DB
}

func (c *conn) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

func (c *conn) Query(ctx context.Context, sqlText string) (*driver.QueryResult, error) {
	return sqlutil.Query(ctx, c.db, sqlText, 5000)
}

func (c *conn) Databases(ctx context.Context) ([]string, error) {
	return sqlutil.ListStrings(ctx, c.db, "SHOW DATABASES")
}

func (c *conn) Close() error {
	return c.db.Close()
}
