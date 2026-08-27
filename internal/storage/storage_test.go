package storage_test

import (
	"path/filepath"
	"testing"
	"time"

	"lotustab/internal/core/connection"
	"lotustab/internal/storage"
)

func sample(id, name, color string) connection.Profile {
	now := time.Now().UTC()
	return connection.Profile{
		ID:        id,
		Name:      name,
		DriverID:  "postgres",
		Params:    map[string]string{"host": "localhost"},
		Color:     color,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestPinRecentColorAndDelete(t *testing.T) {
	dir := t.TempDir()
	store := storage.NewAt(filepath.Join(dir, "connections.json"))

	a := sample("a", "Alpha", "blue")
	b := sample("b", "Beta", "green")
	c := sample("c", "Gamma", "red")
	if err := store.Upsert(a); err != nil {
		t.Fatal(err)
	}
	if err := store.Upsert(b); err != nil {
		t.Fatal(err)
	}
	if err := store.Upsert(c); err != nil {
		t.Fatal(err)
	}

	if err := store.SetColor("b", "violet"); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get("b")
	if err != nil {
		t.Fatal(err)
	}
	if got.Color != "violet" {
		t.Fatalf("color=%q", got.Color)
	}

	if err := store.SetPinned("b", true); err != nil {
		t.Fatal(err)
	}
	if err := store.SetPinned("a", true); err != nil {
		t.Fatal(err)
	}
	if err := store.SetPinned("c", true); err != nil {
		t.Fatal(err)
	}
	if err := store.SetPinned("b", false); err != nil {
		t.Fatal(err)
	}

	if err := store.TouchRecent("c"); err != nil {
		t.Fatal(err)
	}
	if err := store.TouchRecent("a"); err != nil {
		t.Fatal(err)
	}
	if err := store.TouchRecent("c"); err != nil {
		t.Fatal(err)
	}

	_, pinned, recent, err := store.ListState()
	if err != nil {
		t.Fatal(err)
	}
	if len(pinned) != 2 || pinned[0] != "a" || pinned[1] != "c" {
		t.Fatalf("pinned=%v", pinned)
	}
	if len(recent) != 2 || recent[0] != "c" || recent[1] != "a" {
		t.Fatalf("recent=%v", recent)
	}

	if err := store.Delete("c"); err != nil {
		t.Fatal(err)
	}
	_, pinned, recent, err = store.ListState()
	if err != nil {
		t.Fatal(err)
	}
	if len(pinned) != 1 || pinned[0] != "a" {
		t.Fatalf("pinned after delete=%v", pinned)
	}
	if len(recent) != 1 || recent[0] != "a" {
		t.Fatalf("recent after delete=%v", recent)
	}
}

func TestRecentCap(t *testing.T) {
	dir := t.TempDir()
	store := storage.NewAt(filepath.Join(dir, "connections.json"))
	for i := 0; i < 10; i++ {
		id := string(rune('a' + i))
		if err := store.Upsert(sample(id, id, "blue")); err != nil {
			t.Fatal(err)
		}
		if err := store.TouchRecent(id); err != nil {
			t.Fatal(err)
		}
	}
	_, _, recent, err := store.ListState()
	if err != nil {
		t.Fatal(err)
	}
	if len(recent) != 8 {
		t.Fatalf("recent cap=%d %v", len(recent), recent)
	}
	if recent[0] != "j" {
		t.Fatalf("most recent=%q", recent[0])
	}
}
