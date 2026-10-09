package app_test

import (
	"path/filepath"
	"testing"

	"lotustab/internal/app"
	"lotustab/internal/storage"
)

func TestColumnWidthsServicePersistsByTableScope(t *testing.T) {
	store := storage.NewColumnWidthsAt(filepath.Join(t.TempDir(), "column-widths.json"))
	svc := app.NewColumnWidthsService(store)

	if err := svc.SetTableColumnWidth("db", "public", "users", "email", 180); err != nil {
		t.Fatal(err)
	}

	got, err := svc.GetTableWidths("db", "public", "users")
	if err != nil {
		t.Fatal(err)
	}
	if got["email"] != 180 {
		t.Fatalf("got %#v", got)
	}

	other, err := svc.GetTableWidths("db", "public", "orders")
	if err != nil {
		t.Fatal(err)
	}
	if len(other) != 0 {
		t.Fatalf("other table leaked widths: %#v", other)
	}

	schemaOther, err := svc.GetTableWidths("db", "analytics", "users")
	if err != nil {
		t.Fatal(err)
	}
	if len(schemaOther) != 0 {
		t.Fatalf("other schema leaked widths: %#v", schemaOther)
	}
}

func TestColumnWidthsServiceSkipsEmptyTable(t *testing.T) {
	store := storage.NewColumnWidthsAt(filepath.Join(t.TempDir(), "column-widths.json"))
	svc := app.NewColumnWidthsService(store)

	got, err := svc.GetTableWidths("db", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty, got %#v", got)
	}
	if err := svc.SetTableColumnWidth("db", "", "", "id", 100); err == nil {
		t.Fatal("expected error for empty table")
	}
}
