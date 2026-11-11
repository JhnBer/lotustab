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
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import LotusLoader from "@/components/LotusLoader.vue";
import { Switch } from "@/components/ui/switch";
import DatabaseCombobox from "@/components/DatabaseCombobox.vue";
import ExportOutputForm from "@/features/export/ExportOutputForm.vue";
import {
  formatBytes,
  formatDuration,
  formatRows,
  isDirNotEmptyError,
  isSchemaExport,
  suggestedExportFileName,
  suggestedExportFolderName,
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
const content = ref("");
const gzip = ref(false);
const folderMode = ref(false);
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
const schemaExport = computed(() => isSchemaExport(content.value));
const selectedProfile = computed(() => connections.profiles.find((p) => p.id === profileId.value));
const showSchema = computed(
  () => new Set(tables.value.map((t) => t.schema).filter(Boolean)).size > 1,
);
const filteredTables = computed(() => filterTables(tables.value, tableQuery.value));
const selectedTables = computed(() => {
  const rows = exportFull.value
    ? tables.value
    : tables.value.filter((t) => selectedKeys.value[tableKey(t)]);
  if (!schemaExport.value) {
    return rows;
  }
  return rows.filter((t) => t.type !== "view");
});
const choosableTables = computed(() =>
  filteredTables.value.filter((table) => !(schemaExport.value && table.type === "view")),
);
const allFilteredSelected = computed(
  () =>
    choosableTables.value.length > 0 &&
    choosableTables.value.every((t) => selectedKeys.value[tableKey(t)]),
);
const someFilteredSelected = computed(() =>
  choosableTables.value.some((t) => selectedKeys.value[tableKey(t)]),
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

watch(schemaExport, (on) => {
  if (!on) {
    return;
  }
  const next = { ...selectedKeys.value };
  for (const table of tables.value) {
    if (table.type === "view") {
      next[tableKey(table)] = false;
    }
  }
  selectedKeys.value = next;
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
  content.value = "";
  folderMode.value = false;
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
    if (schemaExport.value && table.type === "view") {
      next[tableKey(table)] = false;
      continue;
    }
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

  let countCall: ReturnType<typeof ExportService.CountTables> | null = null;
  if (!schemaExport.value) {
    phase.value = "counting";
    countCall = ExportService.CountTables({
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
  }

  phase.value = "exporting";
  const req: BatchExportRequest = {
    profileId: profileId.value,
    database: database.value,
    tables: targets,
    format: schemaExport.value ? "sql" : format.value,
    content: content.value,
    gzip: gzip.value,
    singleFile: !folderMode.value,
    outputDir: outputDir.value.trim(),
    fileName: fileName.value.trim(),
    folderName: folderName.value.trim(),
    prefix: prefix.value,
    prefixEnabled: folderMode.value && prefixEnabled.value,
    force: true,
    includeCreate: !schemaExport.value && format.value === "sql" && includeCreate.value,
    includeDrop: !schemaExport.value && format.value === "sql" && includeDrop.value,
    insertBatchSize: parseCount(insertBatchSize.value, 500) || 500,
    wrapTransaction: !schemaExport.value && format.value === "sql" && wrapTransaction.value,
    deleteOnAbort: settings.deleteOnAbortExport,
  };
  const label = folderMode.value
    ? folderName.value.trim() || database.value
    : fileName.value.trim() || database.value;
  const id = transfers.start(label, schemaExport.value ? null : countedTotal.value);
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
          <LotusLoader class="size-4" label="Exporting" />
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
          <template v-if="schemaExport">Schema</template>
          <template v-else-if="countedTotal != null">
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
        <p v-if="schemaExport && tables.some((table) => table.type === 'view')" class="text-muted-foreground text-xs">
          Views are not included in a schema export.
        </p>

        <div v-if="!exportFull" class="grid gap-1.5">
          <div class="flex items-center gap-2">
            <Checkbox
              :model-value="allFilteredSelected ? true : someFilteredSelected ? 'indeterminate' : false"
              :disabled="choosableTables.length === 0"
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
              class="hover:bg-accent flex items-center gap-2 rounded-md px-2 py-1 text-sm"
              :class="schemaExport && table.type === 'view' ? 'cursor-not-allowed opacity-50' : 'cursor-pointer'"
            >
              <Checkbox
                :model-value="Boolean(selectedKeys[tableKey(table)])"
                :disabled="schemaExport && table.type === 'view'"
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
          variant="batch"
          :browsing="browsing"
          :overwrite-warn="overwriteWarn"
          @browse="choosePath"
        />
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
