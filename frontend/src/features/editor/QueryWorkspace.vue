<script setup lang="ts">
import { computed, watch } from "vue";
import { Play, Plug } from "@lucide/vue";
import { toast } from "vue-sonner";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import DatabaseCombobox from "@/components/DatabaseCombobox.vue";
import { errorMessage, isCanceled } from "@/lib/errors";
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from "@/components/ui/resizable";
import SqlEditor from "@/features/editor/SqlEditor.vue";
import WorkspaceTabBar from "@/features/editor/WorkspaceTabBar.vue";
import ResultsTable from "@/features/results/ResultsTable.vue";
import TableBrowser from "@/features/results/TableBrowser.vue";
import TablePager from "@/features/results/TablePager.vue";
import { useConnectionsStore } from "@/stores/connections";
import { useExplorerStore } from "@/stores/explorer";
import { isQueryTab, isTableTab, useQueryStore, type QueryTab, type TableTab } from "@/stores/query";

const NONE = "__none__";

const connections = useConnectionsStore();
const queries = useQueryStore();
const explorer = useExplorerStore();

const connecting = computed(() => {
  const id = connections.selectedId;
  return Boolean(id && connections.isConnecting(id));
});

const selectedOffline = computed(() => {
  const id = connections.selectedId;
  return Boolean(id && !connections.isConnected(id) && !connections.isConnecting(id));
});

const profileMountedIds = computed(() => new Set(queries.mountedTabIds(connections.selectedId ?? "")));
const profileTableTabs = computed(() =>
  queries.tabs.filter(
    (t): t is TableTab =>
      isTableTab(t) && t.profileId === connections.selectedId && profileMountedIds.value.has(t.id),
  ),
);
const profileQueryTabs = computed(() =>
  queries.tabs.filter(
    (t): t is QueryTab =>
      isQueryTab(t) && t.profileId === connections.selectedId && profileMountedIds.value.has(t.id),
  ),
);

watch(
  () => connections.selectedId,
  (id) => queries.activateProfile(id),
  { immediate: true },
);

async function run() {
  const tab = queries.activeTab;
  if (!tab || tab.kind !== "query") {
    return;
  }
  try {
    if (!connections.isConnected(tab.profileId)) {
      await connections.connect(tab.profileId);
      if (!connections.isConnected(tab.profileId)) {
        return;
      }
    }
    await queries.run(tab.id);
  } catch (err) {
    if (isCanceled(err)) {
      return;
    }
    toast.error(errorMessage(err));
  }
}

async function connectSelected() {
  const id = connections.selectedId;
  if (!id) {
    return;
  }
  try {
    await connections.connect(id);
    if (!connections.isConnected(id)) {
      return;
    }
    const profile = connections.selected;
    if (profile) {
      toast.success(`Connected to ${profile.name}`);
    }
  } catch (err) {
    if (isCanceled(err)) {
      return;
    }
    toast.error(errorMessage(err));
  }
}

function onDatabase(value: string) {
  const tab = queries.activeTab;
  if (!isQueryTab(tab)) {
    return;
  }
  queries.setDatabase(tab.id, value === NONE ? "" : value);
}
</script>

<template>
  <section class="flex h-full min-h-0 flex-1 flex-col -m-4 overflow-hidden">
    <WorkspaceTabBar />

    <div
      v-if="connecting"
      class="text-muted-foreground flex flex-1 flex-col items-center justify-center gap-2 p-8 text-sm"
    >
      <Spinner />
      <span>Connecting…</span>
    </div>

    <div
      v-else-if="!connections.selectedId"
      class="text-muted-foreground flex flex-1 flex-col items-center justify-center gap-2 p-8 text-sm"
    >
      <span>Select a connection to start querying.</span>
      <RouterLink to="/connections" class="text-foreground hover:underline">
        Open connections
      </RouterLink>
    </div>

    <div
      v-else-if="selectedOffline && !queries.activeTab"
      class="text-muted-foreground flex flex-1 flex-col items-center justify-center gap-3 p-8 text-sm"
    >
      <span>Connect this database to start querying.</span>
      <Button size="sm" @click="connectSelected">
        <Plug class="size-3.5" />
        Connect
      </Button>
    </div>

    <template v-else>
      <div class="relative min-h-0 flex-1">
        <TableBrowser
          v-for="tab in profileTableTabs"
          :key="tab.id"
          class="absolute inset-0"
          :class="tab.id === queries.activeTabId ? 'z-10' : 'invisible pointer-events-none'"
          :tab="tab"
          :active="tab.id === queries.activeTabId"
        />
        <ResizablePanelGroup
          v-for="tab in profileQueryTabs"
          :key="tab.id"
          direction="vertical"
          class="absolute inset-0"
          :class="tab.id === queries.activeTabId ? 'z-10 bg-background' : 'invisible pointer-events-none'"
        >
      <ResizablePanel :default-size="45" :min-size="20">
        <div class="flex h-full min-h-0 flex-col">
          <div class="flex items-center gap-2 border-b px-3 py-2 text-xs" style="--wails-draggable: no-drag">
            <DatabaseCombobox
              class="h-7 w-44"
              :model-value="tab.database || NONE"
              :options="explorer.databases"
              :none-value="NONE"
              none-label="None"
              placeholder="Database"
              @update:model-value="onDatabase"
            />
            <span class="text-muted-foreground min-w-0 flex-1 truncate">
              You can choose a database here or specify it in the query.
            </span>
            <Button
              v-if="tab.running"
              size="sm"
              variant="outline"
              class="shrink-0"
              @click="queries.cancel(tab.id)"
            >
              Cancel
            </Button>
            <Button size="sm" class="shrink-0" :disabled="tab.running" @click="run">
              <Spinner v-if="tab.running" class="size-3.5" />
              <Play v-else class="size-3.5" />
              Run
            </Button>
          </div>
          <SqlEditor
            class="min-h-0 flex-1"
            :model-value="tab.sql"
            @update:model-value="queries.setSql(tab.id, $event)"
            @run="run"
          />
        </div>
      </ResizablePanel>
      <ResizableHandle />
      <ResizablePanel :default-size="55" :min-size="20">
        <div class="flex h-full min-h-0 flex-col">
          <ResultsTable
            class="min-h-0 flex-1"
            :result="tab.result"
            :error="tab.error"
            :running="tab.running"
            :show-status="false"
            :row-offset="(tab.page - 1) * tab.pageSize"
          />
          <TablePager
            :page="tab.page"
            :page-size="tab.pageSize"
            :row-count="tab.result?.rows?.length ?? 0"
            :running="tab.running"
            :truncated="tab.result?.truncated"
            :duration-ms="tab.result?.durationMs"
            @page="queries.setPage(tab.id, $event)"
            @page-size="queries.setPageSize(tab.id, $event)"
          />
        </div>
      </ResizablePanel>
    </ResizablePanelGroup>
        <div
          v-if="queries.coldMountId && queries.coldMountId === queries.activeTabId"
          class="absolute inset-0 z-10 flex items-center justify-center bg-background/60"
        >
          <Spinner />
        </div>
      </div>
    </template>
  </section>
</template>
