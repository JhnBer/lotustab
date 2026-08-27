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

export const useExplorerStore = defineStore("explorer", () => {
  const profileId = ref("");
  const databases = ref<string[]>([]);
  const tablesByDb = ref<Record<string, TableInfo[]>>({});
  const expanded = ref<Record<string, boolean>>({});
  const loadingDbs = ref(false);
  const loadingTables = ref<Record<string, boolean>>({});
  const error = ref("");

  function reset() {
    profileId.value = "";
    databases.value = [];
    tablesByDb.value = {};
    expanded.value = {};
    loadingDbs.value = false;
    loadingTables.value = {};
    error.value = "";
  }

  async function loadDatabases(id: string) {
    if (!id) {
      reset();
      return;
    }
    if (profileId.value !== id) {
      tablesByDb.value = {};
      expanded.value = {};
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

  return {
    profileId,
    databases,
    tablesByDb,
    expanded,
    loadingDbs,
    loadingTables,
    error,
    reset,
    loadDatabases,
    loadTables,
    toggleDatabase,
  };
});
