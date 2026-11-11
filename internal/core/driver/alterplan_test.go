package driver

import (
	"strings"
	"testing"
)

func strPtr(s string) *string { return &s }

func TestNormalizeDefault(t *testing.T) {
	empty := ""
	cast := "''::text"
	quoted := `""`
	expr := "now()"
	if got := NormalizeDefault(nil); got != nil {
		t.Fatal("nil default should stay nil")
	}
	if got := NormalizeDefault(&empty); got == nil || *got != "" {
		t.Fatalf("empty string: %v", got)
	}
	if got := NormalizeDefault(&cast); got == nil || *got != "" {
		t.Fatalf("cast empty: %v", got)
	}
	if got := NormalizeDefault(&quoted); got == nil || *got != "" {
		t.Fatalf("quoted empty: %v", got)
	}
	if got := NormalizeDefault(&expr); got == nil || *got != "now()" {
		t.Fatalf("expression: %v", got)
	}
}

func TestPlanAlterTypeOnly(t *testing.T) {
	def := "0"
	base := []ColumnInfo{{
		Name: "age", Type: "int", Nullable: false, Default: &def, AutoIncrement: true,
		IsPrimaryKey: true, Extra: "auto_increment", Comment: "years",
	}}
	ops, err := PlanAlter(base, []ColumnDraft{{
		OriginalName: strPtr("age"),
		Name:         "age",
		Type:         "bigint",
		Nullable:     false,
		Default:      &def,
		Comment:      "years",
	}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 1 || ops[0].Kind != ColumnOpModify {
		t.Fatalf("ops: %+v", ops)
	}
	patch := ops[0].Patch
	if patch == nil || patch.Type == nil || *patch.Type != "bigint" || patch.Name != nil || patch.Nullable != nil || patch.Default != nil || patch.ClearDefault || patch.Comment != nil {
		t.Fatalf("patch: %+v", patch)
	}
	col := ops[0].Column
	if col == nil || !col.AutoIncrement || col.Extra != "auto_increment" || !col.IsPrimaryKey || col.Default == nil || *col.Default != "0" || col.Comment != "years" {
		t.Fatalf("assembled: %+v", col)
	}
}

func TestPlanAlterCommentOnly(t *testing.T) {
	base := []ColumnInfo{{Name: "name", Type: "text", Nullable: true, Comment: ""}}
	ops, err := PlanAlter(base, []ColumnDraft{{
		OriginalName: strPtr("name"),
		Name:         "name",
		Type:         "text",
		Nullable:     true,
		Comment:      "full name",
	}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 1 || ops[0].Patch == nil || ops[0].Patch.Comment == nil || *ops[0].Patch.Comment != "full name" {
		t.Fatalf("ops: %+v", ops)
	}
	if ops[0].Patch.Type != nil {
		t.Fatal("comment change should not patch the type")
	}
}

func TestPlanAlterClearDefault(t *testing.T) {
	def := "guest"
	base := []ColumnInfo{{Name: "name", Type: "text", Nullable: true, Default: &def}}
	ops, err := PlanAlter(base, []ColumnDraft{{
		OriginalName: strPtr("name"),
		Name:         "name",
		Type:         "text",
		Nullable:     true,
	}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 1 || ops[0].Patch == nil || !ops[0].Patch.ClearDefault || ops[0].Patch.Default != nil {
		t.Fatalf("ops: %+v", ops)
	}
	if ops[0].Column == nil || ops[0].Column.Default != nil {
		t.Fatal("cleared default should be nil on the assembled column")
	}
}

func TestPlanAlterEmptyStringDefault(t *testing.T) {
	base := []ColumnInfo{{Name: "name", Type: "text", Nullable: true}}
	empty := ""
	ops, err := PlanAlter(base, []ColumnDraft{{
		OriginalName: strPtr("name"),
		Name:         "name",
		Type:         "text",
		Nullable:     true,
		Default:      &empty,
	}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 1 || ops[0].Patch == nil || ops[0].Patch.ClearDefault || ops[0].Patch.Default == nil || *ops[0].Patch.Default != "" {
		t.Fatalf("ops: %+v", ops)
	}
}

func TestPlanAlterNormalizedDefaultIsClean(t *testing.T) {
	cast := "''::text"
	empty := ""
	base := []ColumnInfo{{Name: "name", Type: "text", Nullable: true, Default: &cast}}
	ops, err := PlanAlter(base, []ColumnDraft{{
		OriginalName: strPtr("name"),
		Name:         "name",
		Type:         "text",
		Nullable:     true,
		Default:      &empty,
	}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 0 {
		t.Fatalf("normalized empty default should not change the column, got %+v", ops)
	}
}

func TestPlanAlterReorderKeptColumns(t *testing.T) {
	base := []ColumnInfo{
		{Name: "id", Type: "int"},
		{Name: "name", Type: "text", Nullable: true},
	}
	ops, err := PlanAlter(base, []ColumnDraft{
		{OriginalName: strPtr("name"), Name: "name", Type: "text", Nullable: true},
		{OriginalName: strPtr("id"), Name: "id", Type: "int"},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 2 || ops[0].Kind != ColumnOpReorder || ops[1].Kind != ColumnOpReorder {
		t.Fatalf("ops: %+v", ops)
	}
	if ops[0].After == nil || *ops[0].After != "" || ops[1].After == nil || *ops[1].After != "name" {
		t.Fatalf("positions: %+v %+v", ops[0].After, ops[1].After)
	}
}

func TestPlanAlterRefusesMaterializedRewrite(t *testing.T) {
	base := []ColumnInfo{{Name: "v", Type: "String", Extra: "MATERIALIZED", Comment: "expr"}}
	_, err := PlanAlter(base, []ColumnDraft{{
		OriginalName: strPtr("v"),
		Name:         "v",
		Type:         "String",
		Comment:      "changed",
	}}, false)
	if err == nil || !strings.Contains(err.Error(), "MATERIALIZED") {
		t.Fatalf("expected protected extra error, got %v", err)
	}
}
