package export

import (
	"bytes"
	"encoding/json"

	"lotustab/internal/core/driver"
)

// JSONSet is one in-memory result set for JSON output.
type JSONSet struct {
	Index   int
	SQL     string
	Columns []driver.Column
	Rows    [][]any
}

// JSONRows encodes rows as an array of objects with keys in column order.
func JSONRows(cols []driver.Column, rows [][]any) ([]byte, error) {
	var buf bytes.Buffer
	if err := writeJSONRowArray(&buf, cols, rows); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// JSONResultSets encodes several result sets as [{index, sql, rows}, ...].
func JSONResultSets(items []JSONSet) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('[')
	for i, item := range items {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.WriteString(`{"index":`)
		idx, err := marshalJSONValue(item.Index)
		if err != nil {
			return nil, err
		}
		buf.Write(idx)
		buf.WriteString(`,"sql":`)
		sql, err := marshalJSONValue(item.SQL)
		if err != nil {
			return nil, err
		}
		buf.Write(sql)
		buf.WriteString(`,"rows":`)
		if err := writeJSONRowArray(&buf, item.Columns, item.Rows); err != nil {
			return nil, err
		}
		buf.WriteByte('}')
	}
	buf.WriteByte(']')
	return buf.Bytes(), nil
}

func writeJSONRowArray(buf *bytes.Buffer, cols []driver.Column, rows [][]any) error {
	buf.WriteByte('[')
	for i, row := range rows {
		if i > 0 {
			buf.WriteByte(',')
		}
		if err := writeJSONObject(buf, cols, row); err != nil {
			return err
		}
	}
	buf.WriteByte(']')
	return nil
}

func writeJSONObject(buf *bytes.Buffer, cols []driver.Column, row []any) error {
	buf.WriteByte('{')
	for i, col := range cols {
		if i > 0 {
			buf.WriteByte(',')
		}
		key, err := marshalJSONValue(col.Name)
		if err != nil {
			return err
		}
		buf.Write(key)
		buf.WriteByte(':')
		var v any
		if i < len(row) {
			v = row[i]
		}
		val, err := marshalJSONValue(v)
		if err != nil {
			return err
		}
		buf.Write(val)
	}
	buf.WriteByte('}')
	return nil
}

func marshalJSONValue(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte{'\n'}), nil
}
