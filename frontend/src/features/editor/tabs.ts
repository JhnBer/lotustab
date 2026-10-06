import type { QueryResult, StatementResult, TableStructure } from "@/lib/wails";
import type { CellDraft } from "@/features/results/edits";
import type { TableFilter } from "@/features/results/filters";
import type { DraftColumn, TableView } from "@/features/results/structure";

export type { DraftColumn, TableView } from "@/features/results/structure";

export type SortDir = "asc" | "desc" | "";

interface TabBase {
  id: string;
  title: string;
  profileId: string;
  running: boolean;
  error: string;
}

export interface QueryTab extends TabBase {
  kind: "query";
  database: string;
  sql: string;
  executedSql: string;
  page: number;
  pageSize: number;
  result: QueryResult | null;
  results: StatementResult[];
  selectedResult: number;
  followResults: boolean;
  runId: string;
}

export interface TableTab extends TabBase {
  kind: "table";
  database: string;
  schema: string;
  name: string;
  page: number;
  pageSize: number;
  orderBy: string;
  orderDir: SortDir;
  result: QueryResult | null;
  total: number;
  primaryKey: string[];
  edits: CellDraft[];
  filtersEnabled: boolean;
  filtersOpen: boolean;
  filters: TableFilter[];
  view: TableView;
  structure: TableStructure | null;
  structureDraft: DraftColumn[] | null;
  structureError: string;
  manuallyCancelled: boolean;
}

export type WorkspaceTab = QueryTab | TableTab;

export interface WorkspaceSnapshot {
  tabs: WorkspaceTab[];
  activeTabId: string;
}

export function isQueryTab(tab: WorkspaceTab | undefined): tab is QueryTab {
  return tab?.kind === "query";
}

export function isTableTab(tab: WorkspaceTab | undefined): tab is TableTab {
  return tab?.kind === "table";
}

export function tableTitle(schema: string, name: string, database: string): string {
  if (schema && schema !== database) {
    return `${schema}.${name}`;
  }
  return name;
}

export function afterPaint(fn: () => void) {
  requestAnimationFrame(() => {
    requestAnimationFrame(fn);
  });
}
