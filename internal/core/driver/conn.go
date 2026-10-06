package driver

import "context"

// ScriptSession is one pinned database session. Statements run in order and
// share transaction state. The driver reports whether a transaction is open;
// callers must not infer that from the SQL text.
type ScriptSession interface {
	Query(ctx context.Context, sql string, maxRows int) (*QueryResult, error)
	InTransaction(ctx context.Context) (bool, error)
	Rollback(ctx context.Context) error
}

// Conn is an open session to a database server.
type Conn interface {
	Ping(ctx context.Context) error
	Query(ctx context.Context, sql, database string, limit, offset int) (*QueryResult, error)
	WithSession(ctx context.Context, database string, fn func(ScriptSession) error) error
	Databases(ctx context.Context) ([]string, error)
	ListTables(ctx context.Context, database string) ([]TableInfo, error)
	SearchTables(ctx context.Context, database, query string, limit int) ([]TableInfo, error)
	BrowseTable(ctx context.Context, req TableBrowseRequest) (*TableBrowseResult, error)
	ApplyEdits(ctx context.Context, req ApplyEditsRequest) error
	DescribeTable(ctx context.Context, ref TableRef) (*TableStructure, error)
	AlterTable(ctx context.Context, req AlterTableRequest) error
	StreamTable(ctx context.Context, req StreamTableRequest, fn func(cols []Column, rows [][]any) error) error
	TableDDL(ctx context.Context, ref TableRef) (string, error)
	Close() error
}
