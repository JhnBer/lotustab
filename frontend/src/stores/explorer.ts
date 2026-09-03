import { defineStore } from "pinia";
import { computed, ref } from "vue";
import { watchDebounced } from "@vueuse/core";
import { errorMessage, isCanceled } from "@/lib/errors";
import { readJson, writeJson } from "@/lib/persist";
import { ConnectionsService, QueryService, type TableInfo } from "@/lib/wails";

const EXPLORER_KEY = "lotustab.explorer";

export const SEARCH_TABLES_LIMIT = 500;

const SYSTEM_DBS = new Set([
  "mysql",
  "sys",
  "information_schema",
  "performance_schema",
  "template0",
  "template1",
  "system",
]);

export type ExplorerSlice = {
  databases: string[];
  tablesByDb: Record<string, TableInfo[]>;
  expanded: Record<string, boolean>;
  activeDatabase: string;
};

type SearchSlice = {
  query: string;
  results: TableInfo[];
  truncated: boolean;
  loading: boolean;
  error: string;
};

type PersistedSlice = {
  databases: string[];
  expanded: Record<string, boolean>;
  activeDatabase: string;
};

function sortDatabases(names: string[]): string[] {
  const user: string[] = [];
  const system: string[] = [];
  for (const name of names) {
    if (SYSTEM_DBS.has(name.toLowerCase())) {
      system.push(name);
    } else {
      user.push(name);
    }
  }
  user.sort((a, b) => a.localeCompare(b));
  system.sort((a, b) => a.localeCompare(b));
  return [...user, ...system];
}

export function groupTables(tables: TableInfo[]): { schema: string; tables: TableInfo[] }[] | null {
  const schemas = [...new Set(tables.map((t) => t.schema).filter(Boolean))];
  if (schemas.length <= 1) {
    return null;
  }
  return schemas.sort((a, b) => a.localeCompare(b)).map((schema) => ({
    schema,
    tables: tables.filter((t) => t.schema === schema),
  }));
}

export function filterTables(tables: TableInfo[], query: string): TableInfo[] {
  const q = query.trim().toLowerCase();
  if (!q) {
    return tables;
  }
  return tables.filter((table) => table.name.toLowerCase().includes(q));
}

export function tableDatabase(table: TableInfo, fallback = ""): string {
  return table.database || fallback || table.schema;
}

function emptySlice(): ExplorerSlice {
  return { databases: [], tablesByDb: {}, expanded: {}, activeDatabase: "" };
}

function emptySearch(): SearchSlice {
  return { query: "", results: [], truncated: false, loading: false, error: "" };
}

function parseSlices(raw: unknown): Record<string, ExplorerSlice> {
  if (!raw || typeof raw !== "object") {
    return {};
  }
  const out: Record<string, ExplorerSlice> = {};
  for (const [id, value] of Object.entries(raw as Record<string, unknown>)) {
    if (!id || !value || typeof value !== "object") {
      continue;
    }
    const row = value as Record<string, unknown>;
    const databases = Array.isArray(row.databases)
      ? row.databases.filter((name): name is string => typeof name === "string")
      : [];
    const expanded: Record<string, boolean> = {};
    if (row.expanded && typeof row.expanded === "object") {
      for (const [db, open] of Object.entries(row.expanded as Record<string, unknown>)) {
        expanded[db] = Boolean(open);
      }
    }
    out[id] = {
      databases,
      tablesByDb: {},
      expanded,
      activeDatabase: typeof row.activeDatabase === "string" ? row.activeDatabase : "",
    };
  }
  return out;
}

function persistable(slices: Record<string, ExplorerSlice>): Record<string, PersistedSlice> {
  const out: Record<string, PersistedSlice> = {};
  for (const [id, slice] of Object.entries(slices)) {
    out[id] = {
      databases: slice.databases,
      expanded: slice.expanded,
      activeDatabase: slice.activeDatabase,
    };
  }
  return out;
}

export const useExplorerStore = defineStore("explorer", () => {
  const slices = ref<Record<string, ExplorerSlice>>(parseSlices(readJson(EXPLORER_KEY, {})));
  const profileId = ref("");
  const loadingDbsById = ref<Record<string, boolean>>({});
  const loadingTablesById = ref<Record<string, Record<string, boolean>>>({});
  const errorById = ref<Record<string, string>>({});
  const searchById = ref<Record<string, SearchSlice>>({});
  const inflight = new Map<string, { cancel: () => void }>();

  const current = computed(() => (profileId.value ? slices.value[profileId.value] : undefined));
  const databases = computed(() => current.value?.databases ?? []);
  const tablesByDb = computed(() => current.value?.tablesByDb ?? {});
  const expanded = computed(() => current.value?.expanded ?? {});
  const activeDatabase = computed(() => current.value?.activeDatabase ?? "");
  const loadingDbs = computed(() => Boolean(loadingDbsById.value[profileId.value]));
  const loadingTables = computed(() => loadingTablesById.value[profileId.value] ?? {});
  const error = computed(() => errorById.value[profileId.value] ?? "");
  const search = computed(() => searchById.value[profileId.value] ?? emptySearch());

  watchDebounced(
    () => persistable(slices.value),
    (value) => writeJson(EXPLORER_KEY, value),
    { debounce: 300, deep: true },
  );

  function sliceOf(id: string): ExplorerSlice {
    return slices.value[id] ?? emptySlice();
  }

  function patch(id: string, next: Partial<ExplorerSlice>) {
    slices.value = { ...slices.value, [id]: { ...sliceOf(id), ...next } };
  }

  function hasCatalog(id: string): boolean {
    return (slices.value[id]?.databases.length ?? 0) > 0;
  }

  function activate(id: string) {
    profileId.value = id;
  }

  function removeProfile(id: string) {
    if (!slices.value[id]) {
      return;
    }
    const next = { ...slices.value };
    delete next[id];
    slices.value = next;
    if (profileId.value === id) {
      profileId.value = "";
    }
    if (searchById.value[id]) {
      const nextSearch = { ...searchById.value };
      delete nextSearch[id];
      searchById.value = nextSearch;
    }
  }

  function setLoadingTables(id: string, database: string, on: boolean) {
    const currentTables = loadingTablesById.value[id] ?? {};
    loadingTablesById.value = {
      ...loadingTablesById.value,
      [id]: { ...currentTables, [database]: on },
    };
  }

  async function loadDatabases(id: string) {
    if (!id) {
      return;
    }
    profileId.value = id;
    const key = `db:${id}`;
    inflight.get(key)?.cancel();
    loadingDbsById.value = { ...loadingDbsById.value, [id]: true };
    errorById.value = { ...errorById.value, [id]: "" };
    const pending = ConnectionsService.Databases(id);
    inflight.set(key, pending);
    try {
      const list = (await pending) ?? [];
      patch(id, { databases: sortDatabases(list) });
    } catch (err) {
      if (isCanceled(err)) {
        return;
      }
      errorById.value = { ...errorById.value, [id]: errorMessage(err) };
      throw err;
    } finally {
      if (inflight.get(key) === pending) {
        inflight.delete(key);
        loadingDbsById.value = { ...loadingDbsById.value, [id]: false };
      }
    }
  }

  async function loadTables(id: string, database: string) {
    const key = `tables:${id}:${database}`;
    inflight.get(key)?.cancel();
    setLoadingTables(id, database, true);
    const pending = QueryService.ListTables(id, database);
    inflight.set(key, pending);
    try {
      const tables = (await pending) ?? [];
      patch(id, { tablesByDb: { ...sliceOf(id).tablesByDb, [database]: tables } });
    } catch (err) {
      if (isCanceled(err)) {
        return;
      }
      patch(id, { tablesByDb: { ...sliceOf(id).tablesByDb, [database]: [] } });
      throw err;
    } finally {
      if (inflight.get(key) === pending) {
        inflight.delete(key);
        setLoadingTables(id, database, false);
      }
    }
  }

  async function toggleDatabase(id: string, database: string) {
    const next = !sliceOf(id).expanded[database];
    patch(id, { expanded: { ...sliceOf(id).expanded, [database]: next } });
    if (next && !sliceOf(id).tablesByDb[database]) {
      await loadTables(id, database);
    }
  }

  async function setActiveDatabase(id: string, database: string) {
    patch(id, { activeDatabase: database });
    if (database && !(database in sliceOf(id).tablesByDb)) {
      await loadTables(id, database);
    }
  }

  async function searchTables(id: string, query: string) {
    if (!id) {
      return;
    }
    const key = `search:${id}`;
    inflight.get(key)?.cancel();
    const q = query.trim();
    if (!q) {
      searchById.value = { ...searchById.value, [id]: emptySearch() };
      return;
    }
    searchById.value = {
      ...searchById.value,
      [id]: { ...(searchById.value[id] ?? emptySearch()), query: q, loading: true, error: "" },
    };
    const pending = QueryService.SearchTables(id, sliceOf(id).activeDatabase, q, SEARCH_TABLES_LIMIT);
    inflight.set(key, pending);
    try {
      const tables = (await pending) ?? [];
      if (inflight.get(key) !== pending) {
        return;
      }
      searchById.value = {
        ...searchById.value,
        [id]: {
          query: q,
          results: tables,
          truncated: tables.length >= SEARCH_TABLES_LIMIT,
          loading: false,
          error: "",
        },
      };
    } catch (err) {
      if (isCanceled(err)) {
        return;
      }
      searchById.value = {
        ...searchById.value,
        [id]: {
          query: q,
          results: [],
          truncated: false,
          loading: false,
          error: errorMessage(err),
        },
      };
    } finally {
      if (inflight.get(key) === pending) {
        inflight.delete(key);
      }
    }
  }

  async function refresh(id: string) {
    if (!id) {
      return;
    }
    const currentSlice = sliceOf(id);
    const expandedDbs = Object.keys(currentSlice.expanded).filter((db) => currentSlice.expanded[db]);
    const keepActive = currentSlice.activeDatabase;
    await loadDatabases(id);
    const names = sliceOf(id).databases;
    const stillOpen = expandedDbs.filter((db) => names.includes(db));
    const reload = [...stillOpen];
    if (keepActive && names.includes(keepActive) && !reload.includes(keepActive)) {
      reload.push(keepActive);
    }
    await Promise.all(reload.map((db) => loadTables(id, db)));
    const keep = new Set(reload);
    patch(id, {
      tablesByDb: Object.fromEntries(
        Object.entries(sliceOf(id).tablesByDb).filter(([db]) => keep.has(db)),
      ),
      activeDatabase: keepActive && names.includes(keepActive) ? keepActive : "",
    });
  }

  return {
    profileId,
    databases,
    tablesByDb,
    expanded,
    loadingDbs,
    loadingTables,
    error,
    activeDatabase,
    search,
    hasCatalog,
    activate,
    removeProfile,
    loadDatabases,
    loadTables,
    toggleDatabase,
    setActiveDatabase,
    searchTables,
    refresh,
  };
});
