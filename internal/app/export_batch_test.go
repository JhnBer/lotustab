package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lotustab/internal/core/driver"
)

func TestCountTablesValidatesInput(t *testing.T) {
	s := &ExportService{}
	_, err := s.CountTables(context.Background(), CountTablesRequest{})
	if err == nil || !strings.Contains(err.Error(), "profile") {
		t.Fatalf("expected profile error, got %v", err)
	}
	_, err = s.CountTables(context.Background(), CountTablesRequest{ProfileID: "abc"})
	if err == nil || !strings.Contains(err.Error(), "table") {
		t.Fatalf("expected table error, got %v", err)
	}
}

func TestExportTablesValidatesInput(t *testing.T) {
	s := &ExportService{}
	_, err := s.ExportTables(context.Background(), BatchExportRequest{})
	if err == nil || !strings.Contains(err.Error(), "profile") {
		t.Fatalf("expected profile error, got %v", err)
	}
	_, err = s.ExportTables(context.Background(), BatchExportRequest{
		ProfileID: "abc",
		Tables:    []driver.TableRef{{Name: "  "}},
	})
	if err == nil || !strings.Contains(err.Error(), "table") {
		t.Fatalf("expected table error, got %v", err)
	}
}

func TestCheckOutput(t *testing.T) {
	s := &ExportService{}
	dir := t.TempDir()
	st, err := s.CheckOutput(filepath.Join(dir, "missing"), true)
	if err != nil {
		t.Fatal(err)
	}
	if st.Exists || st.HasFiles {
		t.Fatalf("missing dir: %+v", st)
	}

	empty := filepath.Join(dir, "empty")
	if err := os.Mkdir(empty, 0o755); err != nil {
		t.Fatal(err)
	}
	st, err = s.CheckOutput(empty, true)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Exists || st.HasFiles {
		t.Fatalf("empty dir: %+v", st)
	}

	file := filepath.Join(dir, "out.sql")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err = s.CheckOutput(file, false)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Exists || !st.HasFiles {
		t.Fatalf("existing file: %+v", st)
	}

	filled := filepath.Join(dir, "filled")
	if err := os.Mkdir(filled, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filled, "a.sql"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err = s.CheckOutput(filled, true)
	if err != nil {
		t.Fatal(err)
	}
	if !st.HasFiles {
		t.Fatal("expected files")
	}
}

func TestResolveFolderDestOverwrite(t *testing.T) {
	dir := t.TempDir()
	parent := filepath.Join(dir, "parent")
	if err := os.Mkdir(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	dest, err := resolveFolderDest(parent, "dump", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "a.sql"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = resolveFolderDest(parent, "dump", false)
	if !errors.Is(err, ErrDirNotEmpty) {
		t.Fatalf("expected dir_not_empty, got %v", err)
	}
	got, err := resolveFolderDest(parent, "dump", true)
	if err != nil {
		t.Fatal(err)
	}
	if got != dest {
		t.Fatalf("got %s want %s", got, dest)
	}
}

func TestDumpFileBaseQualifiesCollisions(t *testing.T) {
	tables := []driver.TableRef{
		{Schema: "public", Name: "users"},
		{Schema: "other", Name: "users"},
		{Schema: "public", Name: "orders"},
	}
	if got := dumpFileBase("pre_", tables, 0); got != "pre_public_users" {
		t.Fatalf("got %q", got)
	}
	if got := dumpFileBase("", tables, 2); got != "orders" {
		t.Fatalf("got %q", got)
	}
}
