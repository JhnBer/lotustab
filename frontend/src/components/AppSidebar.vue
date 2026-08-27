<script setup lang="ts">
import type { SidebarProps } from "@/components/ui/sidebar";
import {
  ChevronDown,
  ChevronRight,
  Database,
  Eye,
  Pin,
  Plug,
  Table2,
} from "@lucide/vue";
import { toast } from "vue-sonner";
import { storeToRefs } from "pinia";
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
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
  useSidebar,
} from "@/components/ui/sidebar";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { errorMessage } from "@/lib/errors";
import { colorBgClass } from "@/lib/colors";
import type { Profile, TableInfo } from "@/lib/wails";
import { useConnectionsStore } from "@/stores/connections";
import { groupTables, useExplorerStore } from "@/stores/explorer";
import { useQueryStore } from "@/stores/query";

const props = withDefaults(defineProps<SidebarProps>(), {
  collapsible: "icon",
});

const route = useRoute();
const router = useRouter();
const connections = useConnectionsStore();
const explorer = useExplorerStore();
const queries = useQueryStore();
const { setOpen } = useSidebar();
const { selected, pinnedProfiles, recentProfiles } = storeToRefs(connections);
const recentsExpanded = ref(true);

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
  <Sidebar
    class="overflow-hidden *:data-[sidebar=sidebar]:flex-row"
    v-bind="props"
  >
    <Sidebar
      collapsible="none"
      class="w-[calc(var(--sidebar-width-icon)+1px)]! border-r"
    >
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupContent class="px-1.5 md:px-0">
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton
                  tooltip="Connections"
                  :is-active="route.path === '/connections'"
                  class="px-2.5 md:px-2"
                  @click="router.push('/connections'); setOpen(true)"
                >
                  <Database />
                  <span>Connections</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        <template v-if="pinnedProfiles.length > 0">
          <SidebarSeparator />
          <SidebarGroup>
            <SidebarGroupContent class="px-1.5 md:px-0">
              <SidebarMenu>
                <SidebarMenuItem v-for="profile in pinnedProfiles" :key="profile.id">
                  <Tooltip>
                    <TooltipTrigger as-child>
                      <button
                        type="button"
                        class="mx-auto flex size-8 items-center justify-center rounded-md text-white shadow-sm ring-offset-background focus-visible:ring-2 focus-visible:outline-none"
                        :class="[
                          colorBgClass(profile.color),
                          connections.selectedId === profile.id ? 'ring-2 ring-sidebar-ring' : 'opacity-90 hover:opacity-100',
                        ]"
                        @click="activateProfile(profile)"
                      >
                        <Database class="size-4" />
                        <span class="sr-only">{{ profile.name }}</span>
                      </button>
                    </TooltipTrigger>
                    <TooltipContent side="right" align="center">
                      {{ profile.name }}
                    </TooltipContent>
                  </Tooltip>
                </SidebarMenuItem>
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        </template>
      </SidebarContent>
    </Sidebar>

    <Sidebar collapsible="none" class="hidden min-h-0 flex-1 md:flex">
      <SidebarHeader>
        <div v-if="selected" class="flex items-start gap-2 px-1 py-0.5">
          <span
            class="mt-1.5 size-2.5 shrink-0 rounded-full"
            :class="colorBgClass(selected.color)"
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
            </div>
            <div class="text-muted-foreground flex items-center gap-1.5 text-xs">
              <span
                class="size-1.5 rounded-full"
                :class="currentConnected ? 'bg-emerald-500' : 'bg-muted-foreground/40'"
              />
              {{ currentConnected ? "Connected" : "Offline" }}
            </div>
          </div>
        </div>
        <div v-else class="text-muted-foreground px-2 py-1 text-xs">
          Select a connection to browse tables.
        </div>
      </SidebarHeader>

      <SidebarSeparator />

      <SidebarContent>
        <SidebarGroup v-if="selected && !currentConnected">
          <SidebarGroupContent class="px-2 py-2">
            <Button size="sm" class="w-full" @click="connectSelected">
              <Plug class="size-3.5" />
              Connect
            </Button>
          </SidebarGroupContent>
        </SidebarGroup>

        <SidebarGroup v-else-if="selected && explorer.loadingDbs">
          <SidebarGroupContent>
            <p class="text-muted-foreground px-2 py-2 text-xs">Loading databases…</p>
          </SidebarGroupContent>
        </SidebarGroup>

        <SidebarGroup v-else-if="selected && explorer.error">
          <SidebarGroupContent>
            <p class="text-destructive px-2 py-2 text-xs">{{ explorer.error }}</p>
          </SidebarGroupContent>
        </SidebarGroup>

        <SidebarGroup v-else-if="selected && explorer.databases.length === 0">
          <SidebarGroupContent>
            <p class="text-muted-foreground px-2 py-2 text-xs">No databases.</p>
          </SidebarGroupContent>
        </SidebarGroup>

        <SidebarGroup v-else-if="selected">
          <SidebarGroupLabel>Databases</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
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

      <SidebarFooter v-if="recentProfiles.length > 0" class="px-0">
        <SidebarSeparator class="mx-0" />
        <SidebarContent>
            <button
                type="button"
                class="text-muted-foreground hover:text-foreground flex w-full items-center justify-between px-2 py-1 text-xs"
                @click="recentsExpanded = !recentsExpanded"
            >
                <span>Recent</span>
                <ChevronDown
                    class="size-3.5 transition-transform"
                    :class="recentsExpanded ? '' : '-rotate-90'"
                />
            </button>
            <SidebarMenu v-if="recentsExpanded">
                <SidebarMenuItem v-for="profile in recentProfiles" :key="profile.id">
                    <SidebarMenuButton :tooltip="profile.name" @click="activateProfile(profile)">
              <span
                  class="size-2 shrink-0 rounded-full"
                  :class="colorBgClass(profile.color)"
              />
                        <span>{{ profile.name }}</span>
                    </SidebarMenuButton>
                </SidebarMenuItem>
            </SidebarMenu>
        </SidebarContent>
      </SidebarFooter>
    </Sidebar>
  </Sidebar>
</template>
