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

// ClampTableRef trims table identity fields.
func ClampTableRef(ref TableRef) TableRef {
	ref.Database = strings.TrimSpace(ref.Database)
	ref.Schema = strings.TrimSpace(ref.Schema)
	ref.Name = strings.TrimSpace(ref.Name)
	return ref
}

// ClampAlterTable trims table identity and operation fields.
func ClampAlterTable(req AlterTableRequest) AlterTableRequest {
	req.Database = strings.TrimSpace(req.Database)
	req.Schema = strings.TrimSpace(req.Schema)
	req.Name = strings.TrimSpace(req.Name)
	for i := range req.Ops {
		req.Ops[i].Kind = strings.ToLower(strings.TrimSpace(req.Ops[i].Kind))
		req.Ops[i].Name = strings.TrimSpace(req.Ops[i].Name)
		req.Ops[i].NewName = strings.TrimSpace(req.Ops[i].NewName)
		if req.Ops[i].Column != nil {
			req.Ops[i].Column.Name = strings.TrimSpace(req.Ops[i].Column.Name)
			req.Ops[i].Column.Type = strings.TrimSpace(req.Ops[i].Column.Type)
			req.Ops[i].Column.Comment = strings.TrimSpace(req.Ops[i].Column.Comment)
			if req.Ops[i].Column.Default != nil {
				s := strings.TrimSpace(*req.Ops[i].Column.Default)
				req.Ops[i].Column.Default = &s
			}
		}
		if req.Ops[i].After != nil {
			s := strings.TrimSpace(*req.Ops[i].After)
			req.Ops[i].After = &s
		}
	}
	return req
}
