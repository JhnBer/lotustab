import type { Column } from "@/lib/wails";

export const MIN_COL_WIDTH = 72;
export const MAX_COL_WIDTH = 360;
export const MAX_LARGE_COL_WIDTH = 220;
export const SAMPLE_ROW_LIMIT = 40;
export const LONG_VALUE_CHARS = 120;
export const CELL_PAD_X = 16;

const LARGE_TYPE_RE = /\b(jsonb?|text|clob|blob|bytea|binary)\b/i;

const MONO_XS_FONT =
  '12px "Geist Mono", ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace';

let measureCtx: CanvasRenderingContext2D | null = null;

function measureContext(): CanvasRenderingContext2D | null {
  if (typeof document === "undefined") {
    return null;
  }
  if (!measureCtx) {
    const canvas = document.createElement("canvas");
    measureCtx = canvas.getContext("2d");
  }
  if (measureCtx) {
    measureCtx.font = MONO_XS_FONT;
  }
  return measureCtx;
}

/** Pixel width of text in the grid's `text-xs font-mono` face. */
export function measureMonoXs(text: string): number {
  const ctx = measureContext();
  if (!ctx) {
    // SSR / tests without canvas: ~7px per mono character at 12px.
    return Math.ceil(text.length * 7);
  }
  return Math.ceil(ctx.measureText(text).width);
}

export function isLargeColumnType(type: string | undefined | null): boolean {
  if (!type) {
    return false;
  }
  return LARGE_TYPE_RE.test(type);
}

export function isBinaryColumnType(type: string | undefined | null): boolean {
  if (!type) {
    return false;
  }
  return /\b(blob|bytea|binary)\b/i.test(type);
}

export function cellDisplayText(value: unknown): string {
  if (value == null) {
    return "NULL";
  }
  if (value === "" || value === "''") {
    return "empty string";
  }
  if (typeof value === "object") {
    try {
      return JSON.stringify(value);
    } catch {
      return String(value);
    }
  }
  return String(value);
}

function clampWidth(raw: number, large: boolean): number {
  const max = large ? MAX_LARGE_COL_WIDTH : MAX_COL_WIDTH;
  return Math.min(max, Math.max(MIN_COL_WIDTH, raw));
}

/**
 * Compute column widths from headers and a bounded sample of already-loaded rows.
 * Does not scan beyond SAMPLE_ROW_LIMIT rows.
 */
export function computeColumnWidths(
  columns: Column[],
  rows: Array<unknown[] | null | undefined>,
): Record<string, number> {
  const sample = rows.slice(0, SAMPLE_ROW_LIMIT);
  const widths: Record<string, number> = {};

  for (let colIndex = 0; colIndex < columns.length; colIndex++) {
    const column = columns[colIndex]!;
    const headerPx = measureMonoXs(column.name) + CELL_PAD_X;
    let contentPx = 0;
    let longSample = false;

    for (const row of sample) {
      const text = cellDisplayText(row?.[colIndex]);
      if (text.length > LONG_VALUE_CHARS) {
        longSample = true;
      }
      contentPx = Math.max(contentPx, measureMonoXs(text) + CELL_PAD_X);
    }

    const large = isLargeColumnType(column.type) || longSample;
    widths[column.name] = clampWidth(Math.max(headerPx, contentPx), large);
  }

  return widths;
}

export function columnSetKey(columns: { name: string }[]): string {
  return columns.map((column) => column.name).join("\0");
}

export function isLargeCell(type: string | undefined | null, value: unknown): boolean {
  if (isLargeColumnType(type)) {
    return true;
  }
  return cellDisplayText(value).length > LONG_VALUE_CHARS;
}
