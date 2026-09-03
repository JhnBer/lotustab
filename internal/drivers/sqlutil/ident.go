package sqlutil

import (
	"strconv"
	"strings"
)

// QuoteMySQL wraps an identifier in backticks, doubling internal backticks.
func QuoteMySQL(ident string) string {
	return "`" + strings.ReplaceAll(ident, "`", "``") + "`"
}

// QuoteClickHouse wraps an identifier in backticks, doubling internal backticks.
func QuoteClickHouse(ident string) string {
	return QuoteMySQL(ident)
}

// QuotePostgres wraps an identifier in double quotes, doubling internal quotes.
func QuotePostgres(ident string) string {
	return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
}

// QuoteString wraps a value in single quotes, doubling internal quotes.
func QuoteString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// FormatDefault returns a SQL DEFAULT expression. Bare keywords, numbers, and
// already-quoted / cast expressions are passed through; other values are quoted.
func FormatDefault(expr string) string {
	if expr == "" {
		return QuoteString("")
	}
	s := strings.TrimSpace(expr)
	if s == "" {
		return QuoteString("")
	}
	upper := strings.ToUpper(s)
	switch {
	case upper == "NULL", upper == "TRUE", upper == "FALSE":
		return s
	case strings.HasPrefix(upper, "CURRENT_"), strings.HasPrefix(upper, "LOCALTIME"), strings.HasPrefix(upper, "LOCALTIMESTAMP"), strings.HasPrefix(upper, "NOW("):
		return s
	case strings.Contains(s, "::"), strings.Contains(s, "("):
		return s
	case (strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'")) || (strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`)):
		return s
	}
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return s
	}
	return QuoteString(s)
}

// LikeContains returns a LIKE/ILIKE pattern that matches any string containing query.
// %, _, and \ in query are escaped with a backslash.
func LikeContains(query string) string {
	return "%" + EscapeLike(query) + "%"
}

// EscapeLike escapes LIKE wildcards so they match literally.
func EscapeLike(query string) string {
	var b strings.Builder
	b.Grow(len(query))
	for i := 0; i < len(query); i++ {
		c := query[i]
		if c == '%' || c == '_' || c == '\\' {
			b.WriteByte('\\')
		}
		b.WriteByte(c)
	}
	return b.String()
}

// OrderClause returns a leading ORDER BY fragment, or empty if orderBy is unset.
func OrderClause(quote func(string) string, orderBy, orderDir string) string {
	if orderBy == "" {
		return ""
	}
	dir := "ASC"
	if strings.EqualFold(orderDir, "desc") {
		dir = "DESC"
	}
	return " ORDER BY " + quote(orderBy) + " " + dir
}
