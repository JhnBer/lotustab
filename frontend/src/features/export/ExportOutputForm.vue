<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
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
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { formatRows, isSchemaExport, withExportExt } from "@/features/export/format";
import { useSettingsStore } from "@/stores/settings";

const props = defineProps<{
  variant: "table" | "batch";
  schemaDisabled?: boolean;
  browsing?: boolean;
  counting?: boolean;
  checkedTotal?: number | null;
  overwriteWarn?: boolean;
}>();

const emit = defineEmits<{
  browse: [];
  checkCount: [];
}>();

const format = defineModel<string>("format", { required: true });
const gzip = defineModel<boolean>("gzip", { required: true });
const folderMode = defineModel<boolean>("folderMode", { required: true });
const fileName = defineModel<string>("fileName", { required: true });
const folderName = defineModel<string>("folderName", { required: true });
const prefixEnabled = defineModel<boolean>("prefixEnabled", { required: true });
const prefix = defineModel<string>("prefix", { required: true });
const outputDir = defineModel<string>("outputDir", { required: true });
const content = defineModel<string>("content", { required: true });
const includeCreate = defineModel<boolean>("includeCreate", { required: true });
const includeDrop = defineModel<boolean>("includeDrop", { required: true });
const wrapTransaction = defineModel<boolean>("wrapTransaction", { required: true });
const insertBatchSize = defineModel<string>("insertBatchSize", { required: true });
const chunkSize = defineModel<string>("chunkSize", { default: "0" });
const customWhere = defineModel<boolean>("customWhere", { default: false });
const whereSQL = defineModel<string>("whereSql", { default: "" });

const settings = useSettingsStore();
const { t } = useI18n();
const schemaMode = ref<"schemaAdd" | "schemaReplace">("schemaAdd");
const rememberedFormat = ref(format.value);
const schema = computed(() => isSchemaExport(content.value));
const tab = computed({
  get: () => (schema.value ? "schema" : "data"),
  set: (next: string) => {
    if (next === "schema") {
      if (props.schemaDisabled) {
        return;
      }
      rememberedFormat.value = format.value;
      content.value = schemaMode.value;
      if (format.value !== "sql") {
        format.value = "sql";
      }
      return;
    }
    content.value = "";
    format.value = rememberedFormat.value || "sql";
  },
});

watch(content, (value) => {
  if (value === "schemaAdd" || value === "schemaReplace") {
    schemaMode.value = value;
  }
});

watch(
  () => props.schemaDisabled,
  (disabled) => {
    if (disabled && schema.value) {
      content.value = "";
      format.value = rememberedFormat.value || "sql";
    }
  },
);

watch(format, (value) => {
  if (value === "csv" && !schema.value) {
    folderMode.value = true;
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

function chooseSchema(mode: "schemaAdd" | "schemaReplace") {
  schemaMode.value = mode;
  content.value = mode;
}

function onSchemaMode(value: unknown) {
  if (value === "schemaAdd" || value === "schemaReplace") {
    chooseSchema(value);
  }
}
</script>

<template>
  <div class="grid gap-1.5">
    <Label>{{ t('export.format') }}</Label>
    <Select
      :model-value="format"
      :disabled="schema"
      @update:model-value="(v) => (format = String(v ?? 'sql'))"
    >
      <SelectTrigger class="w-full">
        <SelectValue :placeholder="t('export.format')" />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="sql">SQL</SelectItem>
        <SelectItem value="csv">CSV</SelectItem>
      </SelectContent>
    </Select>
  </div>

  <div class="flex items-center justify-between gap-4">
    <Label :for="variant === 'batch' ? 'export-single-file' : 'export-folder-mode'">
      {{ variant === "batch" ? t('export.singleFile') : t('export.separateFolder') }}
    </Label>
    <Switch
      :id="variant === 'batch' ? 'export-single-file' : 'export-folder-mode'"
      :model-value="variant === 'batch' ? !folderMode : folderMode"
      :disabled="format === 'csv'"
      @update:model-value="
        (v) => (folderMode = variant === 'batch' ? v !== true : v === true)
      "
    />
  </div>

  <div v-if="folderMode" class="grid gap-1.5">
    <Label :for="`${variant}-export-foldername`">{{ t('export.folderName') }}</Label>
    <Input
      :id="`${variant}-export-foldername`"
      v-model="folderName"
      placeholder="export_20260101"
      spellcheck="false"
    />
    <p class="text-muted-foreground text-xs">{{ t('export.eachTableFile') }}</p>
  </div>
  <div v-else class="grid gap-1.5">
    <Label :for="`${variant}-export-filename`">{{ t('export.filename') }}</Label>
    <Input :id="`${variant}-export-filename`" v-model="fileName" placeholder="export.sql" spellcheck="false" />
  </div>

  <div v-if="folderMode" class="flex items-center justify-between gap-4">
    <Label :for="`${variant}-export-prefix-on`">{{ t('export.filenamePrefix') }}</Label>
    <Switch
      :id="`${variant}-export-prefix-on`"
      :model-value="prefixEnabled"
      @update:model-value="(v) => (prefixEnabled = v === true)"
    />
  </div>
  <div v-if="folderMode && prefixEnabled" class="grid gap-1.5">
    <Input v-model="prefix" placeholder="prod_" spellcheck="false" />
  </div>

  <div class="grid gap-1.5">
    <Label :for="`${variant}-export-path`">{{ t('export.path') }}</Label>
    <InputGroup>
      <InputGroupInput
        :id="`${variant}-export-path`"
        v-model="outputDir"
        :placeholder="t('export.chooseFolder')"
        spellcheck="false"
      />
      <InputGroupAddon align="inline-end">
        <InputGroupButton :disabled="browsing" @click="emit('browse')">
          <Spinner v-if="browsing" class="size-3.5" />
          {{ t('common.browse') }}
        </InputGroupButton>
      </InputGroupAddon>
    </InputGroup>
  </div>

  <Tabs v-model="tab" class="gap-3">
    <TabsList class="grid w-full grid-cols-2">
      <TabsTrigger value="data">{{ t('export.data') }}</TabsTrigger>
      <TabsTrigger value="schema" :disabled="schemaDisabled">{{ t('export.schemaOnly') }}</TabsTrigger>
    </TabsList>

    <TabsContent value="data" class="grid gap-3">
      <template v-if="variant === 'table'">
        <div v-if="!folderMode" class="grid gap-1.5">
          <Label :for="`${variant}-export-chunk`">{{ t('export.chunkSize') }}</Label>
          <Input
            :id="`${variant}-export-chunk`"
            v-model="chunkSize"
            type="number"
            min="0"
            :placeholder="t('export.chunkPlaceholder')"
          />
        </div>
        <div class="flex items-center justify-between gap-4">
          <Label :for="`${variant}-export-where-on`">{{ t('export.where') }}</Label>
          <Switch
            :id="`${variant}-export-where-on`"
            :model-value="customWhere"
            @update:model-value="(v) => (customWhere = v === true)"
          />
        </div>
        <div v-if="customWhere" class="grid gap-1.5">
          <Textarea
            :id="`${variant}-export-where`"
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
              @click="emit('checkCount')"
            >
              <Spinner v-if="counting" class="size-3.5" />
              {{ t('export.checkCount') }}
            </Button>
            <span v-if="checkedTotal != null" class="text-muted-foreground text-xs">
              {{ t('export.rowsMatch', { n: formatRows(checkedTotal) }) }}
            </span>
          </div>
        </div>
      </template>

      <template v-if="format === 'sql'">
        <div class="flex items-center gap-2">
          <Checkbox
            :id="`${variant}-export-create`"
            :model-value="includeCreate"
            @update:model-value="(v) => (includeCreate = Boolean(v))"
          />
          <Label :for="`${variant}-export-create`" class="font-normal">{{ t('export.includeCreate') }}</Label>
        </div>
        <div class="flex items-center gap-2">
          <Checkbox
            :id="`${variant}-export-drop`"
            :model-value="includeDrop"
            @update:model-value="(v) => (includeDrop = Boolean(v))"
          />
          <Label :for="`${variant}-export-drop`" class="font-normal">{{ t('export.includeDrop') }}</Label>
        </div>
        <div class="flex items-center gap-2">
          <Checkbox
            :id="`${variant}-export-tx`"
            :model-value="wrapTransaction"
            @update:model-value="(v) => (wrapTransaction = Boolean(v))"
          />
          <Label :for="`${variant}-export-tx`" class="font-normal">{{ t('export.wrapTx') }}</Label>
        </div>
        <div class="grid gap-1.5">
          <Label :for="`${variant}-export-batch`">{{ t('export.insertBatch') }}</Label>
          <Input :id="`${variant}-export-batch`" v-model="insertBatchSize" type="number" min="1" />
        </div>
      </template>
    </TabsContent>

    <TabsContent value="schema">
      <RadioGroup :model-value="schemaMode" class="gap-3" @update:model-value="onSchemaMode">
        <div class="flex items-center gap-2.5">
          <RadioGroupItem :id="`${variant}-schema-create`" value="schemaAdd" />
          <Label :for="`${variant}-schema-create`" class="grid gap-0.5 font-normal leading-snug">
            <span class="font-medium leading-none">{{ t('export.create') }}</span>
            <span class="text-muted-foreground text-xs font-normal">
              {{ t('export.createHelp') }}
            </span>
          </Label>
        </div>
        <div class="flex items-center gap-2.5">
          <RadioGroupItem :id="`${variant}-schema-drop`" value="schemaReplace" />
          <Label :for="`${variant}-schema-drop`" class="grid gap-0.5 font-normal leading-snug">
            <span class="font-medium leading-none">{{ t('export.drop') }}</span>
            <span class="text-muted-foreground text-xs font-normal">
              {{ t('export.dropHelp') }}
            </span>
          </Label>
        </div>
      </RadioGroup>
    </TabsContent>
  </Tabs>

  <div class="flex items-center gap-2">
    <Checkbox
      :id="`${variant}-export-gzip`"
      :model-value="gzip"
      @update:model-value="(v) => (gzip = Boolean(v))"
    />
    <Label :for="`${variant}-export-gzip`" class="font-normal">{{ t('export.gzip') }}</Label>
  </div>

  <div class="flex items-center gap-2">
    <Checkbox
      :id="`${variant}-export-delete-abort`"
      :model-value="settings.deleteOnAbortExport"
      @update:model-value="(v) => settings.setDeleteOnAbortExport(Boolean(v))"
    />
    <Label :for="`${variant}-export-delete-abort`" class="font-normal">{{ t('export.deleteOnAbort') }}</Label>
  </div>

  <p v-if="overwriteWarn" class="text-destructive text-xs">
    {{ t('common.overwriteFiles') }}
  </p>
</template>
