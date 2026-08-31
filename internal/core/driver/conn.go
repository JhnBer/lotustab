package driver

import "context"

// Conn is an open session to a database server.
type Conn interface {
	Ping(ctx context.Context) error
	Query(ctx context.Context, sql, database string, limit, offset int) (*QueryResult, error)
	Databases(ctx context.Context) ([]string, error)
	ListTables(ctx context.Context, database string) ([]TableInfo, error)
	BrowseTable(ctx context.Context, req TableBrowseRequest) (*TableBrowseResult, error)
	ApplyEdits(ctx context.Context, req ApplyEditsRequest) error
	Close() error
}
