package sqlutil

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"lotustab/internal/core/driver"
)

// Placeholder formats a 1-based parameter index for a SQL dialect.
type Placeholder func(n int) string

// Dollar is a PostgreSQL $n placeholder.
func Dollar(n int) string { return fmt.Sprintf("$%d", n) }

// Question is a MySQL ? placeholder.
func Question(int) string { return "?" }

// ExecAffected runs a statement and returns how many rows it changed.
type ExecAffected func(ctx context.Context, sql string, args ...any) (int64, error)

// PrepareUpdate builds a parameterized UPDATE and its argument list.
// SET columns are sorted for stable SQL. WHERE columns keep pk order.
func PrepareUpdate(quote func(string) string, ph Placeholder, table string, cells map[string]any, pk []string, key map[string]any) (string, []any, error) {
	if strings.TrimSpace(table) == "" {
		return "", nil, fmt.Errorf("table name is required")
	}
	if len(pk) == 0 {
		return "", nil, fmt.Errorf("table has no primary key")
	}
	if len(cells) == 0 {
		return "", nil, fmt.Errorf("no columns to update")
	}

	setCols := make([]string, 0, len(cells))
	for col := range cells {
		if strings.TrimSpace(col) == "" {
			return "", nil, fmt.Errorf("column name is required")
		}
		setCols = append(setCols, col)
	}
	sort.Strings(setCols)

	var b strings.Builder
	b.WriteString("UPDATE ")
	b.WriteString(table)
	b.WriteString(" SET ")
	args := make([]any, 0, len(setCols)+len(pk))
	for i, col := range setCols {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(quote(col))
		b.WriteByte('=')
		b.WriteString(ph(len(args) + 1))
		args = append(args, cells[col])
	}

	b.WriteString(" WHERE ")
	for i, col := range pk {
		if strings.TrimSpace(col) == "" {
			return "", nil, fmt.Errorf("key column name is required")
		}
		v, ok := key[col]
		if !ok {
			return "", nil, fmt.Errorf("missing key column %s", col)
		}
		if i > 0 {
			b.WriteString(" AND ")
		}
		b.WriteString(quote(col))
		b.WriteByte('=')
		b.WriteString(ph(len(args) + 1))
		args = append(args, v)
	}
	return b.String(), args, nil
}

// ApplyRowEdits runs one UPDATE per changed row. Callers wrap this in a transaction.
func ApplyRowEdits(ctx context.Context, exec ExecAffected, quote func(string) string, ph Placeholder, table string, pk []string, rows []driver.RowEdit) error {
	if len(pk) == 0 {
		return fmt.Errorf("table has no primary key")
	}
	applied := 0
	for i, row := range rows {
		if len(row.Cells) == 0 {
			continue
		}
		q, args, err := PrepareUpdate(quote, ph, table, row.Cells, pk, row.Key)
		if err != nil {
			return err
		}
		n, err := exec(ctx, q, args...)
		if err != nil {
			return err
		}
		if n != 1 {
			return fmt.Errorf("row %d: expected 1 row updated, got %d", i, n)
		}
		applied++
	}
	if applied == 0 {
		return fmt.Errorf("no edits")
	}
	return nil
}
