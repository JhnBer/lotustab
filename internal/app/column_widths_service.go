package app

import (
	"fmt"
	"strings"

	"lotustab/internal/storage"
)

// ColumnWidthsService is a thin Wails binding for manual grid column widths.
type ColumnWidthsService struct {
	store *storage.ColumnWidthStore
}

// NewColumnWidthsService constructs the Wails column-widths service.
func NewColumnWidthsService(store *storage.ColumnWidthStore) *ColumnWidthsService {
	return &ColumnWidthsService{store: store}
}

// GetTableWidths returns saved manual widths for a database/schema/table scope.
// Schema may be empty. Missing cache yields an empty map.
func (s *ColumnWidthsService) GetTableWidths(database, schema, table string) (map[string]float64, error) {
	if s.store == nil {
		return nil, fmt.Errorf("column width store is not configured")
	}
	table = strings.TrimSpace(table)
	if table == "" {
		return map[string]float64{}, nil
	}
	return s.store.Get(storage.TableScopeKey(database, schema, table))
}

// SetTableColumnWidth persists one manual column width for a table scope.
func (s *ColumnWidthsService) SetTableColumnWidth(database, schema, table, column string, width float64) error {
	if s.store == nil {
		return fmt.Errorf("column width store is not configured")
	}
	table = strings.TrimSpace(table)
	column = strings.TrimSpace(column)
	if table == "" {
		return fmt.Errorf("table is required")
	}
	if column == "" {
		return fmt.Errorf("column is required")
	}
	return s.store.SetColumn(storage.TableScopeKey(database, schema, table), column, width)
}
