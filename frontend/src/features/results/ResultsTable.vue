<script setup lang="ts">
import { computed, nextTick, ref, watch } from "vue";
import { useVirtualizer } from "@tanstack/vue-virtual";
import { ArrowDown, ArrowUp, ArrowUpDown } from "@lucide/vue";
import { Spinner } from "@/components/ui/spinner";
import type { QueryResult } from "@/lib/wails";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import { useClipboardText } from "@/composables/useClipboardText";
import { usePointerResize } from "@/composables/usePointerResize";
import {
  draftLookup,
  editInputText,
  isNull,
  rowKey,
  type CellDraft,
} from "@/features/results/edits";

const DEFAULT_COL_WIDTH = 160;
const MIN_COL_WIDTH = 72;
const ROW_HEIGHT = 28;
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
    editable?: boolean;
    edits?: CellDraft[];
    primaryKey?: string[];
  }>(),
  {
    sortable: false,
    sortColumn: "",
    sortDir: "",
    emptyHint: "Run a query to see results.",
    showStatus: true,
    rowOffset: 0,
    editable: false,
    edits: () => [],
    primaryKey: () => [],
  },
);

const emit = defineEmits<{
  sort: [column: string];
  edit: [rowIndex: number, column: string, value: unknown];
}>();

const columns = computed(() => props.result?.columns ?? []);
const rows = computed(() => props.result?.rows ?? []);
const widths = ref<Record<string, number>>({});
const scrollRef = ref<HTMLElement | null>(null);
const editing = ref<{ row: number; col: number; start: string } | null>(null);
const editInput = ref("");
const contextCell = ref<{ row: number; col: number } | null>(null);

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

function columnKey(cols: { name: string }[]) {
  return cols.map((column) => column.name).join("\0");
}

let prevColumnKey = columnKey(columns.value);

watch(
  () => props.result,
  () => {
    const nextKey = columnKey(columns.value);
    const colsChanged = nextKey !== prevColumnKey;
    prevColumnKey = nextKey;
    rowVirtualizer.value.scrollToOffset(0);
    const scroller = scrollRef.value;
    if (scroller) {
      scroller.scrollTop = 0;
      if (colsChanged) {
        scroller.scrollLeft = 0;
        columnVirtualizer.value.scrollToOffset(0);
      }
    }
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

const draftMap = computed(() => {
  const map = new Map<string, Record<string, unknown>>();
  for (const edit of props.edits) {
    if (!edit.key) {
      continue;
    }
    map.set(draftLookup(edit.key), edit.cells ?? {});
  }
  return map;
});

const { writeClipboard } = useClipboardText();

const { start: startResize } = usePointerResize<{ name: string; startWidth: number }>({
  onMove(event, drag) {
    const next = Math.max(MIN_COL_WIDTH, drag.startWidth + (event.clientX - drag.startX));
    widths.value = { ...widths.value, [drag.name]: next };
  },
});

function onStartResize(event: PointerEvent, name: string) {
  startResize(event, { name, startWidth: colWidth(name) });
}

function cellText(value: unknown): string {
  if (isNull(value)) {
    return "NULL";
  }
  if (isEmptyStringValue(value)) {
    return "empty string";
  }
  if (typeof value === "object") {
    return JSON.stringify(value);
  }
  return String(value);
}

function isEmptyStringValue(value: unknown) {
  return value === "" || value === "''";
}

function isPlaceholderValue(value: unknown) {
  return isNull(value) || isEmptyStringValue(value);
}

function originalValue(rowIndex: number, colIndex: number) {
  return rows.value[rowIndex]?.[colIndex];
}

function cellValue(rowIndex: number, colIndex: number) {
  const name = columns.value[colIndex]?.name;
  if (!name) {
    return originalValue(rowIndex, colIndex);
  }
  const key = rowKey(rows.value[rowIndex], columns.value, props.primaryKey);
  if (!key) {
    return originalValue(rowIndex, colIndex);
  }
  const cells = draftMap.value.get(draftLookup(key));
  if (cells && Object.prototype.hasOwnProperty.call(cells, name)) {
    return cells[name];
  }
  return originalValue(rowIndex, colIndex);
}

function isDirty(rowIndex: number, colIndex: number) {
  const name = columns.value[colIndex]?.name;
  if (!name) {
    return false;
  }
  const key = rowKey(rows.value[rowIndex], columns.value, props.primaryKey);
  if (!key) {
    return false;
  }
  const cells = draftMap.value.get(draftLookup(key));
  return Boolean(cells && Object.prototype.hasOwnProperty.call(cells, name));
}

function onHeaderClick(name: string) {
  if (props.sortable) {
    emit("sort", name);
  }
}

function commitEdit() {
  const session = editing.value;
  if (!session) {
    return;
  }
  editing.value = null;
  const name = columns.value[session.col]?.name;
  if (!name || editInput.value === session.start) {
    return;
  }
  emit("edit", session.row, name, editInput.value);
}

function cancelEdit() {
  editing.value = null;
}

function startEdit(row: number, col: number) {
  if (!props.editable) {
    return;
  }
  if (editing.value) {
    commitEdit();
  }
  const start = editInputText(cellValue(row, col));
  editing.value = { row, col, start };
  editInput.value = start;
  void nextTick(() => {
    scrollRef.value?.querySelector("input")?.focus();
  });
}

function isEditing(row: number, col: number) {
  return editing.value?.row === row && editing.value?.col === col;
}

watch(virtualRows, (items) => {
  if (editing.value && !items.some((item) => item.index === editing.value?.row)) {
    commitEdit();
  }
});

function onContextMenu(event: MouseEvent) {
  const el = (event.target as HTMLElement | null)?.closest("[data-cell]");
  if (!(el instanceof HTMLElement)) {
    contextCell.value = null;
    return;
  }
  contextCell.value = {
    row: Number(el.dataset.row),
    col: Number(el.dataset.col),
  };
}

function contextValue() {
  if (!contextCell.value) {
    return undefined;
  }
  return cellValue(contextCell.value.row, contextCell.value.col);
}

async function copyCell() {
  if (!contextCell.value) {
    return;
  }
  await writeClipboard(editInputText(contextValue()));
}

function setNull() {
  if (!props.editable || !contextCell.value) {
    return;
  }
  const name = columns.value[contextCell.value.col]?.name;
  if (!name) {
    return;
  }
  emit("edit", contextCell.value.row, name, null);
}

async function pasteCell() {
  if (!props.editable || !contextCell.value) {
    return;
  }
  const name = columns.value[contextCell.value.col]?.name;
  if (!name) {
    return;
  }
  try {
    emit("edit", contextCell.value.row, name, await navigator.clipboard.readText());
  } catch {
    // clipboard may be denied in the webview
  }
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col">
    <div v-if="running && !result" class="text-muted-foreground flex flex-1 items-center justify-center p-4">
      <Spinner />
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
        <ContextMenu>
          <div ref="scrollRef" class="absolute inset-0 overflow-auto" @contextmenu.capture="onContextMenu">
            <ContextMenuTrigger as-child>
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
                      @pointerdown="onStartResize($event, columns[col.index]?.name ?? '')"
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
                    class="absolute top-0 left-0 flex w-full border-b"
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
                      data-cell
                      :data-row="item.index"
                      :data-col="col.index"
                      class="box-border flex-none truncate px-2 font-mono text-xs leading-7"
                      :class="[
                        editable ? 'cursor-text' : '',
                        isDirty(item.index, col.index)
                          ? 'bg-yellow-500/15 ring-yellow-500 hover:bg-yellow-500/25 ring-1 ring-inset'
                          : 'hover:bg-muted/50',
                        isPlaceholderValue(cellValue(item.index, col.index)) ? 'text-muted-foreground italic' : '',
                      ]"
                      :style="{ width: `${col.size}px` }"
                      :title="isEditing(item.index, col.index) ? undefined : cellText(cellValue(item.index, col.index))"
                      @dblclick="startEdit(item.index, col.index)"
                    >
                      <input
                        v-if="isEditing(item.index, col.index)"
                        v-model="editInput"
                        class="h-full w-full bg-transparent px-0 font-mono text-xs not-italic outline-none"
                        @keydown.enter.prevent="commitEdit"
                        @keydown.escape.prevent="cancelEdit"
                        @blur="commitEdit"
                      >
                      <template v-else>
                        {{ cellText(cellValue(item.index, col.index)) }}
                      </template>
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
            </ContextMenuTrigger>
          </div>
          <ContextMenuContent>
            <ContextMenuItem :disabled="!contextCell" @select="copyCell">
              Copy
            </ContextMenuItem>
            <ContextMenuItem :disabled="!editable || !contextCell" @select="setNull">
              Set NULL
            </ContextMenuItem>
            <ContextMenuItem :disabled="!editable || !contextCell" @select="pasteCell">
              Paste
            </ContextMenuItem>
          </ContextMenuContent>
        </ContextMenu>
      </div>
    </template>
  </div>
</template>
