<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { ArrowDown, ArrowUp, ArrowUpDown } from "@lucide/vue";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { QueryResult } from "@/lib/wails";

const DEFAULT_COL_WIDTH = 160;
const MIN_COL_WIDTH = 72;

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
    <div v-if="running" class="text-muted-foreground p-4 text-sm">
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
        <div class="absolute inset-0 overflow-hidden">
          <Table class="table-fixed" :style="{ minWidth: `${tableMinWidth}px` }">
            <colgroup>
              <col
                v-for="column in columns"
                :key="column.name"
                :style="{ width: `${colWidth(column.name)}px` }"
              />
            </colgroup>
            <TableHeader>
              <TableRow>
                <TableHead
                  v-for="column in columns"
                  :key="column.name"
                  class="bg-background sticky top-0 whitespace-nowrap"
                  :class="sortable ? 'cursor-pointer select-none hover:bg-muted/60' : ''"
                  @click="onHeaderClick(column.name)"
                >
                  <div class="flex items-center gap-1 pr-2">
                    <span>{{ column.name }}</span>
                    <ArrowUp
                      v-if="sortable && sortColumn === column.name && sortDir === 'asc'"
                      class="size-3"
                    />
                    <ArrowDown
                      v-else-if="sortable && sortColumn === column.name && sortDir === 'desc'"
                      class="size-3"
                    />
                    <ArrowUpDown
                      v-else-if="sortable"
                      class="text-muted-foreground size-3 opacity-40"
                    />
                  </div>
                  <div class="text-muted-foreground/60 pr-2 text-[10px] leading-none font-normal">
                    {{ column.type }}
                  </div>
                  <span
                    class="hover:bg-border absolute inset-y-0 right-0 z-10 w-1.5 cursor-col-resize"
                    @pointerdown="startResize($event, column.name)"
                    @click.stop
                  />
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-for="(row, rowIndex) in rows" :key="rowIndex">
                <TableCell
                  v-for="(value, colIndex) in row ?? []"
                  :key="colIndex"
                  class="truncate font-mono text-xs"
                  :class="isNull(value) ? 'text-muted-foreground italic' : ''"
                  :title="cellText(value)"
                >
                  {{ cellText(value) }}
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </div>
      </div>
    </template>
  </div>
</template>
