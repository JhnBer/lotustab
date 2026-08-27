package sqlutil

import "strings"

// QuoteMySQL wraps an identifier in backticks, doubling internal backticks.
func QuoteMySQL(ident string) string {
	return "`" + strings.ReplaceAll(ident, "`", "``") + "`"
}

// QuotePostgres wraps an identifier in double quotes, doubling internal quotes.
func QuotePostgres(ident string) string {
	return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
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
