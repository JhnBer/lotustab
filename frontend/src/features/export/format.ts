export function formatBytes(n: number): string {
  if (n < 1024) {
    return `${n} B`;
  }
  if (n < 1024 * 1024) {
    return `${(n / 1024).toFixed(1)} KB`;
  }
  if (n < 1024 * 1024 * 1024) {
    return `${(n / (1024 * 1024)).toFixed(1)} MB`;
  }
  return `${(n / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

export function formatRows(n: number): string {
  return n.toLocaleString();
}

export function formatDuration(ms: number): string {
  const sec = Math.max(0, Math.floor(ms / 1000));
  if (sec < 60) {
    return `${sec}s`;
  }
  const min = Math.floor(sec / 60);
  const rem = sec % 60;
  if (min < 60) {
    return rem ? `${min}m ${rem}s` : `${min}m`;
  }
  const hr = Math.floor(min / 60);
  const minRem = min % 60;
  return minRem ? `${hr}h ${minRem}m` : `${hr}h`;
}

export function exportFileExt(format: string, gzipOn: boolean): string {
  const ext = format === "csv" ? ".csv" : format === "json" ? ".json" : ".sql";
  return gzipOn ? `${ext}.gz` : ext;
}

export function suggestedExportFileName(table: string, format: string, gzipOn: boolean): string {
  const base =
    table
      .trim()
      .replace(/[^A-Za-z0-9._-]+/g, "_")
      .replace(/^[._]+|[._]+$/g, "") || "table";
  const d = new Date();
  const stamp = `${d.getFullYear()}${String(d.getMonth() + 1).padStart(2, "0")}${String(d.getDate()).padStart(2, "0")}`;
  return `${base}_${stamp}${exportFileExt(format, gzipOn)}`;
}

export function withExportExt(name: string, format: string, gzipOn: boolean): string {
  const stem = name.replace(/\.(sql|csv|json)(\.gz)?$/i, "").trim() || "table";
  return `${stem}${exportFileExt(format, gzipOn)}`;
}

export function suggestedExportFolderName(name: string): string {
  const base =
    name
      .trim()
      .replace(/[^A-Za-z0-9._-]+/g, "_")
      .replace(/^[._]+|[._]+$/g, "") || "export";
  const d = new Date();
  const stamp = `${d.getFullYear()}${String(d.getMonth() + 1).padStart(2, "0")}${String(d.getDate()).padStart(2, "0")}`;
  return `${base}_${stamp}`;
}

export function isDirNotEmptyError(err: unknown): boolean {
  const message = err == null ? "" : err instanceof Error ? err.message : String(err);
  return message.includes("dir_not_empty");
}
