<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { FileCode2, Play, Plus, Table2, X } from "@lucide/vue";
import { toast } from "vue-sonner";
import { Button } from "@/components/ui/button";
import { errorMessage } from "@/lib/errors";
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from "@/components/ui/resizable";
import SqlEditor from "@/features/editor/SqlEditor.vue";
import ResultsTable from "@/features/results/ResultsTable.vue";
import TableBrowser from "@/features/results/TableBrowser.vue";
import { useConnectionsStore } from "@/stores/connections";
import { isQueryTab, isTableTab, useQueryStore } from "@/stores/query";

const connections = useConnectionsStore();
const queries = useQueryStore();

const visibleTabs = computed(() =>
  queries.tabs.filter((t) => t.profileId === connections.selectedId),
);

watch(
  () => connections.selectedId,
  (id) => {
    if (queries.activeTab?.profileId === id) {
      return;
    }
    queries.activeTabId = queries.tabs.find((t) => t.profileId === id)?.id ?? "";
  },
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
          @click="queries.activeTabId = tab.id"
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
      <div v-if="isQueryTab(queries.activeTab)" style="--wails-draggable: no-drag">
        <Button size="sm" :disabled="queries.activeTab.running" @click="run">
          <Play class="size-3.5" />
          Run
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
        <SqlEditor
          :key="queries.activeTab.id"
          :model-value="queries.activeTab.sql"
          @update:model-value="queries.setSql(queries.activeTab!.id, $event)"
          @run="run"
        />
      </ResizablePanel>
      <ResizableHandle />
      <ResizablePanel :default-size="55" :min-size="20">
        <ResultsTable
          :result="queries.activeTab.result"
          :error="queries.activeTab.error"
          :running="queries.activeTab.running"
        />
      </ResizablePanel>
    </ResizablePanelGroup>
  </section>
</template>
