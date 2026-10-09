package storage_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lotustab/internal/storage"
)

func TestTableScopeKeyIncludesSchema(t *testing.T) {
	a := storage.TableScopeKey("db", "public", "users")
	b := storage.TableScopeKey("db", "other", "users")
	c := storage.TableScopeKey("other", "public", "users")
	if a == "" || a == b || a == c || b == c {
		t.Fatalf("expected distinct opaque keys, got %q %q %q", a, b, c)
	}
	same := storage.TableScopeKey("db", "public", "users")
	if a != same {
		t.Fatalf("key should be stable: %q vs %q", a, same)
	}
}

func TestColumnWidthStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := storage.NewColumnWidthsAt(filepath.Join(dir, "column-widths.json"))

	got, err := store.Get("missing")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty map, got %#v", got)
	}

	key := storage.TableScopeKey("app", "public", "orders")
	if err := store.SetColumn(key, "id", 120); err != nil {
		t.Fatal(err)
	}
	if err := store.SetColumn(key, "name", 240); err != nil {
		t.Fatal(err)
	}

	got, err = store.Get(key)
	if err != nil {
		t.Fatal(err)
	}
	if got["id"] != 120 || got["name"] != 240 {
		t.Fatalf("unexpected widths: %#v", got)
	}

	other := storage.TableScopeKey("app", "public", "items")
	got, err = store.Get(other)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("other table should not inherit widths: %#v", got)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "column-widths.json"))
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	body := strings.ToLower(string(raw))
	for _, secret := range []string{"password", "postgres://", "secret"} {
		if strings.Contains(body, secret) {
			t.Fatalf("cache must not contain %q: %s", secret, string(raw))
		}
	}
}
