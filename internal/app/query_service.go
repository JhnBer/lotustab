package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"lotustab/internal/core/connection"
	"lotustab/internal/core/driver"
	"lotustab/internal/secrets"
	"lotustab/internal/storage"
)

const queryProgressEvent = "query:progress"

// QueryService executes SQL against an open (or lazily opened) connection.
type QueryService struct {
	connections *ConnectionsService
}

// QueryProgress is emitted on query:progress after each statement of a script.
type QueryProgress struct {
	RunID  string              `json:"runId"`
	Index  int                 `json:"index"`
	Total  int                 `json:"total"`
	SQL    string              `json:"sql"`
	Result *driver.QueryResult `json:"result,omitempty"`
	Error  string              `json:"error,omitempty"`
}

// NewQueryService constructs the Wails query service.
func NewQueryService(
	registry *driver.Registry,
	store *storage.Store,
	secretsStore *secrets.Store,
	manager *connection.Manager,
) *QueryService {
	return &QueryService{
		connections: NewConnectionsService(registry, store, secretsStore, manager),
	}
}

// NewQueryServiceFromConnections reuses an existing connections service (same manager).
func NewQueryServiceFromConnections(connections *ConnectionsService) *QueryService {
	return &QueryService{connections: connections}
}

// Execute runs SQL for the given profile. The profile is connected if needed.
// database is optional; empty uses the connection default catalog.
// timeoutSeconds is 0 to wait until the caller cancels.
// runID correlates query:progress events when more than one statement runs.
func (s *QueryService) Execute(ctx context.Context, profileID, sqlText, database string, limit, offset, timeoutSeconds int, runID string) (*driver.QueryBatch, error) {
	if strings.TrimSpace(profileID) == "" {
		return nil, fmt.Errorf("profile is required")
	}
	if strings.TrimSpace(sqlText) == "" {
		return nil, fmt.Errorf("query is empty")
	}
	statements := driver.SplitStatements(sqlText)
	if len(statements) == 0 {
		return nil, fmt.Errorf("query is empty")
	}
	limit, offset = driver.ClampLimitOffset(limit, offset)
	database = strings.TrimSpace(database)
	runID = strings.TrimSpace(runID)

	queryCtx := ctx
	if timeoutSeconds > 0 {
		var cancel context.CancelFunc
		queryCtx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
		defer cancel()
	}

	batch, err := withLiveConn(s, queryCtx, profileID, func(liveCtx context.Context, conn driver.Conn) (*driver.QueryBatch, error) {
		if len(statements) == 1 {
			result, err := conn.Query(liveCtx, sqlText, database, limit, offset)
			if err != nil {
				return nil, err
			}
			return &driver.QueryBatch{Results: []driver.StatementResult{{
				Index:  0,
				SQL:    statements[0],
				Result: result,
			}}}, nil
		}
		return runScriptBatch(liveCtx, ctx, conn, statements, database, timeoutSeconds, runID)
	})
	if err != nil {
		if batch != nil && len(batch.Results) > 0 {
			return batch, nil
		}
		if queryCtx.Err() == context.DeadlineExceeded && ctx.Err() == nil {
			return nil, fmt.Errorf("query timed out after %d seconds", timeoutSeconds)
		}
		return nil, err
	}
	return batch, nil
}

func runScriptBatch(queryCtx, parent context.Context, conn driver.Conn, statements []string, database string, timeoutSeconds int, runID string) (*driver.QueryBatch, error) {
	batch := &driver.QueryBatch{Results: make([]driver.StatementResult, 0, len(statements))}
	err := conn.WithSession(queryCtx, database, func(sess driver.ScriptSession) error {
		return driver.RunScript(queryCtx, sess, statements, driver.MaxBrowseLimit, func(i int, result *driver.QueryResult, qerr error) error {
			if qerr != nil && isStaleConn(qerr) && len(batch.Results) == 0 {
				return nil
			}
			item := driver.StatementResult{
				Index:  i,
				SQL:    statements[i],
				Result: result,
				Error:  statementErrText(qerr, queryCtx, parent, timeoutSeconds),
			}
			batch.Results = append(batch.Results, item)
			emitQueryProgress(QueryProgress{
				RunID:  runID,
				Index:  item.Index,
				Total:  len(statements),
				SQL:    item.SQL,
				Result: item.Result,
				Error:  item.Error,
			})
			return nil
		})
	})
	return batch, err
}

func statementErrText(err error, queryCtx, parent context.Context, timeoutSeconds int) string {
	if err == nil {
		return ""
	}
	if queryCtx.Err() == context.DeadlineExceeded && parent.Err() == nil {
		if timeoutSeconds > 0 {
			return fmt.Sprintf("query timed out after %d seconds", timeoutSeconds)
		}
		return "query timed out"
	}
	if errors.Is(err, context.Canceled) || parent.Err() == context.Canceled {
		return "query was cancelled"
	}
	return err.Error()
}

func emitQueryProgress(p QueryProgress) {
	if p.RunID == "" {
		return
	}
	if appl := application.Get(); appl != nil {
		appl.Event.Emit(queryProgressEvent, p)
	}
}

// ListTables returns tables and views in the given database.
func (s *QueryService) ListTables(ctx context.Context, profileID, database string) ([]driver.TableInfo, error) {
	if strings.TrimSpace(profileID) == "" {
		return nil, fmt.Errorf("profile is required")
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	tables, err := withLiveConn(s, ctx, profileID, func(ctx context.Context, conn driver.Conn) ([]driver.TableInfo, error) {
		return conn.ListTables(ctx, strings.TrimSpace(database))
	})
	if err != nil {
		return nil, err
	}
	if tables == nil {
		tables = []driver.TableInfo{}
	}
	return tables, nil
}

// SearchTables finds tables by name substring, capped by limit.
// database, when set, restricts the search to that catalog; empty uses the driver default.
func (s *QueryService) SearchTables(ctx context.Context, profileID, database, query string, limit int) ([]driver.TableInfo, error) {
	if strings.TrimSpace(profileID) == "" {
		return nil, fmt.Errorf("profile is required")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return []driver.TableInfo{}, nil
	}
	limit = driver.ClampSearchLimit(limit)
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	tables, err := withLiveConn(s, ctx, profileID, func(ctx context.Context, conn driver.Conn) ([]driver.TableInfo, error) {
		return conn.SearchTables(ctx, strings.TrimSpace(database), query, limit)
	})
	if err != nil {
		return nil, err
	}
	if tables == nil {
		tables = []driver.TableInfo{}
	}
	return tables, nil
}

// BrowseTable returns a page of rows from a table.
func (s *QueryService) BrowseTable(ctx context.Context, profileID string, req driver.TableBrowseRequest) (*driver.TableBrowseResult, error) {
	if strings.TrimSpace(profileID) == "" {
		return nil, fmt.Errorf("profile is required")
	}
	req = driver.ClampTableBrowse(req)
	if req.Name == "" {
		return nil, fmt.Errorf("table is required")
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	return withLiveConn(s, ctx, profileID, func(ctx context.Context, conn driver.Conn) (*driver.TableBrowseResult, error) {
		return conn.BrowseTable(ctx, req)
	})
}

// ApplyEdits writes accumulated cell edits in a single transaction.
func (s *QueryService) ApplyEdits(ctx context.Context, profileID string, req driver.ApplyEditsRequest) error {
	if strings.TrimSpace(profileID) == "" {
		return fmt.Errorf("profile is required")
	}
	req = driver.ClampApplyEdits(req)
	if req.Name == "" {
		return fmt.Errorf("table is required")
	}
	if len(req.Rows) == 0 {
		return fmt.Errorf("no edits")
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	_, err := withLiveConn(s, ctx, profileID, func(ctx context.Context, conn driver.Conn) (struct{}, error) {
		return struct{}{}, conn.ApplyEdits(ctx, req)
	})
	return err
}

// DescribeTable returns column metadata for a table.
func (s *QueryService) DescribeTable(ctx context.Context, profileID string, ref driver.TableRef) (*driver.TableStructure, error) {
	if strings.TrimSpace(profileID) == "" {
		return nil, fmt.Errorf("profile is required")
	}
	ref = driver.ClampTableRef(ref)
	if ref.Name == "" {
		return nil, fmt.Errorf("table is required")
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	return withLiveConn(s, ctx, profileID, func(ctx context.Context, conn driver.Conn) (*driver.TableStructure, error) {
		return conn.DescribeTable(ctx, ref)
	})
}

// AlterTable applies accumulated column operations in a single request.
func (s *QueryService) AlterTable(ctx context.Context, profileID string, req driver.AlterTableRequest) error {
	if strings.TrimSpace(profileID) == "" {
		return fmt.Errorf("profile is required")
	}
	req = driver.ClampAlterTable(req)
	if req.Name == "" {
		return fmt.Errorf("table is required")
	}
	if len(req.Ops) == 0 {
		return fmt.Errorf("no schema changes")
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	_, err := withLiveConn(s, ctx, profileID, func(ctx context.Context, conn driver.Conn) (struct{}, error) {
		return struct{}{}, conn.AlterTable(ctx, req)
	})
	return err
}

func withLiveConn[T any](s *QueryService, ctx context.Context, profileID string, fn func(context.Context, driver.Conn) (T, error)) (T, error) {
	return withTrackedConn(s.connections, ctx, profileID, fn)
}

func withTrackedConn[T any](connections *ConnectionsService, ctx context.Context, profileID string, fn func(context.Context, driver.Conn) (T, error)) (T, error) {
	var zero T
	ctx, done := connections.Track(ctx, profileID)
	defer done()
	conn, err := connections.ensureOpen(ctx, profileID)
	if err != nil {
		return zero, err
	}
	out, err := fn(ctx, conn)
	if err == nil || ctx.Err() != nil || !isStaleConn(err) {
		return out, err
	}
	conn, err = connections.reconnect(ctx, profileID)
	if err != nil {
		return zero, err
	}
	return fn(ctx, conn)
}
