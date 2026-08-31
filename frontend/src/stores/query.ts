import { defineStore } from "pinia";
import { computed, ref } from "vue";
import { watchDebounced } from "@vueuse/core";
import { errorMessage } from "@/lib/errors";
import { readJson, writeJson } from "@/lib/persist";
import {
  QueryService,
  type QueryResult,
  type TableBrowseResult,
} from "@/lib/wails";
import { useConnectionsStore } from "@/stores/connections";
import { useSettingsStore } from "@/stores/settings";
import { useExplorerStore } from "@/stores/explorer";
import {
  draftLookup,
  rowKey,
  valuesEqual,
  type CellDraft,
} from "@/features/results/edits";
import {
  completeFilters,
  emptyFilter,
  type TableFilter,
} from "@/features/results/filters";

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
  page: number;
  pageSize: number;
  result: QueryResult | null;
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
}

export type WorkspaceTab = QueryTab | TableTab;

export interface WorkspaceSnapshot {
  tabs: WorkspaceTab[];
  activeTabId: string;
}

const WORKSPACE_KEY = "lotustab.workspace";

type PersistedQueryTab = Pick<QueryTab, "kind" | "id" | "title" | "profileId" | "sql" | "page" | "pageSize"> & {
  database?: string;
};
type PersistedTableTab = Pick<
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
>;
type PersistedTab = PersistedQueryTab | PersistedTableTab;
type PersistedWorkspace = {
  tabs: PersistedTab[];
  activeTabId: string;
  activeByProfile: Record<string, string>;
};

function storedPageSize(): number {
  return useSettingsStore().defaultPageSize;
}

function defaultQueryDatabase(): string {
  if (useSettingsStore().showAllDatabases) {
    return "";
  }
  return useExplorerStore().activeDatabase || "";
}

function newQueryTab(profileId: string, title: string): QueryTab {
  return {
    kind: "query",
    id: crypto.randomUUID(),
    title,
    profileId,
    database: defaultQueryDatabase(),
    sql: "SELECT 1;",
    page: 1,
    pageSize: storedPageSize(),
    running: false,
    error: "",
    result: null,
  };
}

function tableTitle(schema: string, name: string, database: string): string {
  if (schema && schema !== database) {
    return `${schema}.${name}`;
  }
  return name;
}

function asQueryResult(result: TableBrowseResult | null): QueryResult | null {
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

export function isQueryTab(tab: WorkspaceTab | undefined): tab is QueryTab {
  return tab?.kind === "query";
}

export function isTableTab(tab: WorkspaceTab | undefined): tab is TableTab {
  return tab?.kind === "table";
}

function persistTab(tab: WorkspaceTab): PersistedTab {
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
  };
}

function restoreQueryTab(tab: PersistedQueryTab): QueryTab {
  return {
    kind: "query",
    id: tab.id,
    title: tab.title || "Query",
    profileId: tab.profileId,
    database: tab.database ?? "",
    sql: tab.sql,
    page: tab.page >= 1 ? tab.page : 1,
    pageSize: tab.pageSize || storedPageSize(),
    running: false,
    error: "",
    result: null,
  };
}

function restoreTableTab(tab: PersistedTableTab): TableTab {
  return {
    kind: "table",
    id: tab.id,
    title: tab.title || tab.name,
    profileId: tab.profileId,
    database: tab.database,
    schema: tab.schema ?? "",
    name: tab.name,
    page: tab.page >= 1 ? tab.page : 1,
    pageSize: tab.pageSize || storedPageSize(),
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

function parseActiveByProfile(
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

function parseWorkspace(raw: unknown): PersistedWorkspace {
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

function hydrateTabs(stored: PersistedWorkspace): WorkspaceTab[] {
  return stored.tabs.map((tab) => (tab.kind === "query" ? restoreQueryTab(tab) : restoreTableTab(tab)));
}

export const useQueryStore = defineStore("query", () => {
  const stored = parseWorkspace(readJson(WORKSPACE_KEY, {}));
  const tabs = ref<WorkspaceTab[]>(hydrateTabs(stored));
  const activeTabId = ref(
    stored.activeTabId && tabs.value.some((t) => t.id === stored.activeTabId)
      ? stored.activeTabId
      : (tabs.value[0]?.id ?? ""),
  );
  const activeByProfile = ref<Record<string, string>>(
    parseActiveByProfile(stored.activeByProfile, tabs.value, activeTabId.value),
  );

  watchDebounced(
    () => ({
      tabs: tabs.value.map(persistTab),
      activeTabId: activeTabId.value,
      activeByProfile: activeByProfile.value,
    }),
    (value) => writeJson(WORKSPACE_KEY, value),
    { debounce: 300, deep: true },
  );

  const activeTab = computed(() => tabs.value.find((t) => t.id === activeTabId.value));

  function selectTab(id: string) {
    if (!id) {
      activeTabId.value = "";
      return;
    }
    const tab = tabs.value.find((t) => t.id === id);
    if (!tab) {
      activeTabId.value = "";
      return;
    }
    activeTabId.value = id;
    if (activeByProfile.value[tab.profileId] === id) {
      return;
    }
    activeByProfile.value = { ...activeByProfile.value, [tab.profileId]: id };
  }

  function activateProfile(profileId: string) {
    if (!profileId) {
      activeTabId.value = "";
      return;
    }
    const remembered = activeByProfile.value[profileId];
    const stillThere =
      Boolean(remembered) &&
      tabs.value.some((t) => t.id === remembered && t.profileId === profileId);
    selectTab(stillThere ? remembered! : (tabs.value.find((t) => t.profileId === profileId)?.id ?? ""));
  }

  function forgetProfile(profileId: string) {
    if (!(profileId in activeByProfile.value)) {
      return;
    }
    const next = { ...activeByProfile.value };
    delete next[profileId];
    activeByProfile.value = next;
  }

  function openForProfile(profileId: string, name: string) {
    const existing = tabs.value.find((t) => t.kind === "query" && t.profileId === profileId);
    if (existing) {
      selectTab(existing.id);
      return existing;
    }
    return add(profileId, name);
  }

  function add(profileId: string, name: string) {
    const count = tabs.value.filter((t) => t.kind === "query" && t.profileId === profileId).length;
    const tab = newQueryTab(profileId, count === 0 ? name : `${name} ${count + 1}`);
    tabs.value.push(tab);
    selectTab(tab.id);
    return tab;
  }

  function openTable(opts: {
    profileId: string;
    database: string;
    schema: string;
    name: string;
  }) {
    const existing = tabs.value.find(
      (t): t is TableTab =>
        t.kind === "table" &&
        t.profileId === opts.profileId &&
        t.database === opts.database &&
        t.schema === opts.schema &&
        t.name === opts.name,
    );
    if (existing) {
      selectTab(existing.id);
      void loadTable(existing.id);
      return existing;
    }
    const tab: TableTab = {
      kind: "table",
      id: crypto.randomUUID(),
      title: tableTitle(opts.schema, opts.name, opts.database),
      profileId: opts.profileId,
      database: opts.database,
      schema: opts.schema,
      name: opts.name,
      page: 1,
      pageSize: storedPageSize(),
      orderBy: "",
      orderDir: "",
      running: false,
      error: "",
      result: null,
      total: 0,
      primaryKey: [],
      edits: [],
      filtersEnabled: false,
      filtersOpen: false,
      filters: [],
    };
    tabs.value.push(tab);
    selectTab(tab.id);
    void loadTable(tab.id);
    return tab;
  }

  function close(id: string) {
    const index = tabs.value.findIndex((t) => t.id === id);
    if (index < 0) {
      return;
    }
    const profileId = tabs.value[index]!.profileId;
    const groupIndex = tabs.value.filter((t) => t.profileId === profileId).findIndex((t) => t.id === id);
    tabs.value.splice(index, 1);
    if (activeTabId.value !== id) {
      return;
    }
    const group = tabs.value.filter((t) => t.profileId === profileId);
    const next = group[groupIndex] ?? group[groupIndex - 1];
    if (next) {
      selectTab(next.id);
      return;
    }
    activeTabId.value = "";
    forgetProfile(profileId);
  }

  function snapshotForProfile(profileId: string): WorkspaceSnapshot {
    const group = tabs.value.filter((t) => t.profileId === profileId);
    const remembered = activeByProfile.value[profileId];
    return {
      tabs: group.map((t) => {
        if (t.kind === "table") {
          return { ...t, running: false, edits: [] };
        }
        return { ...t, running: false };
      }),
      activeTabId: group.some((t) => t.id === remembered) ? remembered : (group[0]?.id ?? ""),
    };
  }

  function removeForProfile(profileId: string) {
    const removingActive =
      tabs.value.find((t) => t.id === activeTabId.value)?.profileId === profileId;
    tabs.value = tabs.value.filter((t) => t.profileId !== profileId);
    forgetProfile(profileId);
    if (removingActive) {
      selectTab(tabs.value[0]?.id ?? "");
    }
  }

  function restore(snapshot: WorkspaceSnapshot) {
    if (!snapshot.tabs.length) {
      return;
    }
    const profileId = snapshot.tabs[0]!.profileId;
    const group = tabs.value.filter((t) => t.profileId === profileId);
    const placeholder = group[0];
    if (
      group.length === 1 &&
      placeholder &&
      placeholder.kind === "query" &&
      placeholder.sql === "SELECT 1;" &&
      !placeholder.result
    ) {
      tabs.value = tabs.value.filter((t) => t.id !== placeholder.id);
    }
    const existing = new Set(tabs.value.map((t) => t.id));
    for (const tab of snapshot.tabs) {
      if (!existing.has(tab.id)) {
        if (tab.kind === "query") {
          tabs.value.push({
            ...tab,
            running: false,
            page: tab.page >= 1 ? tab.page : 1,
            pageSize: tab.pageSize || storedPageSize(),
          });
        } else {
          tabs.value.push({
            ...tab,
            running: false,
            edits: [],
            primaryKey: tab.primaryKey ?? [],
            filtersEnabled: tab.filtersEnabled ?? false,
            filtersOpen: tab.filtersOpen ?? false,
            filters: tab.filters ?? [],
          });
        }
      }
    }
    selectTab(
      snapshot.activeTabId && tabs.value.some((t) => t.id === snapshot.activeTabId)
        ? snapshot.activeTabId
        : snapshot.tabs[0]!.id,
    );
  }

  function reorder(profileId: string, fromIndex: number, toIndex: number) {
    if (fromIndex < 0 || fromIndex === toIndex) {
      return;
    }
    const group = tabs.value.filter((t) => t.profileId === profileId);
    const [moved] = group.splice(fromIndex, 1);
    if (!moved) {
      return;
    }
    group.splice(toIndex, 0, moved);
    let i = 0;
    tabs.value = tabs.value.map((t) => (t.profileId === profileId ? group[i++]! : t));
  }

  function setSql(id: string, sql: string) {
    const tab = tabs.value.find((t) => t.id === id);
    if (tab && tab.kind === "query") {
      tab.sql = sql;
    }
  }

  function setDatabase(id: string, database: string) {
    const tab = tabs.value.find((t) => t.id === id);
    if (tab && tab.kind === "query") {
      tab.database = database;
    }
  }

  function dropTableData(profileId: string) {
    for (const tab of tabs.value) {
      if (tab.profileId !== profileId || tab.kind !== "table") {
        continue;
      }
      tab.result = null;
      tab.total = 0;
      tab.primaryKey = [];
      tab.edits = [];
      tab.error = "";
      tab.running = false;
    }
  }

  async function execute(id: string) {
    const tab = tabs.value.find((t) => t.id === id);
    if (!tab || tab.kind !== "query") {
      return;
    }
    if (!useConnectionsStore().isConnected(tab.profileId)) {
      return;
    }
    if (!tab.sql.trim()) {
      tab.error = "Query is empty";
      return;
    }
    if (!tab.page || tab.page < 1) {
      tab.page = 1;
    }
    if (!tab.pageSize) {
      tab.pageSize = storedPageSize();
    }
    tab.running = true;
    tab.error = "";
    try {
      tab.result = await QueryService.Execute(
        tab.profileId,
        tab.sql,
        tab.database ?? "",
        tab.pageSize,
        (tab.page - 1) * tab.pageSize,
      );
    } catch (err) {
      tab.result = null;
      tab.error = errorMessage(err);
    } finally {
      tab.running = false;
    }
  }

  async function run(id: string) {
    const tab = tabs.value.find((t) => t.id === id);
    if (!tab || tab.kind !== "query") {
      return;
    }
    tab.page = 1;
    await execute(id);
  }

  async function loadTable(id: string) {
    const tab = tabs.value.find((t) => t.id === id);
    if (!tab || tab.kind !== "table") {
      return;
    }
    if (!useConnectionsStore().isConnected(tab.profileId)) {
      return;
    }
    if (!tab.edits) {
      tab.edits = [];
    }
    if (!tab.filters) {
      tab.filters = [];
    }
    tab.running = true;
    tab.error = "";
    try {
      const result = await QueryService.BrowseTable(tab.profileId, {
        database: tab.database,
        schema: tab.schema,
        name: tab.name,
        limit: tab.pageSize,
        offset: (tab.page - 1) * tab.pageSize,
        orderBy: tab.orderBy,
        orderDir: tab.orderDir,
        filters: tab.filtersEnabled ? completeFilters(tab.filters) : [],
      });
      tab.result = asQueryResult(result);
      tab.total = result?.total ?? 0;
      tab.primaryKey = result?.primaryKey ?? [];
    } catch (err) {
      tab.result = null;
      tab.total = 0;
      tab.primaryKey = [];
      tab.error = errorMessage(err);
    } finally {
      tab.running = false;
    }
  }

  function setPage(id: string, page: number) {
    const tab = tabs.value.find((t) => t.id === id);
    if (!tab) {
      return;
    }
    if (tab.kind === "table") {
      const maxPage = Math.max(1, Math.ceil(tab.total / tab.pageSize) || 1);
      tab.page = Math.min(Math.max(1, page), maxPage);
      void loadTable(id);
      return;
    }
    if (tab.kind === "query") {
      tab.page = Math.max(1, page);
      void execute(id);
    }
  }

  function setPageSize(id: string, pageSize: number) {
    const tab = tabs.value.find((t) => t.id === id);
    if (!tab || (tab.kind !== "table" && tab.kind !== "query")) {
      return;
    }
    tab.pageSize = pageSize;
    tab.page = 1;
    useSettingsStore().setDefaultPageSize(pageSize);
    if (tab.kind === "table") {
      void loadTable(id);
      return;
    }
    void execute(id);
  }

  function cycleSort(id: string, column: string) {
    const tab = tabs.value.find((t) => t.id === id);
    if (!tab || tab.kind !== "table") {
      return;
    }
    if (tab.orderBy !== column) {
      tab.orderBy = column;
      tab.orderDir = "asc";
    } else if (tab.orderDir === "asc") {
      tab.orderDir = "desc";
    } else {
      tab.orderBy = "";
      tab.orderDir = "";
    }
    tab.page = 1;
    void loadTable(id);
  }

  function setFilters(id: string, filters: TableFilter[]) {
    const tab = tabs.value.find((t) => t.id === id);
    if (!tab || tab.kind !== "table") {
      return;
    }
    tab.filters = filters;
  }

  function addFilter(id: string) {
    const tab = tabs.value.find((t) => t.id === id);
    if (!tab || tab.kind !== "table" || !tab.filtersEnabled) {
      return;
    }
    tab.filters = [...(tab.filters ?? []), emptyFilter()];
    tab.filtersOpen = true;
  }

  function setFiltersEnabled(id: string, enabled: boolean) {
    const tab = tabs.value.find((t) => t.id === id);
    if (!tab || tab.kind !== "table") {
      return;
    }
    tab.filtersEnabled = enabled;
    if (enabled) {
      tab.filtersOpen = true;
      if (!tab.filters?.length) {
        tab.filters = [emptyFilter()];
      }
    }
    tab.page = 1;
    void loadTable(id);
  }

  function toggleFiltersOpen(id: string) {
    const tab = tabs.value.find((t) => t.id === id);
    if (!tab || tab.kind !== "table") {
      return;
    }
    tab.filtersOpen = !tab.filtersOpen;
    if (tab.filtersOpen && !tab.filters?.length) {
      tab.filters = [emptyFilter()];
    }
  }

  function applyFilters(id: string) {
    const tab = tabs.value.find((t) => t.id === id);
    if (!tab || tab.kind !== "table" || !tab.filtersEnabled) {
      return;
    }
    tab.page = 1;
    void loadTable(id);
  }

  function setCellDraft(id: string, rowIndex: number, column: string, value: unknown) {
    const tab = tabs.value.find((t) => t.id === id);
    if (!tab || tab.kind !== "table" || !tab.primaryKey?.length) {
      return;
    }
    if (!tab.edits) {
      tab.edits = [];
    }
    const columns = tab.result?.columns ?? [];
    const row = tab.result?.rows?.[rowIndex];
    const key = rowKey(row, columns, tab.primaryKey);
    if (!key) {
      return;
    }
    const colIndex = columns.findIndex((c) => c.name === column);
    if (colIndex < 0) {
      return;
    }
    const original = row?.[colIndex];
    const lookup = draftLookup(key);
    const edits = tab.edits.slice();
    const index = edits.findIndex((e) => draftLookup(e.key) === lookup);
    const cells = { ...(index >= 0 ? edits[index]!.cells : {}) };
    if (valuesEqual(original, value)) {
      delete cells[column];
    } else {
      cells[column] = value;
    }
    if (Object.keys(cells).length === 0) {
      if (index >= 0) {
        edits.splice(index, 1);
      }
    } else if (index >= 0) {
      edits[index] = { key, cells };
    } else {
      edits.push({ key, cells });
    }
    tab.edits = edits;
  }

  function clearEdits(id: string) {
    const tab = tabs.value.find((t) => t.id === id);
    if (!tab || tab.kind !== "table") {
      return;
    }
    tab.edits = [];
  }

  async function applyEdits(id: string) {
    const tab = tabs.value.find((t) => t.id === id);
    if (!tab || tab.kind !== "table" || !tab.edits?.length) {
      return;
    }
    await QueryService.ApplyEdits(tab.profileId, {
      database: tab.database,
      schema: tab.schema,
      name: tab.name,
      rows: tab.edits,
    });
    tab.edits = [];
    await loadTable(id);
  }

  return {
    tabs,
    activeTabId,
    activeTab,
    selectTab,
    activateProfile,
    openForProfile,
    add,
    openTable,
    close,
    snapshotForProfile,
    removeForProfile,
    restore,
    dropTableData,
    reorder,
    setSql,
    setDatabase,
    run,
    loadTable,
    setPage,
    setPageSize,
    cycleSort,
    setFilters,
    addFilter,
    setFiltersEnabled,
    toggleFiltersOpen,
    applyFilters,
    setCellDraft,
    clearEdits,
    applyEdits,
  };
});
