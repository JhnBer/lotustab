package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lotustab/internal/core/driver"
	"lotustab/internal/core/export"
)

// SaveResultItem is one in-memory result set to write as SQL or JSON.
type SaveResultItem struct {
	Index     int             `json:"index"`
	SQL       string          `json:"sql"`
	FileBase  string          `json:"fileBase"`
	TableName string          `json:"tableName"`
	Columns   []driver.Column `json:"columns"`
	Rows      [][]any         `json:"rows"`
}

// SaveResultSetsRequest writes query results already in memory.
type SaveResultSetsRequest struct {
	ProfileID  string           `json:"profileId"`
	Format     string           `json:"format"`
	SingleFile bool             `json:"singleFile"`
	OutputDir  string           `json:"outputDir"`
	FileName   string           `json:"fileName"`
	FolderName string           `json:"folderName"`
	Force      bool             `json:"force"`
	Items      []SaveResultItem `json:"items"`
}

// SaveResultSets writes in-memory query results as SQL INSERTs or JSON.
func (s *ExportService) SaveResultSets(req SaveResultSetsRequest) (*ExportResult, error) {
	req.Format = strings.ToLower(strings.TrimSpace(req.Format))
	if req.Format == "" {
		req.Format = export.FormatSQL
	}
	if req.Format != export.FormatSQL && req.Format != export.FormatJSON {
		return nil, fmt.Errorf("unsupported format %q", req.Format)
	}
	if len(req.Items) == 0 {
		return nil, fmt.Errorf("at least one result set is required")
	}
	if len(req.Items) == 1 {
		req.SingleFile = true
	}
	if strings.TrimSpace(req.ProfileID) == "" {
		return nil, fmt.Errorf("profile is required")
	}
	profile, err := s.connections.store.Get(req.ProfileID)
	if err != nil {
		return nil, err
	}
	dialect := strings.ToLower(strings.TrimSpace(profile.DriverID))

	if !s.begin() {
		return nil, fmt.Errorf("an export is already running")
	}
	defer s.end()

	started := time.Now()
	if req.SingleFile {
		path, err := resolveSaveFile(req)
		if err != nil {
			return nil, err
		}
		body, rows, err := encodeResultSets(req.Format, dialect, req.Items)
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, body, 0o644); err != nil {
			return nil, err
		}
		info, _ := os.Stat(path)
		var bytes int64
		if info != nil {
			bytes = info.Size()
		}
		return &ExportResult{
			Files:      []string{path},
			Rows:       rows,
			Bytes:      bytes,
			DurationMs: time.Since(started).Milliseconds(),
		}, nil
	}

	dir, err := resolveFolderDest(req.OutputDir, req.FolderName, req.Force)
	if err != nil {
		return nil, err
	}
	ext := saveExt(req.Format)
	var (
		files []string
		rows  int64
		bytes int64
	)
	for _, item := range req.Items {
		body, n, err := encodeResultSets(req.Format, dialect, []SaveResultItem{item})
		if err != nil {
			return nil, err
		}
		base := strings.TrimSpace(item.FileBase)
		if base == "" {
			base = fmt.Sprintf("query_%d", item.Index+1)
		}
		base = export.SafeFileBase(base)
		path := filepath.Join(dir, base+ext)
		if !req.Force {
			if st, stErr := os.Stat(path); stErr == nil && !st.IsDir() {
				return nil, ErrDirNotEmpty
			} else if stErr != nil && !errors.Is(stErr, os.ErrNotExist) {
				return nil, stErr
			}
		}
		if err := os.WriteFile(path, body, 0o644); err != nil {
			return nil, err
		}
		info, _ := os.Stat(path)
		if info != nil {
			bytes += info.Size()
		}
		files = append(files, path)
		rows += n
	}
	if files == nil {
		files = []string{}
	}
	return &ExportResult{
		Files:      files,
		Rows:       rows,
		Bytes:      bytes,
		DurationMs: time.Since(started).Milliseconds(),
	}, nil
}

func resolveSaveFile(req SaveResultSetsRequest) (string, error) {
	folder := strings.TrimSpace(req.OutputDir)
	if folder == "" {
		return "", fmt.Errorf("output directory is required")
	}
	name := filepath.Base(strings.TrimSpace(req.FileName))
	if name == "" {
		name = exportFileName("query", req.Format, false)
	}
	if extOf(name) == "" {
		name += saveExt(req.Format)
	}
	path := filepath.Join(folder, name)
	if !req.Force {
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			return "", ErrDirNotEmpty
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
	}
	return path, nil
}

func encodeResultSets(format, dialect string, items []SaveResultItem) ([]byte, int64, error) {
	var rows int64
	for _, item := range items {
		rows += int64(len(item.Rows))
	}
	if format == export.FormatJSON {
		if len(items) == 1 {
			body, err := export.JSONRows(items[0].Columns, items[0].Rows)
			return body, rows, err
		}
		sets := make([]export.JSONSet, len(items))
		for i, item := range items {
			sets[i] = export.JSONSet{
				Index:   item.Index,
				SQL:     item.SQL,
				Columns: item.Columns,
				Rows:    item.Rows,
			}
		}
		body, err := export.JSONResultSets(sets)
		return body, rows, err
	}
	var b strings.Builder
	for i, item := range items {
		if i > 0 {
			b.WriteByte('\n')
		}
		if err := export.WriteInserts(&b, item.TableName, dialect, item.Columns, item.Rows, 500); err != nil {
			return nil, 0, err
		}
	}
	return []byte(b.String()), rows, nil
}

func saveExt(format string) string {
	if format == export.FormatJSON {
		return ".json"
	}
	return ".sql"
}
