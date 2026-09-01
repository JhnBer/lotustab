package app

import (
	"database/sql/driver"
	"errors"
	"io"
	"net"
	"testing"
)

func TestIsStaleConn(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{errors.New("syntax error at or near \"SELEC\""), false},
		{io.EOF, true},
		{io.ErrUnexpectedEOF, true},
		{net.ErrClosed, true},
		{driver.ErrBadConn, true},
		{errors.New("invalid connection"), true},
		{errors.New("read tcp 127.0.0.1:5432: unexpected EOF"), true},
		{errors.New("ssh: client is closed"), true},
		{errors.New("write: broken pipe"), true},
	}
	for _, tc := range cases {
		if got := isStaleConn(tc.err); got != tc.want {
			t.Fatalf("isStaleConn(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}
}
