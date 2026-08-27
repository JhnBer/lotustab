package driver

import "strings"

const (
	DefaultBrowseLimit = 100
	MinBrowseLimit     = 50
	MaxBrowseLimit     = 1000
)

// ClampTableBrowse normalizes paging and sort fields for a table browse request.
func ClampTableBrowse(req TableBrowseRequest) TableBrowseRequest {
	req.Database = strings.TrimSpace(req.Database)
	req.Schema = strings.TrimSpace(req.Schema)
	req.Name = strings.TrimSpace(req.Name)
	req.OrderBy = strings.TrimSpace(req.OrderBy)

	switch {
	case req.Limit <= 0:
		req.Limit = DefaultBrowseLimit
	case req.Limit < MinBrowseLimit:
		req.Limit = MinBrowseLimit
	case req.Limit > MaxBrowseLimit:
		req.Limit = MaxBrowseLimit
	}

	if req.Offset < 0 {
		req.Offset = 0
	}

	dir := strings.ToLower(strings.TrimSpace(req.OrderDir))
	if req.OrderBy == "" {
		req.OrderDir = ""
	} else if dir == "desc" {
		req.OrderDir = "desc"
	} else {
		req.OrderDir = "asc"
	}
	return req
}

// BrowseFromQuery copies a query result into a table browse page.
func BrowseFromQuery(result *QueryResult, total int64) *TableBrowseResult {
	if result == nil {
		result = &QueryResult{}
	}
	columns := result.Columns
	if columns == nil {
		columns = []Column{}
	}
	rows := result.Rows
	if rows == nil {
		rows = [][]any{}
	}
	return &TableBrowseResult{
		Columns:      columns,
		Rows:         rows,
		RowsAffected: result.RowsAffected,
		DurationMs:   result.DurationMs,
		Truncated:    result.Truncated,
		Total:        total,
	}
}
