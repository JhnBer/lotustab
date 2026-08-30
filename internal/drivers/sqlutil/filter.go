package sqlutil

import (
	"fmt"
	"strings"

	"lotustab/internal/core/driver"
)

type filterKind int

const (
	filterCompare filterKind = iota
	filterIn
	filterNull
)

type filterOp struct {
	sql  string
	kind filterKind
}

var filterOps = map[string]filterOp{
	"eq":    {sql: "=", kind: filterCompare},
	"neq":   {sql: "!=", kind: filterCompare},
	"gt":    {sql: ">", kind: filterCompare},
	"gte":   {sql: ">=", kind: filterCompare},
	"lt":    {sql: "<", kind: filterCompare},
	"lte":   {sql: "<=", kind: filterCompare},
	"like":  {sql: "LIKE", kind: filterCompare},
	"nlike": {sql: "NOT LIKE", kind: filterCompare},
	"in":    {sql: "IN", kind: filterIn},
	"nin":   {sql: "NOT IN", kind: filterIn},
	"null":  {sql: "IS NULL", kind: filterNull},
	"nnull": {sql: "IS NOT NULL", kind: filterNull},
}

// WhereClause builds a parameterized WHERE fragment from browse filters.
// The first filter's Join is ignored; later joins must be "and" or "or".
func WhereClause(quote func(string) string, ph Placeholder, filters []driver.TableFilter) (string, []any, error) {
	if len(filters) == 0 {
		return "", nil, nil
	}

	var b strings.Builder
	args := make([]any, 0, len(filters))
	for i, f := range filters {
		col := strings.TrimSpace(f.Column)
		if col == "" {
			return "", nil, fmt.Errorf("filter column is required")
		}
		op, ok := filterOps[strings.ToLower(strings.TrimSpace(f.Op))]
		if !ok {
			return "", nil, fmt.Errorf("unknown filter operator %q", f.Op)
		}
		pred, extra, err := filterPredicate(quote, ph, col, op, f.Value, len(args)+1)
		if err != nil {
			return "", nil, err
		}
		if i == 0 {
			b.WriteString(" WHERE ")
		} else {
			join := strings.ToLower(strings.TrimSpace(f.Join))
			if join == "" {
				join = "and"
			}
			if join != "and" && join != "or" {
				return "", nil, fmt.Errorf("unknown filter join %q", f.Join)
			}
			b.WriteByte(' ')
			b.WriteString(strings.ToUpper(join))
			b.WriteByte(' ')
		}
		b.WriteByte('(')
		b.WriteString(pred)
		b.WriteByte(')')
		args = append(args, extra...)
	}
	return b.String(), args, nil
}

func filterPredicate(quote func(string) string, ph Placeholder, col string, op filterOp, value string, start int) (string, []any, error) {
	qcol := quote(col)
	switch op.kind {
	case filterNull:
		return qcol + " " + op.sql, nil, nil
	case filterIn:
		vals := splitInValues(value)
		if len(vals) == 0 {
			return "", nil, fmt.Errorf("IN filter needs a value")
		}
		var b strings.Builder
		b.WriteString(qcol)
		b.WriteByte(' ')
		b.WriteString(op.sql)
		b.WriteString(" (")
		args := make([]any, len(vals))
		for i, v := range vals {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(ph(start + i))
			args[i] = v
		}
		b.WriteByte(')')
		return b.String(), args, nil
	default:
		return qcol + " " + op.sql + " " + ph(start), []any{value}, nil
	}
}

func splitInValues(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
