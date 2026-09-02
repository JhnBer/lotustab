package app

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"lotustab/internal/core/connection"
	"lotustab/internal/core/driver"
	"lotustab/internal/storage"
)

type stubDriver struct {
	mu    sync.Mutex
	opens int
	delay time.Duration
}

func (d *stubDriver) ID() string          { return "stub" }
func (d *stubDriver) DisplayName() string { return "Stub" }
func (d *stubDriver) ConnectionSpec() []driver.Field {
	return nil
}
func (d *stubDriver) Open(context.Context, driver.ConnConfig) (driver.Conn, error) {
	if d.delay > 0 {
		time.Sleep(d.delay)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.opens++
	return &stubConn{id: d.opens}, nil
}

func (d *stubDriver) openCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.opens
}

type stubConn struct {
	id           int
	ping         error
	query        error
	closed       bool
	queries      int
	blockQuery   bool
	queryStarted chan struct{}
}

func (c *stubConn) Ping(context.Context) error { return c.ping }
func (c *stubConn) Query(ctx context.Context, _ string, _ string, _ int, _ int) (*driver.QueryResult, error) {
	c.queries++
	if c.queryStarted != nil {
		select {
		case <-c.queryStarted:
		default:
			close(c.queryStarted)
		}
	}
	if c.blockQuery {
		<-ctx.Done()
		return nil, ctx.Err()
	}
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
func (c *stubConn) DescribeTable(context.Context, driver.TableRef) (*driver.TableStructure, error) {
	return &driver.TableStructure{Columns: []driver.ColumnInfo{}}, nil
}
func (c *stubConn) AlterTable(context.Context, driver.AlterTableRequest) error { return nil }
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
	if d.openCount() != 1 {
		t.Fatalf("opens = %d, want 1", d.openCount())
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
	if d.openCount() != 0 {
		t.Fatalf("opens = %d, want 0", d.openCount())
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
	_, err := qs.Execute(ctx, "p1", "SELECT 1", "", 10, 0, 0)
	if err == nil {
		t.Fatal("expected canceled query to fail")
	}
	if d.openCount() != 0 {
		t.Fatalf("opens = %d, want 0", d.openCount())
	}
}

func TestExecuteRetriesStaleQuery(t *testing.T) {
	d := &stubDriver{}
	s := stubService(t, d)
	stale := &stubConn{query: fmt.Errorf("read: unexpected EOF")}
	s.manager.Put("p1", stale)

	qs := NewQueryServiceFromConnections(s)
	_, err := qs.Execute(context.Background(), "p1", "SELECT 1", "", 10, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !stale.closed {
		t.Fatal("stale connection was not closed after retry")
	}
	if d.openCount() != 1 {
		t.Fatalf("opens = %d, want 1", d.openCount())
	}
}

func TestOpenOnceSharesConcurrentOpens(t *testing.T) {
	d := &stubDriver{delay: 80 * time.Millisecond}
	s := stubService(t, d)
	var wg sync.WaitGroup
	wg.Add(2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			_, err := s.ensureOpen(context.Background(), "p1")
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if d.openCount() != 1 {
		t.Fatalf("opens = %d, want 1", d.openCount())
	}
}

func TestDisconnectCancelsQuery(t *testing.T) {
	d := &stubDriver{}
	s := stubService(t, d)
	started := make(chan struct{})
	s.manager.Put("p1", &stubConn{blockQuery: true, queryStarted: started})
	qs := NewQueryServiceFromConnections(s)

	done := make(chan error, 1)
	go func() {
		_, err := qs.Execute(context.Background(), "p1", "SELECT 1", "", 10, 0, 0)
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("query did not start")
	}
	if err := s.Disconnect("p1"); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected canceled query")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("query was not canceled")
	}
}

func TestExecuteTimeout(t *testing.T) {
	d := &stubDriver{}
	s := stubService(t, d)
	s.manager.Put("p1", &stubConn{blockQuery: true})
	qs := NewQueryServiceFromConnections(s)
	_, err := qs.Execute(context.Background(), "p1", "SELECT 1", "", 10, 0, 1)
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected timeout error, got %v", err)
	}
}
