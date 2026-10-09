import type { TableInfo } from "@/lib/wails";

export const SEARCH_TABLES_LIMIT = 500;

export const ROW_DB = 32;
export const ROW_TABLE = 28;
export const ROW_META = 24;

export type ExplorerRow =
  | { key: string; kind: "db"; name: string; open: boolean; height: number }
  | { key: string; kind: "schema"; schema: string; height: number }
  | { key: string; kind: "table"; database: string; table: TableInfo; indent: number; qualify: boolean; height: number }
  | { key: string; kind: "status"; text: string; height: number };

export type ExplorerCatalogInput = {
  query: string;
  databases: string[];
  tablesByDb: Record<string, TableInfo[]>;
  expanded: Record<string, boolean>;
  activeDatabase: string;
  showAllDatabases: boolean;
  loadingTables: Record<string, boolean>;
  search: {
    query: string;
    results: TableInfo[];
    truncated: boolean;
    loading: boolean;
    error: string;
  };
};

const SYSTEM_DBS = new Set([
  "mysql",
  "sys",
  "information_schema",
  "performance_schema",
  "template0",
  "template1",
  "system",
]);

export function sortDatabases(names: string[]): string[] {
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

export function tableLabel(row: ExplorerRow): string {
  if (row.kind !== "table") {
    return "";
  }
  if (!row.qualify) {
    return row.table.name;
  }
  if (row.table.schema && row.table.schema !== row.database) {
    return `${row.database}.${row.table.schema} › ${row.table.name}`;
  }
  return `${row.database} › ${row.table.name}`;
}

export function preferredDatabase(profileDb: string | undefined, databases: string[]): string {
  const name = profileDb ?? "";
  return name && databases.includes(name) ? name : "";
}

export function nextActiveDatabase(opts: {
  showAllDatabases: boolean;
  connected: boolean;
  current: string;
  databases: string[];
  tablesLoaded: boolean;
  preferred: string;
}): { skip: true } | { database: string; alreadyLoaded: boolean } {
  if (opts.showAllDatabases || !opts.connected) {
    return { skip: true };
  }
  if (opts.current && opts.databases.includes(opts.current)) {
    return { database: opts.current, alreadyLoaded: opts.tablesLoaded };
  }
  return { database: opts.preferred, alreadyLoaded: false };
}

export function pushTables(
  rows: ExplorerRow[],
  database: string,
  tables: TableInfo[],
  indent: number,
  qualify: boolean,
) {
  const groups = groupTables(tables);
  if (groups) {
    for (const group of groups) {
      rows.push({
        key: `schema:${database}.${group.schema}`,
        kind: "schema",
        schema: group.schema,
        height: ROW_META,
      });
      for (const table of group.tables) {
        rows.push({
          key: `table:${database}.${table.schema}.${table.name}`,
          kind: "table",
          database,
          table,
          indent: indent + 1,
          qualify,
          height: ROW_TABLE,
        });
      }
    }
    return;
  }
  if (tables.length === 0) {
    rows.push({ key: `empty:${database}`, kind: "status", text: "No tables", height: ROW_META });
    return;
  }
  for (const table of tables) {
    rows.push({
      key: `table:${database}.${table.schema}.${table.name}`,
      kind: "table",
      database,
      table,
      indent,
      qualify,
      height: ROW_TABLE,
    });
  }
}

export function buildExplorerRows(input: ExplorerCatalogInput): ExplorerRow[] {
  const rows: ExplorerRow[] = [];
  const query = input.query.trim();
  const q = query.toLowerCase();

  if (q) {
    const seen = new Set<string>();
    for (const name of input.databases) {
      if (!name.toLowerCase().includes(q)) {
        continue;
      }
      rows.push({
        key: `db:${name}`,
        kind: "db",
        name,
        open: Boolean(input.expanded[name]),
        height: ROW_DB,
      });
    }
    const scanDbs = input.showAllDatabases
      ? Object.keys(input.tablesByDb)
      : input.activeDatabase
        ? [input.activeDatabase]
        : [];
    for (const database of scanDbs) {
      for (const table of filterTables(input.tablesByDb[database] ?? [], query)) {
        const id = `${database}\0${table.schema}\0${table.name}`;
        if (seen.has(id)) {
          continue;
        }
        seen.add(id);
        rows.push({
          key: `table:${id}`,
          kind: "table",
          database,
          table,
          indent: 0,
          qualify: true,
          height: ROW_TABLE,
        });
      }
    }
    for (const table of input.search.results) {
      const database = tableDatabase(table);
      const id = `${database}\0${table.schema}\0${table.name}`;
      if (seen.has(id)) {
        continue;
      }
      seen.add(id);
      rows.push({
        key: `search:${id}`,
        kind: "table",
        database,
        table,
        indent: 0,
        qualify: true,
        height: ROW_TABLE,
      });
    }
    if (input.search.loading || input.search.query !== q) {
      rows.push({ key: "search-loading", kind: "status", text: "Searching…", height: ROW_META });
    } else if (input.search.error) {
      rows.push({ key: "search-error", kind: "status", text: input.search.error, height: ROW_META });
    } else if (input.search.truncated) {
      rows.push({
        key: "search-truncated",
        kind: "status",
        text: `Showing first ${SEARCH_TABLES_LIMIT} table matches. Refine your search.`,
        height: ROW_META,
      });
    }
    return rows;
  }

  if (!input.showAllDatabases) {
    const name = input.activeDatabase;
    if (!name) {
      return rows;
    }
    if (input.loadingTables[name] && !(name in input.tablesByDb)) {
      rows.push({ key: `loading:${name}`, kind: "status", text: "Loading…", height: ROW_META });
      return rows;
    }
    pushTables(rows, name, input.tablesByDb[name] ?? [], 0, false);
    return rows;
  }

  for (const name of input.databases) {
    const open = Boolean(input.expanded[name]);
    rows.push({ key: `db:${name}`, kind: "db", name, open, height: ROW_DB });
    if (!open) {
      continue;
    }
    // Expanded state is persisted; tablesByDb is not — treat missing as loading.
    if (!(name in input.tablesByDb)) {
      rows.push({ key: `loading:${name}`, kind: "status", text: "Loading…", height: ROW_META });
      continue;
    }
    pushTables(rows, name, input.tablesByDb[name] ?? [], 1, false);
  }
  return rows;
}
