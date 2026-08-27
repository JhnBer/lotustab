package driver_test

import (
	"testing"

	"lotustab/internal/core/driver"
	"lotustab/internal/drivers/mysql"
	"lotustab/internal/drivers/postgres"
)

func TestRegistryRejectsDuplicates(t *testing.T) {
	r := driver.NewRegistry()
	if err := r.Register(postgres.New()); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(postgres.New()); err == nil {
		t.Fatal("expected duplicate registration to fail")
	}
	if err := r.Register(mysql.New()); err != nil {
		t.Fatal(err)
	}
	if got := len(r.All()); got != 2 {
		t.Fatalf("got %d drivers, want 2", got)
	}
}

func TestNormalizeValue(t *testing.T) {
	if driver.NormalizeValue(nil) != nil {
		t.Fatal("nil should stay nil")
	}
	if driver.NormalizeValue([]byte("abc")) != "abc" {
		t.Fatal("bytes should become string")
	}
	if driver.NormalizeValue(int32(7)) != int64(7) {
		t.Fatal("int32 should become int64")
	}
}
