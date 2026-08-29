package app

import (
	"strings"
	"testing"
)

func TestOpenConnectionRequiresProfile(t *testing.T) {
	s := NewWindowsService()
	err := s.OpenConnection("  ", "db")
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
