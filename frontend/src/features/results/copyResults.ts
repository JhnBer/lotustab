import { cellRangeToJson, rowsClipboardText } from "@/features/results/edits";
import type { QueryTab } from "@/features/editor/tabs";
import type { Column, QueryResult, StatementResult } from "@/lib/wails";

export type CopyableResult = {
  index: number;
  sql: string;
  columns: Column[];
  rows: unknown[][];
};

function hasTable(result: QueryResult | null | undefined): result is QueryResult {
  return Boolean(result?.columns?.length);
}

function tableGrid(result: QueryResult): Pick<CopyableResult, "columns" | "rows"> {
  return {
    columns: result.columns ?? [],
    rows: (result.rows ?? []).map((row) => row ?? []),
  };
}

function fromStatement(stmt: StatementResult): CopyableResult | null {
  if (!hasTable(stmt.result)) {
    return null;
  }
  return {
    index: stmt.index,
    sql: stmt.sql,
    ...tableGrid(stmt.result),
  };
}

export function copyableResults(tab: QueryTab): CopyableResult[] {
  const statements = tab.results ?? [];
  if (statements.length > 0) {
    return statements.flatMap((stmt) => {
      const item = fromStatement(stmt);
      return item ? [item] : [];
    });
  }
  if (!hasTable(tab.result)) {
    return [];
  }
  return [
    {
      index: 0,
      sql: tab.executedSql || tab.sql,
      ...tableGrid(tab.result),
    },
  ];
}

export function currentCopyable(tab: QueryTab): CopyableResult | null {
  const all = copyableResults(tab);
  if (all.length === 0) {
    return null;
  }
  return all.find((item) => item.index === tab.selectedResult) ?? null;
}

export function resultTsv(item: CopyableResult) {
  return rowsClipboardText(item.rows);
}

function resultObjects(item: CopyableResult) {
  const names = item.columns.map((col) => col.name);
  return item.rows.map((values) => {
    const row: Record<string, unknown> = {};
    for (let i = 0; i < names.length; i++) {
      row[names[i] ?? ""] = values[i] ?? null;
    }
    return row;
  });
}

export function resultJson(item: CopyableResult) {
  return cellRangeToJson(
    item.columns.map((col) => col.name),
    item.rows,
  );
}

export function resultsTsv(items: CopyableResult[]) {
  return items.map(resultTsv).join("\n\n");
}

export function resultsJson(items: CopyableResult[]) {
  return JSON.stringify(
    items.map((item) => ({
      index: item.index,
      sql: item.sql,
      rows: resultObjects(item),
    })),
  );
}

export function statementLabel(item: CopyableResult) {
  const sql = item.sql.replace(/\s+/g, " ").trim();
  const preview = sql.length > 80 ? `${sql.slice(0, 80)}…` : sql;
  return preview ? `${item.index + 1} · ${preview}` : String(item.index + 1);
}
