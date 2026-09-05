import type { QueryTab, TableTab, WorkspaceTab } from "@/features/editor/tabs";

export type PersistedQueryTab = Pick<QueryTab, "kind" | "id" | "title" | "profileId" | "sql" | "page" | "pageSize"> & {
  database?: string;
};

export type PersistedTableTab = Pick<
  TableTab,
  | "kind"
  | "id"
  | "title"
  | "profileId"
  | "database"
  | "schema"
  | "name"
  | "page"
  | "pageSize"
  | "orderBy"
  | "orderDir"
  | "filtersEnabled"
  | "filtersOpen"
  | "filters"
  | "view"
>;

export type PersistedTab = PersistedQueryTab | PersistedTableTab;

export type PersistedWorkspace = {
  tabs: PersistedTab[];
  activeTabId: string;
  activeByProfile: Record<string, string>;
};

export function persistTab(tab: WorkspaceTab): PersistedTab {
  if (tab.kind === "query") {
    return {
      kind: "query",
      id: tab.id,
      title: tab.title,
      profileId: tab.profileId,
      database: tab.database ?? "",
      sql: tab.sql,
      page: tab.page,
      pageSize: tab.pageSize,
    };
  }
  return {
    kind: "table",
    id: tab.id,
    title: tab.title,
    profileId: tab.profileId,
    database: tab.database,
    schema: tab.schema,
    name: tab.name,
    page: tab.page,
    pageSize: tab.pageSize,
    orderBy: tab.orderBy,
    orderDir: tab.orderDir,
    filtersEnabled: tab.filtersEnabled,
    filtersOpen: tab.filtersOpen,
    filters: tab.filters ?? [],
    view: tab.view === "structure" ? "structure" : "data",
  };
}

export function restoreQueryTab(tab: PersistedQueryTab, defaultPageSize: number): QueryTab {
  return {
    kind: "query",
    id: tab.id,
    title: tab.title || "Query",
    profileId: tab.profileId,
    database: tab.database ?? "",
    sql: tab.sql,
    page: tab.page >= 1 ? tab.page : 1,
    pageSize: tab.pageSize || defaultPageSize,
    running: false,
    error: "",
    result: null,
  };
}

export function restoreTableTab(tab: PersistedTableTab, defaultPageSize: number): TableTab {
  return {
    kind: "table",
    id: tab.id,
    title: tab.title || tab.name,
    profileId: tab.profileId,
    database: tab.database,
    schema: tab.schema ?? "",
    name: tab.name,
    page: tab.page >= 1 ? tab.page : 1,
    pageSize: tab.pageSize || defaultPageSize,
    orderBy: tab.orderBy ?? "",
    orderDir: tab.orderDir === "asc" || tab.orderDir === "desc" ? tab.orderDir : "",
    running: false,
    error: "",
    result: null,
    total: 0,
    primaryKey: [],
    edits: [],
    filtersEnabled: Boolean(tab.filtersEnabled),
    filtersOpen: Boolean(tab.filtersOpen),
    filters: tab.filters ?? [],
    view: tab.view === "structure" ? "structure" : "data",
    structure: null,
    structureDraft: null,
    structureError: "",
    manuallyCancelled: false,
  };
}

function isPersistedQueryTab(tab: unknown): tab is PersistedQueryTab {
  if (!tab || typeof tab !== "object") {
    return false;
  }
  const row = tab as PersistedQueryTab;
  return (
    row.kind === "query" &&
    typeof row.id === "string" &&
    typeof row.profileId === "string" &&
    typeof row.sql === "string"
  );
}

function isPersistedTableTab(tab: unknown): tab is PersistedTableTab {
  if (!tab || typeof tab !== "object") {
    return false;
  }
  const row = tab as PersistedTableTab;
  return (
    row.kind === "table" &&
    typeof row.id === "string" &&
    typeof row.profileId === "string" &&
    typeof row.database === "string" &&
    typeof row.name === "string"
  );
}

export function parseActiveByProfile(
  raw: unknown,
  tabs: WorkspaceTab[],
  activeTabId: string,
): Record<string, string> {
  const byId = new Map(tabs.map((tab) => [tab.id, tab]));
  const out: Record<string, string> = {};
  if (raw && typeof raw === "object" && !Array.isArray(raw)) {
    for (const [profileId, tabId] of Object.entries(raw as Record<string, unknown>)) {
      if (typeof tabId !== "string" || !profileId) {
        continue;
      }
      const tab = byId.get(tabId);
      if (tab && tab.profileId === profileId) {
        out[profileId] = tabId;
      }
    }
  }
  const active = byId.get(activeTabId);
  if (active && !out[active.profileId]) {
    out[active.profileId] = active.id;
  }
  return out;
}

export function parseWorkspace(raw: unknown): PersistedWorkspace {
  if (!raw || typeof raw !== "object") {
    return { tabs: [], activeTabId: "", activeByProfile: {} };
  }
  const row = raw as PersistedWorkspace;
  const tabs = Array.isArray(row.tabs)
    ? row.tabs.filter((tab): tab is PersistedTab => isPersistedQueryTab(tab) || isPersistedTableTab(tab))
    : [];
  return {
    tabs,
    activeTabId: typeof row.activeTabId === "string" ? row.activeTabId : "",
    activeByProfile: row.activeByProfile && typeof row.activeByProfile === "object" ? row.activeByProfile : {},
  };
}

export function hydrateTabs(stored: PersistedWorkspace, defaultPageSize: number): WorkspaceTab[] {
  return stored.tabs.map((tab) =>
    tab.kind === "query" ? restoreQueryTab(tab, defaultPageSize) : restoreTableTab(tab, defaultPageSize),
  );
}
