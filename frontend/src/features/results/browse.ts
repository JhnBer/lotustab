import type { QueryResult, TableBrowseResult } from "@/lib/wails";

export function asQueryResult(result: TableBrowseResult | null): QueryResult | null {
  if (!result) {
    return null;
  }
  return {
    columns: result.columns,
    rows: result.rows,
    rowsAffected: result.rowsAffected,
    durationMs: result.durationMs,
    truncated: result.truncated,
  };
}
