<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { FileCode2, Play, Plus, Table2, X } from "@lucide/vue";
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
import { errorMessage } from "@/lib/errors";
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from "@/components/ui/resizable";
import SqlEditor from "@/features/editor/SqlEditor.vue";
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

const visibleTabs = computed(() =>
  queries.tabs.filter((t) => t.profileId === connections.selectedId),
);

watch(
  () => connections.selectedId,
  (id) => queries.activateProfile(id),
  { immediate: true },
);

const fromIndex = ref(-1);

function onDragStart(event: DragEvent, index: number) {
  fromIndex.value = index;
  event.dataTransfer?.setData("text/plain", String(index));
}

function onDragOver(event: DragEvent) {
  event.preventDefault();
}

function onDrop(index: number) {
  if (fromIndex.value < 0) {
    return;
  }
  queries.reorder(connections.selectedId, fromIndex.value, index);
  fromIndex.value = -1;
}

function addTab() {
  const profile = connections.selected;
  if (!profile) {
    return;
  }
  queries.add(profile.id, profile.name);
}

async function run() {
  const tab = queries.activeTab;
  if (!tab || tab.kind !== "query") {
    return;
  }
  try {
    if (!connections.isConnected(tab.profileId)) {
      await connections.connect(tab.profileId);
    }
    await queries.run(tab.id);
  } catch (err) {
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
    <div class="flex items-center gap-1 border-b px-2 py-1 min-h-10">
      <div class="flex min-w-0 flex-1 items-center gap-1 overflow-x-auto" style="--wails-draggable: no-drag">
        <div
          v-for="(tab, index) in visibleTabs"
          :key="tab.id"
          draggable="true"
          class="flex max-w-[12rem] cursor-grab items-center gap-1 rounded-md px-2 py-1 text-xs active:cursor-grabbing"
          :class="[
            tab.id === queries.activeTabId ? 'bg-muted text-foreground' : 'text-muted-foreground hover:bg-muted/60',
            fromIndex === index ? 'opacity-50' : '',
          ]"
          @click="queries.selectTab(tab.id)"
          @dragstart="onDragStart($event, index)"
          @dragover="onDragOver"
          @drop="onDrop(index)"
          @dragend="fromIndex = -1"
        >
          <Table2 v-if="tab.kind === 'table'" class="size-3 shrink-0" />
          <FileCode2 v-else class="size-3 shrink-0" />
          <span class="truncate">{{ tab.title }}</span>
          <span
            class="hover:bg-background rounded-sm p-0.5"
            @click.stop="queries.close(tab.id)"
            @mousedown.stop
          >
            <X class="size-3" />
          </span>
        </div>
        <Button size="icon-xs" variant="ghost" :disabled="!connections.selected" @click="addTab">
          <Plus class="size-3.5" />
        </Button>
      </div>
    </div>

    <div v-if="!queries.activeTab" class="text-muted-foreground flex flex-1 flex-col items-center justify-center gap-2 p-8 text-sm">
      <span>Select a connection to start querying.</span>
      <RouterLink to="/connections" class="text-foreground hover:underline">
        Open connections
      </RouterLink>
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
