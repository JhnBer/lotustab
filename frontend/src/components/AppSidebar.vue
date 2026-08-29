<script setup lang="ts">
import {
  ChevronDown,
  ChevronRight,
  Database,
  Eye,
  Pin,
  Plug,
  RefreshCw,
  SquareArrowOutUpRight,
  Table2,
} from "@lucide/vue";
import { toast } from "vue-sonner";
import { storeToRefs } from "pinia";
import { computed, onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { Window } from "@wailsio/runtime";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupAction,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  SidebarSeparator,
} from "@/components/ui/sidebar";
import { Button } from "@/components/ui/button";
import { errorMessage } from "@/lib/errors";
import { connBg } from "@/lib/colors";
import type { Profile, TableInfo } from "@/lib/wails";
import { useConnectionsStore } from "@/stores/connections";
import { groupTables, useExplorerStore } from "@/stores/explorer";
import { useQueryStore } from "@/stores/query";

const router = useRouter();
const connections = useConnectionsStore();
const explorer = useExplorerStore();
const queries = useQueryStore();
const { selected, recentProfiles } = storeToRefs(connections);
const recentsExpanded = ref(true);
const isMainWindow = ref(true);

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

watch(
  () => [selected.value?.id, currentConnected.value] as const,
  async ([id, connected]) => {
    if (!id || !connected) {
      explorer.reset();
      return;
    }
    try {
      await explorer.loadDatabases(id);
    } catch (err) {
      toast.error(errorMessage(err));
    }
  },
  { immediate: true },
);

function activateProfile(profile: Profile) {
  connections.select(profile.id);
  router.push("/query");
}

async function connectSelected() {
  if (!selected.value) {
    return;
  }
  try {
    await connections.connect(selected.value.id);
    toast.success(`Connected to ${selected.value.name}`);
    router.push("/query");
  } catch (err) {
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

const explorerRefreshing = ref(false);

async function refreshExplorer() {
  if (!selected.value || explorerRefreshing.value) {
    return;
  }
  explorerRefreshing.value = true;
  try {
    await explorer.refresh(selected.value.id);
  } catch (err) {
    toast.error(errorMessage(err));
  } finally {
    explorerRefreshing.value = false;
  }
}

function openTable(database: string, table: TableInfo) {
  if (!selected.value) {
    return;
  }
  queries.openTable({
    profileId: selected.value.id,
    database,
    schema: table.schema,
    name: table.name,
  });
  router.push("/query");
}

function grouped(database: string) {
  const tables = explorer.tablesByDb[database] ?? [];
  return groupTables(tables);
}
</script>

<template>
  <Sidebar collapsible="none" class="bg-sidebar h-full min-h-0 w-full! overflow-hidden">
    <SidebarHeader>
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
          <div v-else class="text-muted-foreground px-2 py-1 text-xs">
              Select a connection to browse tables.
          </div>
      </div>
    </SidebarHeader>

    <SidebarSeparator class="m-0 -translate-y-full" />

    <SidebarContent class="overflow-hidden">
      <SidebarGroup v-if="selected && !currentConnected">
        <SidebarGroupContent class="px-2 py-2">
          <Button size="sm" class="w-full" @click="connectSelected">
            <Plug class="size-3.5" />
            Connect
          </Button>
        </SidebarGroupContent>
      </SidebarGroup>

      <SidebarGroup v-else-if="selected" class="min-h-0 flex-1 overflow-hidden">
        <SidebarGroupLabel class="bg-sidebar shrink-0">Databases</SidebarGroupLabel>
        <SidebarGroupAction
          as="button"
          title="Reload tables"
          :disabled="explorerRefreshing"
          @click="refreshExplorer"
        >
          <RefreshCw :class="explorerRefreshing ? 'animate-spin' : undefined" />
        </SidebarGroupAction>
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
          <SidebarMenu v-else>
            <SidebarMenuItem v-for="database in explorer.databases" :key="database">
              <SidebarMenuButton
                :is-active="explorer.expanded[database]"
                @click="onToggleDatabase(database)"
              >
                <ChevronDown v-if="explorer.expanded[database]" class="size-3.5" />
                <ChevronRight v-else class="size-3.5" />
                <Database class="size-3.5" />
                <span>{{ database }}</span>
              </SidebarMenuButton>
              <SidebarMenuSub v-if="explorer.expanded[database]">
                <SidebarMenuSubItem v-if="explorer.loadingTables[database]">
                  <span class="text-muted-foreground px-2 py-1 text-xs">Loading…</span>
                </SidebarMenuSubItem>
                <template v-else-if="grouped(database)">
                  <SidebarMenuSubItem
                    v-for="group in grouped(database)"
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
                        <SidebarMenuSubButton as="button" @click="openTable(database, table)">
                          <Eye v-if="table.type === 'view'" class="size-3.5" />
                          <Table2 v-else class="size-3.5" />
                          <span>{{ table.name }}</span>
                        </SidebarMenuSubButton>
                      </SidebarMenuSubItem>
                    </SidebarMenuSub>
                  </SidebarMenuSubItem>
                </template>
                <template v-else>
                  <SidebarMenuSubItem
                    v-for="table in explorer.tablesByDb[database] ?? []"
                    :key="`${table.schema}.${table.name}`"
                  >
                    <SidebarMenuSubButton as="button" @click="openTable(database, table)">
                      <Eye v-if="table.type === 'view'" class="size-3.5" />
                      <Table2 v-else class="size-3.5" />
                      <span>{{ table.name }}</span>
                    </SidebarMenuSubButton>
                  </SidebarMenuSubItem>
                  <SidebarMenuSubItem
                    v-if="(explorer.tablesByDb[database] ?? []).length === 0"
                  >
                    <span class="text-muted-foreground px-2 py-1 text-xs">No tables</span>
                  </SidebarMenuSubItem>
                </template>
              </SidebarMenuSub>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroupContent>
      </SidebarGroup>
    </SidebarContent>

    <SidebarFooter v-if="recentProfiles.length > 0" class="gap-0 p-0">
      <SidebarSeparator class="mx-0" />
      <button
        type="button"
        class="text-muted-foreground hover:text-foreground flex w-full items-center justify-between px-2 py-3 text-xs"
        @click="recentsExpanded = !recentsExpanded"
      >
        <span>Recent</span>
        <ChevronDown
          class="size-3.5 transition-transform duration-100 ease-linear"
          :class="recentsExpanded ? '' : '-rotate-90'"
        />
      </button>
      <div
        class="grid overflow-hidden transition-[grid-template-rows] duration-100 ease-linear"
        :class="recentsExpanded ? 'grid-rows-[1fr]' : 'grid-rows-[0fr]'"
      >
        <div class="min-h-0 overflow-hidden" :class="recentsExpanded ? '' : 'pointer-events-none'">
          <SidebarMenu class="pb-1">
            <SidebarMenuItem v-for="profile in recentProfiles" :key="profile.id">
              <SidebarMenuButton :tooltip="profile.name" @click="activateProfile(profile)">
                <span
                  class="size-2 shrink-0 rounded-full"
                  :class="connBg(profile.color)"
                />
                <span>{{ profile.name }}</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </div>
      </div>
    </SidebarFooter>
  </Sidebar>
</template>
