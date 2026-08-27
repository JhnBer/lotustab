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
func (s *QueryService) Execute(ctx context.Context, profileID, sqlText string) (*driver.QueryResult, error) {
	if strings.TrimSpace(profileID) == "" {
		return nil, fmt.Errorf("profile is required")
	}
	if strings.TrimSpace(sqlText) == "" {
		return nil, fmt.Errorf("query is empty")
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	conn, err := s.connections.ensureOpen(ctx, profileID)
	if err != nil {
		return nil, err
	}
	return conn.Query(ctx, sqlText)
}
