import { isBinaryColumnType, isLargeColumnType } from "@/features/results/columnWidth";

export type ValueInspectorMode = "tree" | "text" | "binary";

export type ValueInspectorPayload = {
  token: string;
  sourceWindow: string;
  columnName: string;
  columnType: string;
  /** Current editor text (may include unsaved edits). */
  text: string;
  editable: boolean;
  row: number;
  col: number;
};

export function valueToEditorText(value: unknown): string {
  if (value == null) {
    return "";
  }
  if (typeof value === "object") {
    try {
      return JSON.stringify(value, null, 2);
    } catch {
      return String(value);
    }
  }
  return String(value);
}

export function tryParseJson(text: string): unknown | undefined {
  const trimmed = text.trim();
  if (!trimmed) {
    return undefined;
  }
  if (!(trimmed.startsWith("{") || trimmed.startsWith("["))) {
    return undefined;
  }
  try {
    const parsed = JSON.parse(trimmed) as unknown;
    if (parsed != null && typeof parsed === "object") {
      return parsed;
    }
  } catch {
    // not JSON
  }
  return undefined;
}

export function isJsonLike(columnType: string, text: string, original?: unknown): boolean {
  if (/\bjsonb?\b/i.test(columnType)) {
    return true;
  }
  if (original != null && typeof original === "object") {
    return true;
  }
  return tryParseJson(text) !== undefined;
}

export function inspectorModeFor(
  columnType: string,
  text: string,
  original?: unknown,
): ValueInspectorMode {
  if (isBinaryColumnType(columnType)) {
    return "binary";
  }
  if (isJsonLike(columnType, text, original)) {
    const parsed = tryParseJson(text);
    if (parsed !== undefined || /\bjsonb?\b/i.test(columnType)) {
      return "tree";
    }
  }
  return "text";
}

export function formatBinaryPreview(text: string, maxChars = 512): string {
  const trimmed = text.trim();
  if (!trimmed) {
    return "(empty)";
  }
  // Prefer showing as-is when it already looks like hex or base64; truncate hard.
  const body = trimmed.length > maxChars ? `${trimmed.slice(0, maxChars)}…` : trimmed;
  const looksHex = /^[0-9a-fA-F\s]+$/.test(trimmed) && trimmed.replace(/\s+/g, "").length % 2 === 0;
  if (looksHex) {
    return `hex:\n${body}`;
  }
  return `base64:\n${body}`;
}

export function newInspectorToken(): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `v-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
}

export { isLargeColumnType, isBinaryColumnType };
