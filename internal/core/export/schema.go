package export

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"
)

// Content selects what a dump file contains.
// The empty value is a row dump, so existing callers stay on that path.
type Content string

const (
	ContentData          Content = ""
	ContentSchemaReplace Content = "schemaReplace"
	ContentSchemaAdd     Content = "schemaAdd"
)

const (
	mysqlFKChecksOff = "/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;\n"
	mysqlFKChecksOn  = "/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;\n"
)

// ParseContent accepts the Wails field. Empty means a row dump.
func ParseContent(raw string) (Content, error) {
	switch Content(strings.TrimSpace(raw)) {
	case ContentData:
		return ContentData, nil
	case ContentSchemaAdd, ContentSchemaReplace:
		return Content(strings.TrimSpace(raw)), nil
	default:
		return "", fmt.Errorf("unsupported export content %q", raw)
	}
}

// SchemaOnly reports a Create or Drop schema dump.
func (c Content) SchemaOnly() bool {
	return c == ContentSchemaAdd || c == ContentSchemaReplace
}

func (o Options) schemaOnly() (bool, error) {
	switch o.Content {
	case ContentData:
		return false, nil
	case ContentSchemaAdd, ContentSchemaReplace:
		return true, nil
	default:
		return false, fmt.Errorf("unsupported export content %q", o.Content)
	}
}

func (o Options) mysqlFKGuard() bool {
	return strings.ToLower(strings.TrimSpace(o.Dialect)) == DialectMySQL && o.Content == ContentSchemaReplace
}

// schemaCreateSQL returns the CREATE statement for a schema dump.
// Add rewrites the leading CREATE TABLE into CREATE TABLE IF NOT EXISTS.
func schemaCreateSQL(ddl string, ifNotExists bool) (string, error) {
	trimmed := strings.TrimSpace(ddl)
	const prefix = "CREATE TABLE"
	if len(trimmed) < len(prefix) || !strings.EqualFold(trimmed[:len(prefix)], prefix) {
		return "", fmt.Errorf("schema export requires a CREATE TABLE statement")
	}
	rest := trimmed[len(prefix):]
	if rest == "" || !isSQLSpace(rest[0]) {
		return "", fmt.Errorf("schema export requires a CREATE TABLE statement")
	}
	if ifNotExists && !hasIfNotExists(strings.TrimLeft(rest, " \t\r\n")) {
		trimmed = prefix + " IF NOT EXISTS" + rest
	}
	return ensureSemicolon(trimmed), nil
}

func isSQLSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

func hasIfNotExists(head string) bool {
	const token = "IF NOT EXISTS"
	if len(head) < len(token) || !strings.EqualFold(head[:len(token)], token) {
		return false
	}
	if len(head) == len(token) {
		return true
	}
	return isSQLSpace(head[len(token)])
}

func ensureSemicolon(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, ";") {
		return s
	}
	return s + ";"
}

func writeSchemaTable(w io.Writer, qualified, ddl string, content Content) error {
	stmt, err := schemaCreateSQL(ddl, content == ContentSchemaAdd)
	if err != nil {
		return err
	}
	if content == ContentSchemaReplace {
		if _, err := io.WriteString(w, "DROP TABLE IF EXISTS "+qualified+";\n"); err != nil {
			return err
		}
	}
	_, err = io.WriteString(w, stmt+"\n")
	return err
}

func runSchema(ctx context.Context, opts Options, progress func(Progress)) (*Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	format := strings.ToLower(strings.TrimSpace(opts.Format))
	if format == "" {
		format = FormatSQL
	}
	if format != FormatSQL {
		return nil, fmt.Errorf("schema export requires SQL format")
	}
	dialect := strings.ToLower(strings.TrimSpace(opts.Dialect))
	if dialect == "" {
		dialect = DialectPostgres
	}
	if strings.TrimSpace(opts.Table) == "" {
		return nil, fmt.Errorf("table is required")
	}
	table := qualifiedTable(dialect, opts.Database, opts.Schema, opts.Table)
	base := strings.TrimSpace(opts.FileBase)
	if base == "" {
		base = safeFileBase(opts.Table)
	}

	ch := newChunker(opts.OutputDir, opts.OutputPath, base, fileExt(format, opts.Gzip), opts.Gzip, 0)
	if err := ch.Open(); err != nil {
		return nil, err
	}
	ok := false
	defer func() {
		if !ok {
			ch.Abort()
			if opts.DeleteOnAbort {
				ch.DeleteChunks()
			}
		}
	}()

	start := time.Now()
	if opts.mysqlFKGuard() {
		if _, err := io.WriteString(ch.Writer(), mysqlFKChecksOff); err != nil {
			return nil, err
		}
	}
	if err := writeSchemaTable(ch.Writer(), table, opts.DDL, opts.Content); err != nil {
		return nil, err
	}
	if opts.mysqlFKGuard() {
		if _, err := io.WriteString(ch.Writer(), mysqlFKChecksOn); err != nil {
			return nil, err
		}
	}
	if err := ch.Close(); err != nil {
		return nil, err
	}
	ok = true
	if progress != nil {
		progress(Progress{Chunk: 1, Rows: 0, Bytes: ch.Bytes()})
	}
	return &Result{
		Files:      ch.Files(),
		Rows:       0,
		Bytes:      ch.Bytes(),
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

func runSchemaMulti(ctx context.Context, tables []MultiTable, opts Options, progress func(Progress)) (*Result, error) {
	if len(tables) == 0 {
		return nil, fmt.Errorf("at least one table is required")
	}
	format := strings.ToLower(strings.TrimSpace(opts.Format))
	if format == "" {
		format = FormatSQL
	}
	if format != FormatSQL {
		return nil, fmt.Errorf("schema export requires SQL format")
	}
	dialect := strings.ToLower(strings.TrimSpace(opts.Dialect))
	if dialect == "" {
		dialect = DialectPostgres
	}
	for i, t := range tables {
		if strings.TrimSpace(t.Table) == "" {
			return nil, fmt.Errorf("table is required")
		}
		if _, err := schemaCreateSQL(t.DDL, opts.Content == ContentSchemaAdd); err != nil {
			return nil, fmt.Errorf("table %d: %w", i, err)
		}
	}
	base := strings.TrimSpace(opts.FileBase)
	if base == "" {
		base = safeFileBase(opts.Table)
		if base == "table" {
			base = safeFileBase(tables[0].Table)
		}
	}

	ch := newChunker(opts.OutputDir, opts.OutputPath, base, fileExt(format, opts.Gzip), opts.Gzip, 0)
	if err := ch.Open(); err != nil {
		return nil, err
	}
	ok := false
	defer func() {
		if !ok {
			ch.Abort()
			if opts.DeleteOnAbort {
				ch.DeleteChunks()
			}
		}
	}()

	start := time.Now()
	if opts.mysqlFKGuard() {
		if _, err := io.WriteString(ch.Writer(), mysqlFKChecksOff); err != nil {
			return nil, err
		}
	}
	for i, t := range tables {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if i > 0 {
			if _, err := fmt.Fprint(ch.Writer(), "\n"); err != nil {
				return nil, err
			}
		}
		qualified := qualifiedTable(dialect, t.Database, t.Schema, t.Table)
		if err := writeSchemaTable(ch.Writer(), qualified, t.DDL, opts.Content); err != nil {
			return nil, err
		}
		if progress != nil {
			progress(Progress{Chunk: i + 1, Rows: 0, Bytes: ch.Bytes()})
		}
	}
	if opts.mysqlFKGuard() {
		if _, err := io.WriteString(ch.Writer(), mysqlFKChecksOn); err != nil {
			return nil, err
		}
	}
	if err := ch.Close(); err != nil {
		return nil, err
	}
	ok = true
	if progress != nil {
		progress(Progress{Chunk: len(tables), Rows: 0, Bytes: ch.Bytes()})
	}
	return &Result{
		Files:      ch.Files(),
		Rows:       0,
		Bytes:      ch.Bytes(),
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}
