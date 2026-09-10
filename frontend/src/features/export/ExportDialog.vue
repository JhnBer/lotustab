<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from "vue";
import { Events } from "@wailsio/runtime";
import { toast } from "vue-sonner";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "@/components/ui/input-group";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { formatBytes, formatDuration, formatRows, suggestedExportFileName, withExportExt } from "@/features/export/format";
import { errorMessage, isCanceled } from "@/lib/errors";
import { ExportService, type ExportTableRequest } from "@/lib/wails";
import { useSettingsStore } from "@/stores/settings";
import { useTransfersStore } from "@/stores/transfers";

export type ExportTarget = {
  profileId: string;
  database: string;
  schema: string;
  name: string;
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

const canExport = computed(
  () => fileName.value.trim().length > 0 && outputDir.value.trim().length > 0,
);
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
    gzip.value = false;
    chunkSize.value = "0";
    customWhere.value = false;
    whereSQL.value = "";
    includeCreate.value = true;
    includeDrop.value = false;
    insertBatchSize.value = "500";
    wrapTransaction.value = true;
    fileName.value = suggestedExportFileName(target.name, "sql", false);
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

watch([format, gzip], () => {
  fileName.value = withExportExt(fileName.value, format.value, gzip.value);
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
  const req: ExportTableRequest = {
    profileId: target.profileId,
    database: target.database,
    schema: target.schema,
    table: target.name,
    format: format.value,
    gzip: gzip.value,
    chunkSize: parseCount(chunkSize.value, 0),
    whereSQL: customWhere.value ? whereSQL.value.trim() : "",
    includeCreate: format.value === "sql" && includeCreate.value,
    includeDrop: format.value === "sql" && includeDrop.value,
    insertBatchSize: parseCount(insertBatchSize.value, 500) || 500,
    wrapTransaction: format.value === "sql" && wrapTransaction.value,
    outputPath: "",
    outputDir: outputDir.value.trim(),
    fileName: fileName.value.trim(),
    deleteOnAbort: settings.deleteOnAbortExport,
  };

  lastExportDir = req.outputDir;

  running.value = true;
  progress.value = { chunk: 0, rows: 0, bytes: 0, query: "" };
  const id = transfers.start(target.name, checkedTotal.value);
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
      files > 1
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
          <Spinner class="size-4" />
          <span>{{ progress.rows === 0 && progress.chunk === 0 ? "Starting…" : "Exporting…" }}</span>
        </div>
        <p class="text-muted-foreground text-sm">
          Chunk {{ progress.chunk || 1 }}
          · {{ rowsProgressLabel() }}
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
        <div class="grid gap-1.5">
          <Label for="export-filename">Filename</Label>
          <Input
            id="export-filename"
            v-model="fileName"
            placeholder="table.sql"
            spellcheck="false"
          />
        </div>

        <div class="grid gap-1.5">
          <Label for="export-path">Path</Label>
          <InputGroup>
            <InputGroupInput
              id="export-path"
              v-model="outputDir"
              placeholder="Choose a folder…"
              spellcheck="false"
            />
            <InputGroupAddon align="inline-end">
              <InputGroupButton :disabled="browsing" @click="choosePath">
                <Spinner v-if="browsing" class="size-3.5" />
                Browse…
              </InputGroupButton>
            </InputGroupAddon>
          </InputGroup>
        </div>

        <div class="grid gap-1.5">
          <Label>Format</Label>
          <Select :model-value="format" @update:model-value="(v) => (format = String(v ?? 'sql'))">
            <SelectTrigger class="w-full">
              <SelectValue placeholder="Format" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="sql">SQL</SelectItem>
              <SelectItem value="csv">CSV</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div class="grid gap-1.5">
          <Label for="export-chunk">Chunk size (rows per file)</Label>
          <Input
            id="export-chunk"
            v-model="chunkSize"
            type="number"
            min="0"
            placeholder="0 = single file"
          />
        </div>

        <div class="flex items-center justify-between gap-4">
          <Label for="export-where-on">Custom where condition</Label>
          <Switch
            id="export-where-on"
            :model-value="customWhere"
            @update:model-value="(v) => (customWhere = v === true)"
          />
        </div>

        <template v-if="customWhere">
          <div class="grid gap-1.5">
            <Textarea
              id="export-where"
              v-model="whereSQL"
              placeholder="created_at >= NOW() - INTERVAL '7 days'"
              class="font-mono text-xs"
              rows="2"
            />
            <div v-if="whereSQL.trim()" class="flex items-center gap-2">
              <Button
                type="button"
                size="sm"
                variant="outline"
                :disabled="counting"
                @click="checkCount"
              >
                <Spinner v-if="counting" class="size-3.5" />
                Check count
              </Button>
              <span v-if="checkedTotal != null" class="text-muted-foreground text-xs">
                ~{{ formatRows(checkedTotal) }} rows match
              </span>
            </div>
          </div>
        </template>

        <div class="flex items-center gap-2">
          <Checkbox
            id="export-gzip"
            :model-value="gzip"
            @update:model-value="(v) => (gzip = Boolean(v))"
          />
          <Label for="export-gzip" class="font-normal">Compress with gzip</Label>
        </div>

        <template v-if="format === 'sql'">
          <div class="flex items-center gap-2">
            <Checkbox
              id="export-create"
              :model-value="includeCreate"
              @update:model-value="(v) => (includeCreate = Boolean(v))"
            />
            <Label for="export-create" class="font-normal">Include CREATE TABLE</Label>
          </div>
          <div class="flex items-center gap-2">
            <Checkbox
              id="export-drop"
              :model-value="includeDrop"
              @update:model-value="(v) => (includeDrop = Boolean(v))"
            />
            <Label for="export-drop" class="font-normal">Include DROP TABLE IF EXISTS</Label>
          </div>
          <div class="flex items-center gap-2">
            <Checkbox
              id="export-tx"
              :model-value="wrapTransaction"
              @update:model-value="(v) => (wrapTransaction = Boolean(v))"
            />
            <Label for="export-tx" class="font-normal">Wrap in a transaction</Label>
          </div>
          <div class="grid gap-1.5">
            <Label for="export-batch">Insert batch size</Label>
            <Input id="export-batch" v-model="insertBatchSize" type="number" min="1" />
          </div>
        </template>

        <div class="flex items-center gap-2">
          <Checkbox
            id="export-delete-abort"
            :model-value="settings.deleteOnAbortExport"
            @update:model-value="(v) => settings.setDeleteOnAbortExport(Boolean(v))"
          />
          <Label for="export-delete-abort" class="font-normal">Delete files on abort/error</Label>
        </div>
      </div>

      <DialogFooter>
        <Button v-if="running" type="button" variant="outline" @click="cancelExport">
          Cancel
        </Button>
        <Button v-else type="button" variant="outline" @click="emit('update:open', false)">
          Close
        </Button>
        <Button v-if="!running" type="button" :disabled="!canExport" @click="startExport">
          Export
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
