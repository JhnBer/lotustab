package sqlutil

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"lotustab/internal/core/driver"
)

const defaultMaxRows = 5000

// Query runs SQL against database/sql and maps the result to QueryResult.
func Query(ctx context.Context, db *sql.DB, query string, maxRows int) (*driver.QueryResult, error) {
	if maxRows <= 0 {
		maxRows = defaultMaxRows
	}
	start := time.Now()
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, fmt.Errorf("empty query")
	}

	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		res, execErr := db.ExecContext(ctx, q)
		if execErr != nil {
			return nil, err
		}
		affected, _ := res.RowsAffected()
		return &driver.QueryResult{
			Columns:      []driver.Column{},
			Rows:         [][]any{},
			RowsAffected: affected,
			DurationMs:   time.Since(start).Milliseconds(),
		}, nil
	}
	defer rows.Close()

	result, err := scanRows(rows, maxRows)
	if err != nil {
		return nil, err
	}
	result.DurationMs = time.Since(start).Milliseconds()
	return result, nil
}

func scanRows(rows *sql.Rows, maxRows int) (*driver.QueryResult, error) {
	colNames, err := rows.Columns()
	if err != nil {
		return &driver.QueryResult{Columns: []driver.Column{}, Rows: [][]any{}}, nil
	}
	colTypes, _ := rows.ColumnTypes()

	columns := make([]driver.Column, len(colNames))
	for i, name := range colNames {
		colType := ""
		if i < len(colTypes) && colTypes[i] != nil {
			colType = colTypes[i].DatabaseTypeName()
		}
		columns[i] = driver.Column{Name: name, Type: colType}
	}

	out := make([][]any, 0)
	truncated := false
	for rows.Next() {
		if len(out) >= maxRows {
			truncated = true
			break
		}
		holders := make([]any, len(colNames))
		ptrs := make([]any, len(colNames))
		for i := range holders {
			ptrs[i] = &holders[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		row := make([]any, len(holders))
		for i, v := range holders {
			row[i] = driver.NormalizeValue(v)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &driver.QueryResult{
		Columns:      columns,
		Rows:         out,
		RowsAffected: int64(len(out)),
		Truncated:    truncated,
	}, nil
}

// ListStrings runs a single-column query and returns string values.
func ListStrings(ctx context.Context, db *sql.DB, query string) ([]string, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []string{}
	}
	return out, nil
}
