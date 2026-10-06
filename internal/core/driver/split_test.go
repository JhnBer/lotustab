package driver_test

import (
	"strings"
	"testing"

	"lotustab/internal/core/driver"
)

func TestSplitStatements(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want []string
	}{
		{name: "empty", sql: "  \n  ", want: []string{}},
		{name: "only comments", sql: "-- hi\n/* x */", want: []string{}},
		{name: "empty semis", sql: " ; ; \n;", want: []string{}},
		{
			name: "two selects",
			sql:  "SELECT 1; SELECT 2",
			want: []string{"SELECT 1", "SELECT 2"},
		},
		{
			name: "trailing semi",
			sql:  "SELECT 1;",
			want: []string{"SELECT 1"},
		},
		{
			name: "semi in string",
			sql:  "SELECT 'a;b'; SELECT 2",
			want: []string{"SELECT 'a;b'", "SELECT 2"},
		},
		{
			name: "escaped quote",
			sql:  `SELECT 'it''s;ok'; SELECT 2`,
			want: []string{`SELECT 'it''s;ok'`, "SELECT 2"},
		},
		{
			name: "line comment hides semi",
			sql:  "SELECT 1; -- skip; this\nSELECT 2",
			want: []string{"SELECT 1", "-- skip; this\nSELECT 2"},
		},
		{
			name: "block comment hides semi",
			sql:  "SELECT 1 /* ; */ ; SELECT 2",
			want: []string{"SELECT 1 /* ; */", "SELECT 2"},
		},
		{
			name: "dollar quote",
			sql:  "DO $$ BEGIN INSERT INTO t VALUES (1); END; $$; SELECT 1",
			want: []string{"DO $$ BEGIN INSERT INTO t VALUES (1); END; $$", "SELECT 1"},
		},
		{
			name: "tagged dollar quote",
			sql:  "SELECT $tag$ a;b $tag$; SELECT 2",
			want: []string{"SELECT $tag$ a;b $tag$", "SELECT 2"},
		},
		{
			name: "backticks",
			sql:  "SELECT `a;b`; SELECT 2",
			want: []string{"SELECT `a;b`", "SELECT 2"},
		},
		{
			name: "double quotes",
			sql:  `SELECT "a;b"; SELECT 2`,
			want: []string{`SELECT "a;b"`, "SELECT 2"},
		},
		{
			name: "comment-only chunk dropped",
			sql:  "SELECT 1; -- only\n  ; SELECT 2",
			want: []string{"SELECT 1", "SELECT 2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := driver.SplitStatements(tt.sql)
			if len(got) != len(tt.want) {
				t.Fatalf("len=%d want %d\ngot %q", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("stmt %d:\n got %q\nwant %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSplitStatementsKeepsLeadingCommentWithSQL(t *testing.T) {
	got := driver.SplitStatements("-- setup\nSELECT 1;")
	if len(got) != 1 || !strings.Contains(got[0], "SELECT 1") {
		t.Fatalf("got %q", got)
	}
}
