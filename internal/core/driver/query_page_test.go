package driver_test

import (
	"strings"
	"testing"

	"lotustab/internal/core/driver"
)

func TestClampLimitOffset(t *testing.T) {
	limit, offset := driver.ClampLimitOffset(0, -3)
	if limit != driver.DefaultBrowseLimit || offset != 0 {
		t.Fatalf("defaults: limit=%d offset=%d", limit, offset)
	}
	limit, _ = driver.ClampLimitOffset(10, 0)
	if limit != driver.MinBrowseLimit {
		t.Fatalf("min: %d", limit)
	}
	limit, _ = driver.ClampLimitOffset(50000, 0)
	if limit != driver.MaxBrowseLimit {
		t.Fatalf("max: %d", limit)
	}
	limit, offset = driver.ClampLimitOffset(250, 40)
	if limit != 250 || offset != 40 {
		t.Fatalf("passthrough: limit=%d offset=%d", limit, offset)
	}
}

func TestPrepareQueryPageSelect(t *testing.T) {
	page := driver.PrepareQueryPage("  select * from users;  ", 100, 50)
	if !page.Paged {
		t.Fatal("expected paged SELECT")
	}
	if page.Limit != 100 || page.Offset != 50 || page.MaxRows != 101 {
		t.Fatalf("paging: %+v", page)
	}
	if !strings.Contains(page.SQL, "select * from users") {
		t.Fatalf("inner sql lost: %s", page.SQL)
	}
	if strings.Contains(page.SQL, "users;") {
		t.Fatalf("trailing semicolon should be stripped: %s", page.SQL)
	}
	if !strings.Contains(page.SQL, "AS _lotustab_page LIMIT 101 OFFSET 50") {
		t.Fatalf("wrap: %s", page.SQL)
	}
}

func TestPrepareQueryPageWith(t *testing.T) {
	page := driver.PrepareQueryPage("WITH t AS (SELECT 1) SELECT * FROM t", 50, 0)
	if !page.Paged {
		t.Fatal("expected paged WITH")
	}
}

func TestPrepareQueryPageNonSelect(t *testing.T) {
	sql := "UPDATE users SET name = 'x'"
	page := driver.PrepareQueryPage(sql, 100, 10)
	if page.Paged {
		t.Fatal("UPDATE should not wrap")
	}
	if page.SQL != sql {
		t.Fatalf("sql rewritten: %s", page.SQL)
	}
	if page.MaxRows != 100 {
		t.Fatalf("maxRows: %d", page.MaxRows)
	}
}

func TestFinishQueryPage(t *testing.T) {
	page := driver.QueryPage{Limit: 2, Paged: true}
	got := driver.FinishQueryPage(&driver.QueryResult{
		Rows: [][]any{{1}, {2}, {3}},
	}, page)
	if !got.Truncated || len(got.Rows) != 2 {
		t.Fatalf("lookahead: truncated=%v rows=%d", got.Truncated, len(got.Rows))
	}

	got = driver.FinishQueryPage(&driver.QueryResult{
		Rows:      [][]any{{1}, {2}},
		Truncated: true,
	}, page)
	if got.Truncated || len(got.Rows) != 2 {
		t.Fatalf("exact page should not be truncated: %+v", got)
	}

	raw := &driver.QueryResult{Rows: [][]any{{1}, {2}, {3}}, Truncated: true}
	got = driver.FinishQueryPage(raw, driver.QueryPage{Limit: 2, Paged: false})
	if !got.Truncated || len(got.Rows) != 3 {
		t.Fatalf("non-paged should pass through: %+v", got)
	}
}
