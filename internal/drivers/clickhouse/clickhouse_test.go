package clickhouse

import (
	"context"
	"testing"

	"lotustab/internal/core/driver"
)

func TestSearchTablesEmpty(t *testing.T) {
	c := &conn{}
	got, err := c.SearchTables(context.Background(), "", "  ", 10)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("empty query should not hit the server, got %v", got)
	}
}

func TestClickhouseQualified(t *testing.T) {
	if got := clickhouseQualified("", "hits"); got != "`hits`" {
		t.Fatalf("got %q", got)
	}
	if got := clickhouseQualified("analytics", "hits"); got != "`analytics`.`hits`" {
		t.Fatalf("got %q", got)
	}
}

func TestClickhouseDBName(t *testing.T) {
	if got := clickhouseDBName("analytics", "ignored"); got != "analytics" {
		t.Fatalf("got %q", got)
	}
	if got := clickhouseDBName("", "analytics"); got != "analytics" {
		t.Fatalf("got %q", got)
	}
	if got := clickhouseDBName("  ", " analytics "); got != "analytics" {
		t.Fatalf("got %q", got)
	}
}

func TestClickhouseObjectType(t *testing.T) {
	tests := []struct {
		engine string
		want   string
	}{
		{"MergeTree", driver.ObjectTable},
		{"ReplicatedMergeTree", driver.ObjectTable},
		{"View", driver.ObjectView},
		{"MaterializedView", driver.ObjectView},
		{"LiveView", driver.ObjectView},
		{"WindowView", driver.ObjectView},
	}
	for _, tt := range tests {
		if got := clickhouseObjectType(tt.engine); got != tt.want {
			t.Errorf("clickhouseObjectType(%q)=%q want %q", tt.engine, got, tt.want)
		}
	}
}

func TestClickhouseUse(t *testing.T) {
	if got := clickhouseUse("analytics"); got != "USE `analytics`" {
		t.Fatalf("got %q", got)
	}
}

func TestClickhouseLiteral(t *testing.T) {
	tests := []struct {
		in   any
		want string
	}{
		{nil, "NULL"},
		{"hello", "'hello'"},
		{"a'b", "'a''b'"},
		{true, "1"},
		{false, "0"},
		{int64(42), "42"},
		{float64(1.5), "1.5"},
		{[]byte("raw"), "'raw'"},
	}
	for _, tt := range tests {
		if got := clickhouseLiteral(tt.in); got != tt.want {
			t.Errorf("clickhouseLiteral(%v)=%q want %q", tt.in, got, tt.want)
		}
	}
}

func TestClickhouseMutationSQL(t *testing.T) {
	sql, err := clickhouseMutationSQL("`analytics`.`hits`", map[string]any{
		"status": "ok",
		"n":      int64(2),
	}, []string{"id", "date"}, map[string]any{"id": int64(1), "date": "2024-01-01"})
	if err != nil {
		t.Fatal(err)
	}
	want := "ALTER TABLE `analytics`.`hits` UPDATE `n` = 2, `status` = 'ok' WHERE `id` = 1 AND `date` = '2024-01-01' SETTINGS mutations_sync = 1"
	if sql != want {
		t.Fatalf("got %q want %q", sql, want)
	}
}

func TestClickhouseDeleteAndInsertSQL(t *testing.T) {
	del, err := clickhouseDeleteSQL("`t`", []string{"id"}, map[string]any{"id": int64(1)})
	if err != nil {
		t.Fatal(err)
	}
	wantDel := "ALTER TABLE `t` DELETE WHERE `id` = 1 SETTINGS mutations_sync = 1"
	if del != wantDel {
		t.Fatalf("delete=%q want %q", del, wantDel)
	}
	ins, err := clickhouseInsertSQL("`t`", map[string]any{"n": int64(2), "name": "a"})
	if err != nil {
		t.Fatal(err)
	}
	wantIns := "INSERT INTO `t` (`n`, `name`) VALUES (2, 'a')"
	if ins != wantIns {
		t.Fatalf("insert=%q want %q", ins, wantIns)
	}
}

func TestClickhouseMutationSQLRejectsEmpty(t *testing.T) {
	if _, err := clickhouseMutationSQL("`t`", nil, []string{"id"}, map[string]any{"id": 1}); err == nil {
		t.Fatal("expected error")
	}
	if _, err := clickhouseMutationSQL("`t`", map[string]any{"n": 1}, nil, nil); err == nil {
		t.Fatal("expected sorting key error")
	}
	if _, err := clickhouseMutationSQL("`t`", map[string]any{"n": 1}, []string{"id"}, map[string]any{}); err == nil {
		t.Fatal("expected missing key error")
	}
}
