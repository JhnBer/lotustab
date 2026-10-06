package driver_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"lotustab/internal/core/driver"
)

type fakeSession struct {
	queries   []string
	inTx      bool
	rollbacks int
	failAt    int
	failErr   error
}

func (s *fakeSession) Query(_ context.Context, sql string, _ int) (*driver.QueryResult, error) {
	s.queries = append(s.queries, sql)
	upper := strings.ToUpper(strings.TrimSpace(sql))
	switch {
	case strings.HasPrefix(upper, "BEGIN") || strings.HasPrefix(upper, "START TRANSACTION"):
		s.inTx = true
	case strings.HasPrefix(upper, "COMMIT") || strings.HasPrefix(upper, "ROLLBACK"):
		s.inTx = false
	}
	if s.failAt > 0 && len(s.queries) == s.failAt {
		err := s.failErr
		if err == nil {
			err = errors.New("boom")
		}
		return nil, err
	}
	return &driver.QueryResult{}, nil
}

func (s *fakeSession) InTransaction(context.Context) (bool, error) {
	return s.inTx, nil
}

func (s *fakeSession) Rollback(context.Context) error {
	s.rollbacks++
	s.inTx = false
	return nil
}

func TestRunScriptSameSessionStopOnError(t *testing.T) {
	sess := &fakeSession{failAt: 2}
	var got []int
	err := driver.RunScript(context.Background(), sess, []string{"SELECT 1", "SELECT 2", "SELECT 3"}, 100, func(i int, _ *driver.QueryResult, _ error) error {
		got = append(got, i)
		return nil
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Join(sess.queries, "|") != "SELECT 1|SELECT 2" {
		t.Fatalf("queries %q", sess.queries)
	}
	if len(got) != 2 {
		t.Fatalf("emits %v", got)
	}
}

func TestRunScriptRollbackOpenTransaction(t *testing.T) {
	sess := &fakeSession{}
	err := driver.RunScript(context.Background(), sess, []string{"BEGIN", "INSERT INTO t VALUES (1)"}, 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	if sess.rollbacks != 1 {
		t.Fatalf("rollbacks=%d", sess.rollbacks)
	}
	if sess.inTx {
		t.Fatal("transaction still open")
	}
}

func TestRunScriptNoRollbackAfterCommit(t *testing.T) {
	sess := &fakeSession{}
	err := driver.RunScript(context.Background(), sess, []string{"BEGIN", "INSERT INTO t VALUES (1)", "COMMIT"}, 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	if sess.rollbacks != 0 {
		t.Fatalf("rollbacks=%d", sess.rollbacks)
	}
}

func TestRunScriptStartTransactionRollback(t *testing.T) {
	sess := &fakeSession{}
	err := driver.RunScript(context.Background(), sess, []string{"START TRANSACTION", "INSERT INTO t VALUES (1)"}, 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	if sess.rollbacks != 1 {
		t.Fatalf("rollbacks=%d", sess.rollbacks)
	}
}

func TestRunScriptCancelEmitsCurrent(t *testing.T) {
	sess := &fakeSession{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var emitted error
	err := driver.RunScript(ctx, sess, []string{"SELECT 1"}, 100, func(_ int, _ *driver.QueryResult, qerr error) error {
		emitted = qerr
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
	if !errors.Is(emitted, context.Canceled) {
		t.Fatalf("emitted=%v", emitted)
	}
	if len(sess.queries) != 0 {
		t.Fatalf("should not query after cancel, got %q", sess.queries)
	}
}
