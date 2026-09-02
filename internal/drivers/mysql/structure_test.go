package mysql

import (
	"strings"
	"testing"

	"lotustab/internal/core/driver"
)

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
