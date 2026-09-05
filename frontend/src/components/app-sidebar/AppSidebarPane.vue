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
import { useProfileActions } from "@/composables/useProfileActions";
import {
  buildExplorerRows,
  nextActiveDatabase,
  preferredDatabase,
  ROW_TABLE,
  tableDatabase,
  tableLabel,
} from "@/features/explorer/catalogRows";
import { errorMessage } from "@/lib/errors";
import type { TableInfo } from "@/lib/wails";
import { useConnectionsStore } from "@/stores/connections";
import { useExplorerStore } from "@/stores/explorer";
import { useQueryStore } from "@/stores/query";
import { useSettingsStore } from "@/stores/settings";

const router = useRouter();
const connections = useConnectionsStore();
const explorer = useExplorerStore();
const queries = useQueryStore();
const settings = useSettingsStore();
const { connect, disconnect } = useProfileActions();
const { selected } = storeToRefs(connections);
const disconnecting = ref(false);

const currentConnected = computed(() =>
  selected.value ? connections.isConnected(selected.value.id) : false,
);

const tableQuery = ref("");
const filtering = computed(() => tableQuery.value.trim().length > 0);
const catalogScrollRef = ref<HTMLElement | null>(null);

const explorerRows = computed(() =>
  buildExplorerRows({
    query: tableQuery.value,
    databases: explorer.databases,
    tablesByDb: explorer.tablesByDb,
    expanded: explorer.expanded,
    activeDatabase: explorer.activeDatabase,
    showAllDatabases: settings.showAllDatabases,
    loadingTables: explorer.loadingTables,
    search: explorer.search,
  }),
);

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

function syncActiveDatabase() {
  if (!selected.value) {
    return;
  }
  const current = explorer.activeDatabase;
  const next = nextActiveDatabase({
    showAllDatabases: settings.showAllDatabases,
    connected: currentConnected.value,
    current,
    databases: explorer.databases,
    tablesLoaded: current in explorer.tablesByDb,
    preferred: preferredDatabase(selected.value.params?.database, explorer.databases),
  });
  if ("skip" in next || next.alreadyLoaded) {
    return;
  }
  void explorer.setActiveDatabase(selected.value.id, next.database).catch((err) => {
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
  await connect(selected.value, { navigate: true });
}

async function disconnectSelected() {
  if (!selected.value || disconnecting.value) {
    return;
  }
  disconnecting.value = true;
  try {
    await disconnect(selected.value);
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
