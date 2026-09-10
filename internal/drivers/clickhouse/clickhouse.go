package clickhouse

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"

	"lotustab/internal/core/driver"
	"lotustab/internal/drivers/sqlutil"
)

const (
	id          = "clickhouse"
	defaultHost = "localhost"
	defaultPort = "9000"
	defaultUser = "default"
)

// Driver implements driver.Driver for ClickHouse.
type Driver struct{}

// New returns a ClickHouse driver.
func New() *Driver {
	return &Driver{}
}

func (d *Driver) ID() string { return id }

func (d *Driver) DisplayName() string { return "ClickHouse" }

func (d *Driver) ConnectionSpec() []driver.Field {
	return []driver.Field{
		{Key: "host", Label: "Host", Type: driver.FieldText, Required: true, Default: defaultHost, Placeholder: "localhost", Group: "host_port"},
		{Key: "port", Label: "Port", Type: driver.FieldNumber, Required: true, Default: defaultPort, Placeholder: defaultPort, Group: "host_port"},
		{Key: "user", Label: "User", Type: driver.FieldText, Required: true, Default: defaultUser, Placeholder: defaultUser, Group: "auth"},
		{Key: "password", Label: "Password", Type: driver.FieldPassword, Secret: true, Placeholder: "stored in keychain", Group: "auth"},
		{Key: "database", Label: "Database", Type: driver.FieldText, Placeholder: "default", Fetchable: true},
		{Key: "secure", Label: "TLS", Type: driver.FieldCheckbox, Default: "false"},
	}
}

func (d *Driver) Open(ctx context.Context, cfg driver.ConnConfig) (driver.Conn, error) {
	opts, err := clickhouseOptions(cfg)
	if err != nil {
		return nil, err
	}
	db := clickhouse.OpenDB(opts)
	db.SetMaxOpenConns(4)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(4 * time.Minute)

	connectCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := db.PingContext(connectCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("clickhouse ping: %w", err)
	}
	return &conn{db: db, database: strings.TrimSpace(cfg.Param("database", ""))}, nil
}

func clickhouseOptions(cfg driver.ConnConfig) (*clickhouse.Options, error) {
	port := cfg.Param("port", defaultPort)
	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		return nil, fmt.Errorf("invalid port: %s", port)
	}
	opts := &clickhouse.Options{
		Addr: []string{net.JoinHostPort(cfg.Param("host", defaultHost), port)},
		Auth: clickhouse.Auth{
			Database: cfg.Param("database", ""),
			Username: cfg.Param("user", defaultUser),
			Password: cfg.Password,
		},
		DialTimeout: 15 * time.Second,
	}
	secure, err := strconv.ParseBool(cfg.Param("secure", "false"))
	if err != nil {
		return nil, fmt.Errorf("invalid secure: %s", cfg.Param("secure", "false"))
	}
	if secure {
		opts.TLS = &tls.Config{}
	}
	if cfg.Dial != nil {
		opts.DialContext = func(ctx context.Context, addr string) (net.Conn, error) {
			return cfg.Dial(ctx, "tcp", addr)
		}
	}
	return opts, nil
}

type conn struct {
	db       *sql.DB
	database string
}

func (c *conn) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

func (c *conn) withSession(ctx context.Context, fn func(*sql.Conn) error) error {
	session, err := c.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer session.Close()
	return fn(session)
}

func (c *conn) Query(ctx context.Context, sqlText, database string, limit, offset int) (*driver.QueryResult, error) {
	page := driver.PrepareQueryPage(sqlText, limit, offset)
	database = strings.TrimSpace(database)
	var result *driver.QueryResult
	err := c.withSession(ctx, func(session *sql.Conn) error {
		if database != "" && database != c.database {
			if _, err := session.ExecContext(ctx, clickhouseUse(database)); err != nil {
				return err
			}
			defer func() {
				if c.database != "" {
					_, _ = session.ExecContext(ctx, clickhouseUse(c.database))
				}
			}()
		}
		var err error
		result, err = sqlutil.Query(ctx, session, page.SQL, page.MaxRows)
		return err
	})
	if err != nil {
		return nil, err
	}
	return driver.FinishQueryPage(result, page), nil
}

func clickhouseUse(database string) string {
	return "USE " + sqlutil.QuoteClickHouse(database)
}

func (c *conn) Databases(ctx context.Context) ([]string, error) {
	var names []string
	err := c.withSession(ctx, func(session *sql.Conn) error {
		var err error
		names, err = sqlutil.ListStrings(ctx, session, "SELECT name FROM system.databases ORDER BY name")
		return err
	})
	return names, err
}

func (c *conn) ListTables(ctx context.Context, database string) ([]driver.TableInfo, error) {
	database = strings.TrimSpace(database)
	var out []driver.TableInfo
	err := c.withSession(ctx, func(session *sql.Conn) error {
		var (
			rows *sql.Rows
			err  error
		)
		if database == "" {
			rows, err = session.QueryContext(ctx, `
			SELECT database, name, engine
			FROM system.tables
			WHERE database = currentDatabase()
			ORDER BY name`)
		} else {
			rows, err = session.QueryContext(ctx, `
			SELECT database, name, engine
			FROM system.tables
			WHERE database = ?
			ORDER BY name`, database)
		}
		if err != nil {
			return err
		}
		defer rows.Close()

		out = make([]driver.TableInfo, 0)
		for rows.Next() {
			var dbName, name, engine string
			if err := rows.Scan(&dbName, &name, &engine); err != nil {
				return err
			}
			out = append(out, driver.TableInfo{
				Database: dbName,
				Schema:   dbName,
				Name:     name,
				Type:     clickhouseObjectType(engine),
			})
		}
		return rows.Err()
	})
	return out, err
}

func (c *conn) SearchTables(ctx context.Context, database, query string, limit int) ([]driver.TableInfo, error) {
	query = strings.TrimSpace(query)
	limit = driver.ClampSearchLimit(limit)
	if query == "" {
		return []driver.TableInfo{}, nil
	}
	pattern := sqlutil.LikeContains(query)
	database = strings.TrimSpace(database)
	var out []driver.TableInfo
	err := c.withSession(ctx, func(session *sql.Conn) error {
		var (
			rows *sql.Rows
			err  error
		)
		if database == "" {
			rows, err = session.QueryContext(ctx, `
			SELECT database, name, engine
			FROM system.tables
			WHERE name ILIKE ? ESCAPE '\\'
			ORDER BY database, name
			LIMIT ?`, pattern, limit)
		} else {
			rows, err = session.QueryContext(ctx, `
			SELECT database, name, engine
			FROM system.tables
			WHERE database = ?
			  AND name ILIKE ? ESCAPE '\\'
			ORDER BY name
			LIMIT ?`, database, pattern, limit)
		}
		if err != nil {
			return err
		}
		defer rows.Close()

		out = make([]driver.TableInfo, 0)
		for rows.Next() {
			var dbName, name, engine string
			if err := rows.Scan(&dbName, &name, &engine); err != nil {
				return err
			}
			out = append(out, driver.TableInfo{
				Database: dbName,
				Schema:   dbName,
				Name:     name,
				Type:     clickhouseObjectType(engine),
			})
		}
		return rows.Err()
	})
	return out, err
}

func (c *conn) BrowseTable(ctx context.Context, req driver.TableBrowseRequest) (*driver.TableBrowseResult, error) {
	req = driver.ClampTableBrowse(req)
	if req.Name == "" {
		return nil, fmt.Errorf("table name is required")
	}
	dbName := clickhouseDBName(req.Database, req.Schema)
	qualified := clickhouseQualified(dbName, req.Name)
	where, args, err := sqlutil.WhereClause(sqlutil.QuoteClickHouse, sqlutil.Question, req.Filters)
	if err != nil {
		return nil, err
	}

	var out *driver.TableBrowseResult
	err = c.withSession(ctx, func(session *sql.Conn) error {
		start := time.Now()
		total, err := sqlutil.Count(ctx, session, "SELECT COUNT(*) FROM "+qualified+where, args...)
		if err != nil {
			return err
		}

		selectSQL := "SELECT * FROM " + qualified + where + sqlutil.OrderClause(sqlutil.QuoteClickHouse, req.OrderBy, req.OrderDir) + " LIMIT ? OFFSET ?"
		selectArgs := append(append([]any{}, args...), req.Limit, req.Offset)
		result, err := sqlutil.QueryContext(ctx, session, req.Limit, selectSQL, selectArgs...)
		if err != nil {
			return err
		}
		out = driver.BrowseFromQuery(result, total)
		out.DurationMs = time.Since(start).Milliseconds()
		pk, err := clickhousePrimaryKey(ctx, session, dbName, req.Name)
		if err != nil {
			return err
		}
		out.PrimaryKey = pk
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *conn) ApplyEdits(ctx context.Context, req driver.ApplyEditsRequest) error {
	req = driver.ClampApplyEdits(req)
	if req.Name == "" {
		return fmt.Errorf("table name is required")
	}
	dbName := clickhouseDBName(req.Database, req.Schema)
	return c.withSession(ctx, func(session *sql.Conn) error {
		pk, err := clickhousePrimaryKey(ctx, session, dbName, req.Name)
		if err != nil {
			return err
		}
		if len(pk) == 0 {
			return fmt.Errorf("table has no sorting key to identify rows")
		}

		qualified := clickhouseQualified(dbName, req.Name)
		applied := 0
		for i, row := range req.Rows {
			var (
				q   string
				err error
			)
			switch {
			case row.Deleted:
				q, err = clickhouseDeleteSQL(qualified, pk, row.Key)
			case row.Inserted:
				q, err = clickhouseInsertSQL(qualified, row.Cells)
			default:
				if len(row.Cells) == 0 {
					continue
				}
				q, err = clickhouseMutationSQL(qualified, row.Cells, pk, row.Key)
			}
			if err != nil {
				return err
			}
			if _, err := session.ExecContext(ctx, q); err != nil {
				return fmt.Errorf("row %d: %w", i, err)
			}
			applied++
		}
		if applied == 0 {
			return fmt.Errorf("no edits")
		}
		return nil
	})
}

func clickhousePrimaryKey(ctx context.Context, q sqlutil.Querier, database, name string) ([]string, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if database == "" {
		rows, err = q.QueryContext(ctx, `
			SELECT name
			FROM system.columns
			WHERE database = currentDatabase()
			  AND table = ?
			  AND is_in_primary_key = 1
			ORDER BY position`, name)
	} else {
		rows, err = q.QueryContext(ctx, `
			SELECT name
			FROM system.columns
			WHERE database = ?
			  AND table = ?
			  AND is_in_primary_key = 1
			ORDER BY position`, database, name)
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

func clickhouseMutationSQL(qualified string, cells map[string]any, pk []string, key map[string]any) (string, error) {
	if strings.TrimSpace(qualified) == "" {
		return "", fmt.Errorf("table name is required")
	}
	if len(pk) == 0 {
		return "", fmt.Errorf("table has no sorting key to identify rows")
	}
	if len(cells) == 0 {
		return "", fmt.Errorf("no columns to update")
	}

	setCols := make([]string, 0, len(cells))
	for col := range cells {
		if strings.TrimSpace(col) == "" {
			return "", fmt.Errorf("column name is required")
		}
		setCols = append(setCols, col)
	}
	sort.Strings(setCols)

	var b strings.Builder
	b.WriteString("ALTER TABLE ")
	b.WriteString(qualified)
	b.WriteString(" UPDATE ")
	for i, col := range setCols {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(sqlutil.QuoteClickHouse(col))
		b.WriteString(" = ")
		b.WriteString(clickhouseLiteral(cells[col]))
	}
	b.WriteString(" WHERE ")
	where, err := clickhouseWhere(pk, key)
	if err != nil {
		return "", err
	}
	b.WriteString(where)
	b.WriteString(" SETTINGS mutations_sync = 1")
	return b.String(), nil
}

func clickhouseDeleteSQL(qualified string, pk []string, key map[string]any) (string, error) {
	if strings.TrimSpace(qualified) == "" {
		return "", fmt.Errorf("table name is required")
	}
	where, err := clickhouseWhere(pk, key)
	if err != nil {
		return "", err
	}
	return "ALTER TABLE " + qualified + " DELETE WHERE " + where + " SETTINGS mutations_sync = 1", nil
}

func clickhouseInsertSQL(qualified string, cells map[string]any) (string, error) {
	if strings.TrimSpace(qualified) == "" {
		return "", fmt.Errorf("table name is required")
	}
	if len(cells) == 0 {
		return "", fmt.Errorf("no columns to insert")
	}
	cols := make([]string, 0, len(cells))
	for col := range cells {
		if strings.TrimSpace(col) == "" {
			return "", fmt.Errorf("column name is required")
		}
		cols = append(cols, col)
	}
	sort.Strings(cols)

	var b strings.Builder
	b.WriteString("INSERT INTO ")
	b.WriteString(qualified)
	b.WriteString(" (")
	for i, col := range cols {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(sqlutil.QuoteClickHouse(col))
	}
	b.WriteString(") VALUES (")
	for i, col := range cols {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(clickhouseLiteral(cells[col]))
	}
	b.WriteByte(')')
	return b.String(), nil
}

func clickhouseWhere(pk []string, key map[string]any) (string, error) {
	if len(pk) == 0 {
		return "", fmt.Errorf("table has no sorting key to identify rows")
	}
	var b strings.Builder
	for i, col := range pk {
		if strings.TrimSpace(col) == "" {
			return "", fmt.Errorf("key column name is required")
		}
		v, ok := key[col]
		if !ok {
			return "", fmt.Errorf("missing key column %s", col)
		}
		if i > 0 {
			b.WriteString(" AND ")
		}
		b.WriteString(sqlutil.QuoteClickHouse(col))
		b.WriteString(" = ")
		b.WriteString(clickhouseLiteral(v))
	}
	return b.String(), nil
}

func clickhouseLiteral(v any) string {
	if v == nil {
		return "NULL"
	}
	switch x := v.(type) {
	case string:
		return sqlutil.QuoteString(x)
	case []byte:
		return sqlutil.QuoteString(string(x))
	case bool:
		if x {
			return "1"
		}
		return "0"
	case int:
		return strconv.Itoa(x)
	case int8:
		return strconv.FormatInt(int64(x), 10)
	case int16:
		return strconv.FormatInt(int64(x), 10)
	case int32:
		return strconv.FormatInt(int64(x), 10)
	case int64:
		return strconv.FormatInt(x, 10)
	case uint:
		return strconv.FormatUint(uint64(x), 10)
	case uint8:
		return strconv.FormatUint(uint64(x), 10)
	case uint16:
		return strconv.FormatUint(uint64(x), 10)
	case uint32:
		return strconv.FormatUint(uint64(x), 10)
	case uint64:
		return strconv.FormatUint(x, 10)
	case float32:
		return strconv.FormatFloat(float64(x), 'g', -1, 32)
	case float64:
		return strconv.FormatFloat(x, 'g', -1, 64)
	default:
		return sqlutil.QuoteString(fmt.Sprint(x))
	}
}

func clickhouseDBName(database, schema string) string {
	if strings.TrimSpace(database) != "" {
		return strings.TrimSpace(database)
	}
	return strings.TrimSpace(schema)
}

func clickhouseQualified(database, name string) string {
	if database == "" {
		return sqlutil.QuoteClickHouse(name)
	}
	return sqlutil.QuoteClickHouse(database) + "." + sqlutil.QuoteClickHouse(name)
}

func clickhouseObjectType(engine string) string {
	if strings.Contains(strings.ToLower(engine), "view") {
		return driver.ObjectView
	}
	return driver.ObjectTable
}

func (c *conn) Close() error {
	return c.db.Close()
}
