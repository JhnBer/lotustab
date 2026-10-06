package app

import (
	"context"
	"strings"
	"testing"

	"lotustab/internal/core/driver"
)

func TestBrowseTableValidatesInput(t *testing.T) {
	s := &QueryService{}
	_, err := s.BrowseTable(context.Background(), "", driver.TableBrowseRequest{Name: "users"})
	if err == nil || !strings.Contains(err.Error(), "profile") {
		t.Fatalf("expected profile error, got %v", err)
	}
	_, err = s.BrowseTable(context.Background(), "abc", driver.TableBrowseRequest{Name: "  "})
	if err == nil || !strings.Contains(err.Error(), "table") {
		t.Fatalf("expected table error, got %v", err)
	}
	_, err = s.BrowseTable(context.Background(), "abc", driver.TableBrowseRequest{Limit: 10})
	if err == nil || !strings.Contains(err.Error(), "table") {
		t.Fatalf("expected table error after clamp, got %v", err)
	}
}

func TestListTablesRequiresProfile(t *testing.T) {
	s := &QueryService{}
	_, err := s.ListTables(context.Background(), "", "db")
	if err == nil || !strings.Contains(err.Error(), "profile") {
		t.Fatalf("expected profile error, got %v", err)
	}
}

func TestSearchTablesValidatesInput(t *testing.T) {
	s := &QueryService{}
	_, err := s.SearchTables(context.Background(), "", "", "users", 50)
	if err == nil || !strings.Contains(err.Error(), "profile") {
		t.Fatalf("expected profile error, got %v", err)
	}
	got, err := s.SearchTables(context.Background(), "abc", "", "  ", 50)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("empty query should skip the server, got %v", got)
	}
}

func TestExecuteValidatesInput(t *testing.T) {
	s := &QueryService{}
	_, err := s.Execute(context.Background(), "", "SELECT 1", "", 10, 0, 0, "")
	if err == nil || !strings.Contains(err.Error(), "profile") {
		t.Fatalf("expected profile error, got %v", err)
	}
	_, err = s.Execute(context.Background(), "abc", "  ", "db", 10, 0, 0, "")
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected empty query error, got %v", err)
	}
}

func TestExecuteEmptyComments(t *testing.T) {
	s := &QueryService{}
	_, err := s.Execute(context.Background(), "abc", "-- just a comment\n/* x */", "", 10, 0, 0, "")
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected empty query error, got %v", err)
	}
}

func TestApplyEditsValidatesInput(t *testing.T) {
	s := &QueryService{}
	row := driver.RowEdit{Key: map[string]any{"id": 1}, Cells: map[string]any{"name": "a"}}
	err := s.ApplyEdits(context.Background(), "", driver.ApplyEditsRequest{Name: "users", Rows: []driver.RowEdit{row}})
	if err == nil || !strings.Contains(err.Error(), "profile") {
		t.Fatalf("expected profile error, got %v", err)
	}
	err = s.ApplyEdits(context.Background(), "abc", driver.ApplyEditsRequest{Name: "  ", Rows: []driver.RowEdit{row}})
	if err == nil || !strings.Contains(err.Error(), "table") {
		t.Fatalf("expected table error, got %v", err)
	}
	err = s.ApplyEdits(context.Background(), "abc", driver.ApplyEditsRequest{Name: "users"})
	if err == nil || !strings.Contains(err.Error(), "no edits") {
		t.Fatalf("expected no edits error, got %v", err)
	}
}

func TestDescribeTableValidatesInput(t *testing.T) {
	s := &QueryService{}
	_, err := s.DescribeTable(context.Background(), "", driver.TableRef{Name: "users"})
	if err == nil || !strings.Contains(err.Error(), "profile") {
		t.Fatalf("expected profile error, got %v", err)
	}
	_, err = s.DescribeTable(context.Background(), "abc", driver.TableRef{Name: "  "})
	if err == nil || !strings.Contains(err.Error(), "table") {
		t.Fatalf("expected table error, got %v", err)
	}
}

func TestAlterTableValidatesInput(t *testing.T) {
	s := &QueryService{}
	op := driver.ColumnOp{Kind: driver.ColumnOpDrop, Name: "age"}
	err := s.AlterTable(context.Background(), "", driver.AlterTableRequest{Name: "users", Ops: []driver.ColumnOp{op}})
	if err == nil || !strings.Contains(err.Error(), "profile") {
		t.Fatalf("expected profile error, got %v", err)
	}
	err = s.AlterTable(context.Background(), "abc", driver.AlterTableRequest{Name: "  ", Ops: []driver.ColumnOp{op}})
	if err == nil || !strings.Contains(err.Error(), "table") {
		t.Fatalf("expected table error, got %v", err)
	}
	err = s.AlterTable(context.Background(), "abc", driver.AlterTableRequest{Name: "users"})
	if err == nil || !strings.Contains(err.Error(), "no schema") {
		t.Fatalf("expected no schema changes error, got %v", err)
	}
}
