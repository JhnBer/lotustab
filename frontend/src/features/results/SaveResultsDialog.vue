<script setup lang="ts">
import { computed, ref, watch } from "vue";
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
import {
  isDirNotEmptyError,
  suggestedExportFileName,
  suggestedExportFolderName,
  withExportExt,
} from "@/features/export/format";
import { statementLabel, type CopyableResult } from "@/features/results/copyResults";
import { errorMessage, isCanceled } from "@/lib/errors";
import { ExportService, type SaveResultItem, type SaveResultSetsRequest } from "@/lib/wails";

const props = defineProps<{
  open: boolean;
  profileId: string;
  sets: CopyableResult[];
}>();

const emit = defineEmits<{
  "update:open": [value: boolean];
}>();

let lastSaveDir = "";

const format = ref("sql");
const singleFile = ref(true);
const separateTables = ref(true);
const tableName = ref("");
const tableNames = ref<Record<number, string>>({});
const selectedKeys = ref<Record<number, boolean>>({});
const outputDir = ref("");
const fileName = ref("");
const folderName = ref("");
const overwriteWarn = ref(false);
const browsing = ref(false);
const saving = ref(false);

const multi = computed(() => props.sets.length > 1);
const selectedSets = computed(() => props.sets.filter((item) => selectedKeys.value[item.index]));
const folderMode = computed(() => multi.value && selectedSets.value.length > 1 && !singleFile.value);
const allSelected = computed(
  () => props.sets.length > 0 && props.sets.every((item) => selectedKeys.value[item.index]),
);
const someSelected = computed(() => props.sets.some((item) => selectedKeys.value[item.index]));

const canSave = computed(() => {
  if (saving.value || selectedSets.value.length === 0 || !outputDir.value.trim()) {
    return false;
  }
  if (folderMode.value) {
    if (!folderName.value.trim()) {
      return false;
    }
  } else if (!fileName.value.trim()) {
    return false;
  }
  if (format.value !== "sql") {
    return true;
  }
  if (multi.value && separateTables.value) {
    return selectedSets.value.every((item) => tableNames.value[item.index]?.trim());
  }
  return tableName.value.trim().length > 0;
});

watch(
  () => props.open,
  (open) => {
    if (!open) {
      return;
    }
    format.value = "sql";
    singleFile.value = true;
    separateTables.value = true;
    tableName.value = "";
    overwriteWarn.value = false;
    outputDir.value = lastSaveDir;
    const names: Record<number, string> = {};
    const keys: Record<number, boolean> = {};
    for (const item of props.sets) {
      names[item.index] = "";
      keys[item.index] = true;
    }
    tableNames.value = names;
    selectedKeys.value = keys;
    suggestNames();
  },
);

watch(format, () => {
  suggestNames();
  overwriteWarn.value = false;
});

watch(singleFile, () => {
  overwriteWarn.value = false;
});

function suggestNames() {
  fileName.value = suggestedExportFileName("query", format.value, false);
  folderName.value = suggestedExportFolderName("query");
}

function onOpenChange(next: boolean) {
  if (!saving.value) {
    emit("update:open", next);
  }
}

function toggleFiltered(on: boolean) {
  const next = { ...selectedKeys.value };
  for (const item of props.sets) {
    next[item.index] = on;
  }
  selectedKeys.value = next;
}

function outputCheckPath(): { path: string; folder: boolean } {
  const dir = outputDir.value.trim();
  if (folderMode.value) {
    const name = folderName.value.trim();
    return { path: name ? `${dir.replace(/[/\\]+$/, "")}/${name}` : dir, folder: true };
  }
  return { path: `${dir.replace(/[/\\]+$/, "")}/${fileName.value.trim()}`, folder: false };
}

async function choosePath() {
  if (browsing.value) {
    return;
  }
  browsing.value = true;
  try {
    const result = await ExportService.ChoosePath("query", format.value, false, true);
    if (!result || result.canceled) {
      return;
    }
    const dir = result.dir || "";
    outputDir.value = dir;
    lastSaveDir = dir;
  } catch (err) {
    toast.error(errorMessage(err));
  } finally {
    browsing.value = false;
  }
}

function items(): SaveResultItem[] {
  const shared = tableName.value.trim();
  return selectedSets.value.map((item) => ({
    index: item.index,
    sql: item.sql,
    fileBase: `query_${item.index + 1}`,
    tableName:
      format.value === "sql" && multi.value && separateTables.value
        ? (tableNames.value[item.index] ?? "").trim()
        : shared,
    columns: item.columns,
    rows: item.rows,
  }));
}

async function save() {
  if (saving.value || !canSave.value) {
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

  const req: SaveResultSetsRequest = {
    profileId: props.profileId,
    format: format.value,
    singleFile: !folderMode.value,
    outputDir: outputDir.value.trim(),
    fileName: withExportExt(fileName.value.trim(), format.value, false),
    folderName: folderName.value.trim(),
    force: true,
    items: items(),
  };
  saving.value = true;
  lastSaveDir = outputDir.value.trim();
  try {
    const result = await ExportService.SaveResultSets(req);
    const files = result?.files?.length ?? 0;
    const rows = result?.rows ?? 0;
    toast.success(
      files > 1 ? `Saved ${rows} rows to ${files} files` : `Saved ${rows} rows`,
    );
    emit("update:open", false);
  } catch (err) {
    if (isCanceled(err)) {
      return;
    }
    if (isDirNotEmptyError(err)) {
      overwriteWarn.value = true;
      return;
    }
    toast.error(errorMessage(err));
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <Dialog :open="open" @update:open="onOpenChange">
    <DialogContent class="sm:max-w-lg max-h-[90vh] overflow-y-auto" :show-close-button="true">
      <DialogHeader>
        <DialogTitle>Save results</DialogTitle>
        <DialogDescription>
          Write the current query results as SQL inserts or JSON.
        </DialogDescription>
      </DialogHeader>

      <div class="grid gap-3">
        <div class="grid gap-1.5">
          <Label>Format</Label>
          <Select :model-value="format" @update:model-value="(v) => (format = String(v ?? 'sql'))">
            <SelectTrigger class="w-full">
              <SelectValue placeholder="Format" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="sql">SQL</SelectItem>
              <SelectItem value="json">JSON</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div v-if="multi" class="grid gap-1.5">
          <div class="flex items-center gap-2">
            <Checkbox
              :model-value="allSelected ? true : someSelected ? 'indeterminate' : false"
              @update:model-value="() => toggleFiltered(!allSelected)"
            />
            <Label class="font-normal">Results</Label>
            <span class="text-muted-foreground text-xs">
              {{ selectedSets.length }} selected
            </span>
          </div>
          <div class="max-h-48 overflow-auto rounded-md border p-1">
            <label
              v-for="item in sets"
              :key="item.index"
              class="hover:bg-accent flex cursor-pointer items-start gap-2 rounded-md px-2 py-1.5 text-sm"
            >
              <Checkbox
                class="mt-0.5"
                :model-value="Boolean(selectedKeys[item.index])"
                @update:model-value="
                  (v) => (selectedKeys = { ...selectedKeys, [item.index]: Boolean(v) })
                "
              />
              <span class="min-w-0 flex-1 truncate" :title="item.sql">{{ statementLabel(item) }}</span>
            </label>
          </div>
        </div>

        <template v-if="format === 'sql'">
          <div v-if="multi" class="flex items-center justify-between gap-4">
            <Label for="save-separate-tables">Insert into separate tables</Label>
            <Switch
              id="save-separate-tables"
              :model-value="separateTables"
              @update:model-value="(v) => (separateTables = v === true)"
            />
          </div>
          <div v-if="multi && separateTables" class="grid gap-1.5">
            <Label>Table names</Label>
            <div class="max-h-48 space-y-1.5 overflow-auto rounded-md border p-2">
              <div
                v-for="item in selectedSets"
                :key="item.index"
                class="grid gap-1"
              >
                <Label class="text-muted-foreground text-xs font-normal" :title="item.sql">
                  {{ statementLabel(item) }}
                </Label>
                <Input
                  :model-value="tableNames[item.index] ?? ''"
                  placeholder="table"
                  spellcheck="false"
                  @update:model-value="
                    (v) => (tableNames = { ...tableNames, [item.index]: String(v ?? '') })
                  "
                />
              </div>
            </div>
          </div>
          <div v-else class="grid gap-1.5">
            <Label for="save-table-name">Table name</Label>
            <Input
              id="save-table-name"
              v-model="tableName"
              placeholder="table"
              spellcheck="false"
            />
          </div>
        </template>

        <div v-if="multi && selectedSets.length > 1" class="flex items-center justify-between gap-4">
          <Label for="save-single-file">Single file</Label>
          <Switch
            id="save-single-file"
            :model-value="singleFile"
            @update:model-value="(v) => (singleFile = v === true)"
          />
        </div>

        <div v-if="folderMode" class="grid gap-1.5">
          <Label for="save-foldername">Folder name</Label>
          <Input
            id="save-foldername"
            v-model="folderName"
            placeholder="query_results"
            spellcheck="false"
          />
          <p class="text-muted-foreground text-xs">
            Each result is saved as a separate file.
          </p>
        </div>
        <div v-else class="grid gap-1.5">
          <Label for="save-filename">Filename</Label>
          <Input
            id="save-filename"
            v-model="fileName"
            :placeholder="format === 'json' ? 'query.json' : 'query.sql'"
            spellcheck="false"
          />
        </div>

        <div class="grid gap-1.5">
          <Label for="save-path">Path</Label>
          <InputGroup>
            <InputGroupInput
              id="save-path"
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

        <p v-if="overwriteWarn" class="text-destructive text-xs">
          Files in this destination will be overwritten.
        </p>
      </div>

      <DialogFooter>
        <Button type="button" variant="outline" :disabled="saving" @click="emit('update:open', false)">
          Close
        </Button>
        <Button type="button" :disabled="!canSave" @click="save">
          <Spinner v-if="saving" class="size-3.5" />
          {{ overwriteWarn ? "Overwrite and save" : "Save" }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
