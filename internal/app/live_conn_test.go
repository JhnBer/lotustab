package app

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"testing"

	"lotustab/internal/core/connection"
	"lotustab/internal/core/driver"
	"lotustab/internal/storage"
)

type stubDriver struct {
	opens int
}

func (d *stubDriver) ID() string          { return "stub" }
func (d *stubDriver) DisplayName() string { return "Stub" }
func (d *stubDriver) ConnectionSpec() []driver.Field {
	return nil
}
func (d *stubDriver) Open(context.Context, driver.ConnConfig) (driver.Conn, error) {
	d.opens++
	return &stubConn{id: d.opens}, nil
}

type stubConn struct {
	id      int
	ping    error
	query   error
	closed  bool
	queries int
}

func (c *stubConn) Ping(context.Context) error { return c.ping }
func (c *stubConn) Query(context.Context, string, string, int, int) (*driver.QueryResult, error) {
	c.queries++
	if c.query != nil {
		return nil, c.query
	}
	return &driver.QueryResult{}, nil
}
func (c *stubConn) Databases(context.Context) ([]string, error) { return nil, nil }
func (c *stubConn) ListTables(context.Context, string) ([]driver.TableInfo, error) {
	return []driver.TableInfo{}, nil
}
func (c *stubConn) BrowseTable(context.Context, driver.TableBrowseRequest) (*driver.TableBrowseResult, error) {
	return &driver.TableBrowseResult{}, nil
}
func (c *stubConn) ApplyEdits(context.Context, driver.ApplyEditsRequest) error { return nil }
func (c *stubConn) Close() error {
	c.closed = true
	return nil
}

func stubService(t *testing.T, d *stubDriver) *ConnectionsService {
	t.Helper()
	reg := driver.NewRegistry()
	if err := reg.Register(d); err != nil {
		t.Fatal(err)
	}
	store := storage.NewAt(filepath.Join(t.TempDir(), "connections.json"))
	profile := connection.Profile{ID: "p1", Name: "stub", DriverID: "stub", Params: map[string]string{}}
	if err := store.Upsert(profile); err != nil {
		t.Fatal(err)
	}
	return NewConnectionsService(reg, store, nil, connection.NewManager())
}

func TestEnsureOpenReconnectsStaleSession(t *testing.T) {
	d := &stubDriver{}
	s := stubService(t, d)
	dead := &stubConn{ping: io.EOF}
	s.manager.Put("p1", dead)

	conn, err := s.ensureOpen(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	if !dead.closed {
		t.Fatal("stale connection was not closed")
	}
	if d.opens != 1 {
		t.Fatalf("opens = %d, want 1", d.opens)
	}
	got, ok := conn.(*stubConn)
	if !ok || got.id != 1 {
		t.Fatalf("got %#v", conn)
	}
}

func TestEnsureOpenKeepsHealthySession(t *testing.T) {
	d := &stubDriver{}
	s := stubService(t, d)
	live := &stubConn{id: 99}
	s.manager.Put("p1", live)

	conn, err := s.ensureOpen(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	if d.opens != 0 {
		t.Fatalf("opens = %d, want 0", d.opens)
	}
	if conn != live {
		t.Fatal("healthy session was replaced")
	}
}

func TestExecuteDoesNotRetryAfterCancel(t *testing.T) {
	d := &stubDriver{}
	s := stubService(t, d)
	stale := &stubConn{query: fmt.Errorf("read: unexpected EOF")}
	s.manager.Put("p1", stale)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	qs := NewQueryServiceFromConnections(s)
	_, err := qs.Execute(ctx, "p1", "SELECT 1", "", 10, 0)
	if err == nil {
		t.Fatal("expected canceled query to fail")
	}
	if d.opens != 0 {
		t.Fatalf("opens = %d, want 0", d.opens)
	}
}

func TestExecuteRetriesStaleQuery(t *testing.T) {
	d := &stubDriver{}
	s := stubService(t, d)
	stale := &stubConn{query: fmt.Errorf("read: unexpected EOF")}
	s.manager.Put("p1", stale)

	qs := NewQueryServiceFromConnections(s)
	_, err := qs.Execute(context.Background(), "p1", "SELECT 1", "", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !stale.closed {
		t.Fatal("stale connection was not closed after retry")
	}
	if d.opens != 1 {
		t.Fatalf("opens = %d, want 1", d.opens)
	}
}
