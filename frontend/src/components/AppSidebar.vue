<script setup lang="ts">
import {
  ChevronDown,
  ChevronRight,
  Database,
  Eye,
  Pencil,
  Pin,
  Plug,
  RefreshCw,
  SquareArrowOutUpRight,
  Table2,
  Unplug,
} from "@lucide/vue";
import { toast } from "vue-sonner";
import { storeToRefs } from "pinia";
import { computed, onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { Window } from "@wailsio/runtime";
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupAction,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarInput,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  SidebarSeparator,
} from "@/components/ui/sidebar";
import { Button } from "@/components/ui/button";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import { Spinner } from "@/components/ui/spinner";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import ConnectionDialog from "@/features/connections/ConnectionDialog.vue";
import { errorMessage, isCanceled } from "@/lib/errors";
import { connBg, connTint } from "@/lib/colors";
import type { Profile, TableInfo } from "@/lib/wails";
import { useConnectionsStore } from "@/stores/connections";
import { filterTables, groupTables, useExplorerStore } from "@/stores/explorer";
import { useQueryStore } from "@/stores/query";
import { useSettingsStore } from "@/stores/settings";

const router = useRouter();
const connections = useConnectionsStore();
const explorer = useExplorerStore();
const queries = useQueryStore();
const settings = useSettingsStore();
const { selected, recentProfiles } = storeToRefs(connections);
const headerTint = computed(() =>
  settings.connectionTint && selected.value ? connTint(selected.value.color) : "",
);
const isMainWindow = ref(true);
const dialogOpen = ref(false);
const editing = ref<Profile | null>(null);

onMounted(async () => {
  try {
    isMainWindow.value = (await Window.Name()) === "main";
  } catch {
    // Preview / non-Wails: keep the pop-out control visible.
  }
});

const currentConnected = computed(() =>
  selected.value ? connections.isConnected(selected.value.id) : false,
);

const tableQuery = ref("");
const ensuringTables = ref(false);
const filtering = computed(() => tableQuery.value.trim().length > 0);

type VisibleDatabase = {
  name: string;
  tables: TableInfo[];
  showHeader: boolean;
};

const visibleDatabases = computed((): VisibleDatabase[] => {
  const query = tableQuery.value;
  if (!settings.showAllDatabases) {
    const name = explorer.activeDatabase;
    if (!name) {
      return [];
    }
    return [
      {
        name,
        tables: filterTables(explorer.tablesByDb[name] ?? [], query),
        showHeader: false,
      },
    ];
  }
  const q = query.trim();
  const out: VisibleDatabase[] = [];
  for (const name of explorer.databases) {
    if (q && !(name in explorer.tablesByDb)) {
      continue;
    }
    const tables = filterTables(explorer.tablesByDb[name] ?? [], query);
    if (q && tables.length === 0) {
      continue;
    }
    out.push({ name, tables, showHeader: true });
  }
  return out;
});

const catalogIncomplete = computed(() => {
  if (!filtering.value || !settings.showAllDatabases) {
    return ensuringTables.value;
  }
  return explorer.databases.some(
    (name) => !(name in explorer.tablesByDb) || explorer.loadingTables[name],
  );
});

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
  () =>
    [
      tableQuery.value.trim(),
      settings.showAllDatabases,
      selected.value?.id,
      currentConnected.value,
      explorer.databases.length,
    ] as const,
  async ([query, showAll, id, connected]) => {
    if (!id || !connected || !showAll || !query) {
      return;
    }
    ensuringTables.value = true;
    try {
      await explorer.ensureTables(id);
    } catch (err) {
      toast.error(errorMessage(err));
    } finally {
      ensuringTables.value = false;
    }
  },
);

function activateProfile(profile: Profile) {
  connections.select(profile.id);
  router.push("/query");
}

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

async function togglePin(profile: Profile) {
  try {
    await connections.setPinned(profile.id, !connections.isPinned(profile.id));
  } catch (err) {
    toast.error(errorMessage(err));
  }
}

async function connectRecent(profile: Profile) {
  if (connections.isConnecting(profile.id)) {
    connections.cancelConnect(profile.id);
    return;
  }
  try {
    await connections.connect(profile.id);
    if (!connections.isConnected(profile.id)) {
      return;
    }
    toast.success(`Connected to ${profile.name}`);
  } catch (err) {
    if (isCanceled(err)) {
      return;
    }
    toast.error(errorMessage(err));
  }
}

async function disconnectRecent(profile: Profile) {
  try {
    await connections.disconnect(profile.id);
    toast.info(`Disconnected from ${profile.name}`);
  } catch (err) {
    toast.error(errorMessage(err));
  }
}

function openEdit(profile: Profile) {
  editing.value = profile;
  dialogOpen.value = true;
}

async function openInNewWindow() {
  if (!selected.value) {
    return;
  }
  try {
    await connections.openInNewWindow(selected.value);
    await router.push("/connections");
  } catch (err) {
    toast.error(errorMessage(err));
  }
}

async function onToggleDatabase(database: string) {
  if (!selected.value) {
    return;
  }
  try {
    await explorer.toggleDatabase(selected.value.id, database);
  } catch (err) {
    toast.error(errorMessage(err));
  }
}

async function onSelectDatabase(value: unknown) {
  if (!selected.value || typeof value !== "string") {
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
    } else if (tableQuery.value.trim()) {
      ensuringTables.value = true;
      try {
        await explorer.ensureTables(selected.value.id);
      } finally {
        ensuringTables.value = false;
      }
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
    database,
    schema: table.schema,
    name: table.name,
    view,
  });
  router.push("/query");
}

function tableGroups(tables: TableInfo[]) {
  return groupTables(tables);
}

function isDbOpen(name: string, showHeader: boolean): boolean {
  if (!showHeader || filtering.value) {
    return true;
  }
  return Boolean(explorer.expanded[name]);
}
</script>

<template>
  <Sidebar collapsible="none" class="bg-sidebar h-full min-h-0 w-full! overflow-hidden">
    <SidebarHeader :class="headerTint">
      <div class="h-12 -m-2 flex items-center p-2">
          <div v-if="selected" class="flex items-start gap-2 px-1 py-0.5">
        <span
            class="mt-1.5 size-2.5 shrink-0 rounded-full"
            :class="connBg(selected.color)"
        />
              <div class="min-w-0 flex-1">
                  <div class="flex items-center gap-1">
                      <span class="truncate text-sm font-medium">{{ selected.name }}</span>
                      <Button
                          size="icon-xs"
                          variant="ghost"
                          class="shrink-0"
                          :title="connections.isPinned(selected.id) ? 'Unpin' : 'Pin'"
                          @click="togglePin(selected)"
                      >
                          <Pin
                              class="size-3.5"
                              :class="connections.isPinned(selected.id) ? 'fill-current' : ''"
                          />
                      </Button>
                      <Button
                          v-if="isMainWindow"
                          size="icon-xs"
                          variant="ghost"
                          class="shrink-0"
                          title="Open in new window"
                          @click="openInNewWindow"
                      >
                          <SquareArrowOutUpRight class="size-3.5" />
                      </Button>
                  </div>
                  <!--          <div class="text-muted-foreground flex items-center gap-1.5 text-xs">-->
                  <!--            {{ currentConnected ? "Connected" : "Offline" }}-->
                  <!--          </div>-->
              </div>
          </div>
          <div v-else class="text-sm font-medium px-2 py-1">
              Connections
          </div>
      </div>
    </SidebarHeader>

    <SidebarSeparator class="m-0 -translate-y-full" />

    <SidebarContent class="overflow-hidden">
      <SidebarGroup v-if="!selected" class="min-h-0 flex-1 overflow-hidden">
        <SidebarGroupLabel class="bg-sidebar shrink-0">Recent</SidebarGroupLabel>
        <SidebarGroupContent class="min-h-0 flex-1 overflow-auto">
          <p v-if="recentProfiles.length === 0" class="text-muted-foreground px-2 py-2 text-xs">
            No recent connections.
          </p>
          <SidebarMenu v-else>
            <SidebarMenuItem v-for="profile in recentProfiles" :key="profile.id">
              <div class="flex items-center gap-0.5">
                <SidebarMenuButton class="min-w-0 flex-1" @click="activateProfile(profile)">
                  <span
                    class="size-2 shrink-0 rounded-full"
                    :class="connBg(profile.color)"
                  />
                  <span class="truncate">{{ profile.name }}</span>
                </SidebarMenuButton>
                <Button
                  size="icon-xs"
                  variant="ghost"
                  class="shrink-0"
                  :title="connections.isPinned(profile.id) ? 'Unpin' : 'Pin'"
                  @click.stop="togglePin(profile)"
                >
                  <Pin
                    class="size-3.5"
                    :class="connections.isPinned(profile.id) ? 'fill-current' : ''"
                  />
                </Button>
                <Button
                  v-if="connections.isConnecting(profile.id)"
                  size="icon-xs"
                  variant="ghost"
                  class="shrink-0"
                  title="Cancel"
                  @click.stop="connections.cancelConnect(profile.id)"
                >
                  <Spinner class="size-3.5" />
                </Button>
                <Button
                  v-else-if="!connections.isConnected(profile.id)"
                  size="icon-xs"
                  variant="ghost"
                  class="shrink-0"
                  title="Connect"
                  @click.stop="connectRecent(profile)"
                >
                  <Plug class="size-3.5" />
                </Button>
                <Button
                  v-else
                  size="icon-xs"
                  variant="ghost"
                  class="shrink-0"
                  title="Disconnect"
                  @click.stop="disconnectRecent(profile)"
                >
                  <Unplug class="size-3.5" />
                </Button>
                <Button
                  size="icon-xs"
                  variant="ghost"
                  class="shrink-0"
                  title="Edit"
                  @click.stop="openEdit(profile)"
                >
                  <Pencil class="size-3.5" />
                </Button>
              </div>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroupContent>
      </SidebarGroup>

      <SidebarGroup v-else-if="!currentConnected">
        <SidebarGroupContent class="px-2 py-2">
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

      <SidebarGroup v-else class="min-h-0 flex-1 gap-1 overflow-hidden">
        <SidebarGroupLabel v-if="settings.showAllDatabases" class="bg-sidebar shrink-0">
          Databases
        </SidebarGroupLabel>
        <div v-else class="flex h-8 shrink-0 items-center pr-8">
          <Select
            :model-value="explorer.activeDatabase || undefined"
            @update:model-value="onSelectDatabase"
          >
            <SelectTrigger size="sm" class="h-7 w-full min-w-0">
              <SelectValue placeholder="Select database" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="database in explorer.databases" :key="database" :value="database">
                {{ database }}
              </SelectItem>
            </SelectContent>
          </Select>
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
        <SidebarGroupContent class="min-h-0 flex-1 overflow-auto">
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
          <p v-else-if="filtering && visibleDatabases.length === 0 && catalogIncomplete" class="text-muted-foreground px-2 py-2 text-xs">
            Loading tables…
          </p>
          <p v-else-if="filtering && visibleDatabases.length === 0" class="text-muted-foreground px-2 py-2 text-xs">
            No matching tables.
          </p>
          <SidebarMenu v-else>
            <SidebarMenuItem v-for="entry in visibleDatabases" :key="entry.name">
              <SidebarMenuButton
                v-if="entry.showHeader"
                :is-active="isDbOpen(entry.name, true)"
                @click="onToggleDatabase(entry.name)"
              >
                <ChevronDown v-if="isDbOpen(entry.name, true)" class="size-3.5" />
                <ChevronRight v-else class="size-3.5" />
                <Database class="size-3.5" />
                <span>{{ entry.name }}</span>
              </SidebarMenuButton>
              <SidebarMenuSub
                v-if="isDbOpen(entry.name, entry.showHeader)"
                :class="entry.showHeader ? undefined : 'mx-0 border-l-0 translate-x-0 px-0'"
              >
                <SidebarMenuSubItem v-if="explorer.loadingTables[entry.name] && entry.tables.length === 0">
                  <span class="text-muted-foreground px-2 py-1 text-xs">Loading…</span>
                </SidebarMenuSubItem>
                <template v-else-if="tableGroups(entry.tables)">
                  <SidebarMenuSubItem
                    v-for="group in tableGroups(entry.tables)"
                    :key="group.schema"
                  >
                    <div class="text-muted-foreground px-2 py-1 text-[11px] font-medium">
                      {{ group.schema }}
                    </div>
                    <SidebarMenuSub>
                      <SidebarMenuSubItem
                        v-for="table in group.tables"
                        :key="`${table.schema}.${table.name}`"
                      >
                        <ContextMenu>
                          <ContextMenuTrigger as-child>
                            <SidebarMenuSubButton as="button" @click="openTable(entry.name, table)">
                              <Eye v-if="table.type === 'view'" class="size-3.5" />
                              <Table2 v-else class="size-3.5" />
                              <span>{{ table.name }}</span>
                            </SidebarMenuSubButton>
                          </ContextMenuTrigger>
                          <ContextMenuContent>
                            <ContextMenuItem @select="openTable(entry.name, table)">
                              Open table
                            </ContextMenuItem>
                            <ContextMenuItem @select="openTable(entry.name, table, 'structure')">
                              Open structure
                            </ContextMenuItem>
                          </ContextMenuContent>
                        </ContextMenu>
                      </SidebarMenuSubItem>
                    </SidebarMenuSub>
                  </SidebarMenuSubItem>
                </template>
                <template v-else>
                  <SidebarMenuSubItem
                    v-for="table in entry.tables"
                    :key="`${table.schema}.${table.name}`"
                  >
                    <ContextMenu>
                      <ContextMenuTrigger as-child>
                        <SidebarMenuSubButton as="button" @click="openTable(entry.name, table)">
                          <Eye v-if="table.type === 'view'" class="size-3.5" />
                          <Table2 v-else class="size-3.5" />
                          <span>{{ table.name }}</span>
                        </SidebarMenuSubButton>
                      </ContextMenuTrigger>
                      <ContextMenuContent>
                        <ContextMenuItem @select="openTable(entry.name, table)">
                          Open table
                        </ContextMenuItem>
                        <ContextMenuItem @select="openTable(entry.name, table, 'structure')">
                          Open structure
                        </ContextMenuItem>
                      </ContextMenuContent>
                    </ContextMenu>
                  </SidebarMenuSubItem>
                  <SidebarMenuSubItem
                    v-if="entry.tables.length === 0"
                  >
                    <span class="text-muted-foreground px-2 py-1 text-xs">No tables</span>
                  </SidebarMenuSubItem>
                </template>
              </SidebarMenuSub>
            </SidebarMenuItem>
            <SidebarMenuItem v-if="catalogIncomplete && filtering">
              <span class="text-muted-foreground px-2 py-1 text-xs">Loading tables…</span>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroupContent>
      </SidebarGroup>
    </SidebarContent>

    <ConnectionDialog v-model:open="dialogOpen" :profile="editing" />
  </Sidebar>
</template>
