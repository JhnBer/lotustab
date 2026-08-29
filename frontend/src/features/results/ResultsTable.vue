<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { useVirtualizer } from "@tanstack/vue-virtual";
import { ArrowDown, ArrowUp, ArrowUpDown } from "@lucide/vue";
import type { QueryResult } from "@/lib/wails";

const DEFAULT_COL_WIDTH = 160;
const MIN_COL_WIDTH = 72;
const ROW_HEIGHT = 36;

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
  }>(),
  {
    sortable: false,
    sortColumn: "",
    sortDir: "",
    emptyHint: "Run a query to see results.",
    showStatus: true,
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

const tableMinWidth = computed(() =>
  columns.value.reduce((sum, column) => sum + (widths.value[column.name] ?? DEFAULT_COL_WIDTH), 0),
);

const virtualizer = useVirtualizer(
  computed(() => ({
    count: rows.value.length,
    getScrollElement: () => scrollRef.value,
    estimateSize: () => ROW_HEIGHT,
    overscan: 10,
  })),
);

watch(
  () => props.result,
  () => {
    virtualizer.value.scrollToOffset(0);
    scrollRef.value?.scrollTo({ top: 0 });
  },
  { flush: "post" },
);

const virtualRows = computed(() => virtualizer.value.getVirtualItems());
const totalSize = computed(() => virtualizer.value.getTotalSize());

function colWidth(name: string) {
  return widths.value[name] ?? DEFAULT_COL_WIDTH;
}

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
            :style="{ minWidth: `${tableMinWidth}px` }"
          >
            <div
              role="row"
              class="bg-background sticky top-0 z-10 flex border-b"
            >
              <div
                v-for="column in columns"
                :key="column.name"
                role="columnheader"
                class="relative box-border flex-none overflow-hidden px-2 py-2"
                :class="sortable ? 'cursor-pointer select-none hover:bg-muted/60' : ''"
                :style="{ width: `${colWidth(column.name)}px` }"
                @click="onHeaderClick(column.name)"
              >
                <div class="flex items-center gap-1 pr-2">
                  <span class="truncate font-medium">{{ column.name }}</span>
                  <ArrowUp
                    v-if="sortable && sortColumn === column.name && sortDir === 'asc'"
                    class="size-3 shrink-0"
                  />
                  <ArrowDown
                    v-else-if="sortable && sortColumn === column.name && sortDir === 'desc'"
                    class="size-3 shrink-0"
                  />
                  <ArrowUpDown
                    v-else-if="sortable"
                    class="text-muted-foreground size-3 shrink-0 opacity-40"
                  />
                </div>
                <div class="text-muted-foreground/60 truncate pr-2 text-[10px] leading-none font-normal">
                  {{ column.type }}
                </div>
                <span
                  class="hover:bg-border absolute inset-y-0 right-0 z-10 w-1.5 cursor-col-resize"
                  @pointerdown="startResize($event, column.name)"
                  @click.stop
                />
              </div>
            </div>
            <div
              class="relative"
              :style="{ height: `${totalSize}px` }"
            >
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
                  v-for="(value, colIndex) in rows[item.index] ?? []"
                  :key="colIndex"
                  role="cell"
                  class="box-border flex-none truncate px-2 font-mono text-xs leading-9"
                  :class="isNull(value) ? 'text-muted-foreground italic' : ''"
                  :style="{ width: `${colWidth(columns[colIndex]?.name ?? '')}px` }"
                  :title="cellText(value)"
                >
                  {{ cellText(value) }}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>
