package postgres

import (
	"context"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"lotustab/internal/core/driver"
)

// cellValue turns one pgx cell into a JSON value. uuid arrives as raw bytes,
// so it is formatted here before the shared normalizer.
func cellValue(oid uint32, v any) any {
	if oid == pgtype.UUIDOID {
		if b, ok := v.([16]byte); ok {
			return pgtype.UUID{Bytes: b, Valid: true}.String()
		}
	}
	return driver.NormalizeValue(v)
}

// safePgType matches format_type output that is safe to splice after a placeholder.
var safePgType = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_ ]*(?:\([0-9]+(?:,\s*[0-9]+)?\))?(?:\[\])*$`)

func castSuffix(formatType string) (string, bool) {
	t := strings.TrimSpace(formatType)
	if t == "" || !safePgType.MatchString(t) {
		return "", false
	}
	return "::" + t, true
}

func editColumnNames(pk []string, rows []driver.RowEdit) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(pk))
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	for _, name := range pk {
		add(name)
	}
	for _, row := range rows {
		for name := range row.Cells {
			add(name)
		}
		for name := range row.Key {
			add(name)
		}
	}
	return out
}

func postgresColumnCasts(ctx context.Context, pool *pgxpool.Pool, schema, name string, columns []string) (map[string]string, error) {
	if len(columns) == 0 {
		return nil, nil
	}
	const q = `
		SELECT a.attname, format_type(a.atttypid, a.atttypmod)
		FROM pg_attribute a
		JOIN pg_class c ON c.oid = a.attrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = CASE WHEN $1 = '' THEN current_schema() ELSE $1 END
		  AND c.relname = $2
		  AND a.attname = ANY($3)
		  AND a.attnum > 0
		  AND NOT a.attisdropped`
	rows, err := pool.Query(ctx, q, schema, name, columns)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]string{}
	for rows.Next() {
		var col, typ string
		if err := rows.Scan(&col, &typ); err != nil {
			return nil, err
		}
		if suffix, ok := castSuffix(typ); ok {
			out[col] = suffix
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
