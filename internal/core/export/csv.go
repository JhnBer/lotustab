package export

import (
	"encoding/csv"
	"io"

	"lotustab/internal/core/driver"
)

type csvDumpWriter struct {
	w      *csv.Writer
	header []string
	wrote  bool
}

func newCSVDumpWriter(w io.Writer, cols []driver.Column) *csvDumpWriter {
	header := make([]string, len(cols))
	for i, col := range cols {
		header[i] = col.Name
	}
	return &csvDumpWriter{w: csv.NewWriter(w), header: header}
}

func (c *csvDumpWriter) writeHeader() error {
	if c.wrote {
		return nil
	}
	if err := c.w.Write(c.header); err != nil {
		return err
	}
	c.wrote = true
	return nil
}

func (c *csvDumpWriter) writeRow(row []any) error {
	if err := c.writeHeader(); err != nil {
		return err
	}
	record := make([]string, len(c.header))
	for i := range record {
		if i < len(row) {
			record[i] = csvCell(row[i])
		}
	}
	return c.w.Write(record)
}

func (c *csvDumpWriter) flush() error {
	c.w.Flush()
	return c.w.Error()
}
