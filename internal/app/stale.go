package app

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"net"
	"strings"
)

// isStaleConn reports whether err means the session (or a pooled socket) is dead
// and a reconnect is likely to recover.
func isStaleConn(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, net.ErrClosed) ||
		errors.Is(err, driver.ErrBadConn) ||
		errors.Is(err, sql.ErrConnDone) {
		return true
	}

	msg := strings.ToLower(err.Error())
	for _, needle := range []string{
		"eof",
		"broken pipe",
		"connection reset",
		"invalid connection",
		"bad connection",
		"conn closed",
		"connection closed",
		"closed network connection",
		"use of closed",
		"sql: database is closed",
		"ssh: client is closed",
		"server closed the connection",
		"unexpected eof",
	} {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}
