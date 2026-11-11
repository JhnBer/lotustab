package driver

import "context"

// Driver is the port every database engine must implement.
type Driver interface {
	ID() string
	DisplayName() string
	ConnectionSpec() []Field
	// ColumnTypes lists ALTER type strings for the structure editor.
	// The first entry is the default type for a new column.
	ColumnTypes() []string
	Open(ctx context.Context, cfg ConnConfig) (Conn, error)
}
