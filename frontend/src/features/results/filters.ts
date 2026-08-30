import type { TableFilter } from "@/lib/wails";

export type { TableFilter };

/** Operator ids must match sqlutil.filterOps. */
export const FILTER_OPS = [
  { id: "eq", label: "=" },
  { id: "neq", label: "!=" },
  { id: "gt", label: ">" },
  { id: "gte", label: ">=" },
  { id: "lt", label: "<" },
  { id: "lte", label: "<=" },
  { id: "like", label: "LIKE" },
  { id: "nlike", label: "NOT LIKE" },
  { id: "in", label: "IN" },
  { id: "nin", label: "NOT IN" },
  { id: "null", label: "IS NULL" },
  { id: "nnull", label: "IS NOT NULL" },
] as const;

export function emptyFilter(): TableFilter {
  return { column: "", op: "eq", value: "", join: "and" };
}

export function opNeedsValue(op: string) {
  return op !== "null" && op !== "nnull";
}

export function valuePlaceholder(op: string) {
  if (op === "in" || op === "nin") {
    return "a, b, c";
  }
  if (op === "like" || op === "nlike") {
    return "%value%";
  }
  return "";
}

export function completeFilters(filters: TableFilter[]): TableFilter[] {
  const out: TableFilter[] = [];
  for (const filter of filters) {
    const column = filter.column.trim();
    if (!column) {
      continue;
    }
    if (opNeedsValue(filter.op) && !filter.value.trim()) {
      continue;
    }
    out.push({
      column,
      op: filter.op,
      value: filter.value,
      join: filter.join === "or" ? "or" : "and",
    });
  }
  return out;
}
