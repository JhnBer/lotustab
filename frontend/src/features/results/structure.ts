import type { ColumnInfo, ColumnOp } from "@/lib/wails";

export type TableView = "data" | "structure";

export type DraftColumn = {
  key: string;
  originalName: string | null;
  dropped: boolean;
  name: string;
  type: string;
  nullable: boolean;
  defaultValue: string | null;
  isPrimaryKey: boolean;
  comment: string;
};

export const COMMON_COLUMN_TYPES = [
  "int",
  "bigint",
  "varchar(255)",
  "text",
  "boolean",
  "date",
  "datetime",
  "timestamp",
  "timestamptz",
  "numeric",
  "decimal(10,2)",
  "json",
  "jsonb",
  "uuid",
  "bytea",
  "blob",
];

export const CLICKHOUSE_COLUMN_TYPES = [
  "String",
  "FixedString(16)",
  "Bool",
  "Int8",
  "Int16",
  "Int32",
  "Int64",
  "Int128",
  "Int256",
  "UInt8",
  "UInt16",
  "UInt32",
  "UInt64",
  "UInt128",
  "UInt256",
  "Float32",
  "Float64",
  "Decimal(18,4)",
  "Date",
  "Date32",
  "DateTime",
  "DateTime64(3)",
  "UUID",
  "IPv4",
  "IPv6",
  "JSON",
  "Array(String)",
  "Map(String, String)",
  "Tuple(String, Int32)",
  "LowCardinality(String)",
  "Enum8('a' = 1, 'b' = 2)",
];

export function columnTypesFor(driverId: string): string[] {
  return driverId === "clickhouse" ? CLICKHOUSE_COLUMN_TYPES : COMMON_COLUMN_TYPES;
}

export function defaultColumnType(driverId: string): string {
  return driverId === "clickhouse" ? "String" : "varchar(255)";
}

export function emptyDraftColumn(name = "column", driverId = ""): DraftColumn {
  return {
    key: crypto.randomUUID(),
    originalName: null,
    dropped: false,
    name,
    type: defaultColumnType(driverId),
    nullable: true,
    defaultValue: null,
    isPrimaryKey: false,
    comment: "",
  };
}

export function nextColumnName(draft: DraftColumn[]): string {
  const names = new Set(
    draft.filter((col) => !col.dropped).map((col) => col.name.trim().toLowerCase()),
  );
  if (!names.has("column")) {
    return "column";
  }
  let i = 2;
  while (names.has(`column_${i}`)) {
    i += 1;
  }
  return `column_${i}`;
}

export function columnsToDraft(columns: ColumnInfo[] | null | undefined): DraftColumn[] {
  return (columns ?? []).map((col) => ({
    key: crypto.randomUUID(),
    originalName: col.name,
    dropped: false,
    name: col.name,
    type: col.type,
    nullable: Boolean(col.nullable),
    defaultValue: normalizeDefault(col.default),
    isPrimaryKey: Boolean(col.isPrimaryKey),
    comment: col.comment ?? "",
  }));
}

/** Catalog defaults like '' / ''::text become a real empty string; SQL NULL stays null. */
export function normalizeDefault(value: string | null | undefined): string | null {
  if (value == null) {
    return null;
  }
  const trimmed = value.trim();
  if (trimmed === "''" || trimmed === '""' || /^''::/.test(trimmed) || /^""::/.test(trimmed)) {
    return "";
  }
  return value;
}

export function defaultsEqual(a: string | null | undefined, b: string | null | undefined): boolean {
  return normalizeDefault(a) === normalizeDefault(b);
}

function toColumnInfo(col: DraftColumn, orig: ColumnInfo | undefined, ordinal: number): ColumnInfo {
  return {
    name: col.name.trim(),
    type: col.type.trim(),
    nullable: col.nullable,
    default: normalizeDefault(col.defaultValue),
    autoIncrement: Boolean(orig?.autoIncrement),
    isPrimaryKey: col.isPrimaryKey,
    comment: col.comment.trim(),
    ordinal,
  };
}

function columnChanged(orig: ColumnInfo, draft: DraftColumn): boolean {
  return (
    orig.name !== draft.name.trim() ||
    orig.type !== draft.type.trim() ||
    Boolean(orig.nullable) !== draft.nullable ||
    !defaultsEqual(orig.default, draft.defaultValue) ||
    (orig.comment ?? "") !== draft.comment.trim()
  );
}

export function structureOps(
  original: ColumnInfo[] | null | undefined,
  draft: DraftColumn[] | null | undefined,
  canReorder: boolean,
): ColumnOp[] {
  const orig = original ?? [];
  const rows = draft ?? [];
  const origByName = new Map(orig.map((col) => [col.name, col]));
  const kept = rows.filter((col) => !col.dropped);
  const keptOriginalNames = new Set(
    kept.map((col) => col.originalName).filter((name): name is string => Boolean(name)),
  );
  const ops: ColumnOp[] = [];

  for (const col of orig) {
    if (!keptOriginalNames.has(col.name)) {
      ops.push({ kind: "drop", name: col.name, after: null });
    }
  }

  const origKeptOrder = orig.filter((col) => keptOriginalNames.has(col.name)).map((col) => col.name);
  const draftKeptOrder = kept
    .map((col) => col.originalName)
    .filter((name): name is string => Boolean(name));
  const orderChanged = canReorder && origKeptOrder.join("\0") !== draftKeptOrder.join("\0");

  kept.forEach((col, i) => {
    const after = orderChanged || col.originalName === null ? (i === 0 ? "" : kept[i - 1]!.name.trim()) : null;
    const source = col.originalName ? origByName.get(col.originalName) : undefined;
    const column = toColumnInfo(col, source, i + 1);
    if (col.originalName === null) {
      ops.push({ kind: "add", name: column.name, column, after });
      return;
    }
    if (!source || columnChanged(source, col)) {
      ops.push({
        kind: "modify",
        name: col.originalName,
        newName: column.name !== col.originalName ? column.name : undefined,
        column,
        after,
      });
      return;
    }
    if (orderChanged) {
      ops.push({
        kind: "reorder",
        name: col.originalName,
        column,
        after,
      });
    }
  });

  return ops;
}

export function isStructureDirty(
  original: ColumnInfo[] | null | undefined,
  draft: DraftColumn[] | null | undefined,
  canReorder: boolean,
): boolean {
  return structureOps(original, draft, canReorder).length > 0;
}

export function filterDraftColumns(draft: DraftColumn[], query: string): DraftColumn[] {
  const q = query.trim().toLowerCase();
  if (!q) {
    return draft;
  }
  return draft.filter(
    (col) =>
      col.name.toLowerCase().includes(q) || (col.originalName ?? "").toLowerCase().includes(q),
  );
}

export function originalFor(col: DraftColumn, columns: ColumnInfo[] | null | undefined): ColumnInfo | null {
  if (!col.originalName) {
    return null;
  }
  return columns?.find((item) => item.name === col.originalName) ?? null;
}
