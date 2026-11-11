<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from "vue";
import { Events } from "@wailsio/runtime";
import { toast } from "vue-sonner";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import LotusLoader from "@/components/LotusLoader.vue";
import ExportOutputForm from "@/features/export/ExportOutputForm.vue";
import { formatBytes, formatDuration, formatRows, isDirNotEmptyError, isSchemaExport, suggestedExportFileName, suggestedExportFolderName } from "@/features/export/format";
import { errorMessage, isCanceled } from "@/lib/errors";
import { ExportService, type ExportTableRequest } from "@/lib/wails";
import { useSettingsStore } from "@/stores/settings";
import { useTransfersStore } from "@/stores/transfers";

export type ExportTarget = {
  profileId: string;
  database: string;
  schema: string;
  name: string;
  type?: string;
};

const props = defineProps<{
  open: boolean;
  target: ExportTarget | null;
}>();

const emit = defineEmits<{
  "update:open": [value: boolean];
}>();

type Progress = { chunk: number; rows: number; bytes: number; query?: string };

const settings = useSettingsStore();
const transfers = useTransfersStore();

let lastExportDir = "";

const format = ref("sql");
const content = ref("");
const gzip = ref(false);
const chunkSize = ref("0");
const customWhere = ref(false);
const whereSQL = ref("");
const includeCreate = ref(true);
const includeDrop = ref(false);
const insertBatchSize = ref("500");
const wrapTransaction = ref(true);
const outputDir = ref("");
const fileName = ref("");
const folderMode = ref(false);
const folderName = ref("");
const prefixEnabled = ref(false);
const prefix = ref("");
const overwriteWarn = ref(false);
const checkedTotal = ref<number | null>(null);
const counting = ref(false);
const browsing = ref(false);

const running = ref(false);
const progress = ref<Progress>({ chunk: 0, rows: 0, bytes: 0 });
const pending = ref<ReturnType<typeof ExportService.ExportTable> | null>(null);
const transferId = ref("");
const exportStartedAt = ref(0);
const elapsedMs = ref(0);
let tick: ReturnType<typeof setInterval> | null = null;

const schemaExport = computed(() => isSchemaExport(content.value));
const schemaDisabled = computed(() => props.target?.type === "view");
const canExport = computed(() => {
  if (outputDir.value.trim().length === 0) {
    return false;
  }
  if (folderMode.value) {
    return folderName.value.trim().length > 0;
  }
  return fileName.value.trim().length > 0;
});
const remainingLabel = computed(() => {
  const total = checkedTotal.value;
  const rows = progress.value.rows;
  const elapsed = elapsedMs.value;
  if (total == null || total <= 0 || rows <= 0 || elapsed <= 0) {
    return "";
  }
  const rate = rows / (elapsed / 1000);
  if (!Number.isFinite(rate) || rate <= 0) {
    return "";
  }
  const leftMs = ((total - rows) / rate) * 1000;
  if (!Number.isFinite(leftMs) || leftMs <= 0) {
    return "";
  }
  return `~${formatDuration(leftMs)} remaining`;
});

watch(
  () => [props.open, props.target] as const,
  ([open, target]) => {
    if (!open || !target) {
      return;
    }
    format.value = "sql";
    content.value = "";
    gzip.value = false;
    chunkSize.value = "0";
    customWhere.value = false;
    whereSQL.value = "";
    includeCreate.value = true;
    includeDrop.value = false;
    insertBatchSize.value = "500";
    wrapTransaction.value = true;
    fileName.value = suggestedExportFileName(target.name, "sql", false);
    folderMode.value = false;
    folderName.value = suggestedExportFolderName(target.name);
    prefixEnabled.value = false;
    prefix.value = "";
    overwriteWarn.value = false;
    outputDir.value = lastExportDir;
    checkedTotal.value = null;
    counting.value = false;
    browsing.value = false;
    progress.value = { chunk: 0, rows: 0, bytes: 0, query: "" };
    running.value = false;
    transferId.value = "";
    elapsedMs.value = 0;
    exportStartedAt.value = 0;
  },
);

watch(whereSQL, () => {
  checkedTotal.value = null;
});

watch(customWhere, (on) => {
  if (!on) {
    checkedTotal.value = null;
  }
});

Events.On("export:progress", (ev) => {
  if (!running.value) {
    return;
  }
  const data = ev.data as Progress | Progress[] | null;
  const next = Array.isArray(data) ? data[0] : data;
  if (!next || typeof next !== "object") {
    return;
  }
  progress.value = {
    chunk: Number(next.chunk) || 0,
    rows: Number(next.rows) || 0,
    bytes: Number(next.bytes) || 0,
    query: typeof next.query === "string" && next.query ? next.query : progress.value.query,
  };
  if (transferId.value) {
    transfers.update(transferId.value, {
      rows: progress.value.rows,
      bytes: progress.value.bytes,
    });
  }
});

onUnmounted(() => {
  pending.value?.cancel();
  stopClock();
});

function startClock() {
  exportStartedAt.value = Date.now();
  elapsedMs.value = 0;
  stopClock();
  tick = setInterval(() => {
    elapsedMs.value = Date.now() - exportStartedAt.value;
  }, 1000);
}

function stopClock() {
  if (tick) {
    clearInterval(tick);
    tick = null;
  }
}

function onOpenChange(next: boolean) {
  if (running.value && !next) {
    return;
  }
  emit("update:open", next);
}

function parseCount(value: string, fallback: number): number {
  const n = Number.parseInt(value, 10);
  return Number.isFinite(n) && n >= 0 ? n : fallback;
}

function rowsProgressLabel(): string {
  if (schemaExport.value) {
    return "Schema";
  }
  const rows = formatRows(progress.value.rows);
  if (checkedTotal.value != null) {
    return `${rows} / ${formatRows(checkedTotal.value)} rows`;
  }
  return `${rows} rows`;
}

async function choosePath() {
  const target = props.target;
  if (!target || browsing.value) {
    return;
  }
  browsing.value = true;
  try {
    const result = await ExportService.ChoosePath(
      target.name,
      format.value,
      gzip.value,
      true,
    );
    if (!result || result.canceled) {
      return;
    }
    const dir = result.dir || "";
    outputDir.value = dir;
    lastExportDir = dir;
  } catch (err) {
    toast.error(errorMessage(err));
  } finally {
    browsing.value = false;
  }
}

async function checkCount() {
  const target = props.target;
  if (!target || counting.value || !whereSQL.value.trim()) {
    return;
  }
  counting.value = true;
  try {
    const n = await ExportService.CountRows(
      target.profileId,
      target.database,
      target.schema,
      target.name,
      whereSQL.value.trim(),
    );
    checkedTotal.value = Number(n) || 0;
  } catch (err) {
    toast.error(errorMessage(err));
  } finally {
    counting.value = false;
  }
}

async function startExport() {
  const target = props.target;
  if (!target || running.value || !canExport.value) {
    return;
  }
  if (!overwriteWarn.value) {
    try {
      const dest = folderMode.value
        ? `${outputDir.value.trim().replace(/[/\\]+$/, "")}/${folderName.value.trim()}`
        : `${outputDir.value.trim().replace(/[/\\]+$/, "")}/${fileName.value.trim()}`;
      const status = await ExportService.CheckOutput(dest, folderMode.value);
      if (status?.hasFiles) {
        overwriteWarn.value = true;
        return;
      }
    } catch (err) {
      toast.error(errorMessage(err));
      return;
    }
  }
  const req: ExportTableRequest = {
    profileId: target.profileId,
    database: target.database,
    schema: target.schema,
    table: target.name,
    format: schemaExport.value ? "sql" : format.value,
    content: content.value,
    gzip: gzip.value,
    chunkSize: folderMode.value ? 0 : parseCount(chunkSize.value, 0),
    whereSQL: customWhere.value ? whereSQL.value.trim() : "",
    includeCreate: format.value === "sql" && includeCreate.value,
    includeDrop: format.value === "sql" && includeDrop.value,
    insertBatchSize: parseCount(insertBatchSize.value, 500) || 500,
    wrapTransaction: format.value === "sql" && wrapTransaction.value,
    outputPath: "",
    outputDir: outputDir.value.trim(),
    fileName: fileName.value.trim(),
    folderMode: folderMode.value,
    folderName: folderName.value.trim(),
    prefix: prefix.value,
    prefixEnabled: folderMode.value && prefixEnabled.value,
    force: overwriteWarn.value,
    deleteOnAbort: settings.deleteOnAbortExport,
  };

  lastExportDir = req.outputDir;
  overwriteWarn.value = false;

  running.value = true;
  progress.value = { chunk: 0, rows: 0, bytes: 0, query: "" };
  const id = transfers.start(target.name, schemaExport.value ? null : checkedTotal.value);
  transferId.value = id;
  startClock();
  const call = ExportService.ExportTable(req);
  pending.value = call;
  try {
    const result = await call;
    if (pending.value !== call) {
      return;
    }
    if (!result || result.canceled) {
      transfers.finish(id, {
        canceled: true,
        rows: progress.value.rows,
        bytes: progress.value.bytes,
      });
      emit("update:open", false);
      return;
    }
    transfers.finish(id, {
      rows: result.rows,
      bytes: result.bytes,
      files: result.files ?? [],
    });
    const files = result.files?.length ?? 0;
    toast.success(
      schemaExport.value
        ? files > 1
          ? `Exported schema to ${files} files`
          : "Exported schema"
        : files > 1
          ? `Exported ${formatRows(result.rows)} rows to ${files} files`
          : `Exported ${formatRows(result.rows)} rows`,
    );
    emit("update:open", false);
  } catch (err) {
    if (isCanceled(err) || pending.value !== call) {
      transfers.finish(id, {
        canceled: true,
        rows: progress.value.rows,
        bytes: progress.value.bytes,
      });
      emit("update:open", false);
      return;
    }
    if (isDirNotEmptyError(err)) {
      overwriteWarn.value = true;
      transfers.fail(id, "Destination is not empty");
      running.value = false;
      return;
    }
    transfers.fail(id, errorMessage(err));
    toast.error(errorMessage(err));
    running.value = false;
  } finally {
    stopClock();
    if (pending.value === call) {
      pending.value = null;
    }
  }
}

function cancelExport() {
  pending.value?.cancel();
}
</script>

<template>
  <Dialog :open="open" @update:open="onOpenChange">
    <DialogContent class="sm:max-w-lg" :show-close-button="!running">
      <DialogHeader>
        <DialogTitle>Export {{ target?.name ?? "table" }}</DialogTitle>
        <DialogDescription>
          Stream rows to a SQL or CSV dump. Optionally add a WHERE condition to export a subset.
        </DialogDescription>
      </DialogHeader>

      <div v-if="running" class="grid gap-3 py-2">
        <div class="flex items-center gap-2 text-sm">
          <LotusLoader class="size-4" label="Exporting" />
          <span>{{ progress.rows === 0 && progress.chunk === 0 ? "Starting…" : "Exporting…" }}</span>
        </div>
        <p class="text-muted-foreground text-sm">
          <template v-if="schemaExport">Schema</template>
          <template v-else>Chunk {{ progress.chunk || 1 }} · {{ rowsProgressLabel() }}</template>
          · {{ formatBytes(progress.bytes) }}
        </p>
        <p class="text-muted-foreground text-sm">
          Elapsed: {{ formatDuration(elapsedMs) }}
          <template v-if="remainingLabel"> · {{ remainingLabel }}</template>
        </p>
        <pre
          v-if="progress.query"
          class="bg-muted text-muted-foreground max-h-32 overflow-auto rounded-md p-2 font-mono text-xs whitespace-pre-wrap break-all"
        >{{ progress.query }}</pre>
      </div>

      <div v-else class="grid gap-3">
        <ExportOutputForm
          v-model:format="format"
          v-model:gzip="gzip"
          v-model:folder-mode="folderMode"
          v-model:file-name="fileName"
          v-model:folder-name="folderName"
          v-model:prefix-enabled="prefixEnabled"
          v-model:prefix="prefix"
          v-model:output-dir="outputDir"
          v-model:content="content"
          v-model:include-create="includeCreate"
          v-model:include-drop="includeDrop"
          v-model:wrap-transaction="wrapTransaction"
          v-model:insert-batch-size="insertBatchSize"
          v-model:chunk-size="chunkSize"
          v-model:custom-where="customWhere"
          v-model:where-sql="whereSQL"
          variant="table"
          :schema-disabled="schemaDisabled"
          :browsing="browsing"
          :counting="counting"
          :checked-total="checkedTotal"
          :overwrite-warn="overwriteWarn"
          @browse="choosePath"
          @check-count="checkCount"
        />
      </div>

      <DialogFooter>
        <Button v-if="running" type="button" variant="outline" @click="cancelExport">
          Cancel
        </Button>
        <Button v-else type="button" variant="outline" @click="emit('update:open', false)">
          Close
        </Button>
        <Button v-if="!running" type="button" :disabled="!canExport" @click="startExport">
          {{ overwriteWarn ? "Overwrite and export" : "Export" }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
