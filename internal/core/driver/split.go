package driver

import (
	"strings"
	"unicode"
)

// SplitStatements cuts SQL on semicolons that are not inside quotes, comments,
// or PostgreSQL dollar-quotes. Empty pieces and comment-only pieces are dropped.
func SplitStatements(sql string) []string {
	var out []string
	var b strings.Builder
	n := len(sql)
	i := 0

	flush := func() {
		s := strings.TrimSpace(b.String())
		b.Reset()
		if s == "" || isCommentOnly(s) {
			return
		}
		out = append(out, s)
	}

	for i < n {
		if sql[i] == '$' {
			if tag, ok, next := dollarTag(sql, i); ok {
				rest := sql[next:]
				end := strings.Index(rest, tag)
				if end < 0 {
					b.WriteString(sql[i:])
					i = n
					break
				}
				end = next + end + len(tag)
				b.WriteString(sql[i:end])
				i = end
				continue
			}
		}

		if i+1 < n && sql[i] == '-' && sql[i+1] == '-' {
			j := i
			for j < n && sql[j] != '\n' {
				j++
			}
			b.WriteString(sql[i:j])
			i = j
			continue
		}

		if i+1 < n && sql[i] == '/' && sql[i+1] == '*' {
			j := i + 2
			for j+1 < n && !(sql[j] == '*' && sql[j+1] == '/') {
				j++
			}
			if j+1 < n {
				j += 2
			} else {
				j = n
			}
			b.WriteString(sql[i:j])
			i = j
			continue
		}

		if sql[i] == '\'' || sql[i] == '"' || sql[i] == '`' {
			q := sql[i]
			b.WriteByte(q)
			i++
			for i < n {
				if sql[i] == '\\' && i+1 < n {
					b.WriteByte(sql[i])
					b.WriteByte(sql[i+1])
					i += 2
					continue
				}
				if sql[i] == q {
					b.WriteByte(q)
					i++
					if i < n && sql[i] == q {
						b.WriteByte(q)
						i++
						continue
					}
					break
				}
				b.WriteByte(sql[i])
				i++
			}
			continue
		}

		if sql[i] == ';' {
			flush()
			i++
			continue
		}

		b.WriteByte(sql[i])
		i++
	}
	flush()
	if out == nil {
		return []string{}
	}
	return out
}

func dollarTag(sql string, i int) (tag string, ok bool, contentStart int) {
	j := i + 1
	for j < len(sql) && isDollarTagChar(sql[j], j == i+1) {
		j++
	}
	if j < len(sql) && sql[j] == '$' {
		return sql[i : j+1], true, j + 1
	}
	return "", false, 0
}

func isDollarTagChar(c byte, first bool) bool {
	if c >= '0' && c <= '9' {
		return !first
	}
	return c == '_' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func isCommentOnly(s string) bool {
	n := len(s)
	i := 0
	for i < n {
		r := rune(s[i])
		if r < 0x80 && unicode.IsSpace(r) {
			i++
			continue
		}
		if i+1 < n && s[i] == '-' && s[i+1] == '-' {
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
		return false
	}
	return true
}
