package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/wailsapp/wails/v3/pkg/application"

	"lotustab/internal/core/driver"
	"lotustab/internal/core/export"
)

const exportProgressEvent = "export:progress"

// ExportTableRequest is the payload for a table dump.
type ExportTableRequest struct {
	ProfileID       string `json:"profileId"`
	Database        string `json:"database"`
	Schema          string `json:"schema"`
	Table           string `json:"table"`
	Format          string `json:"format"`
	Gzip            bool   `json:"gzip"`
	ChunkSize       int    `json:"chunkSize"`
	WhereSQL        string `json:"whereSQL"`
	IncludeCreate   bool   `json:"includeCreate"`
	IncludeDrop     bool   `json:"includeDrop"`
	InsertBatchSize int    `json:"insertBatchSize"`
	WrapTransaction bool   `json:"wrapTransaction"`
	OutputPath      string `json:"outputPath"`
	OutputDir       string `json:"outputDir"`
	FileName        string `json:"fileName"`
	FolderMode      bool   `json:"folderMode"`
	FolderName      string `json:"folderName"`
	Prefix          string `json:"prefix"`
	PrefixEnabled   bool   `json:"prefixEnabled"`
	Force           bool   `json:"force"`
	DeleteOnAbort   bool   `json:"deleteOnAbort"`
}

// ExportPathResult is the file or folder chosen for a dump.
type ExportPathResult struct {
	Path     string `json:"path"`
	Dir      string `json:"dir"`
	Canceled bool   `json:"canceled"`
}

// ExportResult is returned after a dump finishes or is canceled.
type ExportResult struct {
	Files      []string `json:"files"`
	Rows       int64    `json:"rows"`
	Bytes      int64    `json:"bytes"`
	DurationMs int64    `json:"durationMs"`
	Canceled   bool     `json:"canceled"`
}

// ExportProgress is emitted on export:progress while a dump runs.
type ExportProgress struct {
	Chunk int    `json:"chunk"`
	Rows  int64  `json:"rows"`
	Bytes int64  `json:"bytes"`
	Query string `json:"query,omitempty"`
}

// ExportService streams a table to SQL or CSV files.
type ExportService struct {
	connections *ConnectionsService
	mu          sync.Mutex
	busy        bool
}

// NewExportService constructs the Wails export service.
func NewExportService(connections *ConnectionsService) *ExportService {
	return &ExportService{connections: connections}
}

// ExportTable dumps one table to one or more files chosen via a native dialog.
func (s *ExportService) ExportTable(ctx context.Context, req ExportTableRequest) (*ExportResult, error) {
	if strings.TrimSpace(req.ProfileID) == "" {
		return nil, fmt.Errorf("profile is required")
	}
	req.Table = strings.TrimSpace(req.Table)
	if req.Table == "" {
		return nil, fmt.Errorf("table is required")
	}
	req.Format = strings.ToLower(strings.TrimSpace(req.Format))
	if req.Format == "" {
		req.Format = export.FormatSQL
	}
	if req.Format != export.FormatSQL && req.Format != export.FormatCSV {
		return nil, fmt.Errorf("unsupported format %q", req.Format)
	}
	if req.ChunkSize < 0 {
		req.ChunkSize = 0
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

	if req.FolderMode {
		dest, destErr := resolveFolderDest(req.OutputDir, req.FolderName, req.Force)
		if destErr != nil {
			return nil, destErr
		}
		prefix := ""
		if req.PrefixEnabled {
			prefix = req.Prefix
		}
		base := export.FileBaseForTable(prefix, req.Table)
		req.OutputDir = dest
		req.OutputPath = filepath.Join(dest, base+fileExt(req.Format, req.Gzip))
		req.FileName = filepath.Base(req.OutputPath)
		req.ChunkSize = 0
	}

	profile, err := s.connections.store.Get(req.ProfileID)
	if err != nil {
		return nil, err
	}
	dialect := strings.ToLower(strings.TrimSpace(profile.DriverID))
	wrapTx := req.WrapTransaction && dialect != export.DialectClickHouse

	streamReq := driver.ClampStreamTable(driver.StreamTableRequest{
		TableRef: driver.TableRef{
			Database: strings.TrimSpace(req.Database),
			Schema:   strings.TrimSpace(req.Schema),
			Name:     req.Table,
		},
		WhereSQL: strings.TrimSpace(req.WhereSQL),
	})

	query := export.SelectQuery(dialect, streamReq.Database, streamReq.Schema, streamReq.Name, streamReq.WhereSQL)
	log.Printf("export query: %s", query)
	emitExportProgress(ExportProgress{Query: query})

	suggested := exportFileName(req.Table, req.Format, req.Gzip)
	path, dir, fileBase, canceled, err := resolveExportTarget(req, suggested)
	if err != nil {
		return nil, err
	}
	if canceled {
		return &ExportResult{Canceled: true}, nil
	}

	opts := export.Options{
		Format:          req.Format,
		Gzip:            req.Gzip,
		ChunkSize:       req.ChunkSize,
		IncludeCreate:   req.IncludeCreate,
		IncludeDrop:     req.IncludeDrop,
		InsertBatchSize: req.InsertBatchSize,
		WrapTransaction: wrapTx,
		Dialect:         dialect,
		Database:        streamReq.Database,
		Schema:          streamReq.Schema,
		Table:           streamReq.Name,
		OutputPath:      path,
		OutputDir:       dir,
		FileBase:        fileBase,
		DeleteOnAbort:   req.DeleteOnAbort,
	}

	out, err := func() (*export.Result, error) {
		ctx, done := s.connections.Track(ctx, req.ProfileID)
		defer done()
		conn, err := s.connections.ensureOpen(ctx, req.ProfileID)
		if err != nil {
			return nil, err
		}
		if opts.IncludeCreate {
			ddl, ddlErr := conn.TableDDL(ctx, streamReq.TableRef)
			if ddlErr != nil {
				return nil, ddlErr
			}
			opts.DDL = ddl
		}
		return export.Run(ctx, func(fn func(cols []driver.Column, rows [][]any) error) error {
			return conn.StreamTable(ctx, streamReq, fn)
		}, opts, func(p export.Progress) {
			emitExportProgress(ExportProgress{Chunk: p.Chunk, Rows: p.Rows, Bytes: p.Bytes, Query: query})
		})
	}()
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
			return &ExportResult{Canceled: true}, nil
		}
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

// CountRows returns SELECT COUNT(*) for the table and optional WHERE clause.
func (s *ExportService) CountRows(ctx context.Context, profileID, database, schema, table, whereSQL string) (int64, error) {
	if strings.TrimSpace(profileID) == "" {
		return 0, fmt.Errorf("profile is required")
	}
	table = strings.TrimSpace(table)
	if table == "" {
		return 0, fmt.Errorf("table is required")
	}
	profile, err := s.connections.store.Get(profileID)
	if err != nil {
		return 0, err
	}
	dialect := strings.ToLower(strings.TrimSpace(profile.DriverID))
	database = strings.TrimSpace(database)
	schema = strings.TrimSpace(schema)
	query := export.CountQuery(dialect, database, schema, table, whereSQL)
	result, err := withTrackedConn(s.connections, ctx, profileID, func(ctx context.Context, conn driver.Conn) (*driver.QueryResult, error) {
		return conn.Query(ctx, query, database, 1, 0)
	})
	if err != nil {
		return 0, err
	}
	n, ok := countFromResult(result)
	if !ok {
		return 0, fmt.Errorf("count query returned no rows")
	}
	return n, nil
}

// ChoosePath opens a native folder dialog for an export.
func (s *ExportService) ChoosePath(table, format string, gzipOn, chunked bool) (*ExportPathResult, error) {
	suggested := exportFileName(table, strings.ToLower(strings.TrimSpace(format)), gzipOn)
	_, dir, canceled, err := pickExportTarget(true, suggested)
	if err != nil {
		return nil, err
	}
	return &ExportPathResult{Dir: dir, Canceled: canceled}, nil
}

func countFromResult(result *driver.QueryResult) (int64, bool) {
	if result == nil || len(result.Rows) == 0 || len(result.Rows[0]) == 0 {
		return 0, false
	}
	return asInt64(result.Rows[0][0])
}

func asInt64(v any) (int64, bool) {
	switch t := v.(type) {
	case int:
		return int64(t), true
	case int8:
		return int64(t), true
	case int16:
		return int64(t), true
	case int32:
		return int64(t), true
	case int64:
		return t, true
	case uint:
		return int64(t), true
	case uint8:
		return int64(t), true
	case uint16:
		return int64(t), true
	case uint32:
		return int64(t), true
	case uint64:
		if t > uint64(math.MaxInt64) {
			return math.MaxInt64, true
		}
		return int64(t), true
	case float32:
		return int64(t), true
	case float64:
		return int64(t), true
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		return n, err == nil
	case []byte:
		n, err := strconv.ParseInt(strings.TrimSpace(string(t)), 10, 64)
		return n, err == nil
	default:
		if v == nil {
			return 0, false
		}
		n, err := strconv.ParseInt(strings.TrimSpace(fmt.Sprint(t)), 10, 64)
		return n, err == nil
	}
}

func (s *ExportService) begin() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.busy {
		return false
	}
	s.busy = true
	return true
}

func (s *ExportService) end() {
	s.mu.Lock()
	s.busy = false
	s.mu.Unlock()
}

func resolveExportTarget(req ExportTableRequest, suggested string) (path, dir, fileBase string, canceled bool, err error) {
	name := filepath.Base(strings.TrimSpace(req.FileName))
	folder := strings.TrimSpace(req.OutputDir)
	path = strings.TrimSpace(req.OutputPath)
	if name != "" && extOf(name) == "" {
		name += extOf(suggested)
	}
	if name == "" {
		name = suggested
	}
	fileBase = strings.TrimSuffix(name, extOf(name))
	if fileBase == "" {
		fileBase = strings.TrimSuffix(suggested, extOf(suggested))
	}
	if folder != "" {
		if req.ChunkSize > 0 {
			return "", folder, fileBase, false, nil
		}
		return filepath.Join(folder, name), "", fileBase, false, nil
	}
	if path != "" {
		return path, "", fileBase, false, nil
	}
	path, dir, canceled, err = pickExportTarget(req.ChunkSize > 0, name)
	return path, dir, fileBase, canceled, err
}

func pickExportTarget(chunked bool, filename string) (path, dir string, canceled bool, err error) {
	appl := application.Get()
	if appl == nil {
		return "", "", false, fmt.Errorf("application is not running")
	}
	if chunked {
		chosen, dlgErr := appl.Dialog.OpenFile().
			CanChooseFiles(false).
			CanChooseDirectories(true).
			CanCreateDirectories(true).
			SetTitle("Choose export folder").
			PromptForSingleSelection()
		if dlgErr != nil {
			return "", "", false, dlgErr
		}
		if strings.TrimSpace(chosen) == "" {
			return "", "", true, nil
		}
		return "", chosen, false, nil
	}
	filterName, pattern := "SQL", "*.sql"
	if strings.HasSuffix(filename, ".csv.gz") {
		filterName, pattern = "Gzipped CSV", "*.csv.gz"
	} else if strings.HasSuffix(filename, ".csv") {
		filterName, pattern = "CSV", "*.csv"
	} else if strings.HasSuffix(filename, ".sql.gz") {
		filterName, pattern = "Gzipped SQL", "*.sql.gz"
	}
	chosen, dlgErr := appl.Dialog.SaveFile().
		SetFilename(filename).
		AddFilter(filterName, pattern).
		CanCreateDirectories(true).
		PromptForSingleSelection()
	if dlgErr != nil {
		return "", "", false, dlgErr
	}
	if strings.TrimSpace(chosen) == "" {
		return "", "", true, nil
	}
	return chosen, "", false, nil
}

func exportFileName(table, format string, gzipOn bool) string {
	base := sanitizeFilePart(table)
	stamp := time.Now().Format("20060102")
	ext := ".sql"
	if format == export.FormatCSV {
		ext = ".csv"
	}
	if gzipOn {
		ext += ".gz"
	}
	return fmt.Sprintf("%s_%s%s", base, stamp, ext)
}

func sanitizeFilePart(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "table"
	}
	var b strings.Builder
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '-' || r == '_' {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('_')
	}
	out := strings.Trim(b.String(), "._")
	if out == "" {
		return "table"
	}
	return out
}

func extOf(name string) string {
	switch {
	case strings.HasSuffix(name, ".sql.gz"):
		return ".sql.gz"
	case strings.HasSuffix(name, ".csv.gz"):
		return ".csv.gz"
	case strings.HasSuffix(name, ".sql"):
		return ".sql"
	case strings.HasSuffix(name, ".csv"):
		return ".csv"
	default:
		return ""
	}
}

func emitExportProgress(p ExportProgress) {
	if appl := application.Get(); appl != nil {
		appl.Event.Emit(exportProgressEvent, p)
	}
}
