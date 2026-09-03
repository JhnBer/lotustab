import type { ClassValue } from "clsx"
import { clsx } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

/** Converts nullable profile params map to a plain Record<string, string>. */
export function asRecord(params: { [_: string]: string | undefined } | null | undefined): Record<string, string> {
  const out: Record<string, string> = {};
  if (!params) return out;
  for (const [key, value] of Object.entries(params)) {
    if (typeof value === "string") out[key] = value;
  }
  return out;
}
