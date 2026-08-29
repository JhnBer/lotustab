import { defineStore } from "pinia";
import { computed, ref } from "vue";
import { errorMessage } from "@/lib/errors";
import {
  QueryService,
  type QueryResult,
  type TableBrowseResult,
} from "@/lib/wails";

export const PAGE_SIZES = [50, 100, 250, 500, 1000, 5000, 10000] as const;

const DEFAULT_PAGE_SIZE = 100;
const PAGE_SIZE_KEY = "lotustab.tablePageSize";

function storedPageSize(): number {
  const n = Number(localStorage.getItem(PAGE_SIZE_KEY));
  return (PAGE_SIZES as readonly number[]).includes(n) ? n : DEFAULT_PAGE_SIZE;
}

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
  sql: string;
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
}

export type WorkspaceTab = QueryTab | TableTab;

export interface WorkspaceSnapshot {
  tabs: WorkspaceTab[];
  activeTabId: string;
}

function newQueryTab(profileId: string, title: string): QueryTab {
  return {
    kind: "query",
    id: crypto.randomUUID(),
    title,
    profileId,
    sql: "SELECT 1;",
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

export const useQueryStore = defineStore("query", () => {
  const tabs = ref<WorkspaceTab[]>([]);
  const activeTabId = ref("");

  const activeTab = computed(() => tabs.value.find((t) => t.id === activeTabId.value));

  function openForProfile(profileId: string, name: string) {
    const existing = tabs.value.find((t) => t.kind === "query" && t.profileId === profileId);
    if (existing) {
      activeTabId.value = existing.id;
      return existing;
    }
    return add(profileId, name);
  }

  function add(profileId: string, name: string) {
    const count = tabs.value.filter((t) => t.kind === "query" && t.profileId === profileId).length;
    const tab = newQueryTab(profileId, count === 0 ? name : `${name} ${count + 1}`);
    tabs.value.push(tab);
    activeTabId.value = tab.id;
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
      activeTabId.value = existing.id;
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
    };
    tabs.value.push(tab);
    activeTabId.value = tab.id;
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
    if (activeTabId.value === id) {
      const group = tabs.value.filter((t) => t.profileId === profileId);
      const next = group[groupIndex] ?? group[groupIndex - 1];
      activeTabId.value = next?.id ?? "";
    }
  }

  function snapshotForProfile(profileId: string): WorkspaceSnapshot {
    const group = tabs.value.filter((t) => t.profileId === profileId);
    const activeInGroup = group.some((t) => t.id === activeTabId.value);
    return {
      tabs: group.map((t) => ({ ...t, running: false })),
      activeTabId: activeInGroup ? activeTabId.value : (group[0]?.id ?? ""),
    };
  }

  function removeForProfile(profileId: string) {
    const removingActive =
      tabs.value.find((t) => t.id === activeTabId.value)?.profileId === profileId;
    tabs.value = tabs.value.filter((t) => t.profileId !== profileId);
    if (removingActive) {
      activeTabId.value = tabs.value[0]?.id ?? "";
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
        tabs.value.push({ ...tab, running: false });
      }
    }
    if (snapshot.activeTabId && tabs.value.some((t) => t.id === snapshot.activeTabId)) {
      activeTabId.value = snapshot.activeTabId;
    } else {
      activeTabId.value = snapshot.tabs[0]!.id;
    }
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

  async function run(id: string) {
    const tab = tabs.value.find((t) => t.id === id);
    if (!tab || tab.kind !== "query") {
      return;
    }
    if (!tab.sql.trim()) {
      tab.error = "Query is empty";
      return;
    }
    tab.running = true;
    tab.error = "";
    try {
      tab.result = await QueryService.Execute(tab.profileId, tab.sql);
    } catch (err) {
      tab.result = null;
      tab.error = errorMessage(err);
    } finally {
      tab.running = false;
    }
  }

  async function loadTable(id: string) {
    const tab = tabs.value.find((t) => t.id === id);
    if (!tab || tab.kind !== "table") {
      return;
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
      });
      tab.result = asQueryResult(result);
      tab.total = result?.total ?? 0;
    } catch (err) {
      tab.result = null;
      tab.total = 0;
      tab.error = errorMessage(err);
    } finally {
      tab.running = false;
    }
  }

  function setPage(id: string, page: number) {
    const tab = tabs.value.find((t) => t.id === id);
    if (!tab || tab.kind !== "table") {
      return;
    }
    const maxPage = Math.max(1, Math.ceil(tab.total / tab.pageSize) || 1);
    tab.page = Math.min(Math.max(1, page), maxPage);
    void loadTable(id);
  }

  function setPageSize(id: string, pageSize: number) {
    const tab = tabs.value.find((t) => t.id === id);
    if (!tab || tab.kind !== "table") {
      return;
    }
    tab.pageSize = pageSize;
    tab.page = 1;
    if ((PAGE_SIZES as readonly number[]).includes(pageSize)) {
      localStorage.setItem(PAGE_SIZE_KEY, String(pageSize));
    }
    void loadTable(id);
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

  return {
    tabs,
    activeTabId,
    activeTab,
    openForProfile,
    add,
    openTable,
    close,
    snapshotForProfile,
    removeForProfile,
    restore,
    reorder,
    setSql,
    run,
    loadTable,
    setPage,
    setPageSize,
    cycleSort,
  };
});
