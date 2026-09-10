package export

import (
	"io"
	"strings"

	"lotustab/internal/core/driver"
)

const defaultInsertBatch = 500

type sqlDumpWriter struct {
	w         io.Writer
	table     string
	cols      string
	dialect   string
	batchSize int
	inBatch   int
}

func newSQLDumpWriter(w io.Writer, table, dialect string, cols []driver.Column, batchSize int) *sqlDumpWriter {
	if batchSize <= 0 {
		batchSize = defaultInsertBatch
	}
	quoted := make([]string, len(cols))
	for i, col := range cols {
		quoted[i] = quoteIdent(dialect, col.Name)
	}
	return &sqlDumpWriter{
		w:         w,
		table:     table,
		cols:      strings.Join(quoted, ", "),
		dialect:   dialect,
		batchSize: batchSize,
	}
}

func (s *sqlDumpWriter) writeHeader(ddl string, includeDrop, includeCreate, wrapTx, firstChunk bool) error {
	if wrapTx && s.dialect != DialectClickHouse {
		if _, err := io.WriteString(s.w, "BEGIN;\n"); err != nil {
			return err
		}
	}
	if firstChunk && includeDrop {
		if _, err := io.WriteString(s.w, "DROP TABLE IF EXISTS "+s.table+";\n"); err != nil {
			return err
		}
	}
	if firstChunk && includeCreate && strings.TrimSpace(ddl) != "" {
		if _, err := io.WriteString(s.w, strings.TrimSpace(ddl)+"\n"); err != nil {
			return err
		}
	}
	return nil
}

func (s *sqlDumpWriter) writeRow(row []any) error {
	if s.inBatch == 0 {
		if _, err := io.WriteString(s.w, "INSERT INTO "+s.table+" ("+s.cols+") VALUES\n"); err != nil {
			return err
		}
	} else {
		if _, err := io.WriteString(s.w, ",\n"); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(s.w, "("+s.formatRow(row)+")"); err != nil {
		return err
	}
	s.inBatch++
	if s.inBatch >= s.batchSize {
		return s.flushInsert()
	}
	return nil
}

func (s *sqlDumpWriter) formatRow(row []any) string {
	parts := make([]string, len(row))
	for i, v := range row {
		parts[i] = sqlLiteral(s.dialect, v)
	}
	return strings.Join(parts, ", ")
}

func (s *sqlDumpWriter) flushInsert() error {
	if s.inBatch == 0 {
		return nil
	}
	_, err := io.WriteString(s.w, ";\n")
	s.inBatch = 0
	return err
}

func (s *sqlDumpWriter) writeFooter(wrapTx bool) error {
	if err := s.flushInsert(); err != nil {
		return err
	}
	if wrapTx && s.dialect != DialectClickHouse {
		if _, err := io.WriteString(s.w, "COMMIT;\n"); err != nil {
			return err
		}
	}
	return nil
}
