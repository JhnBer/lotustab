package sqlutil

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
)

type fakeResult struct {
	n int64
}

func (f fakeResult) LastInsertId() (int64, error) { return 0, nil }
func (f fakeResult) RowsAffected() (int64, error) { return f.n, nil }

type fakeDB struct {
	queries  int
	execs    int
	affected int64
	queryErr error
	execErr  error
}

func (f *fakeDB) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	f.queries++
	if f.queryErr != nil {
		return nil, f.queryErr
	}
	return nil, fmt.Errorf("no rows")
}

func (f *fakeDB) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	f.execs++
	if f.execErr != nil {
		return nil, f.execErr
	}
	return fakeResult{n: f.affected}, nil
}

func TestQueryExecsDML(t *testing.T) {
	db := &fakeDB{affected: 7}
	got, err := Query(context.Background(), db, "INSERT INTO t VALUES (1)", 100)
	if err != nil {
		t.Fatal(err)
	}
	if db.queries != 0 || db.execs != 1 {
		t.Fatalf("queries=%d execs=%d", db.queries, db.execs)
	}
	if got.RowsAffected != 7 || !got.AffectedKnown {
		t.Fatalf("affected=%d known=%v", got.RowsAffected, got.AffectedKnown)
	}
}

func TestQueryExecsUpdateZeroKnown(t *testing.T) {
	db := &fakeDB{affected: 0}
	got, err := Query(context.Background(), db, "UPDATE t SET a = 1 WHERE false", 100)
	if err != nil {
		t.Fatal(err)
	}
	if db.execs != 1 || got.RowsAffected != 0 || !got.AffectedKnown {
		t.Fatalf("execs=%d affected=%d known=%v", db.execs, got.RowsAffected, got.AffectedKnown)
	}
}

func TestQueryBeginNotKnown(t *testing.T) {
	db := &fakeDB{affected: 0}
	got, err := Query(context.Background(), db, "BEGIN", 100)
	if err != nil {
		t.Fatal(err)
	}
	if db.queries != 0 || db.execs != 1 {
		t.Fatalf("queries=%d execs=%d", db.queries, db.execs)
	}
	if got.AffectedKnown {
		t.Fatal("BEGIN should not report affected")
	}
}

func TestQueryCommentThenInsert(t *testing.T) {
	db := &fakeDB{affected: 1}
	got, err := Query(context.Background(), db, "-- setup\nINSERT INTO t VALUES (1)", 100)
	if err != nil {
		t.Fatal(err)
	}
	if db.queries != 0 || db.execs != 1 || !got.AffectedKnown {
		t.Fatalf("queries=%d execs=%d known=%v", db.queries, db.execs, got.AffectedKnown)
	}
}

func TestQuerySelectUsesQuery(t *testing.T) {
	db := &fakeDB{queryErr: fmt.Errorf("driver query"), execErr: fmt.Errorf("driver exec")}
	_, err := Query(context.Background(), db, "SELECT 1", 100)
	if err == nil {
		t.Fatal("expected error")
	}
	if db.queries != 1 {
		t.Fatalf("SELECT should Query first, queries=%d", db.queries)
	}
}

func TestQueryWithUsesQuery(t *testing.T) {
	db := &fakeDB{queryErr: fmt.Errorf("driver query"), execErr: fmt.Errorf("driver exec")}
	_, err := Query(context.Background(), db, "WITH t AS (SELECT 1) SELECT * FROM t", 100)
	if err == nil {
		t.Fatal("expected error")
	}
	if db.queries != 1 {
		t.Fatalf("WITH should Query, queries=%d", db.queries)
	}
}
