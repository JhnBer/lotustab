package driver_test

import (
	"testing"

	"lotustab/internal/core/driver"
)

func TestStatementVerb(t *testing.T) {
	tests := []struct {
		sql  string
		want string
	}{
		{sql: "INSERT INTO t VALUES (1)", want: "INSERT"},
		{sql: "BEGIN", want: "BEGIN"},
		{sql: "START TRANSACTION", want: "START"},
		{sql: "-- setup\nINSERT INTO t VALUES (1)", want: "INSERT"},
		{sql: "/* x */ UPDATE t SET a = 1", want: "UPDATE"},
		{sql: "WITH t AS (SELECT 1) SELECT * FROM t", want: "WITH"},
		{sql: "(SELECT 1)", want: "SELECT"},
		{sql: "  ", want: ""},
	}
	for _, tt := range tests {
		if got := driver.StatementVerb(tt.sql); got != tt.want {
			t.Fatalf("%q: got %q want %q", tt.sql, got, tt.want)
		}
	}
}

func TestVerbKind(t *testing.T) {
	if !driver.VerbReturnsRows("SELECT") || !driver.VerbReturnsRows("WITH") {
		t.Fatal("SELECT/WITH should return rows")
	}
	if driver.VerbReturnsRows("INSERT") || driver.VerbReturnsRows("BEGIN") {
		t.Fatal("INSERT/BEGIN should not return rows")
	}
	if !driver.VerbAffectsRows("INSERT") || !driver.VerbAffectsRows("UPDATE") || !driver.VerbAffectsRows("DELETE") {
		t.Fatal("DML should affect rows")
	}
	if driver.VerbAffectsRows("BEGIN") || driver.VerbAffectsRows("COMMIT") || driver.VerbAffectsRows("SET") {
		t.Fatal("session commands should not affect rows")
	}
}
