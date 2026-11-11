package driver

import (
	"context"
	"net"
)

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
// Dial, when set, is used instead of the default TCP dialer (SSH tunnel).
type ConnConfig struct {
	Params   map[string]string                                                 `json:"params"`
	Password string                                                            `json:"-"`
	Dial     func(ctx context.Context, network, addr string) (net.Conn, error) `json:"-"`
}

// Column describes a single result-set column.
type Column struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// QueryResult is a JSON-friendly tabular query result.
type QueryResult struct {
	Columns       []Column `json:"columns"`
	Rows          [][]any  `json:"rows"`
	RowsAffected  int64    `json:"rowsAffected"`
	AffectedKnown bool     `json:"affectedKnown"`
	DurationMs    int64    `json:"durationMs"`
	Truncated     bool     `json:"truncated"`
}

// StatementResult is one statement from Execute: a result set, a command, or an error.
type StatementResult struct {
	Index  int          `json:"index"`
	SQL    string       `json:"sql"`
	Result *QueryResult `json:"result,omitempty"`
	Error  string       `json:"error,omitempty"`
}

// QueryBatch is every statement Execute ran, in order.
type QueryBatch struct {
	Results []StatementResult `json:"results"`
}

const (
	ObjectTable = "table"
	ObjectView  = "view"
)

// TableInfo is a catalog entry for a table or view.
type TableInfo struct {
	Database string `json:"database,omitempty"`
	Schema   string `json:"schema"`
	Name     string `json:"name"`
	Type     string `json:"type"`
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
// Deleted and Inserted are mutually exclusive with a regular UPDATE.
type RowEdit struct {
	Key      map[string]any `json:"key"`
	Cells    map[string]any `json:"cells"`
	Deleted  bool           `json:"deleted,omitempty"`
	Inserted bool           `json:"inserted,omitempty"`
}

// ApplyEditsRequest applies accumulated cell edits to a single table.
type ApplyEditsRequest struct {
	Database string    `json:"database"`
	Schema   string    `json:"schema"`
	Name     string    `json:"name"`
	Rows     []RowEdit `json:"rows"`
}

const (
	ColumnOpAdd     = "add"
	ColumnOpDrop    = "drop"
	ColumnOpModify  = "modify"
	ColumnOpReorder = "reorder"
)

// TableRef identifies a table inside a connection catalog.
type TableRef struct {
	Database string `json:"database"`
	Schema   string `json:"schema"`
	Name     string `json:"name"`
}

// StreamTableRequest streams rows from a table, optionally filtered by a WHERE clause.
type StreamTableRequest struct {
	TableRef
	WhereSQL string `json:"whereSQL"`
}

// ColumnInfo is catalog metadata for one table column.
type ColumnInfo struct {
	Name          string  `json:"name"`
	Type          string  `json:"type"`
	Nullable      bool    `json:"nullable"`
	Default       *string `json:"default"`
	AutoIncrement bool    `json:"autoIncrement"`
	IsPrimaryKey  bool    `json:"isPrimaryKey"`
	Extra         string  `json:"extra,omitempty"`
	Comment       string  `json:"comment"`
	Ordinal       int     `json:"ordinal"`
}

// TableStructure is the editable column list for a table.
type TableStructure struct {
	Columns    []ColumnInfo `json:"columns"`
	CanReorder bool         `json:"canReorder"`
}

// ColumnDraft is one editable column in a structure editor apply.
// OriginalName is nil for a column that does not exist in the catalog yet.
// A nil Default clears the default; an empty string sets the default to ”.
type ColumnDraft struct {
	OriginalName *string `json:"originalName"`
	Dropped      bool    `json:"dropped"`
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	Nullable     bool    `json:"nullable"`
	Default      *string `json:"default"`
	Comment      string  `json:"comment"`
}

// ColumnPatch is the subset of a column that a modify actually changes.
// Nil pointers are unchanged. ClearDefault drops the default; a non-nil Default sets it.
type ColumnPatch struct {
	Name         *string `json:"name,omitempty"`
	Type         *string `json:"type,omitempty"`
	Nullable     *bool   `json:"nullable,omitempty"`
	Default      *string `json:"default,omitempty"`
	ClearDefault bool    `json:"clearDefault,omitempty"`
	Comment      *string `json:"comment,omitempty"`
}

// ColumnOp is one pending schema change against a table.
// After is the previous column's (new) name; empty string means FIRST; nil leaves position unchanged.
// For modify, Column is the catalog column with the patch applied, and Patch lists what changed.
// AutoIncrement, Extra, and IsPrimaryKey on Column stay as they were in the catalog.
type ColumnOp struct {
	Kind    string       `json:"kind"`
	Name    string       `json:"name"`
	NewName string       `json:"newName,omitempty"`
	Column  *ColumnInfo  `json:"column,omitempty"`
	Patch   *ColumnPatch `json:"patch,omitempty"`
	After   *string      `json:"after"`
}

// AlterTableRequest applies a structure draft, or a planned op list, to one table.
// The app service reads Draft and plans Ops. Engine adapters consume Ops.
type AlterTableRequest struct {
	Database string        `json:"database"`
	Schema   string        `json:"schema"`
	Name     string        `json:"name"`
	Draft    []ColumnDraft `json:"draft,omitempty"`
	Ops      []ColumnOp    `json:"ops,omitempty"`
}

// DriverInfo is the public description of a registered driver.
type DriverInfo struct {
	ID             string   `json:"id"`
	DisplayName    string   `json:"displayName"`
	ConnectionSpec []Field  `json:"connectionSpec"`
	ColumnTypes    []string `json:"columnTypes"`
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
