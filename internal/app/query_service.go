package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"lotustab/internal/core/connection"
	"lotustab/internal/core/driver"
	"lotustab/internal/secrets"
	"lotustab/internal/storage"
)

// QueryService executes SQL against an open (or lazily opened) connection.
type QueryService struct {
	connections *ConnectionsService
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
func (s *QueryService) Execute(ctx context.Context, profileID, sqlText string, limit, offset int) (*driver.QueryResult, error) {
	if strings.TrimSpace(profileID) == "" {
		return nil, fmt.Errorf("profile is required")
	}
	if strings.TrimSpace(sqlText) == "" {
		return nil, fmt.Errorf("query is empty")
	}
	limit, offset = driver.ClampLimitOffset(limit, offset)
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	conn, err := s.connections.ensureOpen(ctx, profileID)
	if err != nil {
		return nil, err
	}
	return conn.Query(ctx, sqlText, limit, offset)
}

// ListTables returns tables and views in the given database.
func (s *QueryService) ListTables(ctx context.Context, profileID, database string) ([]driver.TableInfo, error) {
	if strings.TrimSpace(profileID) == "" {
		return nil, fmt.Errorf("profile is required")
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	conn, err := s.connections.ensureOpen(ctx, profileID)
	if err != nil {
		return nil, err
	}
	tables, err := conn.ListTables(ctx, strings.TrimSpace(database))
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

	conn, err := s.connections.ensureOpen(ctx, profileID)
	if err != nil {
		return nil, err
	}
	return conn.BrowseTable(ctx, req)
}
