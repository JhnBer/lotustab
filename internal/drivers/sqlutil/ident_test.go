package sqlutil_test

import (
	"testing"

	"lotustab/internal/drivers/sqlutil"
)

func TestQuoteMySQL(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"users", "`users`"},
		{"a`b", "`a``b`"},
		{"", "``"},
	}
	for _, tt := range tests {
		if got := sqlutil.QuoteMySQL(tt.in); got != tt.want {
			t.Fatalf("QuoteMySQL(%q)=%q want %q", tt.in, got, tt.want)
		}
	}
}

func TestQuoteClickHouse(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"users", "`users`"},
		{"a`b", "`a``b`"},
		{"", "``"},
	}
	for _, tt := range tests {
		if got := sqlutil.QuoteClickHouse(tt.in); got != tt.want {
			t.Fatalf("QuoteClickHouse(%q)=%q want %q", tt.in, got, tt.want)
		}
	}
}

func TestQuotePostgres(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"users", `"users"`},
		{`a"b`, `"a""b"`},
		{"", `""`},
	}
	for _, tt := range tests {
		if got := sqlutil.QuotePostgres(tt.in); got != tt.want {
			t.Fatalf("QuotePostgres(%q)=%q want %q", tt.in, got, tt.want)
		}
	}
}

func TestQuoteString(t *testing.T) {
	if got := sqlutil.QuoteString("a'b"); got != "'a''b'" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatDefault(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", "''"},
		{"NULL", "NULL"},
		{"CURRENT_TIMESTAMP", "CURRENT_TIMESTAMP"},
		{"now()", "now()"},
		{"42", "42"},
		{"'foo'::text", "'foo'::text"},
		{"hello", "'hello'"},
		{"a'b", "'a''b'"},
	}
	for _, tt := range tests {
		if got := sqlutil.FormatDefault(tt.in); got != tt.want {
			t.Fatalf("FormatDefault(%q)=%q want %q", tt.in, got, tt.want)
		}
	}
}

func TestOrderClause(t *testing.T) {
	if got := sqlutil.OrderClause(sqlutil.QuotePostgres, "", "asc"); got != "" {
		t.Fatalf("empty orderBy should be empty, got %q", got)
	}
	got := sqlutil.OrderClause(sqlutil.QuoteMySQL, "id", "desc")
	if got != " ORDER BY `id` DESC" {
		t.Fatalf("got %q", got)
	}
	got = sqlutil.OrderClause(sqlutil.QuotePostgres, "name", "asc")
	if got != ` ORDER BY "name" ASC` {
		t.Fatalf("got %q", got)
	}
}

func TestLikeContains(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"users", "%users%"},
		{"a%b", `%a\%b%`},
		{"a_b", `%a\_b%`},
		{`a\b`, `%a\\b%`},
		{"", "%%"},
	}
	for _, tt := range tests {
		if got := sqlutil.LikeContains(tt.in); got != tt.want {
			t.Fatalf("LikeContains(%q)=%q want %q", tt.in, got, tt.want)
		}
	}
}

func TestNormalizeWhere(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"  created_at > 1  ", "created_at > 1"},
		{"created_at > 1;", "created_at > 1"},
		{"WHERE created_at > 1", "created_at > 1"},
		{"where created_at > 1;", "created_at > 1"},
		{"wherever = 1", "wherever = 1"},
	}
	for _, tt := range tests {
		if got := sqlutil.NormalizeWhere(tt.in); got != tt.want {
			t.Errorf("NormalizeWhere(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSelectStar(t *testing.T) {
	if got := sqlutil.SelectStar("`db`.`t`", ""); got != "SELECT * FROM `db`.`t`" {
		t.Fatalf("unfiltered: %q", got)
	}
	if got := sqlutil.SelectStar("`db`.`t`", "WHERE id = 1;"); got != "SELECT * FROM `db`.`t` WHERE id = 1" {
		t.Fatalf("filtered: %q", got)
	}
}

func TestValidIdent(t *testing.T) {
	if !sqlutil.ValidIdent("created_at") || !sqlutil.ValidIdent("_id") || !sqlutil.ValidIdent("col$1") {
		t.Fatal("expected valid identifiers")
	}
	if sqlutil.ValidIdent("") || sqlutil.ValidIdent("1col") || sqlutil.ValidIdent("a;b") || sqlutil.ValidIdent("a b") {
		t.Fatal("expected invalid identifiers")
	}
}
