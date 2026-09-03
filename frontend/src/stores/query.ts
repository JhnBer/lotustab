import { defineStore } from "pinia";
import { computed, ref } from "vue";
import { watchDebounced } from "@vueuse/core";
import { errorMessage, isCanceled } from "@/lib/errors";
import { readJson, writeJson } from "@/lib/persist";
import {
  QueryService,
  type QueryResult,
  type TableBrowseResult,
  type TableStructure,
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
import {
  columnsToDraft,
  emptyDraftColumn,
  isStructureDirty,
  nextColumnName,
  structureOps,
  type DraftColumn,
  type TableView,
} from "@/features/results/structure";

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
  view: TableView;
  structure: TableStructure | null;
  structureDraft: DraftColumn[] | null;
  structureError: string;
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
  | "view"
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

function afterPaint(fn: () => void) {
  requestAnimationFrame(() => {
    requestAnimationFrame(fn);
  });
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
    view: tab.view === "structure" ? "structure" : "data",
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
    view: tab.view === "structure" ? "structure" : "data",
    structure: null,
    structureDraft: null,
    structureError: "",
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

const MOUNTED_TAB_LIMIT = 5;

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
  const recentTabIds = ref<string[]>(activeTabId.value ? [activeTabId.value] : []);
  const coldMountId = ref("");

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

  function touchRecent(id: string) {
    if (!id) {
      return;
    }
    recentTabIds.value = [id, ...recentTabIds.value.filter((x) => x !== id)];
  }

  function forgetRecent(id: string) {
    if (!recentTabIds.value.includes(id)) {
      return;
    }
    recentTabIds.value = recentTabIds.value.filter((x) => x !== id);
  }

  function mountedTabIds(profileId: string): string[] {
    if (!profileId) {
      return [];
    }
    const inProfile = new Set(tabs.value.filter((t) => t.profileId === profileId).map((t) => t.id));
    const out: string[] = [];
    const active = activeTabId.value;
    if (active && inProfile.has(active)) {
      out.push(active);
    }
    for (const id of recentTabIds.value) {
      if (out.length >= MOUNTED_TAB_LIMIT) {
        break;
      }
      if (!inProfile.has(id) || out.includes(id)) {
        continue;
      }
      out.push(id);
    }
    return out;
  }

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
    const isCold = !mountedTabIds(tab.profileId).includes(id);
    activeTabId.value = id;
    touchRecent(id);
    if (isCold) {
      coldMountId.value = id;
      afterPaint(() => {
        if (coldMountId.value === id) {
          coldMountId.value = "";
        }
      });
    }
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
    view?: TableView;
  }) {
    const view = opts.view === "structure" ? "structure" : "data";
    const existing = tabs.value.find(
      (t): t is TableTab =>
        t.kind === "table" &&
        t.profileId === opts.profileId &&
        t.database === opts.database &&
        t.schema === opts.schema &&
        t.name === opts.name &&
        t.view === view,
    );
    if (existing) {
      selectTab(existing.id);
      if (existing.view === "structure") {
        if (existing.structure || existing.structureDraft) {
          return existing;
        }
        existing.running = true;
        afterPaint(() => void loadStructure(existing.id));
      } else if (!existing.result) {
        existing.running = true;
        afterPaint(() => void loadTable(existing.id));
      }
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
      running: true,
      error: "",
      result: null,
      total: 0,
      primaryKey: [],
      edits: [],
      filtersEnabled: false,
      filtersOpen: false,
      filters: [],
      view,
      structure: null,
      structureDraft: null,
      structureError: "",
    };
    tabs.value.push(tab);
    selectTab(tab.id);
    afterPaint(() => void reloadTab(tab.id));
    return tab;
  }

  const inflight = new Map<string, { cancel: () => void }>();

  function cancel(id: string) {
    inflight.get(id)?.cancel();
  }

  function close(id: string) {
    cancel(id);
    const index = tabs.value.findIndex((t) => t.id === id);
    if (index < 0) {
      return;
    }
    const profileId = tabs.value[index]!.profileId;
    const groupIndex = tabs.value.filter((t) => t.profileId === profileId).findIndex((t) => t.id === id);
    tabs.value.splice(index, 1);
    forgetRecent(id);
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
          return { ...t, running: false, edits: [], structureDraft: t.structureDraft ? t.structureDraft.map((col) => ({ ...col })) : null };
        }
        return { ...t, running: false };
      }),
      activeTabId: group.some((t) => t.id === remembered) ? remembered : (group[0]?.id ?? ""),
    };
  }

  function removeForProfile(profileId: string) {
    const removingActive =
      tabs.value.find((t) => t.id === activeTabId.value)?.profileId === profileId;
    const removed = new Set(tabs.value.filter((t) => t.profileId === profileId).map((t) => t.id));
    tabs.value = tabs.value.filter((t) => t.profileId !== profileId);
    recentTabIds.value = recentTabIds.value.filter((id) => !removed.has(id));
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
      forgetRecent(placeholder.id);
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
            view: tab.view === "structure" ? "structure" : "data",
            structure: tab.structure ?? null,
            structureDraft: null,
            structureError: "",
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

  function setOrder(profileId: string, ids: string[]) {
    const group = tabs.value.filter((t) => t.profileId === profileId);
    if (group.length !== ids.length) {
      return;
    }
    const byId = new Map(group.map((t) => [t.id, t]));
    const ordered: WorkspaceTab[] = [];
    for (const id of ids) {
      const tab = byId.get(id);
      if (!tab) {
        return;
      }
      ordered.push(tab);
    }
    let i = 0;
    tabs.value = tabs.value.map((t) => (t.profileId === profileId ? ordered[i++]! : t));
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
      tab.structure = null;
      tab.structureDraft = null;
      tab.structureError = "";
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
    cancel(id);
    tab.running = true;
    tab.error = "";
    const pending = QueryService.Execute(
      tab.profileId,
      tab.sql,
      tab.database ?? "",
      tab.pageSize,
      (tab.page - 1) * tab.pageSize,
      useSettingsStore().queryTimeoutSeconds,
    );
    inflight.set(id, pending);
    try {
      tab.result = await pending;
    } catch (err) {
      if (isCanceled(err)) {
        return;
      }
      tab.result = null;
      tab.error = errorMessage(err);
    } finally {
      if (inflight.get(id) === pending) {
        inflight.delete(id);
        tab.running = false;
      }
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
      tab.running = false;
      return;
    }
    if (!tab.edits) {
      tab.edits = [];
    }
    if (!tab.filters) {
      tab.filters = [];
    }
    cancel(id);
    tab.running = true;
    tab.error = "";
    const pending = QueryService.BrowseTable(tab.profileId, {
      database: tab.database,
      schema: tab.schema,
      name: tab.name,
      limit: tab.pageSize,
      offset: (tab.page - 1) * tab.pageSize,
      orderBy: tab.orderBy,
      orderDir: tab.orderDir,
      filters: tab.filtersEnabled ? completeFilters(tab.filters) : [],
    });
    inflight.set(id, pending);
    try {
      const result = await pending;
      tab.result = asQueryResult(result);
      tab.total = result?.total ?? 0;
      tab.primaryKey = result?.primaryKey ?? [];
    } catch (err) {
      if (isCanceled(err)) {
        return;
      }
      tab.result = null;
      tab.total = 0;
      tab.primaryKey = [];
      tab.error = errorMessage(err);
    } finally {
      if (inflight.get(id) === pending) {
        inflight.delete(id);
        tab.running = false;
      }
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

  function tableTab(id: string): TableTab | undefined {
    const tab = tabs.value.find((t) => t.id === id);
    if (!tab || tab.kind !== "table") {
      return undefined;
    }
    return tab;
  }

  function reloadTab(id: string) {
    const tab = tableTab(id);
    if (!tab) {
      return;
    }
    if (tab.view === "structure") {
      return loadStructure(id);
    }
    return loadTable(id);
  }

  function clearBrowse(tab: TableTab) {
    tab.edits = [];
    tab.result = null;
    tab.total = 0;
    tab.primaryKey = [];
  }

  async function loadStructure(id: string, opts?: { force?: boolean }) {
    const tab = tableTab(id);
    if (!tab) {
      return;
    }
    if (!useConnectionsStore().isConnected(tab.profileId)) {
      tab.running = false;
      return;
    }
    const keepDraft =
      !opts?.force &&
      Boolean(tab.structureDraft) &&
      isStructureDirty(tab.structure?.columns, tab.structureDraft, tab.structure?.canReorder ?? false);
    cancel(id);
    tab.running = true;
    tab.structureError = "";
    const pending = QueryService.DescribeTable(tab.profileId, {
      database: tab.database,
      schema: tab.schema,
      name: tab.name,
    });
    inflight.set(id, pending);
    try {
      const result = await pending;
      tab.structure = result
        ? { columns: result.columns ?? [], canReorder: Boolean(result.canReorder) }
        : { columns: [], canReorder: false };
      if (!keepDraft) {
        tab.structureDraft = columnsToDraft(tab.structure.columns);
      }
    } catch (err) {
      if (isCanceled(err)) {
        return;
      }
      tab.structure = null;
      if (!keepDraft) {
        tab.structureDraft = null;
      }
      tab.structureError = errorMessage(err);
    } finally {
      if (inflight.get(id) === pending) {
        inflight.delete(id);
        tab.running = false;
      }
    }
  }

  function ensureStructureDraft(tab: TableTab): DraftColumn[] {
    if (!tab.structureDraft) {
      tab.structureDraft = columnsToDraft(tab.structure?.columns ?? []);
    }
    return tab.structureDraft;
  }

  function updateStructureColumn(id: string, key: string, patch: Partial<DraftColumn>) {
    const tab = tableTab(id);
    if (!tab) {
      return;
    }
    const draft = ensureStructureDraft(tab);
    const index = draft.findIndex((col) => col.key === key);
    if (index < 0) {
      return;
    }
    tab.structureDraft = draft.map((col, i) => (i === index ? { ...col, ...patch, key: col.key } : col));
  }

  function addStructureColumn(id: string, driverId = "") {
    const tab = tableTab(id);
    if (!tab) {
      return;
    }
    const draft = ensureStructureDraft(tab);
    tab.structureDraft = [...draft, emptyDraftColumn(nextColumnName(draft), driverId)];
  }

  function toggleDropStructureColumn(id: string, key: string) {
    const tab = tableTab(id);
    if (!tab) {
      return;
    }
    const draft = ensureStructureDraft(tab);
    const index = draft.findIndex((col) => col.key === key);
    if (index < 0) {
      return;
    }
    const col = draft[index]!;
    if (col.originalName === null) {
      tab.structureDraft = draft.filter((item) => item.key !== key);
      return;
    }
    tab.structureDraft = draft.map((item) =>
      item.key === key ? { ...item, dropped: !item.dropped } : item,
    );
  }

  function moveStructureColumns(id: string, next: DraftColumn[]) {
    const tab = tableTab(id);
    if (!tab) {
      return;
    }
    tab.structureDraft = next;
  }

  function cancelStructure(id: string) {
    const tab = tableTab(id);
    if (!tab) {
      return;
    }
    tab.structureDraft = columnsToDraft(tab.structure?.columns ?? []);
  }

  async function applyStructure(id: string) {
    const tab = tableTab(id);
    if (!tab) {
      return;
    }
    const ops = structureOps(tab.structure?.columns, tab.structureDraft, tab.structure?.canReorder ?? false);
    if (ops.length === 0) {
      return;
    }
    await QueryService.AlterTable(tab.profileId, {
      database: tab.database,
      schema: tab.schema,
      name: tab.name,
      ops,
    });
    clearBrowse(tab);
    const dataTab = tabs.value.find(
      (t): t is TableTab =>
        t.kind === "table" &&
        t.view === "data" &&
        t.id !== tab.id &&
        t.profileId === tab.profileId &&
        t.database === tab.database &&
        t.schema === tab.schema &&
        t.name === tab.name,
    );
    if (dataTab) {
      clearBrowse(dataTab);
    }
    await loadStructure(id, { force: true });
  }

  return {
    tabs,
    activeTabId,
    activeTab,
    mountedTabIds,
    coldMountId,
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
    setOrder,
    setSql,
    setDatabase,
    run,
    cancel,
    loadTable,
    loadStructure,
    reloadTab,
    updateStructureColumn,
    addStructureColumn,
    toggleDropStructureColumn,
    moveStructureColumns,
    cancelStructure,
    applyStructure,
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
