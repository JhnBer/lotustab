package driver

import (
	"fmt"
	"strings"
)

// QueryPage is a user query prepared for a single result page.
type QueryPage struct {
	SQL     string
	MaxRows int
	Limit   int
	Offset  int
	Paged   bool
}

// ClampLimitOffset applies the same page-size bounds as table browse.
func ClampLimitOffset(limit, offset int) (int, int) {
	switch {
	case limit <= 0:
		limit = DefaultBrowseLimit
	case limit < MinBrowseLimit:
		limit = MinBrowseLimit
	case limit > MaxBrowseLimit:
		limit = MaxBrowseLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

// PrepareQueryPage wraps SELECT/WITH with LIMIT/OFFSET so the database
// applies the cap. Non-select SQL is unchanged; MaxRows still bounds the scanner.
func PrepareQueryPage(sql string, limit, offset int) QueryPage {
	limit, offset = ClampLimitOffset(limit, offset)
	sql = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(sql), ";"))
	if isPaginableQuery(sql) {
		return QueryPage{
			SQL:     fmt.Sprintf("SELECT * FROM (\n%s\n) AS _lotustab_page LIMIT %d OFFSET %d", sql, limit+1, offset),
			MaxRows: limit + 1,
			Limit:   limit,
			Offset:  offset,
			Paged:   true,
		}
	}
	return QueryPage{
		SQL:     sql,
		MaxRows: limit,
		Limit:   limit,
		Offset:  offset,
	}
}

// FinishQueryPage drops the extra lookahead row used to detect a following page.
func FinishQueryPage(result *QueryResult, page QueryPage) *QueryResult {
	if result == nil || !page.Paged {
		return result
	}
	if len(result.Rows) > page.Limit {
		result.Rows = result.Rows[:page.Limit]
		result.Truncated = true
	} else {
		result.Truncated = false
	}
	return result
}

func isPaginableQuery(sql string) bool {
	fields := strings.Fields(sql)
	if len(fields) == 0 {
		return false
	}
	tok := strings.ToUpper(strings.TrimRight(fields[0], ";()"))
	return tok == "SELECT" || tok == "WITH"
}
