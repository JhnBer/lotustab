import type { Column } from "@/lib/wails";

export type CellDraft = {
  key: Record<string, unknown>;
  cells: Record<string, unknown>;
};

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
