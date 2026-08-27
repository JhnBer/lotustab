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
	got = driver.ClampTableBrowse(driver.TableBrowseRequest{Limit: 5000})
	if got.Limit != driver.MaxBrowseLimit {
		t.Fatalf("max clamp: %d", got.Limit)
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
