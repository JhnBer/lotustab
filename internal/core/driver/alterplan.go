package driver

import (
	"fmt"
	"strings"
)

// NormalizeDefault maps catalog empty-string literals to an empty string.
// SQL NULL stays nil. Other expressions are returned unchanged.
func NormalizeDefault(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "''" || trimmed == `""` || strings.HasPrefix(trimmed, "''::") || strings.HasPrefix(trimmed, `""::`) {
		empty := ""
		return &empty
	}
	return value
}

func defaultsEqual(a, b *string) bool {
	na, nb := NormalizeDefault(a), NormalizeDefault(b)
	if na == nil || nb == nil {
		return na == nil && nb == nil
	}
	return *na == *nb
}

func protectedExtra(extra string) bool {
	switch strings.ToUpper(strings.TrimSpace(extra)) {
	case "MATERIALIZED", "ALIAS", "EPHEMERAL":
		return true
	default:
		return false
	}
}

// PlanAlter diffs a structure draft against the catalog and returns the ops
// an engine adapter should render. Unchanged autoIncrement, extra, and primary
// key membership stay on the assembled column and are not taken from the draft.
func PlanAlter(base []ColumnInfo, draft []ColumnDraft, canReorder bool) ([]ColumnOp, error) {
	origByName := make(map[string]ColumnInfo, len(base))
	for _, col := range base {
		origByName[col.Name] = col
	}

	kept := make([]ColumnDraft, 0, len(draft))
	keptOriginal := make(map[string]struct{})
	for _, col := range draft {
		if col.Dropped {
			continue
		}
		if err := requireColumnIdentity(col); err != nil {
			return nil, err
		}
		kept = append(kept, col)
		if col.OriginalName != nil && *col.OriginalName != "" {
			keptOriginal[*col.OriginalName] = struct{}{}
		}
	}

	ops := make([]ColumnOp, 0)
	for _, col := range base {
		if _, ok := keptOriginal[col.Name]; !ok {
			ops = append(ops, ColumnOp{Kind: ColumnOpDrop, Name: col.Name})
		}
	}

	origKept := make([]string, 0)
	for _, col := range base {
		if _, ok := keptOriginal[col.Name]; ok {
			origKept = append(origKept, col.Name)
		}
	}
	draftKept := make([]string, 0, len(kept))
	for _, col := range kept {
		if col.OriginalName != nil && *col.OriginalName != "" {
			draftKept = append(draftKept, *col.OriginalName)
		}
	}
	orderChanged := canReorder && strings.Join(origKept, "\x00") != strings.Join(draftKept, "\x00")

	for i, col := range kept {
		after := positionAfter(orderChanged, col.OriginalName == nil, i, kept)
		if col.OriginalName == nil {
			ops = append(ops, ColumnOp{
				Kind:   ColumnOpAdd,
				Name:   strings.TrimSpace(col.Name),
				Column: assembledNew(col, i+1),
				After:  after,
			})
			continue
		}
		source, ok := origByName[*col.OriginalName]
		if !ok {
			return nil, fmt.Errorf("column %q no longer exists", *col.OriginalName)
		}
		merged, patch, changed := mergeColumn(source, col, i+1)
		if protectedExtra(source.Extra) && (changed || orderChanged) {
			return nil, fmt.Errorf("column %q cannot be altered while its extra is %s", source.Name, source.Extra)
		}
		if changed {
			ops = append(ops, ColumnOp{
				Kind:   ColumnOpModify,
				Name:   source.Name,
				Column: &merged,
				Patch:  &patch,
				After:  after,
			})
			continue
		}
		if orderChanged {
			ops = append(ops, ColumnOp{
				Kind:   ColumnOpReorder,
				Name:   source.Name,
				Column: &merged,
				After:  after,
			})
		}
	}
	return ops, nil
}

func requireColumnIdentity(col ColumnDraft) error {
	if strings.TrimSpace(col.Name) == "" {
		return fmt.Errorf("column name is required")
	}
	if strings.TrimSpace(col.Type) == "" {
		return fmt.Errorf("column type is required")
	}
	return nil
}

func positionAfter(orderChanged, isNew bool, index int, kept []ColumnDraft) *string {
	if !orderChanged && !isNew {
		return nil
	}
	if index == 0 {
		first := ""
		return &first
	}
	prev := strings.TrimSpace(kept[index-1].Name)
	return &prev
}

func assembledNew(col ColumnDraft, ordinal int) *ColumnInfo {
	info := ColumnInfo{
		Name:     strings.TrimSpace(col.Name),
		Type:     strings.TrimSpace(col.Type),
		Nullable: col.Nullable,
		Comment:  strings.TrimSpace(col.Comment),
		Ordinal:  ordinal,
	}
	if col.Default != nil {
		value := *col.Default
		info.Default = &value
	}
	return &info
}

func mergeColumn(base ColumnInfo, draft ColumnDraft, ordinal int) (ColumnInfo, ColumnPatch, bool) {
	merged := base
	merged.Name = strings.TrimSpace(draft.Name)
	merged.Type = strings.TrimSpace(draft.Type)
	merged.Nullable = draft.Nullable
	merged.Comment = strings.TrimSpace(draft.Comment)
	merged.Ordinal = ordinal

	var patch ColumnPatch
	changed := false
	if merged.Name != base.Name {
		name := merged.Name
		patch.Name = &name
		changed = true
	}
	if merged.Type != strings.TrimSpace(base.Type) {
		typ := merged.Type
		patch.Type = &typ
		changed = true
	}
	if merged.Nullable != base.Nullable {
		nullable := merged.Nullable
		patch.Nullable = &nullable
		changed = true
	}
	if !defaultsEqual(base.Default, draft.Default) {
		if draft.Default == nil {
			merged.Default = nil
			patch.ClearDefault = true
		} else {
			value := *NormalizeDefault(draft.Default)
			merged.Default = &value
			patch.Default = &value
		}
		changed = true
	}
	if merged.Comment != strings.TrimSpace(base.Comment) {
		comment := merged.Comment
		patch.Comment = &comment
		changed = true
	}
	return merged, patch, changed
}
