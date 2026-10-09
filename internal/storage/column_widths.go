package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const columnWidthsFile = "column-widths.json"

// ColumnWidthStore persists manual grid column widths in the user config directory.
// The file holds only opaque scope hashes and pixel widths — never passwords or DSNs.
type ColumnWidthStore struct {
	path string
	mu   sync.Mutex
}

type columnWidthsPayload struct {
	Tables map[string]map[string]float64 `json:"tables"`
}

// NewColumnWidths creates a store at os.UserConfigDir()/lotustab/column-widths.json.
func NewColumnWidths() (*ColumnWidthStore, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("user config dir: %w", err)
	}
	appDir := filepath.Join(dir, appDirName)
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}
	return NewColumnWidthsAt(filepath.Join(appDir, columnWidthsFile)), nil
}

// NewColumnWidthsAt creates a store that reads and writes the given JSON file.
func NewColumnWidthsAt(path string) *ColumnWidthStore {
	return &ColumnWidthStore{path: path}
}

// TableScopeKey returns a stable opaque key for a database/schema/table triple.
// Schema is part of the hash material when non-empty so schemas do not share widths.
func TableScopeKey(database, schema, table string) string {
	database = strings.TrimSpace(database)
	schema = strings.TrimSpace(schema)
	table = strings.TrimSpace(table)
	material := database + "\x00" + schema + "\x00" + table
	sum := sha256.Sum256([]byte(material))
	return hex.EncodeToString(sum[:])
}

// Get returns saved manual widths for a scope key. Missing file or key yields an empty map.
func (s *ColumnWidthStore) Get(scopeKey string) (map[string]float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	payload, err := s.read()
	if err != nil {
		return nil, err
	}
	widths := payload.Tables[scopeKey]
	if len(widths) == 0 {
		return map[string]float64{}, nil
	}
	out := make(map[string]float64, len(widths))
	for name, width := range widths {
		out[name] = width
	}
	return out, nil
}

// SetColumn stores one manual column width under a scope key.
func (s *ColumnWidthStore) SetColumn(scopeKey, column string, width float64) error {
	scopeKey = strings.TrimSpace(scopeKey)
	column = strings.TrimSpace(column)
	if scopeKey == "" {
		return fmt.Errorf("scope key is required")
	}
	if column == "" {
		return fmt.Errorf("column is required")
	}
	if width < 1 {
		return fmt.Errorf("width must be positive")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	payload, err := s.read()
	if err != nil {
		return err
	}
	if payload.Tables[scopeKey] == nil {
		payload.Tables[scopeKey] = map[string]float64{}
	}
	payload.Tables[scopeKey][column] = width
	return s.write(payload)
}

func (s *ColumnWidthStore) read() (columnWidthsPayload, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return columnWidthsPayload{Tables: map[string]map[string]float64{}}, nil
		}
		return columnWidthsPayload{}, fmt.Errorf("read column widths: %w", err)
	}
	var payload columnWidthsPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return columnWidthsPayload{}, fmt.Errorf("parse column widths: %w", err)
	}
	if payload.Tables == nil {
		payload.Tables = map[string]map[string]float64{}
	}
	return payload, nil
}

func (s *ColumnWidthStore) write(payload columnWidthsPayload) error {
	if payload.Tables == nil {
		payload.Tables = map[string]map[string]float64{}
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("encode column widths: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write column widths: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("replace column widths: %w", err)
	}
	return nil
}
