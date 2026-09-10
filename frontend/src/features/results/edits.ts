import type { Column } from "@/lib/wails";

export const INSERT_ID_KEY = "__insertId";

export type DraftKind = "update" | "delete" | "insert";

export type CellDraft = {
  key: Record<string, unknown>;
  cells: Record<string, unknown>;
  kind?: DraftKind;
  after?: Record<string, unknown>;
};

export function draftKind(edit: CellDraft): DraftKind {
  return edit.kind ?? "update";
}

export function isInsertDraft(edit: CellDraft): boolean {
  return draftKind(edit) === "insert";
}

export function isDeleteDraft(edit: CellDraft): boolean {
  return draftKind(edit) === "delete";
}

export function insertedDrafts(edits: CellDraft[]): CellDraft[] {
  return edits.filter(isInsertDraft);
}

export function insertIdOf(edit: CellDraft): string {
  return String(edit.key?.[INSERT_ID_KEY] ?? "");
}

export function isRowDeleted(edits: CellDraft[], key: Record<string, unknown> | null): boolean {
  if (!key) {
    return false;
  }
  const lookup = draftLookup(key);
  return edits.some((edit) => isDeleteDraft(edit) && draftLookup(edit.key) === lookup);
}

export function isNull(value: unknown) {
  return value === null || value === undefined;
}

export function valuesEqual(a: unknown, b: unknown) {
  if (isNull(a) && isNull(b)) {
    return true;
  }
  if (isNull(a) || isNull(b)) {
    return false;
  }
  if (typeof a === "object" || typeof b === "object") {
    return JSON.stringify(a) === JSON.stringify(b);
  }
  return String(a) === String(b);
}

export function rowKey(
  row: unknown[] | null | undefined,
  columns: Column[],
  primaryKey: string[],
): Record<string, unknown> | null {
  if (!row || primaryKey.length === 0) {
    return null;
  }
  const key: Record<string, unknown> = {};
  for (const name of primaryKey) {
    const index = columns.findIndex((column) => column.name === name);
    if (index < 0) {
      return null;
    }
    key[name] = row[index] ?? null;
  }
  return key;
}

export function draftLookup(key: Record<string, unknown>) {
  return JSON.stringify(key);
}

export function editInputText(value: unknown) {
  if (isNull(value)) {
    return "";
  }
  if (typeof value === "object") {
    return JSON.stringify(value);
  }
  return String(value);
}

export function rowClipboardText(values: unknown[]) {
  return values.map((value) => editInputText(value)).join("\t");
}

export function rowsClipboardText(grid: unknown[][]) {
  return grid.map((values) => rowClipboardText(values)).join("\n");
}

export function cellRangeToJson(colNames: string[], grid: unknown[][]) {
  const objects = grid.map((values) => {
    const row: Record<string, unknown> = {};
    for (let i = 0; i < colNames.length; i++) {
      row[colNames[i] ?? ""] = values[i] ?? null;
    }
    return row;
  });
  return JSON.stringify(objects);
}

export type VisualRow = {
  values: unknown[];
  liveIndex?: number;
  draft?: CellDraft;
};

export function layoutVisualRows(
  liveRows: (unknown[] | null)[],
  columns: Column[],
  edits: CellDraft[],
  primaryKey: string[],
): VisualRow[] {
  const inserts = insertedDrafts(edits);
  const afterMap = new Map<string, CellDraft[]>();
  const tail: CellDraft[] = [];
  for (const ins of inserts) {
    if (!ins.after) {
      tail.push(ins);
      continue;
    }
    const lookup = draftLookup(ins.after);
    const list = afterMap.get(lookup);
    if (list) {
      list.push(ins);
    } else {
      afterMap.set(lookup, [ins]);
    }
  }

  const out: VisualRow[] = [];
  const placed = new Set<string>();

  function pushInsert(ins: CellDraft) {
    const id = insertIdOf(ins);
    if (placed.has(id)) {
      return;
    }
    placed.add(id);
    out.push({
      values: columns.map((column) => ins.cells[column.name] ?? null),
      draft: ins,
    });
    emitInsertsAfter(ins.key);
  }

  function emitInsertsAfter(key: Record<string, unknown>) {
    const list = afterMap.get(draftLookup(key));
    if (!list) {
      return;
    }
    for (const ins of list) {
      pushInsert(ins);
    }
  }

  for (let i = 0; i < liveRows.length; i++) {
    const row = liveRows[i] ?? [];
    out.push({ values: row, liveIndex: i });
    const key = rowKey(row, columns, primaryKey);
    if (key) {
      emitInsertsAfter(key);
    }
  }
  for (const ins of tail) {
    pushInsert(ins);
  }
  for (const ins of inserts) {
    pushInsert(ins);
  }
  return out;
}

export function visualRowIdentity(
  row: VisualRow,
  columns: Column[],
  primaryKey: string[],
): Record<string, unknown> | null {
  if (row.draft) {
    return row.draft.key;
  }
  return rowKey(row.values, columns, primaryKey);
}
