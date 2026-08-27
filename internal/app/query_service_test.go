package app

import (
	"context"
	"strings"
	"testing"

	"lotustab/internal/core/driver"
)

func TestBrowseTableValidatesInput(t *testing.T) {
	s := &QueryService{}
	_, err := s.BrowseTable(context.Background(), "", driver.TableBrowseRequest{Name: "users"})
	if err == nil || !strings.Contains(err.Error(), "profile") {
		t.Fatalf("expected profile error, got %v", err)
	}
	_, err = s.BrowseTable(context.Background(), "abc", driver.TableBrowseRequest{Name: "  "})
	if err == nil || !strings.Contains(err.Error(), "table") {
		t.Fatalf("expected table error, got %v", err)
	}
	_, err = s.BrowseTable(context.Background(), "abc", driver.TableBrowseRequest{Limit: 10})
	if err == nil || !strings.Contains(err.Error(), "table") {
		t.Fatalf("expected table error after clamp, got %v", err)
	}
}

func TestListTablesRequiresProfile(t *testing.T) {
	s := &QueryService{}
	_, err := s.ListTables(context.Background(), "", "db")
	if err == nil || !strings.Contains(err.Error(), "profile") {
		t.Fatalf("expected profile error, got %v", err)
	}
}
