package mysql

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

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

func ptr(s string) *string { return &s }

func TestMysqlAlterSQL(t *testing.T) {
	def := "guest"
	after := "id"
	sql, err := mysqlAlterSQL("`app`.`users`", []driver.ColumnOp{
		{Kind: driver.ColumnOpDrop, Name: "legacy"},
		{Kind: driver.ColumnOpModify, Name: "name", Column: &driver.ColumnInfo{Name: "full_name", Type: "varchar(255)", Nullable: true}},
		{Kind: driver.ColumnOpAdd, Column: &driver.ColumnInfo{Name: "nickname", Type: "varchar(64)", Nullable: false, Default: &def}, After: &after},
		{Kind: driver.ColumnOpReorder, Name: "id", Column: &driver.ColumnInfo{Name: "id", Type: "int", Nullable: false, AutoIncrement: true}, After: ptr("")},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"ALTER TABLE `app`.`users`",
		"DROP COLUMN `legacy`",
		"CHANGE COLUMN `name` `full_name` varchar(255) NULL",
		"ADD COLUMN `nickname` varchar(64) NOT NULL DEFAULT 'guest' AFTER `id`",
		"CHANGE COLUMN `id` `id` int NOT NULL AUTO_INCREMENT FIRST",
	}
	for _, part := range want {
		if !strings.Contains(sql, part) {
			t.Fatalf("missing %q in %s", part, sql)
		}
	}
}

func TestMysqlAlterSQLRejectsEmpty(t *testing.T) {
	if _, err := mysqlAlterSQL("`t`", nil); err == nil {
		t.Fatal("expected error")
	}
	if _, err := mysqlAlterSQL("`t`", []driver.ColumnOp{{Kind: "nope"}}); err == nil {
		t.Fatal("expected unknown op error")
	}
}

func TestSearchTablesLiveEnsembl(t *testing.T) {
	if os.Getenv("LOTUSTAB_LIVE") == "" {
		t.Skip("set LOTUSTAB_LIVE=1 to hit ensembldb.ensembl.org")
	}
	d := New()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	t0 := time.Now()
	c, err := d.Open(ctx, driver.ConnConfig{
		Params: map[string]string{
			"host": "ensembldb.ensembl.org",
			"port": "5306",
			"user": "anonymous",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	t.Logf("open %s", time.Since(t0))

	t1 := time.Now()
	names, err := c.Databases(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("databases %d in %s", len(names), time.Since(t1))
	if len(names) < 100 {
		t.Fatalf("expected a large catalog, got %d databases", len(names))
	}

	t2 := time.Now()
	none, err := c.SearchTables(ctx, "", "gene", 20)
	if err != nil {
		t.Fatalf("unscoped search: %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("unscoped search must not scan the whole server, got %d", len(none))
	}
	t.Logf("unscoped search %s", time.Since(t2))

	var sample string
	for _, name := range names {
		if strings.Contains(name, "homo_sapiens_core") {
			sample = name
			break
		}
	}
	if sample == "" {
		t.Fatal("no homo_sapiens_core database")
	}
	t3 := time.Now()
	tables, err := c.SearchTables(ctx, sample, "gene", 20)
	if err != nil {
		t.Fatalf("search after %s: %v", time.Since(t3), err)
	}
	t.Logf("search %s -> %d in %s", sample, len(tables), time.Since(t3))
	if len(tables) == 0 {
		t.Fatal("expected gene table matches")
	}
	if len(tables) > 20 {
		t.Fatalf("limit should cap results, got %d", len(tables))
	}
	for _, table := range tables {
		if table.Name == "" || table.Database == "" {
			t.Fatalf("incomplete row: %+v", table)
		}
		if !strings.Contains(strings.ToLower(table.Name), "gene") {
			t.Fatalf("unexpected table name %q", table.Name)
		}
	}
}
