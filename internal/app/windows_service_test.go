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

func TestOpenValueWindowRequiresToken(t *testing.T) {
	s := NewWindowsService()
	err := s.OpenValueWindow("  ", "Value", `{"text":"x"}`)
	if err == nil || !strings.Contains(err.Error(), "token") {
		t.Fatalf("expected token error, got %v", err)
	}
}

func TestParkAndTakeValueHandoff(t *testing.T) {
	s := NewWindowsService()
	if s.parkValue("t1", "") || s.parkValue("t1", "  ") {
		t.Fatal("empty payload should not park")
	}
	if !s.parkValue("t1", `{"text":"hello"}`) {
		t.Fatal("expected park")
	}
	if got := s.TakeValueHandoff("t1"); got != `{"text":"hello"}` {
		t.Fatalf("got %q", got)
	}
	if got := s.TakeValueHandoff("t1"); got != "" {
		t.Fatalf("second take should be empty, got %q", got)
	}
}

func TestParkAndTakeValueReturn(t *testing.T) {
	s := NewWindowsService()
	s.parkValueReturn("t1", `{"text":"edited"}`)
	if got := s.TakeValueReturn("t1"); got != `{"text":"edited"}` {
		t.Fatalf("got %q", got)
	}
	if got := s.TakeValueReturn("t1"); got != "" {
		t.Fatalf("second take should be empty, got %q", got)
	}
}
