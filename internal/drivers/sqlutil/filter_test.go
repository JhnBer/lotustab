package sqlutil_test

import (
	"strings"
	"testing"

	"lotustab/internal/core/driver"
	"lotustab/internal/drivers/sqlutil"
)

func TestWhereClauseEmpty(t *testing.T) {
	sql, args, err := sqlutil.WhereClause(sqlutil.QuotePostgres, sqlutil.Dollar, nil)
	if err != nil {
		t.Fatal(err)
	}
	if sql != "" || args != nil {
		t.Fatalf("sql=%q args=%v", sql, args)
	}
}

func TestWhereClauseCompareAndLike(t *testing.T) {
	sql, args, err := sqlutil.WhereClause(sqlutil.QuotePostgres, sqlutil.Dollar, []driver.TableFilter{
		{Column: "name", Op: "eq", Value: "ada"},
		{Column: "bio", Op: "like", Value: "%x%", Join: "and"},
		{Column: "age", Op: "gte", Value: "18", Join: "or"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := ` WHERE ("name" = $1) AND ("bio" LIKE $2) OR ("age" >= $3)`
	if sql != want {
		t.Fatalf("sql=%q want %q", sql, want)
	}
	if len(args) != 3 || args[0] != "ada" || args[1] != "%x%" || args[2] != "18" {
		t.Fatalf("args=%v", args)
	}
}

func TestWhereClauseOperators(t *testing.T) {
	cases := []struct {
		op, sql string
	}{
		{"neq", `("n" != $1)`},
		{"gt", `("n" > $1)`},
		{"lt", `("n" < $1)`},
		{"lte", `("n" <= $1)`},
		{"nlike", `("n" NOT LIKE $1)`},
	}
	for _, tc := range cases {
		got, args, err := sqlutil.WhereClause(sqlutil.QuotePostgres, sqlutil.Dollar, []driver.TableFilter{
			{Column: "n", Op: tc.op, Value: "v"},
		})
		if err != nil {
			t.Fatalf("%s: %v", tc.op, err)
		}
		want := " WHERE " + tc.sql
		if got != want {
			t.Fatalf("%s: sql=%q want %q", tc.op, got, want)
		}
		if len(args) != 1 || args[0] != "v" {
			t.Fatalf("%s: args=%v", tc.op, args)
		}
	}
}

func TestWhereClauseNullAndIn(t *testing.T) {
	sql, args, err := sqlutil.WhereClause(sqlutil.QuoteMySQL, sqlutil.Question, []driver.TableFilter{
		{Column: "gone", Op: "null"},
		{Column: "id", Op: "in", Value: "1, 2, 3", Join: "and"},
		{Column: "tag", Op: "nin", Value: "a,, b", Join: "or"},
		{Column: "note", Op: "nnull", Join: "and"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := " WHERE (`gone` IS NULL) AND (`id` IN (?, ?, ?)) OR (`tag` NOT IN (?, ?)) AND (`note` IS NOT NULL)"
	if sql != want {
		t.Fatalf("sql=%q want %q", sql, want)
	}
	if len(args) != 5 || args[0] != "1" || args[1] != "2" || args[2] != "3" || args[3] != "a" || args[4] != "b" {
		t.Fatalf("args=%v", args)
	}
}

func TestWhereClauseQuotesSpecialChars(t *testing.T) {
	sql, _, err := sqlutil.WhereClause(sqlutil.QuotePostgres, sqlutil.Dollar, []driver.TableFilter{
		{Column: `a"b`, Op: "eq", Value: "1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sql, `"a""b"`) {
		t.Fatalf("sql=%q", sql)
	}
}

func TestWhereClauseErrors(t *testing.T) {
	_, _, err := sqlutil.WhereClause(sqlutil.QuotePostgres, sqlutil.Dollar, []driver.TableFilter{
		{Column: "", Op: "eq", Value: "1"},
	})
	if err == nil {
		t.Fatal("expected column error")
	}
	_, _, err = sqlutil.WhereClause(sqlutil.QuotePostgres, sqlutil.Dollar, []driver.TableFilter{
		{Column: "n", Op: "between", Value: "1"},
	})
	if err == nil {
		t.Fatal("expected unknown op")
	}
	_, _, err = sqlutil.WhereClause(sqlutil.QuotePostgres, sqlutil.Dollar, []driver.TableFilter{
		{Column: "n", Op: "eq", Value: "1"},
		{Column: "m", Op: "eq", Value: "2", Join: "xor"},
	})
	if err == nil {
		t.Fatal("expected unknown join")
	}
	_, _, err = sqlutil.WhereClause(sqlutil.QuotePostgres, sqlutil.Dollar, []driver.TableFilter{
		{Column: "n", Op: "in", Value: " , "},
	})
	if err == nil {
		t.Fatal("expected empty IN")
	}
}

func TestWhereClauseDollarIndexAfterIn(t *testing.T) {
	sql, args, err := sqlutil.WhereClause(sqlutil.QuotePostgres, sqlutil.Dollar, []driver.TableFilter{
		{Column: "id", Op: "in", Value: "1,2"},
		{Column: "name", Op: "eq", Value: "ada", Join: "and"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := ` WHERE ("id" IN ($1, $2)) AND ("name" = $3)`
	if sql != want {
		t.Fatalf("sql=%q want %q", sql, want)
	}
	if len(args) != 3 || args[2] != "ada" {
		t.Fatalf("args=%v", args)
	}
}
