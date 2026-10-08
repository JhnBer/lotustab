import { computed, ref } from "vue";

export type CellRange = { r1: number; r2: number; c1: number; c2: number };
export type SelectionMode = "none" | "rows" | "cells";
export type CellPos = { row: number; col: number };

function cellKey(row: number, col: number) {
  return `${row}:${col}`;
}

function parseCellKey(key: string): CellPos {
  const sep = key.indexOf(":");
  return { row: Number(key.slice(0, sep)), col: Number(key.slice(sep + 1)) };
}

export function useTableSelection() {
  const rowAnchor = ref<number | null>(null);
  const rowActive = ref<number | null>(null);
  const committedCells = ref(new Set<string>());
  const dragAnchor = ref<CellPos | null>(null);
  const dragActive = ref<CellPos | null>(null);

  const dragRange = computed((): CellRange | null => {
    if (!dragAnchor.value || !dragActive.value) {
      return null;
    }
    return {
      r1: Math.min(dragAnchor.value.row, dragActive.value.row),
      r2: Math.max(dragAnchor.value.row, dragActive.value.row),
      c1: Math.min(dragAnchor.value.col, dragActive.value.col),
      c2: Math.max(dragAnchor.value.col, dragActive.value.col),
    };
  });

  const mode = computed((): SelectionMode => {
    if (rowAnchor.value != null && rowActive.value != null) {
      return "rows";
    }
    if (committedCells.value.size > 0 || dragAnchor.value) {
      return "cells";
    }
    return "none";
  });

  const selectedRowRange = computed((): { lo: number; hi: number } | null => {
    if (rowAnchor.value == null || rowActive.value == null) {
      return null;
    }
    return {
      lo: Math.min(rowAnchor.value, rowActive.value),
      hi: Math.max(rowAnchor.value, rowActive.value),
    };
  });

  const selectedRows = computed(() => {
    const rows = new Set<number>();
    const range = selectedRowRange.value;
    if (!range) {
      return rows;
    }
    for (let i = range.lo; i <= range.hi; i++) {
      rows.add(i);
    }
    return rows;
  });

  const selectedKeys = computed(() => {
    const keys = new Set(committedCells.value);
    const range = dragRange.value;
    if (!range) {
      return keys;
    }
    for (let r = range.r1; r <= range.r2; r++) {
      for (let c = range.c1; c <= range.c2; c++) {
        keys.add(cellKey(r, c));
      }
    }
    return keys;
  });

  const cellSelectionList = computed((): CellPos[] => {
    const list: CellPos[] = [];
    for (const key of selectedKeys.value) {
      list.push(parseCellKey(key));
    }
    return list;
  });

  function isRowSelected(row: number) {
    return selectedRows.value.has(row);
  }

  function isCellSelected(row: number, col: number) {
    return selectedKeys.value.has(cellKey(row, col));
  }

  function cellBorders(row: number, col: number) {
    if (!isCellSelected(row, col)) {
      return null;
    }
    return {
      top: !isCellSelected(row - 1, col),
      bottom: !isCellSelected(row + 1, col),
      left: !isCellSelected(row, col - 1),
      right: !isCellSelected(row, col + 1),
    };
  }

  function clearRows() {
    rowAnchor.value = null;
    rowActive.value = null;
  }

  function clearCells() {
    committedCells.value = new Set();
    dragAnchor.value = null;
    dragActive.value = null;
  }

  function selectRow(row: number) {
    clearCells();
    rowAnchor.value = row;
    rowActive.value = row;
  }

  function extendRowTo(row: number) {
    if (rowAnchor.value == null) {
      selectRow(row);
      return;
    }
    clearCells();
    rowActive.value = row;
  }

  function selectCell(row: number, col: number) {
    clearRows();
    committedCells.value = new Set();
    dragAnchor.value = { row, col };
    dragActive.value = { row, col };
  }

  function extendCellTo(row: number, col: number) {
    if (!dragAnchor.value) {
      selectCell(row, col);
      return;
    }
    clearRows();
    committedCells.value = new Set();
    dragActive.value = { row, col };
  }

  function foldRangeIntoCommitted() {
    const range = dragRange.value;
    if (!range) {
      return;
    }
    const next = new Set(committedCells.value);
    for (let r = range.r1; r <= range.r2; r++) {
      for (let c = range.c1; c <= range.c2; c++) {
        next.add(cellKey(r, c));
      }
    }
    committedCells.value = next;
    dragActive.value = null;
  }

  function toggleCell(row: number, col: number) {
    clearRows();
    foldRangeIntoCommitted();
    const key = cellKey(row, col);
    const next = new Set(committedCells.value);
    if (next.has(key)) {
      next.delete(key);
    } else {
      next.add(key);
    }
    committedCells.value = next;
    if (next.size === 0) {
      dragAnchor.value = null;
      dragActive.value = null;
    }
  }

  function clear() {
    clearRows();
    clearCells();
  }

  return {
    mode,
    selectedRows,
    selectedRowRange,
    selectedKeys,
    cellSelectionList,
    isRowSelected,
    isCellSelected,
    cellBorders,
    selectRow,
    extendRowTo,
    selectCell,
    extendCellTo,
    toggleCell,
    clear,
  };
}
