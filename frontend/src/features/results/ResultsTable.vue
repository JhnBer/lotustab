<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { useVirtualizer } from "@tanstack/vue-virtual";
import { ArrowDown, ArrowUp, ArrowUpDown } from "@lucide/vue";
import type { QueryResult } from "@/lib/wails";

const DEFAULT_COL_WIDTH = 160;
const MIN_COL_WIDTH = 72;
const ROW_HEIGHT = 36;
const ROW_NUM_WIDTH = 56;

const props = withDefaults(
  defineProps<{
    result: QueryResult | null;
    error: string;
    running: boolean;
    sortable?: boolean;
    sortColumn?: string;
    sortDir?: string;
    emptyHint?: string;
    showStatus?: boolean;
    rowOffset?: number;
  }>(),
  {
    sortable: false,
    sortColumn: "",
    sortDir: "",
    emptyHint: "Run a query to see results.",
    showStatus: true,
    rowOffset: 0,
  },
);

const emit = defineEmits<{
  sort: [column: string];
}>();

const columns = computed(() => props.result?.columns ?? []);
const rows = computed(() => props.result?.rows ?? []);
const widths = ref<Record<string, number>>({});
const scrollRef = ref<HTMLElement | null>(null);

watch(
  columns,
  (cols) => {
    const next: Record<string, number> = {};
    for (const column of cols) {
      next[column.name] = widths.value[column.name] ?? DEFAULT_COL_WIDTH;
    }
    widths.value = next;
  },
  { immediate: true },
);

function colWidth(name: string) {
  return widths.value[name] ?? DEFAULT_COL_WIDTH;
}

const rowVirtualizer = useVirtualizer(
  computed(() => ({
    count: rows.value.length,
    getScrollElement: () => scrollRef.value,
    estimateSize: () => ROW_HEIGHT,
    overscan: 10,
  })),
);

const columnVirtualizer = useVirtualizer(
  computed(() => ({
    count: columns.value.length,
    getScrollElement: () => scrollRef.value,
    estimateSize: (index: number) => colWidth(columns.value[index]?.name ?? ""),
    horizontal: true,
    paddingStart: ROW_NUM_WIDTH,
    overscan: 3,
  })),
);

const rowNumOffset = ref(props.rowOffset);

watch(
  () => props.result,
  () => {
    rowNumOffset.value = props.rowOffset;
  },
);

watch(
  () => props.result,
  () => {
    rowVirtualizer.value.scrollToOffset(0);
    columnVirtualizer.value.scrollToOffset(0);
    scrollRef.value?.scrollTo({ top: 0, left: 0 });
  },
  { flush: "post" },
);

watch(widths, () => {
  columnVirtualizer.value.measure();
});

const virtualRows = computed(() => rowVirtualizer.value.getVirtualItems());
const totalSize = computed(() => rowVirtualizer.value.getTotalSize());
const virtualColumns = computed(() => columnVirtualizer.value.getVirtualItems());
const colTotalSize = computed(() => columnVirtualizer.value.getTotalSize());
const padLeft = computed(() => {
  const first = virtualColumns.value[0];
  return first ? Math.max(0, first.start - ROW_NUM_WIDTH) : 0;
});
const padRight = computed(() => {
  const last = virtualColumns.value.at(-1);
  return last ? colTotalSize.value - last.end : 0;
});

let drag: { name: string; startX: number; startWidth: number } | null = null;

function onResizeMove(event: PointerEvent) {
  if (!drag) {
    return;
  }
  const next = Math.max(MIN_COL_WIDTH, drag.startWidth + (event.clientX - drag.startX));
  widths.value = { ...widths.value, [drag.name]: next };
}

function onResizeUp() {
  drag = null;
  document.body.style.cursor = "";
  document.body.style.userSelect = "";
  window.removeEventListener("pointermove", onResizeMove);
  window.removeEventListener("pointerup", onResizeUp);
}

function startResize(event: PointerEvent, name: string) {
  event.preventDefault();
  event.stopPropagation();
  drag = {
    name,
    startX: event.clientX,
    startWidth: colWidth(name),
  };
  document.body.style.cursor = "col-resize";
  document.body.style.userSelect = "none";
  window.addEventListener("pointermove", onResizeMove);
  window.addEventListener("pointerup", onResizeUp);
}

onBeforeUnmount(() => {
  onResizeUp();
});

function cellText(value: unknown): string {
  if (value === null || value === undefined) {
    return "NULL";
  }
  if (typeof value === "object") {
    return JSON.stringify(value);
  }
  return String(value);
}

function isNull(value: unknown) {
  return value === null || value === undefined;
}

function onHeaderClick(name: string) {
  if (props.sortable) {
    emit("sort", name);
  }
}

function cellValue(rowIndex: number, colIndex: number) {
  return rows.value[rowIndex]?.[colIndex];
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col">
    <div v-if="running && !result" class="text-muted-foreground p-4 text-sm">
      Running…
    </div>
    <div v-else-if="error" class="text-destructive overflow-auto p-4 font-mono text-sm whitespace-pre-wrap">
      {{ error }}
    </div>
    <div v-else-if="!result" class="text-muted-foreground p-4 text-sm">
      {{ emptyHint }}
    </div>
    <template v-else>
      <div v-if="showStatus" class="text-muted-foreground flex items-center justify-between border-b px-3 py-2 text-xs">
        <span>
          {{ rows.length }} row{{ rows.length === 1 ? "" : "s" }}
          <span v-if="result.truncated"> (truncated)</span>
        </span>
        <span>{{ result.durationMs }} ms · {{ result.rowsAffected }} affected</span>
      </div>
      <div v-if="columns.length === 0" class="text-muted-foreground p-4 text-sm">
        Statement completed. {{ result.rowsAffected }} row{{ result.rowsAffected === 1 ? "" : "s" }} affected.
      </div>
      <div v-else class="relative min-h-0 flex-1">
        <div ref="scrollRef" class="absolute inset-0 overflow-auto">
          <div
            role="table"
            class="text-sm"
            :style="{ width: `${colTotalSize}px` }"
          >
            <div
              role="row"
              class="bg-background sticky top-0 z-30 flex border-b"
            >
              <div
                role="columnheader"
                class="bg-background text-muted-foreground sticky left-0 z-40 box-border flex flex-none items-center justify-center border-r px-1 py-2 font-medium"
                :style="{ width: `${ROW_NUM_WIDTH}px` }"
              >
                #
              </div>
              <div
                v-if="padLeft"
                aria-hidden="true"
                class="flex-none"
                :style="{ width: `${padLeft}px` }"
              />
              <div
                v-for="col in virtualColumns"
                :key="col.index"
                role="columnheader"
                class="relative box-border flex-none overflow-hidden px-2 py-2"
                :class="sortable ? 'cursor-pointer select-none hover:bg-muted/60' : ''"
                :style="{ width: `${col.size}px` }"
                @click="onHeaderClick(columns[col.index]?.name ?? '')"
              >
                <div class="flex items-center gap-1 pr-2">
                  <span class="truncate font-medium">{{ columns[col.index]?.name }}</span>
                  <ArrowUp
                    v-if="sortable && sortColumn === columns[col.index]?.name && sortDir === 'asc'"
                    class="size-3 shrink-0"
                  />
                  <ArrowDown
                    v-else-if="sortable && sortColumn === columns[col.index]?.name && sortDir === 'desc'"
                    class="size-3 shrink-0"
                  />
                  <ArrowUpDown
                    v-else-if="sortable"
                    class="text-muted-foreground size-3 shrink-0 opacity-40"
                  />
                </div>
                <div class="text-muted-foreground/60 truncate pr-2 text-[10px] leading-none font-normal">
                  {{ columns[col.index]?.type }}
                </div>
                <span
                  class="hover:bg-border absolute inset-y-0 right-0 z-10 w-1.5 cursor-col-resize"
                  @pointerdown="startResize($event, columns[col.index]?.name ?? '')"
                  @click.stop
                />
              </div>
              <div
                v-if="padRight"
                aria-hidden="true"
                class="flex-none"
                :style="{ width: `${padRight}px` }"
              />
            </div>
            <div
              class="relative"
              :style="{ height: `${totalSize}px` }"
            >
              <div
                class="bg-background sticky left-0 z-20 border-r"
                :style="{ width: `${ROW_NUM_WIDTH}px`, height: `${totalSize}px` }"
              >
                <div
                  v-for="item in virtualRows"
                  :key="item.index"
                  role="cell"
                  class="text-muted-foreground absolute top-0 left-0 box-border flex items-center justify-center px-1 font-mono text-xs"
                  :style="{
                    width: `${ROW_NUM_WIDTH}px`,
                    height: `${item.size}px`,
                    transform: `translateY(${item.start}px)`,
                  }"
                >
                  {{ rowNumOffset + item.index + 1 }}
                </div>
              </div>
              <div
                v-for="item in virtualRows"
                :key="item.index"
                role="row"
                class="hover:bg-muted/50 absolute top-0 left-0 flex w-full border-b"
                :style="{
                  height: `${item.size}px`,
                  transform: `translateY(${item.start}px)`,
                }"
              >
                <div
                  aria-hidden="true"
                  class="flex-none"
                  :style="{ width: `${padLeft + ROW_NUM_WIDTH}px` }"
                />
                <div
                  v-for="col in virtualColumns"
                  :key="col.index"
                  role="cell"
                  class="box-border flex-none truncate px-2 font-mono text-xs leading-9"
                  :class="isNull(cellValue(item.index, col.index)) ? 'text-muted-foreground italic' : ''"
                  :style="{ width: `${col.size}px` }"
                  :title="cellText(cellValue(item.index, col.index))"
                >
                  {{ cellText(cellValue(item.index, col.index)) }}
                </div>
                <div
                  v-if="padRight"
                  aria-hidden="true"
                  class="flex-none"
                  :style="{ width: `${padRight}px` }"
                />
              </div>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>
