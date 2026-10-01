package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"lotustab/internal/core/driver"
	"lotustab/internal/core/export"
)

const (
	exportBatchProgressEvent = "export:batch-progress"
	errDirNotEmptyMsg        = "dir_not_empty"
)

// ErrDirNotEmpty is returned when a folder dump would overwrite existing files.
var ErrDirNotEmpty = errors.New(errDirNotEmptyMsg)

// CountTablesRequest is the payload for counting rows across tables.
type CountTablesRequest struct {
	ProfileID string            `json:"profileId"`
	Database  string            `json:"database"`
	Tables    []driver.TableRef `json:"tables"`
}

// TableRowCount is SELECT COUNT(*) for one table.
type TableRowCount struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`
	Rows   int64  `json:"rows"`
}

// BatchExportRequest dumps several tables from one database.
type BatchExportRequest struct {
	ProfileID       string            `json:"profileId"`
	Database        string            `json:"database"`
	Tables          []driver.TableRef `json:"tables"`
	Format          string            `json:"format"`
	Gzip            bool              `json:"gzip"`
	SingleFile      bool              `json:"singleFile"`
	OutputDir       string            `json:"outputDir"`
	FileName        string            `json:"fileName"`
	FolderName      string            `json:"folderName"`
	Prefix          string            `json:"prefix"`
	PrefixEnabled   bool              `json:"prefixEnabled"`
	Force           bool              `json:"force"`
	IncludeCreate   bool              `json:"includeCreate"`
	IncludeDrop     bool              `json:"includeDrop"`
	InsertBatchSize int               `json:"insertBatchSize"`
	WrapTransaction bool              `json:"wrapTransaction"`
	DeleteOnAbort   bool              `json:"deleteOnAbort"`
}

// BatchExportProgress is emitted on export:batch-progress while a dump runs.
type BatchExportProgress struct {
	TableIndex int    `json:"tableIndex"`
	TableTotal int    `json:"tableTotal"`
	TableName  string `json:"tableName"`
	TableRows  int64  `json:"tableRows"`
	TotalRows  int64  `json:"totalRows"`
	TotalBytes int64  `json:"totalBytes"`
	Query      string `json:"query,omitempty"`
}

// OutputStatus describes whether a dump destination already has data.
type OutputStatus struct {
	Exists   bool `json:"exists"`
	HasFiles bool `json:"hasFiles"`
}

// CheckOutput reports whether a file or folder already exists / has contents.
func (s *ExportService) CheckOutput(path string, folder bool) (*OutputStatus, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return &OutputStatus{}, nil
	}
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return &OutputStatus{Exists: false, HasFiles: false}, nil
	}
	if err != nil {
		return nil, err
	}
	if !folder {
		return &OutputStatus{Exists: true, HasFiles: true}, nil
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory")
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	return &OutputStatus{Exists: true, HasFiles: len(entries) > 0}, nil
}

// CountTables returns SELECT COUNT(*) for each table.
func (s *ExportService) CountTables(ctx context.Context, req CountTablesRequest) ([]TableRowCount, error) {
	if strings.TrimSpace(req.ProfileID) == "" {
		return nil, fmt.Errorf("profile is required")
	}
	tables := clampExportTables(req.Tables)
	if len(tables) == 0 {
		return nil, fmt.Errorf("at least one table is required")
	}
	profile, err := s.connections.store.Get(req.ProfileID)
	if err != nil {
		return nil, err
	}
	dialect := strings.ToLower(strings.TrimSpace(profile.DriverID))
	database := strings.TrimSpace(req.Database)

	return withTrackedConn(s.connections, ctx, req.ProfileID, func(ctx context.Context, conn driver.Conn) ([]TableRowCount, error) {
		out := make([]TableRowCount, 0, len(tables))
		for _, t := range tables {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			query := export.CountQuery(dialect, database, t.Schema, t.Name, "")
			result, err := conn.Query(ctx, query, database, 1, 0)
			if err != nil {
				return nil, err
			}
			n, ok := countFromResult(result)
			if !ok {
				return nil, fmt.Errorf("count query returned no rows for %s", t.Name)
			}
			out = append(out, TableRowCount{Schema: t.Schema, Name: t.Name, Rows: n})
		}
		return out, nil
	})
}

// ExportTables dumps the selected tables sequentially into one file or a folder.
func (s *ExportService) ExportTables(ctx context.Context, req BatchExportRequest) (*ExportResult, error) {
	if strings.TrimSpace(req.ProfileID) == "" {
		return nil, fmt.Errorf("profile is required")
	}
	req.Tables = clampExportTables(req.Tables)
	if len(req.Tables) == 0 {
		return nil, fmt.Errorf("at least one table is required")
	}
	req.Format = strings.ToLower(strings.TrimSpace(req.Format))
	if req.Format == "" {
		req.Format = export.FormatSQL
	}
	if req.Format != export.FormatSQL && req.Format != export.FormatCSV {
		return nil, fmt.Errorf("unsupported format %q", req.Format)
	}
	if req.Format == export.FormatCSV {
		req.SingleFile = false
	}
	if req.InsertBatchSize <= 0 {
		req.InsertBatchSize = 500
	}
	if req.InsertBatchSize > 5000 {
		req.InsertBatchSize = 5000
	}

	if !s.begin() {
		return nil, fmt.Errorf("an export is already running")
	}
	defer s.end()

	if ctx.Err() != nil {
		return &ExportResult{Canceled: true}, nil
	}

	profile, err := s.connections.store.Get(req.ProfileID)
	if err != nil {
		return nil, err
	}
	dialect := strings.ToLower(strings.TrimSpace(profile.DriverID))
	wrapTx := req.WrapTransaction && dialect != export.DialectClickHouse
	database := strings.TrimSpace(req.Database)
	prefix := ""
	if req.PrefixEnabled {
		prefix = req.Prefix
	}

	destDir, destPath, err := resolveBatchDest(req)
	if err != nil {
		return nil, err
	}

	ctx, done := s.connections.Track(ctx, req.ProfileID)
	defer done()
	conn, err := s.connections.ensureOpen(ctx, req.ProfileID)
	if err != nil {
		return nil, err
	}

	var (
		allFiles []string
		ok       bool
	)
	defer func() {
		if ok || !req.DeleteOnAbort {
			return
		}
		for _, path := range allFiles {
			_ = os.Remove(path)
		}
	}()

	if req.SingleFile {
		result, runErr := s.exportTablesSingleFile(ctx, conn, req, dialect, wrapTx, database, destPath)
		if runErr != nil {
			if errors.Is(runErr, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
				return &ExportResult{Canceled: true}, nil
			}
			return nil, runErr
		}
		allFiles = result.Files
		ok = true
		return result, nil
	}

	result, runErr := s.exportTablesFolder(ctx, conn, req, dialect, wrapTx, database, destDir, prefix)
	if runErr != nil {
		allFiles = result.Files
		if errors.Is(runErr, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
			return &ExportResult{Files: result.Files, Rows: result.Rows, Bytes: result.Bytes, Canceled: true}, nil
		}
		return nil, runErr
	}
	allFiles = result.Files
	ok = true
	return result, nil
}

func (s *ExportService) exportTablesSingleFile(
	ctx context.Context,
	conn driver.Conn,
	req BatchExportRequest,
	dialect string,
	wrapTx bool,
	database, destPath string,
) (*ExportResult, error) {
	tables := make([]export.MultiTable, 0, len(req.Tables))
	for _, t := range req.Tables {
		ref := t
		if strings.TrimSpace(ref.Database) == "" {
			ref.Database = database
		}
		item := export.MultiTable{
			Database: ref.Database,
			Schema:   ref.Schema,
			Table:    ref.Name,
			Stream: func(fn func(cols []driver.Column, rows [][]any) error) error {
				streamReq := driver.ClampStreamTable(driver.StreamTableRequest{TableRef: ref})
				return conn.StreamTable(ctx, streamReq, fn)
			},
		}
		if req.IncludeCreate {
			ddl, err := conn.TableDDL(ctx, ref)
			if err != nil {
				return nil, err
			}
			item.DDL = ddl
		}
		tables = append(tables, item)
	}

	query := fmt.Sprintf("export %d tables", len(req.Tables))
	emitExportBatchProgress(BatchExportProgress{TableIndex: 1, TableTotal: len(req.Tables), TableName: req.Tables[0].Name, Query: query})

	out, err := export.RunMulti(ctx, tables, export.Options{
		Format:          export.FormatSQL,
		Gzip:            req.Gzip,
		IncludeCreate:   req.IncludeCreate,
		IncludeDrop:     req.IncludeDrop,
		InsertBatchSize: req.InsertBatchSize,
		WrapTransaction: wrapTx,
		Dialect:         dialect,
		Database:        database,
		Table:           req.Tables[0].Name,
		OutputPath:      destPath,
		FileBase:        strings.TrimSuffix(filepath.Base(destPath), extOf(filepath.Base(destPath))),
		DeleteOnAbort:   req.DeleteOnAbort,
	}, func(p export.Progress) {
		name := req.Tables[len(req.Tables)-1].Name
		if p.Chunk > 0 && p.Chunk <= len(req.Tables) {
			name = req.Tables[p.Chunk-1].Name
		}
		emitExportBatchProgress(BatchExportProgress{
			TableIndex: 1,
			TableTotal: len(req.Tables),
			TableName:  name,
			TableRows:  p.Rows,
			TotalRows:  p.Rows,
			TotalBytes: p.Bytes,
			Query:      query,
		})
	})
	if err != nil {
		return nil, err
	}
	files := out.Files
	if files == nil {
		files = []string{}
	}
	return &ExportResult{
		Files:      files,
		Rows:       out.Rows,
		Bytes:      out.Bytes,
		DurationMs: out.DurationMs,
	}, nil
}

func (s *ExportService) exportTablesFolder(
	ctx context.Context,
	conn driver.Conn,
	req BatchExportRequest,
	dialect string,
	wrapTx bool,
	database, destDir, prefix string,
) (*ExportResult, error) {
	var (
		allFiles   []string
		totalRows  int64
		totalBytes int64
		started    = time.Now()
	)
	ext := ".sql"
	if req.Format == export.FormatCSV {
		ext = ".csv"
	}
	if req.Gzip {
		ext += ".gz"
	}

	for i, t := range req.Tables {
		if err := ctx.Err(); err != nil {
			return &ExportResult{Files: allFiles, Rows: totalRows, Bytes: totalBytes}, err
		}
		ref := t
		if strings.TrimSpace(ref.Database) == "" {
			ref.Database = database
		}
		streamReq := driver.ClampStreamTable(driver.StreamTableRequest{TableRef: ref})
		query := export.SelectQuery(dialect, streamReq.Database, streamReq.Schema, streamReq.Name, "")
		emitExportBatchProgress(BatchExportProgress{
			TableIndex: i + 1,
			TableTotal: len(req.Tables),
			TableName:  t.Name,
			TotalRows:  totalRows,
			TotalBytes: totalBytes,
			Query:      query,
		})

		base := dumpFileBase(prefix, req.Tables, i)
		path := filepath.Join(destDir, base+ext)
		opts := export.Options{
			Format:          req.Format,
			Gzip:            req.Gzip,
			IncludeCreate:   req.Format == export.FormatSQL && req.IncludeCreate,
			IncludeDrop:     req.Format == export.FormatSQL && req.IncludeDrop,
			InsertBatchSize: req.InsertBatchSize,
			WrapTransaction: wrapTx,
			Dialect:         dialect,
			Database:        streamReq.Database,
			Schema:          streamReq.Schema,
			Table:           streamReq.Name,
			OutputPath:      path,
			FileBase:        base,
			DeleteOnAbort:   req.DeleteOnAbort,
		}
		if opts.IncludeCreate {
			ddl, err := conn.TableDDL(ctx, streamReq.TableRef)
			if err != nil {
				return &ExportResult{Files: allFiles, Rows: totalRows, Bytes: totalBytes}, err
			}
			opts.DDL = ddl
		}

		var tableRows int64
		out, err := export.Run(ctx, func(fn func(cols []driver.Column, rows [][]any) error) error {
			return conn.StreamTable(ctx, streamReq, fn)
		}, opts, func(p export.Progress) {
			tableRows = p.Rows
			emitExportBatchProgress(BatchExportProgress{
				TableIndex: i + 1,
				TableTotal: len(req.Tables),
				TableName:  t.Name,
				TableRows:  p.Rows,
				TotalRows:  totalRows + p.Rows,
				TotalBytes: totalBytes + p.Bytes,
				Query:      query,
			})
		})
		if err != nil {
			return &ExportResult{Files: allFiles, Rows: totalRows, Bytes: totalBytes}, err
		}
		if out != nil {
			allFiles = append(allFiles, out.Files...)
			totalRows += out.Rows
			totalBytes += out.Bytes
			tableRows = out.Rows
		}
		emitExportBatchProgress(BatchExportProgress{
			TableIndex: i + 1,
			TableTotal: len(req.Tables),
			TableName:  t.Name,
			TableRows:  tableRows,
			TotalRows:  totalRows,
			TotalBytes: totalBytes,
			Query:      query,
		})
	}

	if allFiles == nil {
		allFiles = []string{}
	}
	return &ExportResult{
		Files:      allFiles,
		Rows:       totalRows,
		Bytes:      totalBytes,
		DurationMs: time.Since(started).Milliseconds(),
	}, nil
}

func dumpFileBase(prefix string, tables []driver.TableRef, i int) string {
	t := tables[i]
	name := t.Name
	for j, other := range tables {
		if j != i && other.Name == t.Name && t.Schema != "" {
			name = t.Schema + "_" + t.Name
			break
		}
	}
	return export.FileBaseForTable(prefix, name)
}

func clampExportTables(in []driver.TableRef) []driver.TableRef {
	out := make([]driver.TableRef, 0, len(in))
	for _, t := range in {
		t = driver.ClampTableRef(t)
		if t.Name == "" {
			continue
		}
		out = append(out, t)
	}
	return out
}

func resolveBatchDest(req BatchExportRequest) (dir, path string, err error) {
	if req.SingleFile {
		name := filepath.Base(strings.TrimSpace(req.FileName))
		folder := strings.TrimSpace(req.OutputDir)
		if name == "" {
			name = exportFileName(req.Tables[0].Name, req.Format, req.Gzip)
		}
		if extOf(name) == "" {
			name += fileExt(req.Format, req.Gzip)
		}
		if folder == "" {
			return "", "", fmt.Errorf("output directory is required")
		}
		path = filepath.Join(folder, name)
		if !req.Force {
			if st, stErr := os.Stat(path); stErr == nil && !st.IsDir() {
				return "", "", ErrDirNotEmpty
			} else if stErr != nil && !errors.Is(stErr, os.ErrNotExist) {
				return "", "", stErr
			}
		}
		return folder, path, nil
	}
	dir, err = resolveFolderDest(req.OutputDir, req.FolderName, req.Force)
	return dir, "", err
}

func resolveFolderDest(outputDir, folderName string, force bool) (string, error) {
	dir := strings.TrimSpace(outputDir)
	if dir == "" {
		return "", fmt.Errorf("output directory is required")
	}
	name := strings.TrimSpace(folderName)
	if name != "" {
		dir = filepath.Join(dir, sanitizeFilePart(name))
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	if len(entries) > 0 && !force {
		return "", ErrDirNotEmpty
	}
	return dir, nil
}

func fileExt(format string, gzipOn bool) string {
	ext := ".sql"
	if format == export.FormatCSV {
		ext = ".csv"
	}
	if gzipOn {
		return ext + ".gz"
	}
	return ext
}

func emitExportBatchProgress(p BatchExportProgress) {
	if appl := application.Get(); appl != nil {
		appl.Event.Emit(exportBatchProgressEvent, p)
	}
}
