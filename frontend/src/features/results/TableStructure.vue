<script setup lang="ts">
import { computed, ref } from "vue";
import { useVirtualizer } from "@tanstack/vue-virtual";
import { Plus, RotateCw } from "@lucide/vue";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import {
  filterDraftColumns,
  isStructureDirty,
  type DraftColumn,
} from "@/features/results/structure";
import TableStructureRow from "@/features/results/TableStructureRow.vue";
import { useConnectionsStore } from "@/stores/connections";
import { type TableTab, useQueryStore } from "@/stores/query";

const ROW_HEIGHT = 36;

const props = defineProps<{
  tab: TableTab;
  running?: boolean;
}>();

const queries = useQueryStore();
const connections = useConnectionsStore();
const query = ref("");
const confirmRefresh = ref(false);
const scrollRef = ref<HTMLElement | null>(null);
const dragKey = ref("");

const driverId = computed(() => connections.profileById(props.tab.profileId)?.driverId ?? "");

const draft = computed(() => props.tab.structureDraft ?? []);
const visible = computed(() => filterDraftColumns(draft.value, query.value));
const searching = computed(() => query.value.trim().length > 0);
const canReorder = computed(() => Boolean(props.tab.structure?.canReorder) && !searching.value);
const emptyHint = computed(() =>
  props.tab.structureError
    ? props.tab.structureError
    : searching.value
      ? "No columns match this search."
      : "This table has no columns.",
);

const dirty = computed(() =>
  isStructureDirty(props.tab.structure?.columns, props.tab.structureDraft, Boolean(props.tab.structure?.canReorder)),
);

const rowVirtualizer = useVirtualizer(
  computed(() => ({
    count: visible.value.length,
    getScrollElement: () => scrollRef.value,
    estimateSize: () => ROW_HEIGHT,
    overscan: 8,
  })),
);

const virtualRows = computed(() => rowVirtualizer.value.getVirtualItems());
const totalSize = computed(() => rowVirtualizer.value.getTotalSize());

function onRefresh() {
  if (dirty.value) {
    confirmRefresh.value = true;
    return;
  }
  void queries.loadStructure(props.tab.id);
}

function confirmAndRefresh() {
  confirmRefresh.value = false;
  void queries.loadStructure(props.tab.id, { force: true });
}

function onReorderStart(event: DragEvent, key: string) {
  dragKey.value = key;
  event.dataTransfer?.setData("text/plain", key);
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = "move";
  }
}

function onReorderEnd() {
  dragKey.value = "";
}

function onReorderDrop(event: DragEvent, targetKey: string) {
  const sourceKey = dragKey.value || event.dataTransfer?.getData("text/plain") || "";
  dragKey.value = "";
  if (!sourceKey || sourceKey === targetKey) {
    return;
  }
  const list = [...draft.value];
  const from = list.findIndex((col) => col.key === sourceKey);
  const to = list.findIndex((col) => col.key === targetKey);
  if (from < 0 || to < 0) {
    return;
  }
  const [item] = list.splice(from, 1);
  list.splice(to, 0, item!);
  queries.moveStructureColumns(props.tab.id, list);
}

function rowAt(index: number): DraftColumn | undefined {
  return visible.value[index];
}
</script>

<template>
  <div class="flex min-h-0 flex-1 flex-col">
    <div class="flex items-center gap-2 border-b px-3 py-2">
      <Input
        v-model="query"
        class="h-7 max-w-64"
        placeholder="Search columns…"
      />
      <span class="text-muted-foreground min-w-0 flex-1 truncate text-xs">
        <template v-if="!tab.structure?.canReorder">Column order cannot be changed on this database.</template>
      </span>
      <Button
        size="icon-xs"
        variant="outline"
        :disabled="running"
        aria-label="Refresh structure"
        @click="onRefresh"
      >
        <RotateCw class="size-3.5" :class="running ? 'animate-spin' : ''" />
      </Button>
      <Button size="xs" variant="outline" :disabled="running" @click="queries.addStructureColumn(tab.id, driverId)">
        <Plus class="size-3.5" />
        Column
      </Button>
    </div>

    <div v-if="running && !tab.structureDraft?.length" class="text-muted-foreground flex flex-1 items-center justify-center p-4">
      <Spinner />
    </div>
    <div v-else-if="tab.structureError && !tab.structureDraft?.length" class="text-destructive flex flex-1 items-center justify-center p-6 text-sm">
      {{ tab.structureError }}
    </div>
    <div v-else class="relative min-h-0 flex-1">
      <div ref="scrollRef" class="absolute inset-0 overflow-auto">
        <div class="min-w-208 text-sm">
          <div class="bg-background sticky top-0 z-10 flex border-b" role="row">
            <div class="h-9 w-8 shrink-0" />
            <div class="text-foreground flex h-9 min-w-40 flex-1 items-center px-2 font-medium">Name</div>
            <div class="text-foreground flex h-9 w-48 shrink-0 items-center px-2 font-medium">Type</div>
            <div v-if="driverId !== 'clickhouse'" class="text-foreground flex h-9 w-20 shrink-0 items-center justify-center px-2 font-medium">Null</div>
            <div class="text-foreground flex h-9 min-w-36 flex-1 items-center px-2 font-medium">Default</div>
            <div class="text-foreground flex h-9 w-20 shrink-0 items-center justify-center px-2 font-medium">Primary</div>
            <div class="text-foreground flex h-9 w-56 min-w-40 shrink-0 items-center px-2 font-medium">Extra</div>
            <div class="text-foreground flex h-9 min-w-36 flex-1 items-center px-2 font-medium">Comment</div>
            <div class="h-9 w-10 shrink-0" />
          </div>
          <div v-if="visible.length === 0" class="text-muted-foreground px-3 py-8 text-center text-sm">
            {{ emptyHint }}
          </div>
          <div
            v-else
            class="relative"
            :style="{ height: `${totalSize}px` }"
          >
            <TableStructureRow
              v-for="item in virtualRows"
              :key="rowAt(item.index)?.key"
              class="absolute top-0 left-0 w-full"
              :style="{ height: `${item.size}px`, transform: `translateY(${item.start}px)` }"
              :tab-id="tab.id"
              :col="rowAt(item.index)!"
              :columns="tab.structure?.columns"
              :running="running"
              :can-drag="canReorder"
              :driver-id="driverId"
              @reorderstart="onReorderStart($event, rowAt(item.index)!.key)"
              @reorderend="onReorderEnd"
              @reorderdrop="onReorderDrop($event, rowAt(item.index)!.key)"
            />
          </div>
        </div>
      </div>
    </div>

    <Dialog :open="confirmRefresh" @update:open="confirmRefresh = $event">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Discard structure changes?</DialogTitle>
          <DialogDescription>
            Refreshing will drop unsaved column changes.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" @click="confirmRefresh = false">Cancel</Button>
          <Button @click="confirmAndRefresh">Refresh</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
