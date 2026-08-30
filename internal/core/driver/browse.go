package driver

import "strings"

const (
	DefaultBrowseLimit = 100
	MinBrowseLimit     = 50
	MaxBrowseLimit     = 10000
)

// ClampTableBrowse normalizes paging and sort fields for a table browse request.
func ClampTableBrowse(req TableBrowseRequest) TableBrowseRequest {
	req.Database = strings.TrimSpace(req.Database)
	req.Schema = strings.TrimSpace(req.Schema)
	req.Name = strings.TrimSpace(req.Name)
	req.OrderBy = strings.TrimSpace(req.OrderBy)
	req.Limit, req.Offset = ClampLimitOffset(req.Limit, req.Offset)

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
		PrimaryKey:   []string{},
	}
}

// ClampApplyEdits trims table identity fields on an edit batch.
func ClampApplyEdits(req ApplyEditsRequest) ApplyEditsRequest {
	req.Database = strings.TrimSpace(req.Database)
	req.Schema = strings.TrimSpace(req.Schema)
	req.Name = strings.TrimSpace(req.Name)
	return req
}
