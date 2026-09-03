package clickhouse

import (
	"strings"
	"testing"

	"lotustab/internal/core/driver"
)

func ptr(s string) *string { return &s }

func TestClickhouseAlterSQL(t *testing.T) {
	def := "guest"
	after := "id"
	stmts, err := clickhouseAlterSQL("`analytics`.`users`", []driver.ColumnOp{
		{Kind: driver.ColumnOpDrop, Name: "legacy"},
		{Kind: driver.ColumnOpAdd, Column: &driver.ColumnInfo{Name: "nickname", Type: "String", Default: &def, Comment: "aka"}, After: &after},
		{Kind: driver.ColumnOpModify, Name: "name", Column: &driver.ColumnInfo{Name: "full_name", Type: "Nullable(String)"}},
		{Kind: driver.ColumnOpReorder, Name: "id", Column: &driver.ColumnInfo{Name: "id", Type: "UInt64"}, After: ptr("")},
	})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(stmts, "; ")
	want := []string{
		"ALTER TABLE `analytics`.`users` DROP COLUMN `legacy`",
		"ALTER TABLE `analytics`.`users` ADD COLUMN `nickname` String DEFAULT 'guest' COMMENT 'aka' AFTER `id`",
		"ALTER TABLE `analytics`.`users` RENAME COLUMN `name` TO `full_name`",
		"ALTER TABLE `analytics`.`users` MODIFY COLUMN `full_name` Nullable(String)",
		"ALTER TABLE `analytics`.`users` MODIFY COLUMN `id` UInt64 FIRST",
	}
	for _, part := range want {
		if !strings.Contains(joined, part) {
			t.Fatalf("missing %q in %s", part, joined)
		}
	}
}

func TestClickhouseAlterSQLRejectsEmpty(t *testing.T) {
	if _, err := clickhouseAlterSQL("`t`", nil); err == nil {
		t.Fatal("expected error")
	}
	if _, err := clickhouseAlterSQL("`t`", []driver.ColumnOp{{Kind: "nope"}}); err == nil {
		t.Fatal("expected unknown op error")
	}
}

func TestClickhouseNullable(t *testing.T) {
	if !clickhouseNullable("Nullable(String)") {
		t.Fatal("expected Nullable(String) to be nullable")
	}
	if !clickhouseNullable("LowCardinality(Nullable(String))") {
		t.Fatal("expected nested Nullable to be nullable")
	}
	if clickhouseNullable("String") {
		t.Fatal("String should not be nullable")
	}
}

func TestClickhouseExtra(t *testing.T) {
	tests := []struct {
		kind, want string
	}{
		{"", ""},
		{"DEFAULT", ""},
		{"MATERIALIZED", "MATERIALIZED"},
		{"ALIAS", "ALIAS"},
		{"EPHEMERAL", "EPHEMERAL"},
	}
	for _, tt := range tests {
		if got := clickhouseExtra(tt.kind); got != tt.want {
			t.Errorf("clickhouseExtra(%q)=%q want %q", tt.kind, got, tt.want)
		}
	}
}

func TestClickhouseColumnDefOmitsNullKeyword(t *testing.T) {
	got, err := clickhouseColumnDef(driver.ColumnInfo{Name: "status", Type: "Nullable(String)", Nullable: true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "NULL") && !strings.Contains(got, "Nullable") {
		t.Fatalf("unexpected NULL keyword in %q", got)
	}
	if got != "`status` Nullable(String)" {
		t.Fatalf("got %q", got)
	}
}
