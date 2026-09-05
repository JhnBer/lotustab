<script setup lang="ts">
import {
  ChevronDown,
  ChevronRight,
  Database,
  Eye,
  Plug,
  RefreshCw,
  Table2,
  Unplug,
} from "@lucide/vue";
import { toast } from "vue-sonner";
import { storeToRefs } from "pinia";
import { computed, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { useVirtualizer } from "@tanstack/vue-virtual";
import { watchDebounced } from "@vueuse/core";
import {
  SidebarGroup,
  SidebarGroupAction,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarInput,
  SidebarMenuButton,
  SidebarMenuSubButton,
} from "@/components/ui/sidebar";
import { Button } from "@/components/ui/button";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import { Spinner } from "@/components/ui/spinner";
import DatabaseCombobox from "@/components/DatabaseCombobox.vue";
import { errorMessage, isCanceled } from "@/lib/errors";
import type { TableInfo } from "@/lib/wails";
import { useConnectionsStore } from "@/stores/connections";
import {
  filterTables,
  groupTables,
  SEARCH_TABLES_LIMIT,
  tableDatabase,
  useExplorerStore,
} from "@/stores/explorer";
import { useQueryStore } from "@/stores/query";
import { useSettingsStore } from "@/stores/settings";

const router = useRouter();
const connections = useConnectionsStore();
const explorer = useExplorerStore();
const queries = useQueryStore();
const settings = useSettingsStore();
const { selected } = storeToRefs(connections);
const disconnecting = ref(false);

const currentConnected = computed(() =>
  selected.value ? connections.isConnected(selected.value.id) : false,
);

const tableQuery = ref("");
const filtering = computed(() => tableQuery.value.trim().length > 0);
const catalogScrollRef = ref<HTMLElement | null>(null);

const ROW_DB = 32;
const ROW_TABLE = 28;
const ROW_META = 24;

type ExplorerRow =
  | { key: string; kind: "db"; name: string; open: boolean; height: number }
  | { key: string; kind: "schema"; schema: string; height: number }
  | { key: string; kind: "table"; database: string; table: TableInfo; indent: number; qualify: boolean; height: number }
  | { key: string; kind: "status"; text: string; height: number };

function pushTables(rows: ExplorerRow[], database: string, tables: TableInfo[], indent: number, qualify: boolean) {
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

const explorerRows = computed((): ExplorerRow[] => {
  const rows: ExplorerRow[] = [];
  const query = tableQuery.value.trim();
  const q = query.toLowerCase();

  if (q) {
    const seen = new Set<string>();
    for (const name of explorer.databases) {
      if (!name.toLowerCase().includes(q)) {
        continue;
      }
      rows.push({
        key: `db:${name}`,
        kind: "db",
        name,
        open: Boolean(explorer.expanded[name]),
        height: ROW_DB,
      });
    }
    const scanDbs = settings.showAllDatabases
      ? Object.keys(explorer.tablesByDb)
      : explorer.activeDatabase
        ? [explorer.activeDatabase]
        : [];
    for (const database of scanDbs) {
      for (const table of filterTables(explorer.tablesByDb[database] ?? [], query)) {
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
    for (const table of explorer.search.results) {
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
    if (explorer.search.loading || explorer.search.query !== q) {
      rows.push({ key: "search-loading", kind: "status", text: "Searching…", height: ROW_META });
    } else if (explorer.search.error) {
      rows.push({ key: "search-error", kind: "status", text: explorer.search.error, height: ROW_META });
    } else if (explorer.search.truncated) {
      rows.push({
        key: "search-truncated",
        kind: "status",
        text: `Showing first ${SEARCH_TABLES_LIMIT} table matches. Refine your search.`,
        height: ROW_META,
      });
    }
    return rows;
  }

  if (!settings.showAllDatabases) {
    const name = explorer.activeDatabase;
    if (!name) {
      return rows;
    }
    if (explorer.loadingTables[name] && !(name in explorer.tablesByDb)) {
      rows.push({ key: `loading:${name}`, kind: "status", text: "Loading…", height: ROW_META });
      return rows;
    }
    pushTables(rows, name, explorer.tablesByDb[name] ?? [], 0, false);
    return rows;
  }

  for (const name of explorer.databases) {
    const open = Boolean(explorer.expanded[name]);
    rows.push({ key: `db:${name}`, kind: "db", name, open, height: ROW_DB });
    if (!open) {
      continue;
    }
    if (explorer.loadingTables[name] && !(name in explorer.tablesByDb)) {
      rows.push({ key: `loading:${name}`, kind: "status", text: "Loading…", height: ROW_META });
      continue;
    }
    pushTables(rows, name, explorer.tablesByDb[name] ?? [], 1, false);
  }
  return rows;
});

const rowVirtualizer = useVirtualizer(
  computed(() => ({
    count: explorerRows.value.length,
    getScrollElement: () => catalogScrollRef.value,
    estimateSize: (index: number) => explorerRows.value[index]?.height ?? ROW_TABLE,
    overscan: 12,
  })),
);
const virtualRows = computed(() => rowVirtualizer.value.getVirtualItems());
const catalogTotalSize = computed(() => rowVirtualizer.value.getTotalSize());
const visibleVirtual = computed(() =>
  virtualRows.value.flatMap((v) => {
    const row = explorerRows.value[v.index];
    return row ? [{ v, row }] : [];
  }),
);

function tableLabel(row: ExplorerRow): string {
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

function preferredDatabase(): string {
  const name = selected.value?.params?.database ?? "";
  return name && explorer.databases.includes(name) ? name : "";
}

function syncActiveDatabase() {
  if (settings.showAllDatabases || !selected.value || !currentConnected.value) {
    return;
  }
  const current = explorer.activeDatabase;
  if (current && explorer.databases.includes(current)) {
    if (!(current in explorer.tablesByDb)) {
      void explorer.setActiveDatabase(selected.value.id, current).catch((err) => {
        toast.error(errorMessage(err));
      });
    }
    return;
  }
  void explorer.setActiveDatabase(selected.value.id, preferredDatabase()).catch((err) => {
    toast.error(errorMessage(err));
  });
}

watch(
  () => [selected.value?.id, currentConnected.value] as const,
  async ([id, connected]) => {
    tableQuery.value = "";
    disconnecting.value = false;
    if (!id) {
      return;
    }
    explorer.activate(id);
    if (!connected) {
      return;
    }
    if (explorer.hasCatalog(id)) {
      syncActiveDatabase();
      return;
    }
    try {
      await explorer.loadDatabases(id);
      syncActiveDatabase();
    } catch (err) {
      toast.error(errorMessage(err));
    }
  },
  { immediate: true },
);

watch(
  () => settings.showAllDatabases,
  (showAll) => {
    if (!showAll) {
      syncActiveDatabase();
    }
  },
);

watch(
  () => tableQuery.value.trim(),
  (query) => {
    if (query || !selected.value || !currentConnected.value) {
      return;
    }
    void explorer.searchTables(selected.value.id, "");
  },
);

watchDebounced(
  () => [tableQuery.value.trim(), selected.value?.id, currentConnected.value] as const,
  ([query, id, connected]) => {
    if (!id || !connected || !query) {
      return;
    }
    void explorer.searchTables(id, query);
  },
  { debounce: 300 },
);

async function connectSelected() {
  if (!selected.value) {
    return;
  }
  if (connections.isConnecting(selected.value.id)) {
    connections.cancelConnect(selected.value.id);
    return;
  }
  try {
    await connections.connect(selected.value.id);
    if (!connections.isConnected(selected.value.id)) {
      return;
    }
    toast.success(`Connected to ${selected.value.name}`);
    router.push("/query");
  } catch (err) {
    if (isCanceled(err)) {
      return;
    }
    toast.error(errorMessage(err));
  }
}

async function disconnectSelected() {
  if (!selected.value || disconnecting.value) {
    return;
  }
  disconnecting.value = true;
  try {
    await connections.disconnect(selected.value.id);
    toast.info(`Disconnected from ${selected.value.name}`);
  } catch (err) {
    toast.error(errorMessage(err));
  } finally {
    disconnecting.value = false;
  }
}

async function onToggleDatabase(database: string) {
  if (!selected.value) {
    return;
  }
  try {
    if (!settings.showAllDatabases) {
      await explorer.setActiveDatabase(selected.value.id, database);
      return;
    }
    await explorer.toggleDatabase(selected.value.id, database);
  } catch (err) {
    toast.error(errorMessage(err));
  }
}

async function onSelectDatabase(value: string) {
  if (!selected.value || !value) {
    return;
  }
  try {
    await explorer.setActiveDatabase(selected.value.id, value);
  } catch (err) {
    toast.error(errorMessage(err));
  }
}

const explorerRefreshing = ref(false);

async function refreshExplorer() {
  if (!selected.value || explorerRefreshing.value) {
    return;
  }
  explorerRefreshing.value = true;
  try {
    await explorer.refresh(selected.value.id);
    if (!settings.showAllDatabases) {
      syncActiveDatabase();
    }
  } catch (err) {
    toast.error(errorMessage(err));
  } finally {
    explorerRefreshing.value = false;
  }
}

function openTable(database: string, table: TableInfo, view: "data" | "structure" = "data") {
  if (!selected.value) {
    return;
  }
  queries.openTable({
    profileId: selected.value.id,
    database: tableDatabase(table, database),
    schema: table.schema,
    name: table.name,
    view,
  });
  router.push("/query");
}
</script>

<template>
  <SidebarGroup v-if="selected" class="shrink-0">
    <SidebarGroupContent class="px-2 py-0.5">
      <Button
        v-if="connections.isConnecting(selected.id)"
        size="sm"
        variant="outline"
        class="w-full"
        @click="connections.cancelConnect(selected.id)"
      >
        <Spinner class="size-3.5" />
        Cancel
      </Button>
      <Button
        v-else-if="currentConnected"
        size="sm"
        variant="outline"
        class="w-full"
        @click="disconnectSelected"
      >
        <Spinner v-if="disconnecting" class="size-3.5" />
        <Unplug v-else class="size-3.5" />
        Disconnect
      </Button>
      <Button
        v-else
        size="sm"
        class="w-full"
        @click="connectSelected"
      >
        <Plug class="size-3.5" />
        Connect
      </Button>
    </SidebarGroupContent>
  </SidebarGroup>

  <SidebarGroup v-if="selected && currentConnected" class="min-h-0 flex-1 gap-1 overflow-hidden">
    <SidebarGroupLabel v-if="settings.showAllDatabases" class="bg-sidebar shrink-0">
      Databases
    </SidebarGroupLabel>
    <div v-else class="flex h-8 shrink-0 items-center pr-8">
      <DatabaseCombobox
        class="h-7 w-full min-w-0"
        :model-value="explorer.activeDatabase"
        :options="explorer.databases"
        placeholder="Select database"
        @update:model-value="onSelectDatabase"
      />
    </div>
    <SidebarGroupAction
      as="button"
      title="Reload tables"
      :disabled="explorerRefreshing"
      @click="refreshExplorer"
    >
      <RefreshCw :class="explorerRefreshing ? 'animate-spin' : undefined" />
    </SidebarGroupAction>
    <div class="px-0 pb-1">
      <SidebarInput v-model="tableQuery" placeholder="Filter tables…" />
    </div>
    <SidebarGroupContent class="min-h-0 flex-1 overflow-hidden">
      <p v-if="explorer.loadingDbs && explorer.databases.length === 0" class="text-muted-foreground px-2 py-2 text-xs">
        Loading databases…
      </p>
      <p v-else-if="explorer.error && explorer.databases.length === 0" class="text-destructive px-2 py-2 text-xs">
        {{ explorer.error }}
      </p>
      <p v-else-if="explorer.databases.length === 0" class="text-muted-foreground px-2 py-2 text-xs">
        No databases.
      </p>
      <p v-else-if="!settings.showAllDatabases && !explorer.activeDatabase" class="text-muted-foreground px-2 py-2 text-xs">
        Select a database.
      </p>
      <p v-else-if="filtering && explorerRows.length === 0" class="text-muted-foreground px-2 py-2 text-xs">
        No matching tables.
      </p>
      <div v-else ref="catalogScrollRef" class="h-full overflow-auto">
        <div :style="{ height: `${catalogTotalSize}px`, position: 'relative' }">
          <div
            v-for="item in visibleVirtual"
            :key="item.row.key"
            class="absolute top-0 left-0 w-full"
            :style="{ transform: `translateY(${item.v.start}px)`, height: `${item.v.size}px` }"
          >
            <SidebarMenuButton
              v-if="item.row.kind === 'db'"
              :is-active="item.row.open"
              class="h-full"
              @click="onToggleDatabase(item.row.name)"
            >
              <ChevronDown v-if="item.row.open" class="size-3.5" />
              <ChevronRight v-else class="size-3.5" />
              <Database class="size-3.5" />
              <span>{{ item.row.name }}</span>
            </SidebarMenuButton>
            <div
              v-else-if="item.row.kind === 'schema'"
              class="text-muted-foreground flex h-full items-center px-6 text-[11px] font-medium"
            >
              {{ item.row.schema }}
            </div>
            <ContextMenu v-else-if="item.row.kind === 'table'">
              <ContextMenuTrigger as-child>
                <SidebarMenuSubButton
                  as="button"
                  class="h-full w-full"
                  :class="item.row.indent > 0 ? 'pl-6' : undefined"
                  @click="openTable(item.row.database, item.row.table)"
                >
                  <Eye v-if="item.row.table.type === 'view'" class="size-3.5" />
                  <Table2 v-else class="size-3.5" />
                  <span>{{ tableLabel(item.row) }}</span>
                </SidebarMenuSubButton>
              </ContextMenuTrigger>
              <ContextMenuContent>
                <ContextMenuItem @select="openTable(item.row.database, item.row.table)">
                  Open table
                </ContextMenuItem>
                <ContextMenuItem @select="openTable(item.row.database, item.row.table, 'structure')">
                  Open structure
                </ContextMenuItem>
              </ContextMenuContent>
            </ContextMenu>
            <span
              v-else
              class="text-muted-foreground flex h-full items-center px-2 text-xs"
            >
              {{ item.row.text }}
            </span>
          </div>
        </div>
      </div>
    </SidebarGroupContent>
  </SidebarGroup>
</template>
