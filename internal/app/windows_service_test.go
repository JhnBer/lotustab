package app

import (
	"strings"
	"testing"
)

func TestOpenConnectionRequiresProfile(t *testing.T) {
	s := NewWindowsService()
	err := s.OpenConnection("  ", "db", "")
	if err == nil || !strings.Contains(err.Error(), "profile") {
		t.Fatalf("expected profile error, got %v", err)
	}
}

func TestWindowOptions(t *testing.T) {
	opts := WindowOptions("main", "LotusTab", "/")
	if opts.Name != "main" || opts.Title != "LotusTab" || opts.URL != "/" {
		t.Fatalf("unexpected options: %+v", opts)
	}
	if opts.Width != 1280 || opts.Height != 800 {
		t.Fatalf("unexpected size: %dx%d", opts.Width, opts.Height)
	}
}

func TestParkAndTakeHandoff(t *testing.T) {
	s := NewWindowsService()
	if s.park("p1", "") || s.park("p1", "  ") {
		t.Fatal("empty snapshot should not park")
	}
	if got := s.TakeHandoff("p1"); got != "" {
		t.Fatalf("expected empty take, got %q", got)
	}

	if !s.park("p1", `{"tabs":[]}`) {
		t.Fatal("expected park")
	}
	if got := s.TakeHandoff("p1"); got != `{"tabs":[]}` {
		t.Fatalf("got %q", got)
	}
	if got := s.TakeHandoff("p1"); got != "" {
		t.Fatalf("second take should be empty, got %q", got)
	}
}
