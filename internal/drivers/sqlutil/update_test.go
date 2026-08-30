package sqlutil_test

import (
	"context"
	"fmt"
	"testing"

	"lotustab/internal/core/driver"
	"lotustab/internal/drivers/sqlutil"
)

func TestPrepareUpdatePostgres(t *testing.T) {
	sql, args, err := sqlutil.PrepareUpdate(
		sqlutil.QuotePostgres,
		sqlutil.Dollar,
		`"public"."users"`,
		map[string]any{"name": "ada", "age": 36},
		[]string{"id"},
		map[string]any{"id": 7},
	)
	if err != nil {
		t.Fatal(err)
	}
	want := `UPDATE "public"."users" SET "age"=$1, "name"=$2 WHERE "id"=$3`
	if sql != want {
		t.Fatalf("sql=%q want %q", sql, want)
	}
	if len(args) != 3 || args[0] != 36 || args[1] != "ada" || args[2] != 7 {
		t.Fatalf("args=%v", args)
	}
}

func TestPrepareUpdateMySQLComposite(t *testing.T) {
	sql, args, err := sqlutil.PrepareUpdate(
		sqlutil.QuoteMySQL,
		sqlutil.Question,
		"`app`.`items`",
		map[string]any{"note": nil},
		[]string{"a", "b"},
		map[string]any{"a": 1, "b": "x"},
	)
	if err != nil {
		t.Fatal(err)
	}
	want := "UPDATE `app`.`items` SET `note`=? WHERE `a`=? AND `b`=?"
	if sql != want {
		t.Fatalf("sql=%q want %q", sql, want)
	}
	if len(args) != 3 || args[0] != nil || args[1] != 1 || args[2] != "x" {
		t.Fatalf("args=%v", args)
	}
}

func TestPrepareUpdateQuotesSpecialChars(t *testing.T) {
	sql, _, err := sqlutil.PrepareUpdate(
		sqlutil.QuotePostgres,
		sqlutil.Dollar,
		`"t"`,
		map[string]any{`a"b`: "1"},
		[]string{`c"d`},
		map[string]any{`c"d`: 2},
	)
	if err != nil {
		t.Fatal(err)
	}
	want := `UPDATE "t" SET "a""b"=$1 WHERE "c""d"=$2`
	if sql != want {
		t.Fatalf("sql=%q want %q", sql, want)
	}
}

func TestPrepareUpdateErrors(t *testing.T) {
	_, _, err := sqlutil.PrepareUpdate(sqlutil.QuotePostgres, sqlutil.Dollar, "", map[string]any{"n": 1}, []string{"id"}, map[string]any{"id": 1})
	if err == nil {
		t.Fatal("expected table error")
	}
	_, _, err = sqlutil.PrepareUpdate(sqlutil.QuotePostgres, sqlutil.Dollar, "t", map[string]any{"n": 1}, nil, map[string]any{"id": 1})
	if err == nil {
		t.Fatal("expected pk error")
	}
	_, _, err = sqlutil.PrepareUpdate(sqlutil.QuotePostgres, sqlutil.Dollar, "t", nil, []string{"id"}, map[string]any{"id": 1})
	if err == nil {
		t.Fatal("expected cells error")
	}
	_, _, err = sqlutil.PrepareUpdate(sqlutil.QuotePostgres, sqlutil.Dollar, "t", map[string]any{"n": 1}, []string{"id"}, map[string]any{})
	if err == nil {
		t.Fatal("expected missing key error")
	}
}

func TestApplyRowEditsAllOrNothingCount(t *testing.T) {
	var got []string
	exec := func(_ context.Context, q string, args ...any) (int64, error) {
		got = append(got, fmt.Sprintf("%s %v", q, args))
		return 1, nil
	}
	err := sqlutil.ApplyRowEdits(context.Background(), exec, sqlutil.QuoteMySQL, sqlutil.Question, "`t`", []string{"id"}, []driver.RowEdit{
		{Key: map[string]any{"id": 1}, Cells: map[string]any{"n": "a"}},
		{Key: map[string]any{"id": 2}, Cells: map[string]any{}},
		{Key: map[string]any{"id": 3}, Cells: map[string]any{"n": "b"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 updates, got %d: %v", len(got), got)
	}
}

func TestApplyRowEditsRejectsWrongAffected(t *testing.T) {
	exec := func(context.Context, string, ...any) (int64, error) { return 0, nil }
	err := sqlutil.ApplyRowEdits(context.Background(), exec, sqlutil.QuotePostgres, sqlutil.Dollar, `"t"`, []string{"id"}, []driver.RowEdit{
		{Key: map[string]any{"id": 1}, Cells: map[string]any{"n": "a"}},
	})
	if err == nil {
		t.Fatal("expected affected-count error")
	}
}

func TestApplyRowEditsNoPrimaryKey(t *testing.T) {
	err := sqlutil.ApplyRowEdits(context.Background(), nil, sqlutil.QuotePostgres, sqlutil.Dollar, `"t"`, nil, []driver.RowEdit{
		{Key: map[string]any{"id": 1}, Cells: map[string]any{"n": "a"}},
	})
	if err == nil {
		t.Fatal("expected pk error")
	}
}
