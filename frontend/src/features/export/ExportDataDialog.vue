<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from "vue";
import { Events } from "@wailsio/runtime";
import { Eye, RefreshCw, Table2 } from "@lucide/vue";
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
import DatabaseCombobox from "@/components/DatabaseCombobox.vue";
import {
  formatBytes,
  formatDuration,
  formatRows,
  isDirNotEmptyError,
  suggestedExportFileName,
  suggestedExportFolderName,
  withExportExt,
} from "@/features/export/format";
import { filterTables, sortDatabases } from "@/features/explorer/catalogRows";
import { errorMessage, isCanceled } from "@/lib/errors";
import {
  ConnectionsService,
  ExportService,
  QueryService,
  type BatchExportRequest,
  type TableInfo,
  type TableRef,
} from "@/lib/wails";
import { useConnectionsStore } from "@/stores/connections";
import { useSettingsStore } from "@/stores/settings";
import { useTransfersStore } from "@/stores/transfers";

const props = defineProps<{
  open: boolean;
  profileId?: string;
  database?: string;
}>();

const emit = defineEmits<{
  "update:open": [value: boolean];
}>();

type BatchProgress = {
  tableIndex: number;
  tableTotal: number;
  tableName: string;
  tableRows: number;
  totalRows: number;
  totalBytes: number;
  query?: string;
};

type Phase = "form" | "counting" | "exporting";

const connections = useConnectionsStore();
const settings = useSettingsStore();
const transfers = useTransfersStore();

let lastExportDir = "";

const profileId = ref("");
const database = ref("");
const databases = ref<string[]>([]);
const tables = ref<TableInfo[]>([]);
const selectedKeys = ref<Record<string, boolean>>({});
const tableQuery = ref("");
const exportFull = ref(true);
const loadingDbs = ref(false);
const loadingTables = ref(false);
const dbError = ref("");
const tableError = ref("");
const browsing = ref(false);

const format = ref("sql");
const singleFile = ref(true);
const gzip = ref(false);
const includeCreate = ref(true);
const includeDrop = ref(false);
const insertBatchSize = ref("500");
const wrapTransaction = ref(true);
const outputDir = ref("");
const fileName = ref("");
const folderName = ref("");
const prefixEnabled = ref(false);
const prefix = ref("");
const overwriteWarn = ref(false);

const phase = ref<Phase>("form");
const countedTotal = ref<number | null>(null);
const progress = ref<BatchProgress>({
  tableIndex: 0,
  tableTotal: 0,
  tableName: "",
  tableRows: 0,
  totalRows: 0,
  totalBytes: 0,
});
const pending = ref<{ cancel: () => void } | null>(null);
const transferId = ref("");
const exportStartedAt = ref(0);
const elapsedMs = ref(0);
let tick: ReturnType<typeof setInterval> | null = null;

const running = computed(() => phase.value !== "form");
const folderMode = computed(() => format.value === "csv" || !singleFile.value);
const selectedProfile = computed(() => connections.profiles.find((p) => p.id === profileId.value));
const showSchema = computed(
  () => new Set(tables.value.map((t) => t.schema).filter(Boolean)).size > 1,
);
const filteredTables = computed(() => filterTables(tables.value, tableQuery.value));
const selectedTables = computed(() => {
  if (exportFull.value) {
    return tables.value;
  }
  return tables.value.filter((t) => selectedKeys.value[tableKey(t)]);
});
const allFilteredSelected = computed(
  () =>
    filteredTables.value.length > 0 &&
    filteredTables.value.every((t) => selectedKeys.value[tableKey(t)]),
);
const someFilteredSelected = computed(() =>
  filteredTables.value.some((t) => selectedKeys.value[tableKey(t)]),
);
const canExport = computed(() => {
  if (!profileId.value || !database.value || selectedTables.value.length === 0) {
    return false;
  }
  if (!outputDir.value.trim()) {
    return false;
  }
  if (folderMode.value) {
    return folderName.value.trim().length > 0;
  }
  return fileName.value.trim().length > 0;
});
const percent = computed(() => {
  if (countedTotal.value && countedTotal.value > 0) {
    return Math.min(100, Math.round((progress.value.totalRows / countedTotal.value) * 100));
  }
  if (progress.value.tableTotal > 0) {
    return Math.min(100, Math.round((progress.value.tableIndex / progress.value.tableTotal) * 100));
  }
  return 0;
});
const remainingLabel = computed(() => {
  const total = countedTotal.value;
  const rows = progress.value.totalRows;
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
  () => [props.open, props.profileId, props.database] as const,
  ([open, nextProfile, nextDatabase]) => {
    if (!open) {
      return;
    }
    if (running.value) {
      return;
    }
    resetForm(nextProfile ?? "", nextDatabase ?? "");
  },
);

watch(format, (value) => {
  if (value === "csv") {
    singleFile.value = false;
  }
  if (fileName.value) {
    fileName.value = withExportExt(fileName.value, value, gzip.value);
  }
});

watch(gzip, (on) => {
  if (fileName.value) {
    fileName.value = withExportExt(fileName.value, format.value, on);
  }
});

Events.On("export:batch-progress", (ev) => {
  if (phase.value !== "exporting") {
    return;
  }
  const data = ev.data as BatchProgress | BatchProgress[] | null;
  const next = Array.isArray(data) ? data[0] : data;
  if (!next || typeof next !== "object") {
    return;
  }
  progress.value = {
    tableIndex: Number(next.tableIndex) || 0,
    tableTotal: Number(next.tableTotal) || 0,
    tableName: typeof next.tableName === "string" ? next.tableName : "",
    tableRows: Number(next.tableRows) || 0,
    totalRows: Number(next.totalRows) || 0,
    totalBytes: Number(next.totalBytes) || 0,
    query: typeof next.query === "string" && next.query ? next.query : progress.value.query,
  };
  if (transferId.value) {
    transfers.update(transferId.value, {
      rows: progress.value.totalRows,
      bytes: progress.value.totalBytes,
    });
  }
});

onUnmounted(() => {
  pending.value?.cancel();
  stopClock();
});

function tableKey(table: TableInfo | TableRef): string {
  return `${table.schema ?? ""}\t${table.name}`;
}

function tableLabel(table: TableInfo): string {
  if (showSchema.value && table.schema) {
    return `${table.schema}.${table.name}`;
  }
  return table.name;
}

function suggestNames(name: string) {
  const stem = name || selectedProfile.value?.name || "export";
  fileName.value = suggestedExportFileName(stem, format.value, gzip.value);
  folderName.value = suggestedExportFolderName(stem);
}

function onProfileChange(id: string) {
  profileId.value = id;
  database.value = "";
  tables.value = [];
  selectedKeys.value = {};
  databases.value = [];
  dbError.value = "";
  tableError.value = "";
  suggestNames(id);
  if (id) {
    void loadDatabases(id);
  }
}

function onDatabaseChange(name: string) {
  database.value = name;
  tables.value = [];
  selectedKeys.value = {};
  tableError.value = "";
  suggestNames(name);
  if (profileId.value && name) {
    void loadTables(profileId.value, name);
  }
}

function resetForm(nextProfile: string, nextDatabase: string) {
  profileId.value = nextProfile;
  database.value = nextDatabase;
  databases.value = [];
  tables.value = [];
  selectedKeys.value = {};
  tableQuery.value = "";
  exportFull.value = true;
  loadingDbs.value = false;
  loadingTables.value = false;
  dbError.value = "";
  tableError.value = "";
  format.value = "sql";
  singleFile.value = true;
  gzip.value = false;
  includeCreate.value = true;
  includeDrop.value = false;
  insertBatchSize.value = "500";
  wrapTransaction.value = true;
  outputDir.value = lastExportDir;
  prefixEnabled.value = false;
  prefix.value = "";
  overwriteWarn.value = false;
  suggestNames(nextDatabase || nextProfile);
  phase.value = "form";
  countedTotal.value = null;
  progress.value = {
    tableIndex: 0,
    tableTotal: 0,
    tableName: "",
    tableRows: 0,
    totalRows: 0,
    totalBytes: 0,
  };
  transferId.value = "";
  elapsedMs.value = 0;
  exportStartedAt.value = 0;
  if (nextProfile) {
    void loadDatabases(nextProfile, nextDatabase);
  }
}

async function loadDatabases(id: string, keepDatabase = "") {
  loadingDbs.value = true;
  dbError.value = "";
  try {
    const list = sortDatabases((await ConnectionsService.Databases(id)) ?? []);
    if (profileId.value !== id) {
      return;
    }
    databases.value = list;
    const preferred = keepDatabase && list.includes(keepDatabase) ? keepDatabase : database.value;
    if (preferred && list.includes(preferred)) {
      database.value = preferred;
    } else if (!database.value && list.length === 1) {
      database.value = list[0];
    } else if (database.value && !list.includes(database.value)) {
      database.value = "";
    }
    if (database.value) {
      await loadTables(id, database.value);
    }
  } catch (err) {
    if (isCanceled(err) || profileId.value !== id) {
      return;
    }
    dbError.value = errorMessage(err);
    toast.error(errorMessage(err));
  } finally {
    if (profileId.value === id) {
      loadingDbs.value = false;
    }
  }
}

async function loadTables(id: string, db: string) {
  loadingTables.value = true;
  tableError.value = "";
  try {
    const list = (await QueryService.ListTables(id, db)) ?? [];
    if (profileId.value !== id || database.value !== db) {
      return;
    }
    tables.value = list;
    const next: Record<string, boolean> = {};
    for (const table of list) {
      next[tableKey(table)] = true;
    }
    selectedKeys.value = next;
  } catch (err) {
    if (isCanceled(err) || profileId.value !== id) {
      return;
    }
    tables.value = [];
    tableError.value = errorMessage(err);
    toast.error(errorMessage(err));
  } finally {
    if (profileId.value === id && database.value === db) {
      loadingTables.value = false;
    }
  }
}

function toggleFiltered(on: boolean) {
  const next = { ...selectedKeys.value };
  for (const table of filteredTables.value) {
    next[tableKey(table)] = on;
  }
  selectedKeys.value = next;
}

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
  emit("update:open", next);
}

function parseCount(value: string, fallback: number): number {
  const n = Number.parseInt(value, 10);
  return Number.isFinite(n) && n >= 0 ? n : fallback;
}

async function choosePath() {
  if (browsing.value) {
    return;
  }
  browsing.value = true;
  try {
    const result = await ExportService.ChoosePath(
      database.value || "export",
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

function asRefs(list: TableInfo[]): TableRef[] {
  return list.map((table) => ({
    database: table.database || database.value,
    schema: table.schema,
    name: table.name,
  }));
}

function outputCheckPath(): { path: string; folder: boolean } {
  const dir = outputDir.value.trim();
  if (folderMode.value) {
    const name = folderName.value.trim();
    return { path: name ? `${dir.replace(/[/\\]+$/, "")}/${name}` : dir, folder: true };
  }
  return { path: `${dir.replace(/[/\\]+$/, "")}/${fileName.value.trim()}`, folder: false };
}

async function startExport() {
  if (running.value || !canExport.value) {
    return;
  }
  const dest = outputCheckPath();
  if (!overwriteWarn.value) {
    try {
      const status = await ExportService.CheckOutput(dest.path, dest.folder);
      if (status?.hasFiles) {
        overwriteWarn.value = true;
        return;
      }
    } catch (err) {
      toast.error(errorMessage(err));
      return;
    }
  }

  const targets = asRefs(selectedTables.value);
  lastExportDir = outputDir.value.trim();
  phase.value = "counting";
  countedTotal.value = null;
  progress.value = {
    tableIndex: 0,
    tableTotal: targets.length,
    tableName: "",
    tableRows: 0,
    totalRows: 0,
    totalBytes: 0,
  };
  overwriteWarn.value = false;
  startClock();

  const countCall = ExportService.CountTables({
    profileId: profileId.value,
    database: database.value,
    tables: targets,
  });
  pending.value = countCall;
  try {
    const counts = (await countCall) ?? [];
    if (pending.value !== countCall) {
      stopClock();
      phase.value = "form";
      return;
    }
    countedTotal.value = counts.reduce((sum, row) => sum + (Number(row.rows) || 0), 0);
  } catch (err) {
    const mine = pending.value === countCall;
    if (mine) {
      pending.value = null;
    }
    stopClock();
    phase.value = "form";
    if (isCanceled(err) || !mine) {
      emit("update:open", false);
      return;
    }
    toast.error(errorMessage(err));
    return;
  }

  phase.value = "exporting";
  const req: BatchExportRequest = {
    profileId: profileId.value,
    database: database.value,
    tables: targets,
    format: format.value,
    gzip: gzip.value,
    singleFile: !folderMode.value,
    outputDir: outputDir.value.trim(),
    fileName: fileName.value.trim(),
    folderName: folderName.value.trim(),
    prefix: prefix.value,
    prefixEnabled: folderMode.value && prefixEnabled.value,
    force: true,
    includeCreate: format.value === "sql" && includeCreate.value,
    includeDrop: format.value === "sql" && includeDrop.value,
    insertBatchSize: parseCount(insertBatchSize.value, 500) || 500,
    wrapTransaction: format.value === "sql" && wrapTransaction.value,
    deleteOnAbort: settings.deleteOnAbortExport,
  };
  const label = folderMode.value
    ? folderName.value.trim() || database.value
    : fileName.value.trim() || database.value;
  const id = transfers.start(label, countedTotal.value);
  transferId.value = id;
  const call = ExportService.ExportTables(req);
  pending.value = call;
  try {
    const result = await call;
    if (pending.value !== call) {
      return;
    }
    if (!result || result.canceled) {
      transfers.finish(id, {
        canceled: true,
        rows: progress.value.totalRows,
        bytes: progress.value.totalBytes,
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
        rows: progress.value.totalRows,
        bytes: progress.value.totalBytes,
      });
      emit("update:open", false);
      return;
    }
    if (isDirNotEmptyError(err)) {
      overwriteWarn.value = true;
      phase.value = "form";
      transfers.fail(id, "Destination is not empty");
      return;
    }
    transfers.fail(id, errorMessage(err));
    toast.error(errorMessage(err));
    phase.value = "form";
  } finally {
    stopClock();
    if (pending.value === call || pending.value === countCall) {
      pending.value = null;
    }
    phase.value = "form";
  }
}

function cancelExport() {
  pending.value?.cancel();
}
</script>

<template>
  <Dialog :open="open" @update:open="onOpenChange">
    <DialogContent class="sm:max-w-lg max-h-[90vh] overflow-y-auto" :show-close-button="true">
      <DialogHeader>
        <DialogTitle>Export</DialogTitle>
        <DialogDescription>
          Choose a database and tables, then stream a SQL or CSV dump.
        </DialogDescription>
      </DialogHeader>

      <div v-if="running" class="grid gap-3 py-2">
        <div class="flex items-center gap-2 text-sm">
          <Spinner class="size-4" />
          <span v-if="phase === 'counting'">
            Preparing… Counting rows in {{ selectedTables.length }} tables…
          </span>
          <span v-else>
            {{ progress.totalRows === 0 && progress.tableIndex === 0 ? "Starting…" : "Exporting…" }}
            <template v-if="progress.tableTotal">
              · table {{ progress.tableIndex || 1 }} of {{ progress.tableTotal }}
              <template v-if="progress.tableName"> ({{ progress.tableName }})</template>
            </template>
          </span>
        </div>
        <div class="bg-muted h-2 w-full overflow-hidden rounded-full">
          <div class="bg-primary h-full transition-all" :style="{ width: `${percent}%` }" />
        </div>
        <p class="text-muted-foreground text-sm">
          {{ percent }}%
          ·
          <template v-if="countedTotal != null">
            {{ formatRows(progress.totalRows) }} / {{ formatRows(countedTotal) }} rows
          </template>
          <template v-else>{{ formatRows(progress.totalRows) }} rows</template>
          · {{ formatBytes(progress.totalBytes) }}
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
          <Label>Connection</Label>
          <Select :model-value="profileId" @update:model-value="(v) => onProfileChange(String(v ?? ''))">
            <SelectTrigger class="w-full">
              <SelectValue placeholder="Select connection" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="profile in connections.profiles" :key="profile.id" :value="profile.id">
                {{ profile.name }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div class="grid gap-1.5">
          <Label>Database</Label>
          <div class="flex items-center gap-1.5">
            <DatabaseCombobox
              class="h-8 min-w-0 flex-1"
              :model-value="database"
              :options="databases"
              :disabled="!profileId || loadingDbs"
              placeholder="Select database"
              @update:model-value="onDatabaseChange"
            />
            <Button
              type="button"
              size="icon-sm"
              variant="outline"
              :disabled="!profileId || loadingDbs"
              title="Reload databases"
              @click="profileId && loadDatabases(profileId, database)"
            >
              <RefreshCw :class="loadingDbs ? 'size-3.5 animate-spin' : 'size-3.5'" />
            </Button>
          </div>
          <p v-if="dbError" class="text-destructive text-xs">{{ dbError }}</p>
        </div>

        <div class="flex items-center justify-between gap-4">
          <Label for="export-full-db">Export full database</Label>
          <Switch
            id="export-full-db"
            :model-value="exportFull"
            :disabled="!database"
            @update:model-value="(v) => (exportFull = v === true)"
          />
        </div>
        <p v-if="exportFull && loadingTables" class="text-muted-foreground text-xs">Loading tables…</p>
        <p v-else-if="exportFull && tableError" class="text-destructive text-xs">{{ tableError }}</p>
        <p v-else-if="exportFull && database && tables.length === 0 && !loadingTables" class="text-muted-foreground text-xs">
          No tables in this database.
        </p>

        <div v-if="!exportFull" class="grid gap-1.5">
          <div class="flex items-center gap-2">
            <Checkbox
              :model-value="allFilteredSelected ? true : someFilteredSelected ? 'indeterminate' : false"
              :disabled="filteredTables.length === 0"
              @update:model-value="() => toggleFiltered(!allFilteredSelected)"
            />
            <Label class="font-normal">Tables</Label>
            <span class="text-muted-foreground text-xs">
              {{ selectedTables.length }} selected
            </span>
          </div>
          <Input v-model="tableQuery" placeholder="Filter tables…" />
          <p v-if="loadingTables" class="text-muted-foreground text-xs">Loading tables…</p>
          <p v-else-if="tableError" class="text-destructive text-xs">{{ tableError }}</p>
          <p v-else-if="tables.length === 0" class="text-muted-foreground text-xs">No tables.</p>
          <p v-else-if="filteredTables.length === 0" class="text-muted-foreground text-xs">
            No matching tables.
          </p>
          <div v-else class="max-h-48 overflow-auto rounded-md border p-1">
            <label
              v-for="table in filteredTables"
              :key="tableKey(table)"
              class="hover:bg-accent flex cursor-pointer items-center gap-2 rounded-md px-2 py-1 text-sm"
            >
              <Checkbox
                :model-value="Boolean(selectedKeys[tableKey(table)])"
                @update:model-value="
                  (v) => (selectedKeys = { ...selectedKeys, [tableKey(table)]: Boolean(v) })
                "
              />
              <Eye v-if="table.type === 'view'" class="text-muted-foreground size-3.5 shrink-0" />
              <Table2 v-else class="text-muted-foreground size-3.5 shrink-0" />
              <span class="truncate">{{ tableLabel(table) }}</span>
            </label>
          </div>
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

        <div v-if="format === 'sql'" class="flex items-center justify-between gap-4">
          <Label for="export-single-file">Single file</Label>
          <Switch
            id="export-single-file"
            :model-value="singleFile"
            @update:model-value="(v) => (singleFile = v === true)"
          />
        </div>

        <div v-if="folderMode" class="grid gap-1.5">
          <Label for="export-foldername">Folder name</Label>
          <Input
            id="export-foldername"
            v-model="folderName"
            placeholder="database_20260101"
            spellcheck="false"
          />
          <p class="text-muted-foreground text-xs">
            Each table is saved as a separate file.
          </p>
        </div>
        <div v-else class="grid gap-1.5">
          <Label for="export-batch-filename">Filename</Label>
          <Input
            id="export-batch-filename"
            v-model="fileName"
            placeholder="database.sql"
            spellcheck="false"
          />
        </div>

        <div class="grid gap-1.5">
          <Label for="export-batch-path">Path</Label>
          <InputGroup>
            <InputGroupInput
              id="export-batch-path"
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

        <div v-if="folderMode" class="flex items-center justify-between gap-4">
          <Label for="export-prefix-on">Filename prefix</Label>
          <Switch
            id="export-prefix-on"
            :model-value="prefixEnabled"
            @update:model-value="(v) => (prefixEnabled = v === true)"
          />
        </div>
        <div v-if="folderMode && prefixEnabled" class="grid gap-1.5">
          <Input v-model="prefix" placeholder="prod_" spellcheck="false" />
        </div>

        <div class="flex items-center gap-2">
          <Checkbox
            id="export-batch-gzip"
            :model-value="gzip"
            @update:model-value="(v) => (gzip = Boolean(v))"
          />
          <Label for="export-batch-gzip" class="font-normal">Compress with gzip</Label>
        </div>

        <template v-if="format === 'sql'">
          <div class="flex items-center gap-2">
            <Checkbox
              id="export-batch-create"
              :model-value="includeCreate"
              @update:model-value="(v) => (includeCreate = Boolean(v))"
            />
            <Label for="export-batch-create" class="font-normal">Include CREATE TABLE</Label>
          </div>
          <div class="flex items-center gap-2">
            <Checkbox
              id="export-batch-drop"
              :model-value="includeDrop"
              @update:model-value="(v) => (includeDrop = Boolean(v))"
            />
            <Label for="export-batch-drop" class="font-normal">Include DROP TABLE IF EXISTS</Label>
          </div>
          <div class="flex items-center gap-2">
            <Checkbox
              id="export-batch-tx"
              :model-value="wrapTransaction"
              @update:model-value="(v) => (wrapTransaction = Boolean(v))"
            />
            <Label for="export-batch-tx" class="font-normal">Wrap in a transaction</Label>
          </div>
          <div class="grid gap-1.5">
            <Label for="export-batch-batch">Insert batch size</Label>
            <Input id="export-batch-batch" v-model="insertBatchSize" type="number" min="1" />
          </div>
        </template>

        <div class="flex items-center gap-2">
          <Checkbox
            id="export-batch-delete-abort"
            :model-value="settings.deleteOnAbortExport"
            @update:model-value="(v) => settings.setDeleteOnAbortExport(Boolean(v))"
          />
          <Label for="export-batch-delete-abort" class="font-normal">Delete files on abort/error</Label>
        </div>

        <p v-if="overwriteWarn" class="text-destructive text-xs">
          Files in this destination will be overwritten.
        </p>
      </div>

      <DialogFooter>
        <Button v-if="running" type="button" variant="ghost" @click="emit('update:open', false)">
          Close
        </Button>
        <Button v-if="running" type="button" variant="outline" @click="cancelExport">
          Cancel export
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
