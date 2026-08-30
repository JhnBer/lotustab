import { defineStore } from "pinia";
import { ref } from "vue";
import { errorMessage } from "@/lib/errors";
import { ConnectionsService, QueryService, type TableInfo } from "@/lib/wails";

const SYSTEM_DBS = new Set([
  "mysql",
  "sys",
  "information_schema",
  "performance_schema",
  "template0",
  "template1",
]);

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

export const useExplorerStore = defineStore("explorer", () => {
  const profileId = ref("");
  const databases = ref<string[]>([]);
  const tablesByDb = ref<Record<string, TableInfo[]>>({});
  const expanded = ref<Record<string, boolean>>({});
  const loadingDbs = ref(false);
  const loadingTables = ref<Record<string, boolean>>({});
  const error = ref("");
  const activeDatabase = ref("");

  function reset() {
    profileId.value = "";
    databases.value = [];
    tablesByDb.value = {};
    expanded.value = {};
    loadingDbs.value = false;
    loadingTables.value = {};
    error.value = "";
    activeDatabase.value = "";
  }

  async function loadDatabases(id: string) {
    if (!id) {
      reset();
      return;
    }
    if (profileId.value !== id) {
      tablesByDb.value = {};
      expanded.value = {};
      activeDatabase.value = "";
    }
    profileId.value = id;
    loadingDbs.value = true;
    error.value = "";
    try {
      const list = (await ConnectionsService.Databases(id)) ?? [];
      databases.value = sortDatabases(list);
    } catch (err) {
      error.value = errorMessage(err);
      databases.value = [];
      throw err;
    } finally {
      loadingDbs.value = false;
    }
  }

  async function loadTables(id: string, database: string) {
    loadingTables.value = { ...loadingTables.value, [database]: true };
    try {
      const tables = (await QueryService.ListTables(id, database)) ?? [];
      tablesByDb.value = { ...tablesByDb.value, [database]: tables };
    } catch (err) {
      tablesByDb.value = { ...tablesByDb.value, [database]: [] };
      throw err;
    } finally {
      loadingTables.value = { ...loadingTables.value, [database]: false };
    }
  }

  async function toggleDatabase(id: string, database: string) {
    const next = !expanded.value[database];
    expanded.value = { ...expanded.value, [database]: next };
    if (next && !tablesByDb.value[database]) {
      await loadTables(id, database);
    }
  }

  async function setActiveDatabase(id: string, database: string) {
    activeDatabase.value = database;
    if (database && !(database in tablesByDb.value)) {
      await loadTables(id, database);
    }
  }

  async function ensureTables(id: string) {
    if (!id) {
      return;
    }
    for (const database of databases.value) {
      if (database in tablesByDb.value) {
        continue;
      }
      try {
        await loadTables(id, database);
      } catch {
        // Cached as [] in loadTables; keep scanning the rest.
      }
    }
  }

  async function refresh(id: string) {
    if (!id) {
      return;
    }
    const expandedDbs = Object.keys(expanded.value).filter((db) => expanded.value[db]);
    const keepActive = activeDatabase.value;
    await loadDatabases(id);
    const stillOpen = expandedDbs.filter((db) => databases.value.includes(db));
    const reload = [...stillOpen];
    if (keepActive && databases.value.includes(keepActive) && !reload.includes(keepActive)) {
      reload.push(keepActive);
    }
    await Promise.all(reload.map((db) => loadTables(id, db)));
    const keep = new Set(reload);
    tablesByDb.value = Object.fromEntries(
      Object.entries(tablesByDb.value).filter(([db]) => keep.has(db)),
    );
    if (keepActive && !databases.value.includes(keepActive)) {
      activeDatabase.value = "";
    }
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
    reset,
    loadDatabases,
    loadTables,
    toggleDatabase,
    setActiveDatabase,
    ensureTables,
    refresh,
  };
});
