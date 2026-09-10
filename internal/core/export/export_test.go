package export

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lotustab/internal/core/driver"
)

func TestSQLDumpEscapesAndBatches(t *testing.T) {
	cols := []driver.Column{{Name: "id"}, {Name: "name"}, {Name: "note"}}
	var b strings.Builder
	w := newSQLDumpWriter(&b, "`t`", DialectMySQL, cols, 2)
	if err := w.writeHeader("CREATE TABLE `t` (id int);", true, true, true, true); err != nil {
		t.Fatal(err)
	}
	if err := w.writeRow([]any{int64(1), "a'b", nil}); err != nil {
		t.Fatal(err)
	}
	if err := w.writeRow([]any{int64(2), "x", []byte{0x00, 0xff}}); err != nil {
		t.Fatal(err)
	}
	if err := w.writeRow([]any{true, "z", nil}); err != nil {
		t.Fatal(err)
	}
	if err := w.writeFooter(true); err != nil {
		t.Fatal(err)
	}
	got := b.String()
	want := []string{
		"BEGIN;",
		"DROP TABLE IF EXISTS `t`;",
		"CREATE TABLE `t` (id int);",
		"INSERT INTO `t` (`id`, `name`, `note`) VALUES",
		"(1, 'a''b', NULL),",
		"(2, 'x', X'00ff');",
		"(1, 'z', NULL);",
		"COMMIT;",
	}
	for _, part := range want {
		if !strings.Contains(got, part) {
			t.Fatalf("missing %q in\n%s", part, got)
		}
	}
}

func TestSQLDumpPostgresBool(t *testing.T) {
	var b strings.Builder
	w := newSQLDumpWriter(&b, `"t"`, DialectPostgres, []driver.Column{{Name: "ok"}}, 10)
	if err := w.writeRow([]any{true}); err != nil {
		t.Fatal(err)
	}
	if err := w.writeFooter(false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "(TRUE);") {
		t.Fatalf("got %s", b.String())
	}
}

func TestRunChunksGzipSQL(t *testing.T) {
	dir := t.TempDir()
	rows := [][]any{
		{int64(1), "a"},
		{int64(2), "b"},
		{int64(3), "c"},
	}
	result, err := Run(t.Context(), func(fn func(cols []driver.Column, rows [][]any) error) error {
		return fn([]driver.Column{{Name: "id"}, {Name: "name"}}, rows)
	}, Options{
		Format:          FormatSQL,
		Gzip:            true,
		ChunkSize:       2,
		IncludeCreate:   true,
		IncludeDrop:     true,
		InsertBatchSize: 10,
		WrapTransaction: true,
		Dialect:         DialectMySQL,
		Table:           "users",
		DDL:             "CREATE TABLE `users` (id int);",
		OutputDir:       dir,
		FileBase:        "users_20260101",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Rows != 3 {
		t.Fatalf("rows=%d", result.Rows)
	}
	if len(result.Files) != 2 {
		t.Fatalf("files=%v", result.Files)
	}
	for _, path := range result.Files {
		if filepath.Ext(path) != ".gz" {
			t.Fatalf("expected gzip file, got %s", path)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		gr, err := gzip.NewReader(bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		var decoded strings.Builder
		if _, err := io.Copy(&decoded, gr); err != nil {
			t.Fatal(err)
		}
		_ = gr.Close()
		body := decoded.String()
		if strings.Contains(path, "_001") {
			if !strings.Contains(body, "CREATE TABLE") {
				t.Fatalf("first chunk missing CREATE: %s", body)
			}
		} else if strings.Contains(body, "CREATE TABLE") {
			t.Fatalf("later chunk should not recreate table: %s", body)
		}
		if _, err := os.Stat(path + ".part"); err == nil {
			t.Fatal("part file left behind")
		}
	}
}

func TestRunCSV(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.csv")
	result, err := Run(t.Context(), func(fn func(cols []driver.Column, rows [][]any) error) error {
		return fn([]driver.Column{{Name: "id"}, {Name: "name"}}, [][]any{{int64(1), "a,b"}, {nil, "x"}})
	}, Options{
		Format:     FormatCSV,
		Table:      "t",
		OutputPath: path,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Rows != 2 {
		t.Fatalf("rows=%d", result.Rows)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	if !strings.Contains(got, "id,name") || !strings.Contains(got, `"a,b"`) {
		t.Fatalf("csv=%s", got)
	}
}

func TestSelectQuery(t *testing.T) {
	got := SelectQuery(DialectMySQL, "db", "", "logs", "WHERE created_at > 1;")
	want := "SELECT * FROM `db`.`logs` WHERE created_at > 1"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestCountQuery(t *testing.T) {
	got := CountQuery(DialectMySQL, "db", "", "logs", "created_at > 1;")
	want := "SELECT COUNT(*) FROM `db`.`logs` WHERE created_at > 1"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestRunDeleteOnAbort(t *testing.T) {
	dir := t.TempDir()
	_, err := Run(t.Context(), func(fn func(cols []driver.Column, rows [][]any) error) error {
		if err := fn([]driver.Column{{Name: "id"}}, [][]any{{int64(1)}, {int64(2)}}); err != nil {
			return err
		}
		return fmt.Errorf("boom")
	}, Options{
		Format:        FormatCSV,
		ChunkSize:     1,
		Table:         "t",
		OutputDir:     dir,
		FileBase:      "t",
		DeleteOnAbort: true,
	}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("leftover files: %v", names)
	}
}
