import { defineStore } from "pinia";
import { computed, ref } from "vue";
import { errorMessage } from "@/lib/errors";
import {
  QueryService,
  type QueryResult,
  type TableBrowseResult,
} from "@/lib/wails";

export const PAGE_SIZES = [50, 100, 250, 500, 1000] as const;

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
      pageSize: 100,
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
    tabs.value.splice(index, 1);
    if (activeTabId.value === id) {
      const next = tabs.value[index] ?? tabs.value[index - 1];
      activeTabId.value = next?.id ?? "";
    }
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
    setSql,
    run,
    loadTable,
    setPage,
    setPageSize,
    cycleSort,
  };
});
