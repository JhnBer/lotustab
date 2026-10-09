package export

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"lotustab/internal/drivers/sqlutil"
)

const (
	DialectPostgres   = "postgres"
	DialectMySQL      = "mysql"
	DialectClickHouse = "clickhouse"

	FormatSQL  = "sql"
	FormatCSV  = "csv"
	FormatJSON = "json"
)

func quoteIdent(dialect, ident string) string {
	switch dialect {
	case DialectPostgres:
		return sqlutil.QuotePostgres(ident)
	default:
		return sqlutil.QuoteMySQL(ident)
	}
}

// SelectQuery is the SELECT * statement StreamTable runs for a dump.
func SelectQuery(dialect, database, schema, name, where string) string {
	return sqlutil.SelectStar(qualifiedTable(dialect, database, schema, name), where)
}

// CountQuery is SELECT COUNT(*) for the same table and WHERE as a dump.
func CountQuery(dialect, database, schema, name, where string) string {
	q := "SELECT COUNT(*) FROM " + qualifiedTable(dialect, database, schema, name)
	if where = sqlutil.NormalizeWhere(where); where != "" {
		q += " WHERE " + where
	}
	return q
}

func qualifiedTable(dialect, database, schema, name string) string {
	name = strings.TrimSpace(name)
	schema = strings.TrimSpace(schema)
	database = strings.TrimSpace(database)
	switch dialect {
	case DialectPostgres:
		if schema == "" {
			return quoteIdent(dialect, name)
		}
		return quoteIdent(dialect, schema) + "." + quoteIdent(dialect, name)
	default:
		db := database
		if db == "" {
			db = schema
		}
		if db == "" {
			return quoteIdent(dialect, name)
		}
		return quoteIdent(dialect, db) + "." + quoteIdent(dialect, name)
	}
}

func sqlLiteral(dialect string, v any) string {
	if v == nil {
		return "NULL"
	}
	switch t := v.(type) {
	case bool:
		if dialect == DialectPostgres {
			if t {
				return "TRUE"
			}
			return "FALSE"
		}
		if t {
			return "1"
		}
		return "0"
	case int:
		return strconv.FormatInt(int64(t), 10)
	case int8:
		return strconv.FormatInt(int64(t), 10)
	case int16:
		return strconv.FormatInt(int64(t), 10)
	case int32:
		return strconv.FormatInt(int64(t), 10)
	case int64:
		return strconv.FormatInt(t, 10)
	case uint:
		return strconv.FormatUint(uint64(t), 10)
	case uint8:
		return strconv.FormatUint(uint64(t), 10)
	case uint16:
		return strconv.FormatUint(uint64(t), 10)
	case uint32:
		return strconv.FormatUint(uint64(t), 10)
	case uint64:
		return strconv.FormatUint(t, 10)
	case float32:
		return strconv.FormatFloat(float64(t), 'g', -1, 32)
	case float64:
		return strconv.FormatFloat(t, 'g', -1, 64)
	case string:
		return sqlutil.QuoteString(t)
	case []byte:
		if utf8.Valid(t) {
			return sqlutil.QuoteString(string(t))
		}
		return hexLiteral(dialect, t)
	case time.Time:
		return sqlutil.QuoteString(t.UTC().Format("2006-01-02 15:04:05.999999"))
	case fmt.Stringer:
		return sqlutil.QuoteString(t.String())
	default:
		return sqlutil.QuoteString(fmt.Sprint(t))
	}
}

func hexLiteral(dialect string, b []byte) string {
	h := hex.EncodeToString(b)
	if dialect == DialectPostgres {
		return `'\x` + h + `'`
	}
	return "X'" + h + "'"
}

func csvCell(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case bool:
		if t {
			return "true"
		}
		return "false"
	case []byte:
		if utf8.Valid(t) {
			return string(t)
		}
		return hex.EncodeToString(t)
	case time.Time:
		return t.UTC().Format(time.RFC3339Nano)
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}

func fileExt(format string, gzipOn bool) string {
	ext := ".sql"
	if format == FormatCSV {
		ext = ".csv"
	}
	if gzipOn {
		return ext + ".gz"
	}
	return ext
}

func safeFileBase(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "table"
	}
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_' {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('_')
	}
	out := strings.Trim(b.String(), "._")
	if out == "" {
		return "table"
	}
	return out
}

// SafeFileBase is the exported form of the dump filename sanitizer.
func SafeFileBase(name string) string {
	return safeFileBase(name)
}

// FileBaseForTable builds `{prefix}{table}` for folder-mode dumps.
func FileBaseForTable(prefix, name string) string {
	prefix = strings.TrimSpace(prefix)
	name = strings.TrimSpace(name)
	if prefix == "" {
		return safeFileBase(name)
	}
	return safeFileBase(prefix + name)
}
