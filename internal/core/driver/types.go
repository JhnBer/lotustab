package driver

// FieldType describes how the frontend should render a connection form field.
type FieldType string

const (
	FieldText     FieldType = "text"
	FieldPassword FieldType = "password"
	FieldNumber   FieldType = "number"
	FieldSelect   FieldType = "select"
	FieldCheckbox FieldType = "checkbox"
)

// Option is a selectable value for FieldSelect.
type Option struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// Field is a declarative connection-form field advertised by a driver.
type Field struct {
	Key         string    `json:"key"`
	Label       string    `json:"label"`
	Type        FieldType `json:"type"`
	Required    bool      `json:"required"`
	Secret      bool      `json:"secret"`
	Default     string    `json:"default,omitempty"`
	Placeholder string    `json:"placeholder,omitempty"`
	Options     []Option  `json:"options,omitempty"`
	Group       string    `json:"group,omitempty"`
	Fetchable   bool      `json:"fetchable,omitempty"`
}

// ConnConfig is the runtime configuration used to open a connection.
// Password is supplied separately so it is never persisted with profiles.
type ConnConfig struct {
	Params   map[string]string `json:"params"`
	Password string            `json:"-"`
}

// Column describes a single result-set column.
type Column struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// QueryResult is a JSON-friendly tabular query result.
type QueryResult struct {
	Columns      []Column `json:"columns"`
	Rows         [][]any  `json:"rows"`
	RowsAffected int64    `json:"rowsAffected"`
	DurationMs   int64    `json:"durationMs"`
	Truncated    bool     `json:"truncated"`
}

const (
	ObjectTable = "table"
	ObjectView  = "view"
)

// TableInfo is a catalog entry for a table or view.
type TableInfo struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`
	Type   string `json:"type"`
}

// TableFilter is one column predicate on a table browse request.
type TableFilter struct {
	Column string `json:"column"`
	Op     string `json:"op"`
	Value  string `json:"value"`
	Join   string `json:"join"`
}

// TableBrowseRequest pages and optionally sorts rows from a single table.
type TableBrowseRequest struct {
	Database string        `json:"database"`
	Schema   string        `json:"schema"`
	Name     string        `json:"name"`
	Limit    int           `json:"limit"`
	Offset   int           `json:"offset"`
	OrderBy  string        `json:"orderBy"`
	OrderDir string        `json:"orderDir"`
	Filters  []TableFilter `json:"filters,omitempty"`
}

// TableBrowseResult is a page of table data plus a total row count.
type TableBrowseResult struct {
	Columns      []Column `json:"columns"`
	Rows         [][]any  `json:"rows"`
	RowsAffected int64    `json:"rowsAffected"`
	DurationMs   int64    `json:"durationMs"`
	Truncated    bool     `json:"truncated"`
	Total        int64    `json:"total"`
	PrimaryKey   []string `json:"primaryKey"`
}

// RowEdit is a set of cell changes for one row, addressed by its original primary key.
type RowEdit struct {
	Key   map[string]any `json:"key"`
	Cells map[string]any `json:"cells"`
}

// ApplyEditsRequest applies accumulated cell edits to a single table.
type ApplyEditsRequest struct {
	Database string    `json:"database"`
	Schema   string    `json:"schema"`
	Name     string    `json:"name"`
	Rows     []RowEdit `json:"rows"`
}

// DriverInfo is the public description of a registered driver.
type DriverInfo struct {
	ID             string  `json:"id"`
	DisplayName    string  `json:"displayName"`
	ConnectionSpec []Field `json:"connectionSpec"`
}

// Param returns a connection parameter or a default.
func (c ConnConfig) Param(key, fallback string) string {
	if c.Params == nil {
		return fallback
	}
	if v, ok := c.Params[key]; ok && v != "" {
		return v
	}
	return fallback
}
