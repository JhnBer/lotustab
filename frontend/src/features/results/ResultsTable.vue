<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useVirtualizer } from "@tanstack/vue-virtual";
import { ArrowDown, ArrowUp, ArrowUpDown } from "@lucide/vue";
import { Spinner } from "@/components/ui/spinner";
import type { QueryResult } from "@/lib/wails";
import { Button } from "@/components/ui/button";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { useClipboardText } from "@/composables/useClipboardText";
import { usePointerResize } from "@/composables/usePointerResize";
import {
  cellRangeToJson,
  draftLookup,
  editInputText,
  insertIdOf,
  isNull,
  isRowDeleted,
  layoutVisualRows,
  rowsClipboardText,
  rowKey,
  type CellDraft,
} from "@/features/results/edits";
import { useTableSelection } from "@/features/results/useTableSelection";
import SelectionBorder from "@/features/results/SelectionBorder.vue";

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
  editInsert: [insertId: string, column: string, value: unknown];
  deleteRow: [rowIndex: number];
  cloneRow: [rowIndex: number, afterIndex: number];
}>();

type ContextTarget =
  | { type: "row"; row: number }
  | { type: "cell"; row: number; col: number };

const columns = computed(() => props.result?.columns ?? []);
const rows = computed(() => props.result?.rows ?? []);
const visualList = computed(() =>
  layoutVisualRows(rows.value, columns.value, props.edits, props.primaryKey),
);
const allRows = computed(() => visualList.value.map((row) => row.values));
const widths = ref<Record<string, number>>({});
const scrollRef = ref<HTMLElement | null>(null);
const bodyRef = ref<HTMLElement | null>(null);
const editing = ref<{ row: number; col: number; start: string } | null>(null);
const editInput = ref("");
const selection = useTableSelection();
const dragging = ref<"rows" | "cells" | null>(null);
const contextTarget = ref<ContextTarget | null>(null);
const selectedRowCount = computed(() => selection.selectedRows.value.size);
const setValueOpen = ref(false);
const setValueInput = ref("");
const selectedCellCount = computed(() => selection.cellSelectionList.value.length);

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
    count: allRows.value.length,
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
    selection.clear();
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
    if (edit.kind === "delete" || edit.kind === "insert") {
      continue;
    }
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

function isInsertIndex(rowIndex: number) {
  return Boolean(visualList.value[rowIndex]?.draft);
}

function originalValue(rowIndex: number, colIndex: number) {
  const liveIndex = visualList.value[rowIndex]?.liveIndex;
  if (liveIndex == null) {
    return undefined;
  }
  return rows.value[liveIndex]?.[colIndex];
}

function cellValue(rowIndex: number, colIndex: number) {
  const name = columns.value[colIndex]?.name;
  const visual = visualList.value[rowIndex];
  if (visual?.draft) {
    return visual.draft.cells[name ?? ""] ?? null;
  }
  if (!name) {
    return originalValue(rowIndex, colIndex);
  }
  const liveIndex = visual?.liveIndex;
  const liveRow = liveIndex != null ? rows.value[liveIndex] : undefined;
  const key = rowKey(liveRow, columns.value, props.primaryKey);
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
  if (isInsertIndex(rowIndex)) {
    return false;
  }
  const name = columns.value[colIndex]?.name;
  if (!name) {
    return false;
  }
  const liveIndex = visualList.value[rowIndex]?.liveIndex;
  const liveRow = liveIndex != null ? rows.value[liveIndex] : undefined;
  const key = rowKey(liveRow, columns.value, props.primaryKey);
  if (!key) {
    return false;
  }
  const cells = draftMap.value.get(draftLookup(key));
  return Boolean(cells && Object.prototype.hasOwnProperty.call(cells, name));
}

function isDeletedIndex(rowIndex: number) {
  if (isInsertIndex(rowIndex)) {
    return false;
  }
  const liveIndex = visualList.value[rowIndex]?.liveIndex;
  const liveRow = liveIndex != null ? rows.value[liveIndex] : undefined;
  const key = rowKey(liveRow, columns.value, props.primaryKey);
  return isRowDeleted(props.edits, key);
}

function rowStateClass(rowIndex: number) {
  if (isInsertIndex(rowIndex)) {
    return "bg-emerald-500/10";
  }
  if (isDeletedIndex(rowIndex)) {
    return "bg-destructive/10 opacity-60";
  }
  return "";
}

function isSingleCellSelected() {
  return selection.cellSelectionList.value.length === 1;
}

function cellHighlightClass(rowIndex: number, colIndex: number) {
  if (selection.isCellSelected(rowIndex, colIndex)) {
    return "bg-primary/10";
  }
  return isDirty(rowIndex, colIndex)
    ? "bg-yellow-500/15 ring-yellow-500 hover:bg-yellow-500/25 ring-1 ring-inset"
    : "hover:bg-muted/50";
}

function isRowsMode() {
  return selection.mode.value === "rows";
}

function hasToggleModifier(event: PointerEvent | KeyboardEvent | MouseEvent) {
  return event.metaKey || event.ctrlKey || event.getModifierState("Meta") || event.getModifierState("Control");
}

const toggleModifierHeld = ref(false);

function syncToggleModifier(event: KeyboardEvent) {
  toggleModifierHeld.value = hasToggleModifier(event);
}

function clearToggleModifier() {
  toggleModifierHeld.value = false;
}

onMounted(() => {
  window.addEventListener("keydown", syncToggleModifier);
  window.addEventListener("keyup", syncToggleModifier);
  window.addEventListener("blur", clearToggleModifier);
});

onBeforeUnmount(() => {
  window.removeEventListener("keydown", syncToggleModifier);
  window.removeEventListener("keyup", syncToggleModifier);
  window.removeEventListener("blur", clearToggleModifier);
});

function selectionEdges(row: number, col: number) {
  const lastCol = columns.value.length - 1;

  const rr = selection.selectedRowRange.value;
  if (rr && row >= rr.lo && row <= rr.hi) {
    const isGutter = col === -1;
    return {
      top: row === rr.lo,
      bottom: row === rr.hi,
      left: isGutter,
      right: !isGutter && col === lastCol,
    };
  }

  if (col === -1) {
    return null;
  }
  return selection.cellBorders(row, col);
}

function rowInSelection(row: number) {
  if (selection.isRowSelected(row)) {
    return true;
  }
  return virtualColumns.value.some((col) => selection.isCellSelected(row, col.index));
}

function onHeaderClick(name: string) {
  if (props.sortable) {
    emit("sort", name);
  }
}

function rowIndexAtClientY(y: number) {
  const body = bodyRef.value;
  const count = allRows.value.length;
  if (!body || count === 0) {
    return null;
  }
  const index = Math.floor((y - body.getBoundingClientRect().top) / ROW_HEIGHT);
  return Math.max(0, Math.min(count - 1, index));
}

function colIndexAtClientX(x: number) {
  const scroller = scrollRef.value;
  const cols = columns.value;
  if (!scroller || cols.length === 0) {
    return null;
  }
  let offset = scroller.scrollLeft + (x - scroller.getBoundingClientRect().left) - ROW_NUM_WIDTH;
  if (offset < 0) {
    return 0;
  }
  for (let i = 0; i < cols.length; i++) {
    offset -= colWidth(cols[i]?.name ?? "");
    if (offset < 0) {
      return i;
    }
  }
  return cols.length - 1;
}

function stopDrag() {
  if (!dragging.value) {
    return;
  }
  dragging.value = null;
  document.body.style.userSelect = "";
  window.removeEventListener("pointermove", onDragMove);
  window.removeEventListener("pointerup", stopDrag);
  window.removeEventListener("pointercancel", stopDrag);
}

function startDrag(kind: "rows" | "cells") {
  if (!dragging.value) {
    document.body.style.userSelect = "none";
    window.addEventListener("pointermove", onDragMove);
    window.addEventListener("pointerup", stopDrag);
    window.addEventListener("pointercancel", stopDrag);
  }
  dragging.value = kind;
}

function onDragMove(event: PointerEvent) {
  if (dragging.value === "rows") {
    const row = rowIndexAtClientY(event.clientY);
    if (row != null) {
      selection.extendRowTo(row);
    }
    return;
  }
  if (dragging.value === "cells") {
    const row = rowIndexAtClientY(event.clientY);
    const col = colIndexAtClientX(event.clientX);
    if (row != null && col != null) {
      selection.extendCellTo(row, col);
    }
  }
}

function onRowPointerDown(event: PointerEvent, rowIndex: number) {
  if (event.button !== 0) {
    return;
  }
  event.preventDefault();
  if (event.shiftKey) {
    selection.extendRowTo(rowIndex);
  } else {
    selection.selectRow(rowIndex);
  }
  startDrag("rows");
}

function onCellPointerDown(event: PointerEvent, rowIndex: number, colIndex: number) {
  if (event.button !== 0) {
    return;
  }
  if ((event.target as HTMLElement | null)?.closest("input")) {
    return;
  }
  event.preventDefault();
  if (hasToggleModifier(event) || toggleModifierHeld.value) {
    selection.toggleCell(rowIndex, colIndex);
    return;
  }
  if (event.shiftKey) {
    selection.extendCellTo(rowIndex, colIndex);
  } else {
    selection.selectCell(rowIndex, colIndex);
  }
  startDrag("cells");
}

onBeforeUnmount(stopDrag);

function emitCellChange(rowIndex: number, column: string, value: unknown) {
  const visual = visualList.value[rowIndex];
  if (visual?.draft) {
    emit("editInsert", insertIdOf(visual.draft), column, value);
    return;
  }
  if (visual?.liveIndex == null) {
    return;
  }
  emit("edit", visual.liveIndex, column, value);
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
  emitCellChange(session.row, name, editInput.value);
}

function cancelEdit() {
  editing.value = null;
}

function startEdit(row: number, col: number) {
  if (!props.editable || isDeletedIndex(row)) {
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

function selectedRowIndexes() {
  return [...selection.selectedRows.value].sort((a, b) => a - b);
}

function extractGrid(r1: number, r2: number, c1: number, c2: number, maskUnselected = false) {
  const names: string[] = [];
  for (let c = c1; c <= c2; c++) {
    names.push(columns.value[c]?.name ?? "");
  }
  const grid: unknown[][] = [];
  for (let r = r1; r <= r2; r++) {
    const row: unknown[] = [];
    for (let c = c1; c <= c2; c++) {
      row.push(maskUnselected && !selection.isCellSelected(r, c) ? null : cellValue(r, c));
    }
    grid.push(row);
  }
  return { names, grid };
}

function extractCellSelection() {
  const cells = selection.cellSelectionList.value;
  if (cells.length === 0) {
    return null;
  }
  let r1 = cells[0]!.row;
  let r2 = r1;
  let c1 = cells[0]!.col;
  let c2 = c1;
  for (const cell of cells) {
    r1 = Math.min(r1, cell.row);
    r2 = Math.max(r2, cell.row);
    c1 = Math.min(c1, cell.col);
    c2 = Math.max(c2, cell.col);
  }
  return extractGrid(r1, r2, c1, c2, true);
}

function onContextMenu(event: MouseEvent) {
  const rowEl = (event.target as HTMLElement | null)?.closest("[data-row-num]");
  if (rowEl instanceof HTMLElement) {
    const row = Number(rowEl.dataset.row);
    contextTarget.value = { type: "row", row };
    if (!selection.isRowSelected(row)) {
      selection.selectRow(row);
    }
    return;
  }
  const cellEl = (event.target as HTMLElement | null)?.closest("[data-cell]");
  if (cellEl instanceof HTMLElement) {
    const row = Number(cellEl.dataset.row);
    const col = Number(cellEl.dataset.col);
    contextTarget.value = { type: "cell", row, col };
    if (!selection.isCellSelected(row, col)) {
      selection.selectCell(row, col);
    }
    return;
  }
  contextTarget.value = null;
}

async function copySelection(asJson: boolean) {
  let extracted: { names: string[]; grid: unknown[][] } | null = null;
  if (selection.mode.value === "rows") {
    const indexes = selectedRowIndexes();
    const last = indexes.at(-1);
    if (indexes[0] != null && last != null) {
      extracted = extractGrid(indexes[0], last, 0, columns.value.length - 1);
    }
  } else {
    extracted = extractCellSelection();
  }
  if (!extracted) {
    return;
  }
  await writeClipboard(asJson ? cellRangeToJson(extracted.names, extracted.grid) : rowsClipboardText(extracted.grid));
}

function cloneContextRows() {
  const indexes = selectedRowIndexes();
  const after = indexes.at(-1);
  if (after == null) {
    return;
  }
  for (const row of indexes) {
    emit("cloneRow", row, after);
  }
}

type RowDeleteState = "all-live" | "all-deleted" | "mixed";

function selectedRowDeleteState(): RowDeleteState {
  const applicable = selectedRowIndexes().filter((row) => !isInsertIndex(row));
  if (applicable.length === 0) {
    return "all-live";
  }
  const deletedCount = applicable.filter(isDeletedIndex).length;
  if (deletedCount === 0) {
    return "all-live";
  }
  if (deletedCount === applicable.length) {
    return "all-deleted";
  }
  return "mixed";
}

function deleteSelectedLiveRows() {
  for (const row of selectedRowIndexes().toReversed()) {
    if (!isDeletedIndex(row)) {
      emit("deleteRow", row);
    }
  }
}

function restoreSelectedDeletedRows() {
  for (const row of selectedRowIndexes().toReversed()) {
    if (isDeletedIndex(row)) {
      emit("deleteRow", row);
    }
  }
}

function applyValueToSelection(value: unknown) {
  if (!props.editable) {
    return;
  }
  if (selection.mode.value === "rows") {
    for (const row of selectedRowIndexes()) {
      if (isDeletedIndex(row)) {
        continue;
      }
      for (const column of columns.value) {
        emitCellChange(row, column.name, value);
      }
    }
    return;
  }
  for (const cell of selection.cellSelectionList.value) {
    if (isDeletedIndex(cell.row)) {
      continue;
    }
    const name = columns.value[cell.col]?.name;
    if (name) {
      emitCellChange(cell.row, name, value);
    }
  }
}

function setNullSelection() {
  applyValueToSelection(null);
}

function openSetValue() {
  setValueInput.value = "";
  void nextTick(() => {
    setValueOpen.value = true;
  });
}

function setValueSelection() {
  applyValueToSelection(setValueInput.value);
  setValueOpen.value = false;
  setValueInput.value = "";
}

function selectionEditable() {
  if (!props.editable) {
    return false;
  }
  if (selection.mode.value === "rows") {
    return selectedRowIndexes().some((row) => !isDeletedIndex(row));
  }
  return selection.cellSelectionList.value.some((cell) => !isDeletedIndex(cell.row));
}

async function pasteCell() {
  const cell = selection.cellSelectionList.value[0];
  if (!props.editable || !cell || !isSingleCellSelected() || isDeletedIndex(cell.row)) {
    return;
  }
  const name = columns.value[cell.col]?.name;
  if (!name) {
    return;
  }
  try {
    emitCellChange(cell.row, name, await navigator.clipboard.readText());
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
        <span>{{ result.durationMs }} ms<template v-if="result.affectedKnown"> · {{ result.rowsAffected }} affected</template></span>
      </div>
      <div v-if="columns.length === 0" class="text-muted-foreground p-4 text-sm">
        <template v-if="result.affectedKnown">
          Statement completed. {{ result.rowsAffected }} row{{ result.rowsAffected === 1 ? "" : "s" }} affected.
        </template>
        <template v-else>Statement completed.</template>
      </div>
      <div v-else class="relative min-h-0 flex-1">
        <ContextMenu>
          <div ref="scrollRef" class="absolute inset-0 overflow-auto" @contextmenu.capture="onContextMenu">
            <ContextMenuTrigger as-child>
              <div
                role="table"
                class="text-sm select-none"
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
                  ref="bodyRef"
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
                      data-row-num
                      :data-row="item.index"
                      class="text-muted-foreground absolute top-0 left-0 box-border flex cursor-pointer items-center justify-center border-b px-1 font-mono text-xs"
                      :class="[
                        rowStateClass(item.index),
                        selection.isRowSelected(item.index) ? 'text-foreground' : '',
                        rowInSelection(item.index) ? 'z-10' : '',
                      ]"
                      :style="{
                        width: `${ROW_NUM_WIDTH}px`,
                        height: `${item.size}px`,
                        transform: `translateY(${item.start}px)`,
                      }"
                      @pointerdown="onRowPointerDown($event, item.index)"
                    >
                      <template v-for="e in [selectionEdges(item.index, -1)]" :key="0">
                        <SelectionBorder v-if="e" v-bind="e" />
                      </template>
                      {{ rowNumOffset + item.index + 1 }}
                    </div>
                  </div>
                  <div
                    v-for="item in virtualRows"
                    :key="item.index"
                    role="row"
                    class="absolute top-0 left-0 flex w-full border-b"
                    :class="[rowStateClass(item.index), rowInSelection(item.index) ? 'z-10' : '']"
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
                      class="relative box-border flex-none px-2 font-mono text-xs leading-7"
                      :class="[
                        editable ? 'cursor-text' : 'cursor-pointer',
                        cellHighlightClass(item.index, col.index),
                        isPlaceholderValue(cellValue(item.index, col.index)) ? 'text-muted-foreground italic' : '',
                      ]"
                      :style="{ width: `${col.size}px` }"
                      :title="isEditing(item.index, col.index) ? undefined : cellText(cellValue(item.index, col.index))"
                      @pointerdown="onCellPointerDown($event, item.index, col.index)"
                      @dblclick="startEdit(item.index, col.index)"
                    >
                      <template v-if="isRowsMode()">
                        <template v-for="e in [selectionEdges(item.index, col.index)]" :key="0">
                          <SelectionBorder v-if="e" v-bind="e" />
                        </template>
                      </template>
                      <SelectionBorder
                        v-else
                        :row="item.index"
                        :col="col.index"
                        :selected-keys="selection.selectedKeys.value"
                      />
                      <input
                        v-if="isEditing(item.index, col.index)"
                        v-model="editInput"
                        class="h-full w-full truncate bg-transparent px-0 font-mono text-xs not-italic outline-none select-text"
                        @keydown.enter.prevent="commitEdit"
                        @keydown.escape.prevent="cancelEdit"
                        @blur="commitEdit"
                      >
                      <span v-else class="block truncate">
                        {{ cellText(cellValue(item.index, col.index)) }}
                      </span>
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
            <template v-if="contextTarget?.type === 'row'">
              <ContextMenuItem @select="copySelection(false)">
                Copy {{ selectedRowCount === 1 ? "row" : "rows" }}
              </ContextMenuItem>
              <ContextMenuItem @select="copySelection(true)">
                Copy as JSON
              </ContextMenuItem>
              <ContextMenuItem :disabled="!editable" @select="cloneContextRows">
                Clone {{ selectedRowCount === 1 ? "row" : "rows" }}
              </ContextMenuItem>
              <ContextMenuItem
                v-if="selectedRowDeleteState() !== 'all-deleted'"
                variant="destructive"
                :disabled="!editable"
                @select="deleteSelectedLiveRows"
              >
                Delete {{ selectedRowCount === 1 ? "row" : "rows" }}
              </ContextMenuItem>
              <ContextMenuItem
                v-if="selectedRowDeleteState() !== 'all-live'"
                :disabled="!editable"
                @select="restoreSelectedDeletedRows"
              >
                Restore {{ selectedRowCount === 1 ? "row" : "rows" }}
              </ContextMenuItem>
              <ContextMenuSeparator />
              <ContextMenuItem :disabled="!selectionEditable()" @select="setNullSelection">
                Set NULL
              </ContextMenuItem>
            </template>
            <template v-else-if="contextTarget?.type === 'cell'">
              <ContextMenuItem @select="copySelection(false)">
                Copy
              </ContextMenuItem>
              <ContextMenuItem @select="copySelection(true)">
                Copy as JSON
              </ContextMenuItem>
              <ContextMenuSeparator />
              <ContextMenuItem :disabled="!selectionEditable()" @select="setNullSelection">
                Set NULL
              </ContextMenuItem>
              <ContextMenuItem :disabled="!selectionEditable()" @select="openSetValue">
                Set value for selected
              </ContextMenuItem>
              <ContextMenuItem
                v-if="isSingleCellSelected()"
                :disabled="!selectionEditable()"
                @select="pasteCell"
              >
                Paste
              </ContextMenuItem>
            </template>
          </ContextMenuContent>
        </ContextMenu>
        <Dialog :open="setValueOpen" @update:open="setValueOpen = $event">
          <DialogContent>
            <DialogHeader>
              <DialogTitle>
                Set value for {{ selectedCellCount }} {{ selectedCellCount === 1 ? "cell" : "cells" }}
              </DialogTitle>
            </DialogHeader>
            <Input
              v-model="setValueInput"
              autofocus
              @keydown.enter.prevent="setValueSelection"
            />
            <DialogFooter>
              <Button variant="outline" @click="setValueOpen = false">Cancel</Button>
              <Button @click="setValueSelection">Confirm</Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>
    </template>
  </div>
</template>
