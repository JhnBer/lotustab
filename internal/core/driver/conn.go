package driver

import "context"

// Conn is an open session to a database server.
type Conn interface {
	Ping(ctx context.Context) error
	Query(ctx context.Context, sql, database string, limit, offset int) (*QueryResult, error)
	Databases(ctx context.Context) ([]string, error)
	ListTables(ctx context.Context, database string) ([]TableInfo, error)
	SearchTables(ctx context.Context, database, query string, limit int) ([]TableInfo, error)
	BrowseTable(ctx context.Context, req TableBrowseRequest) (*TableBrowseResult, error)
	ApplyEdits(ctx context.Context, req ApplyEditsRequest) error
	DescribeTable(ctx context.Context, ref TableRef) (*TableStructure, error)
	AlterTable(ctx context.Context, req AlterTableRequest) error
	Close() error
}
