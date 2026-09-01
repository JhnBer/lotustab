<script setup lang="ts">
import { computed, watch } from "vue";
import { Play, Plug } from "@lucide/vue";
import { toast } from "vue-sonner";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
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
import { isQueryTab, isTableTab, useQueryStore } from "@/stores/query";

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

function onDatabase(value: unknown) {
  const tab = queries.activeTab;
  if (!isQueryTab(tab)) {
    return;
  }
  const name = String(value ?? "");
  queries.setDatabase(tab.id, name === NONE ? "" : name);
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

    <TableBrowser
      v-else-if="isTableTab(queries.activeTab)"
      :key="queries.activeTab.id"
      :tab="queries.activeTab"
    />

    <ResizablePanelGroup v-else-if="isQueryTab(queries.activeTab)" direction="vertical" class="min-h-0 flex-1">
      <ResizablePanel :default-size="45" :min-size="20">
        <div class="flex h-full min-h-0 flex-col">
          <div class="flex items-center gap-2 border-b px-3 py-2 text-xs" style="--wails-draggable: no-drag">
            <Select
              :model-value="queries.activeTab.database || undefined"
              @update:model-value="onDatabase"
            >
              <SelectTrigger size="sm" class="h-7 w-44">
                <SelectValue placeholder="Database" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem :value="NONE">None</SelectItem>
                <SelectItem v-for="database in explorer.databases" :key="database" :value="database">
                  {{ database }}
                </SelectItem>
              </SelectContent>
            </Select>
            <span class="text-muted-foreground min-w-0 flex-1 truncate">
              You can choose a database here or specify it in the query.
            </span>
            <Button
              v-if="queries.activeTab.running"
              size="sm"
              variant="outline"
              class="shrink-0"
              @click="queries.cancel(queries.activeTab.id)"
            >
              Cancel
            </Button>
            <Button size="sm" class="shrink-0" :disabled="queries.activeTab.running" @click="run">
              <Spinner v-if="queries.activeTab.running" class="size-3.5" />
              <Play v-else class="size-3.5" />
              Run
            </Button>
          </div>
          <SqlEditor
            :key="queries.activeTab.id"
            class="min-h-0 flex-1"
            :model-value="queries.activeTab.sql"
            @update:model-value="queries.setSql(queries.activeTab!.id, $event)"
            @run="run"
          />
        </div>
      </ResizablePanel>
      <ResizableHandle />
      <ResizablePanel :default-size="55" :min-size="20">
        <div class="flex h-full min-h-0 flex-col">
          <ResultsTable
            class="min-h-0 flex-1"
            :result="queries.activeTab.result"
            :error="queries.activeTab.error"
            :running="queries.activeTab.running"
            :show-status="false"
            :row-offset="(queries.activeTab.page - 1) * queries.activeTab.pageSize"
          />
          <TablePager
            :page="queries.activeTab.page"
            :page-size="queries.activeTab.pageSize"
            :row-count="queries.activeTab.result?.rows?.length ?? 0"
            :running="queries.activeTab.running"
            :truncated="queries.activeTab.result?.truncated"
            :duration-ms="queries.activeTab.result?.durationMs"
            @page="queries.setPage(queries.activeTab!.id, $event)"
            @page-size="queries.setPageSize(queries.activeTab!.id, $event)"
          />
        </div>
      </ResizablePanel>
    </ResizablePanelGroup>
  </section>
</template>
