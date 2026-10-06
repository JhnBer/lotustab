package sqlutil

import (
	"context"
	"database/sql"

	"lotustab/internal/core/driver"
)

// ScriptConn pins a database/sql session for driver.RunScript.
type ScriptConn struct {
	DB *sql.Conn
	Tx func(context.Context, *sql.Conn) (bool, error)
}

func (s ScriptConn) Query(ctx context.Context, q string, maxRows int) (*driver.QueryResult, error) {
	return Query(ctx, s.DB, q, maxRows)
}

func (s ScriptConn) InTransaction(ctx context.Context) (bool, error) {
	if s.Tx == nil {
		return false, nil
	}
	return s.Tx(ctx, s.DB)
}

func (s ScriptConn) Rollback(ctx context.Context) error {
	_, err := s.DB.ExecContext(ctx, "ROLLBACK")
	return err
}

var _ driver.ScriptSession = ScriptConn{}
