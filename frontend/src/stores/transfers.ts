import { computed, ref } from "vue";
import { defineStore } from "pinia";

export type TransferStatus = "running" | "done" | "failed" | "canceled";

export type TransferRecord = {
  id: string;
  table: string;
  status: TransferStatus;
  rows: number;
  totalRows: number | null;
  bytes: number;
  startedAt: number;
  durationMs?: number;
  error?: string;
  files?: string[];
};

let seq = 0;

export const useTransfersStore = defineStore("transfers", () => {
  const records = ref<TransferRecord[]>([]);
  const runningCount = computed(
    () => records.value.filter((r) => r.status === "running").length,
  );

  function start(table: string, totalRows: number | null): string {
    const id = `export-${++seq}-${Date.now()}`;
    records.value.unshift({
      id,
      table,
      status: "running",
      rows: 0,
      totalRows,
      bytes: 0,
      startedAt: Date.now(),
    });
    return id;
  }

  function update(id: string, patch: { rows?: number; bytes?: number }) {
    const rec = records.value.find((r) => r.id === id);
    if (!rec || rec.status !== "running") {
      return;
    }
    if (patch.rows != null) {
      rec.rows = patch.rows;
    }
    if (patch.bytes != null) {
      rec.bytes = patch.bytes;
    }
  }

  function finish(
    id: string,
    result: { rows?: number; bytes?: number; files?: string[]; canceled?: boolean },
  ) {
    const rec = records.value.find((r) => r.id === id);
    if (!rec) {
      return;
    }
    if (result.rows != null) {
      rec.rows = result.rows;
    }
    if (result.bytes != null) {
      rec.bytes = result.bytes;
    }
    rec.files = result.files ?? rec.files;
    rec.durationMs = Date.now() - rec.startedAt;
    rec.status = result.canceled ? "canceled" : "done";
  }

  function fail(id: string, error: string) {
    const rec = records.value.find((r) => r.id === id);
    if (!rec) {
      return;
    }
    rec.status = "failed";
    rec.error = error;
    rec.durationMs = Date.now() - rec.startedAt;
  }

  return { records, runningCount, start, update, finish, fail };
});
