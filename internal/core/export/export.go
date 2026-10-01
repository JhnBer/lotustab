package export

import (
	"context"
	"fmt"
	"strings"
	"time"

	"lotustab/internal/core/driver"
)

type Options struct {
	Format          string
	Gzip            bool
	ChunkSize       int
	IncludeCreate   bool
	IncludeDrop     bool
	InsertBatchSize int
	WrapTransaction bool
	Dialect         string
	Database        string
	Schema          string
	Table           string
	DDL             string
	OutputPath      string
	OutputDir       string
	FileBase        string
	DeleteOnAbort   bool
}

type Progress struct {
	Chunk int   `json:"chunk"`
	Rows  int64 `json:"rows"`
	Bytes int64 `json:"bytes"`
}

type Result struct {
	Files      []string `json:"files"`
	Rows       int64    `json:"rows"`
	Bytes      int64    `json:"bytes"`
	DurationMs int64    `json:"durationMs"`
}

type StreamFunc func(fn func(cols []driver.Column, rows [][]any) error) error

func Run(ctx context.Context, stream StreamFunc, opts Options, progress func(Progress)) (*Result, error) {
	if stream == nil {
		return nil, fmt.Errorf("stream is required")
	}
	format := strings.ToLower(strings.TrimSpace(opts.Format))
	if format == "" {
		format = FormatSQL
	}
	if format != FormatSQL && format != FormatCSV {
		return nil, fmt.Errorf("unsupported format %q", opts.Format)
	}
	dialect := strings.ToLower(strings.TrimSpace(opts.Dialect))
	if dialect == "" {
		dialect = DialectPostgres
	}
	table := qualifiedTable(dialect, opts.Database, opts.Schema, opts.Table)
	base := strings.TrimSpace(opts.FileBase)
	if base == "" {
		base = safeFileBase(opts.Table)
	}
	ext := fileExt(format, opts.Gzip)
	chunkSize := opts.ChunkSize
	if chunkSize < 0 {
		chunkSize = 0
	}

	ch := newChunker(opts.OutputDir, opts.OutputPath, base, ext, opts.Gzip, chunkSize)
	if err := ch.Open(); err != nil {
		return nil, err
	}
	ok := false
	defer func() {
		if !ok {
			ch.Abort()
			if opts.DeleteOnAbort {
				ch.DeleteChunks()
			}
		}
	}()

	start := time.Now()
	var (
		totalRows int64
		sqlW      *sqlDumpWriter
		csvW      *csvDumpWriter
		cols      []driver.Column
		lastEmit  time.Time
	)
	emit := func() {
		if progress == nil {
			return
		}
		now := time.Now()
		if !lastEmit.IsZero() && now.Sub(lastEmit) < 250*time.Millisecond {
			return
		}
		lastEmit = now
		progress(Progress{Chunk: ch.Chunk(), Rows: totalRows, Bytes: ch.Bytes()})
	}

	startChunk := func(first bool) error {
		switch format {
		case FormatCSV:
			csvW = newCSVDumpWriter(ch.Writer(), cols)
			return csvW.writeHeader()
		default:
			sqlW = newSQLDumpWriter(ch.Writer(), table, dialect, cols, opts.InsertBatchSize)
			return sqlW.writeHeader(opts.DDL, opts.IncludeDrop, opts.IncludeCreate, opts.WrapTransaction, first)
		}
	}
	finishChunk := func() error {
		switch format {
		case FormatCSV:
			if csvW != nil {
				return csvW.flush()
			}
		default:
			if sqlW != nil {
				return sqlW.writeFooter(opts.WrapTransaction)
			}
		}
		return nil
	}

	err := stream(func(nextCols []driver.Column, rows [][]any) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if cols == nil {
			cols = nextCols
			if err := startChunk(true); err != nil {
				return err
			}
		}
		for _, row := range rows {
			if ch.ShouldRotate() {
				if err := finishChunk(); err != nil {
					return err
				}
				if err := ch.Rotate(); err != nil {
					return err
				}
				if err := startChunk(false); err != nil {
					return err
				}
			}
			if format == FormatCSV {
				if err := csvW.writeRow(row); err != nil {
					return err
				}
			} else {
				if err := sqlW.writeRow(row); err != nil {
					return err
				}
			}
			ch.AddRows(1)
			totalRows++
			emit()
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if cols == nil {
		cols = []driver.Column{}
		if err := startChunk(true); err != nil {
			return nil, err
		}
	}
	if err := finishChunk(); err != nil {
		return nil, err
	}
	if err := ch.Close(); err != nil {
		return nil, err
	}
	ok = true
	if progress != nil {
		progress(Progress{Chunk: ch.Chunk(), Rows: totalRows, Bytes: ch.Bytes()})
	}
	return &Result{
		Files:      ch.Files(),
		Rows:       totalRows,
		Bytes:      ch.Bytes(),
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

// MultiTable is one table in a single-file SQL dump.
type MultiTable struct {
	Database string
	Schema   string
	Table    string
	DDL      string
	Stream   StreamFunc
}

// RunMulti writes several tables into one SQL file (gzip optional).
func RunMulti(ctx context.Context, tables []MultiTable, opts Options, progress func(Progress)) (*Result, error) {
	if len(tables) == 0 {
		return nil, fmt.Errorf("at least one table is required")
	}
	for i, t := range tables {
		if t.Stream == nil {
			return nil, fmt.Errorf("stream is required for table %d", i)
		}
		if strings.TrimSpace(t.Table) == "" {
			return nil, fmt.Errorf("table is required")
		}
	}
	format := strings.ToLower(strings.TrimSpace(opts.Format))
	if format == "" {
		format = FormatSQL
	}
	if format != FormatSQL {
		return nil, fmt.Errorf("single-file export requires SQL format")
	}
	dialect := strings.ToLower(strings.TrimSpace(opts.Dialect))
	if dialect == "" {
		dialect = DialectPostgres
	}
	wrapTx := opts.WrapTransaction && dialect != DialectClickHouse
	base := strings.TrimSpace(opts.FileBase)
	if base == "" {
		base = safeFileBase(opts.Table)
		if base == "table" {
			base = safeFileBase(tables[0].Table)
		}
	}
	ext := fileExt(format, opts.Gzip)

	ch := newChunker(opts.OutputDir, opts.OutputPath, base, ext, opts.Gzip, 0)
	if err := ch.Open(); err != nil {
		return nil, err
	}
	ok := false
	defer func() {
		if !ok {
			ch.Abort()
			if opts.DeleteOnAbort {
				ch.DeleteChunks()
			}
		}
	}()

	start := time.Now()
	var (
		totalRows int64
		lastEmit  time.Time
	)
	emit := func() {
		if progress == nil {
			return
		}
		now := time.Now()
		if !lastEmit.IsZero() && now.Sub(lastEmit) < 250*time.Millisecond {
			return
		}
		lastEmit = now
		progress(Progress{Chunk: ch.Chunk(), Rows: totalRows, Bytes: ch.Bytes()})
	}

	if wrapTx {
		if _, err := fmt.Fprint(ch.Writer(), "BEGIN;\n"); err != nil {
			return nil, err
		}
	}

	for i, t := range tables {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if i > 0 {
			if _, err := fmt.Fprint(ch.Writer(), "\n"); err != nil {
				return nil, err
			}
		}
		table := qualifiedTable(dialect, t.Database, t.Schema, t.Table)
		var (
			sqlW *sqlDumpWriter
			cols []driver.Column
		)
		err := t.Stream(func(nextCols []driver.Column, rows [][]any) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if cols == nil {
				cols = nextCols
				sqlW = newSQLDumpWriter(ch.Writer(), table, dialect, cols, opts.InsertBatchSize)
				if err := sqlW.writeHeader(t.DDL, opts.IncludeDrop, opts.IncludeCreate, false, true); err != nil {
					return err
				}
			}
			for _, row := range rows {
				if err := sqlW.writeRow(row); err != nil {
					return err
				}
				ch.AddRows(1)
				totalRows++
				emit()
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		if cols == nil {
			cols = []driver.Column{}
			sqlW = newSQLDumpWriter(ch.Writer(), table, dialect, cols, opts.InsertBatchSize)
			if err := sqlW.writeHeader(t.DDL, opts.IncludeDrop, opts.IncludeCreate, false, true); err != nil {
				return nil, err
			}
		}
		if err := sqlW.writeFooter(false); err != nil {
			return nil, err
		}
	}

	if wrapTx {
		if _, err := fmt.Fprint(ch.Writer(), "COMMIT;\n"); err != nil {
			return nil, err
		}
	}
	if err := ch.Close(); err != nil {
		return nil, err
	}
	ok = true
	if progress != nil {
		progress(Progress{Chunk: ch.Chunk(), Rows: totalRows, Bytes: ch.Bytes()})
	}
	return &Result{
		Files:      ch.Files(),
		Rows:       totalRows,
		Bytes:      ch.Bytes(),
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}
