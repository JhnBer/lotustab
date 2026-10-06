package driver

import (
	"strings"
	"unicode"
)

// StatementVerb is the first SQL keyword, skipping whitespace and comments.
func StatementVerb(sql string) string {
	s := skipSQLTrivia(sql)
	for s != "" && (s[0] == '(' || s[0] == ';') {
		s = skipSQLTrivia(s[1:])
	}
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	return strings.ToUpper(strings.Trim(fields[0], ";()"))
}

// VerbReturnsRows reports whether the statement is expected to produce a result set.
func VerbReturnsRows(verb string) bool {
	switch verb {
	case "SELECT", "WITH", "SHOW", "EXPLAIN", "DESCRIBE", "DESC", "TABLE", "VALUES", "CALL":
		return true
	default:
		return false
	}
}

// VerbAffectsRows reports whether the statement has a meaningful rows-affected count.
func VerbAffectsRows(verb string) bool {
	switch verb {
	case "INSERT", "UPDATE", "DELETE", "REPLACE", "MERGE", "MOVE", "FETCH", "COPY":
		return true
	default:
		return false
	}
}

func skipSQLTrivia(s string) string {
	i := 0
	n := len(s)
	for i < n {
		r := rune(s[i])
		if r < 0x80 && unicode.IsSpace(r) {
			i++
			continue
		}
		if i+1 < n && s[i] == '-' && s[i+1] == '-' {
			i += 2
			for i < n && s[i] != '\n' {
				i++
			}
			continue
		}
		if i+1 < n && s[i] == '/' && s[i+1] == '*' {
			i += 2
			for i+1 < n && !(s[i] == '*' && s[i+1] == '/') {
				i++
			}
			if i+1 < n {
				i += 2
			} else {
				i = n
			}
			continue
		}
		break
	}
	return s[i:]
}
