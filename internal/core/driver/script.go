package driver

import (
	"context"
	"time"
)

// StatementFunc is called after each statement in RunScript, including failures.
type StatementFunc func(index int, result *QueryResult, err error) error

const scriptRollbackTimeout = 5 * time.Second

// RunScript executes statements in order on one session. It stops at the first
// error. If a transaction is still open when it returns, it rolls that
// transaction back on a fresh timeout context so cancel/timeout still clean up.
func RunScript(ctx context.Context, sess ScriptSession, statements []string, maxRows int, emit StatementFunc) error {
	if maxRows <= 0 {
		maxRows = MaxBrowseLimit
	}
	defer rollbackOpen(sess)

	for i, stmt := range statements {
		if err := ctx.Err(); err != nil {
			if emit != nil {
				_ = emit(i, nil, err)
			}
			return err
		}
		result, err := sess.Query(ctx, stmt, maxRows)
		if emit != nil {
			if emitErr := emit(i, result, err); emitErr != nil {
				return emitErr
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func rollbackOpen(sess ScriptSession) {
	if sess == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), scriptRollbackTimeout)
	defer cancel()
	inTx, err := sess.InTransaction(ctx)
	if err != nil || !inTx {
		return
	}
	_ = sess.Rollback(ctx)
}
