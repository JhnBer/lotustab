package driver_test

import (
	"testing"

	"lotustab/internal/core/driver"
)

func TestClampTableBrowse(t *testing.T) {
	got := driver.ClampTableBrowse(driver.TableBrowseRequest{
		Database: " db ",
		Schema:   " public ",
		Name:     " users ",
		Limit:    0,
		Offset:   -4,
		OrderBy:  " id ",
		OrderDir: "DESC",
	})
	if got.Database != "db" || got.Schema != "public" || got.Name != "users" {
		t.Fatalf("trimmed identifiers: %+v", got)
	}
	if got.Limit != driver.DefaultBrowseLimit {
		t.Fatalf("limit default: %d", got.Limit)
	}
	if got.Offset != 0 {
		t.Fatalf("offset: %d", got.Offset)
	}
	if got.OrderBy != "id" || got.OrderDir != "desc" {
		t.Fatalf("order: %+v", got)
	}

	got = driver.ClampTableBrowse(driver.TableBrowseRequest{Limit: 10})
	if got.Limit != driver.MinBrowseLimit {
		t.Fatalf("min clamp: %d", got.Limit)
	}
	got = driver.ClampTableBrowse(driver.TableBrowseRequest{Limit: 50000})
	if got.Limit != driver.MaxBrowseLimit {
		t.Fatalf("max clamp: %d", got.Limit)
	}
	got = driver.ClampTableBrowse(driver.TableBrowseRequest{Limit: 5000})
	if got.Limit != 5000 {
		t.Fatalf("5000 should be in range: %d", got.Limit)
	}
	got = driver.ClampTableBrowse(driver.TableBrowseRequest{Limit: 250, OrderBy: "name", OrderDir: "nope"})
	if got.Limit != 250 || got.OrderDir != "asc" {
		t.Fatalf("invalid dir should become asc: %+v", got)
	}
	got = driver.ClampTableBrowse(driver.TableBrowseRequest{OrderDir: "desc"})
	if got.OrderDir != "" {
		t.Fatalf("orderDir without column should clear: %+v", got)
	}
}

func TestClampSearchLimit(t *testing.T) {
	if got := driver.ClampSearchLimit(0); got != driver.DefaultSearchTablesLimit {
		t.Fatalf("zero: %d", got)
	}
	if got := driver.ClampSearchLimit(-1); got != driver.DefaultSearchTablesLimit {
		t.Fatalf("negative: %d", got)
	}
	if got := driver.ClampSearchLimit(50); got != 50 {
		t.Fatalf("in range: %d", got)
	}
	if got := driver.ClampSearchLimit(5000); got != driver.MaxSearchTablesLimit {
		t.Fatalf("over max: %d", got)
	}
}

func TestClampApplyEdits(t *testing.T) {
	got := driver.ClampApplyEdits(driver.ApplyEditsRequest{
		Database: " db ",
		Schema:   " public ",
		Name:     " users ",
		Rows:     []driver.RowEdit{{Key: map[string]any{"id": 1}, Cells: map[string]any{"n": "x"}}},
	})
	if got.Database != "db" || got.Schema != "public" || got.Name != "users" {
		t.Fatalf("trimmed identifiers: %+v", got)
	}
	if len(got.Rows) != 1 {
		t.Fatalf("rows should be kept: %d", len(got.Rows))
	}
}

func TestClampTableRef(t *testing.T) {
	got := driver.ClampTableRef(driver.TableRef{Database: " db ", Schema: " public ", Name: " users "})
	if got.Database != "db" || got.Schema != "public" || got.Name != "users" {
		t.Fatalf("trimmed identifiers: %+v", got)
	}
}

func TestClampAlterTable(t *testing.T) {
	def := "  now()  "
	after := " id "
	got := driver.ClampAlterTable(driver.AlterTableRequest{
		Database: " db ",
		Schema:   " public ",
		Name:     " users ",
		Ops: []driver.ColumnOp{{
			Kind:    " ADD ",
			Name:    " age ",
			NewName: " years ",
			Column:  &driver.ColumnInfo{Name: " years ", Type: " int ", Default: &def},
			After:   &after,
		}},
	})
	if got.Database != "db" || got.Schema != "public" || got.Name != "users" {
		t.Fatalf("trimmed identifiers: %+v", got)
	}
	if len(got.Ops) != 1 {
		t.Fatalf("ops: %d", len(got.Ops))
	}
	op := got.Ops[0]
	if op.Kind != "add" || op.Name != "age" || op.NewName != "years" {
		t.Fatalf("op identity: %+v", op)
	}
	if op.Column == nil || op.Column.Name != "years" || op.Column.Type != "int" || op.Column.Default == nil || *op.Column.Default != "now()" {
		t.Fatalf("column: %+v", op.Column)
	}
	if op.After == nil || *op.After != "id" {
		t.Fatalf("after: %+v", op.After)
	}
}

func TestBrowseFromQueryEmptyPrimaryKey(t *testing.T) {
	got := driver.BrowseFromQuery(nil, 0)
	if got.PrimaryKey == nil {
		t.Fatal("primaryKey should be empty slice, not nil")
	}
}
