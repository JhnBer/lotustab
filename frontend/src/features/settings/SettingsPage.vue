<script setup lang="ts">
import { useColorMode } from "@vueuse/core";
import { useI18n } from "vue-i18n";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Input } from "@/components/ui/input";
import { isAppLocale, LOCALE_OPTIONS } from "@/i18n";
import {
  isCellCopyLayout,
  isSideViewMinTextLength,
  isTabLayout,
  PAGE_SIZES,
  QUERY_TIMEOUTS,
  useSettingsStore,
} from "@/stores/settings";

const settings = useSettingsStore();
const { t, locale } = useI18n();
const mode = useColorMode({ initialValue: "auto" });

function onLocale(value: unknown) {
  if (isAppLocale(value)) {
    settings.setLocale(value);
  }
}

function timeoutLabel(seconds: number): string {
  if (seconds <= 0) {
    return t("settings.queryTimeoutOff");
  }
  if (seconds < 60) {
    return t("settings.queryTimeoutSeconds", { n: seconds });
  }
  return t("settings.queryTimeoutMinutes", { n: seconds / 60 });
}

function onTheme(value: unknown) {
  if (value === "light" || value === "dark" || value === "auto") {
    mode.value = value;
  }
}

function onPageSize(value: unknown) {
  const size = Number(value);
  if (!Number.isFinite(size)) {
    return;
  }
  settings.setDefaultPageSize(size);
}

function onTint(value: unknown) {
  settings.setConnectionTint(Boolean(value));
}

function onShowAllDatabases(value: unknown) {
  settings.setShowAllDatabases(Boolean(value));
}

function onEnableDatabaseChoice(value: unknown) {
  settings.setEnableDatabaseChoice(Boolean(value));
}

function onTabLayout(value: unknown) {
  if (isTabLayout(value)) {
    settings.setTabLayout(value);
  }
}

function onQueryTimeout(value: unknown) {
  const seconds = Number(value);
  if (!Number.isFinite(seconds)) {
    return;
  }
  settings.setQueryTimeout(seconds);
}

function onDeleteOnAbort(value: unknown) {
  settings.setDeleteOnAbortExport(Boolean(value));
}

function onCellCopyLayout(value: unknown) {
  if (isCellCopyLayout(value)) {
    settings.setCellCopyLayout(value);
  }
}

function onUseCustomCellDelimiter(value: unknown) {
  settings.setUseCustomCellDelimiter(Boolean(value));
}

function onUseCustomRowDelimiter(value: unknown) {
  settings.setUseCustomRowDelimiter(Boolean(value));
}

function onCustomCellDelimiter(value: string | number) {
  settings.setCustomCellDelimiter(String(value));
}

function onCustomRowDelimiter(value: string | number) {
  settings.setCustomRowDelimiter(String(value));
}

function onSideViewMinTextLength(value: string | number) {
  if (value === "") {
    return;
  }
  const length = Number(value);
  if (!isSideViewMinTextLength(length)) {
    return;
  }
  settings.setSideViewMinTextLength(length);
}

function onApplySideViewLengthToJson(value: unknown) {
  settings.setApplySideViewLengthToJson(Boolean(value));
}
</script>

<template>
  <div class="h-full overflow-y-auto">
  <section class="mx-auto flex w-full min-w-0 max-w-lg flex-col gap-8 px-4 py-4">
    <h1 class="text-lg font-medium">{{ t('settings.title') }}</h1>

    <div class="flex items-center justify-between gap-4">
      <Label for="settings-language" class="min-w-0">{{ t('settings.language') }}</Label>
      <Select :model-value="settings.locale" @update:model-value="onLocale">
        <SelectTrigger id="settings-language" size="sm" class="w-36 shrink-0">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem v-for="item in LOCALE_OPTIONS" :key="item.id" :value="item.id">
            {{ item.name }}
          </SelectItem>
        </SelectContent>
      </Select>
    </div>

    <div class="flex items-center justify-between gap-4">
      <Label for="settings-theme" class="min-w-0">{{ t('settings.theme') }}</Label>
      <Select :key="locale" :model-value="mode" @update:model-value="onTheme">
        <SelectTrigger id="settings-theme" size="sm" class="w-36 shrink-0">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="light">{{ t('settings.themeLight') }}</SelectItem>
          <SelectItem value="dark">{{ t('settings.themeDark') }}</SelectItem>
          <SelectItem value="auto">{{ t('settings.themeSystem') }}</SelectItem>
        </SelectContent>
      </Select>
    </div>

    <div class="flex items-center justify-between gap-4">
      <Label for="settings-tint" class="min-w-0">{{ t('settings.tint') }}</Label>
      <Switch
        id="settings-tint"
        :model-value="settings.connectionTint"
        @update:model-value="onTint"
      />
    </div>

    <div class="flex items-center justify-between gap-4">
      <Label for="settings-show-all-dbs" class="min-w-0">{{ t('settings.showAllDatabases') }}</Label>
      <Switch
        id="settings-show-all-dbs"
        :model-value="settings.showAllDatabases"
        @update:model-value="onShowAllDatabases"
      />
    </div>

    <div v-if="settings.showAllDatabases" class="flex items-center justify-between gap-4">
      <Label for="settings-enable-db-choice" class="min-w-0">{{ t('settings.enableDatabaseChoice') }}</Label>
      <Switch
        id="settings-enable-db-choice"
        :model-value="settings.enableDatabaseChoice"
        @update:model-value="onEnableDatabaseChoice"
      />
    </div>

    <div class="flex items-center justify-between gap-4">
      <Label for="settings-tab-layout" class="min-w-0">{{ t('settings.tabLayout') }}</Label>
      <Select :key="locale" :model-value="settings.tabLayout" @update:model-value="onTabLayout">
        <SelectTrigger id="settings-tab-layout" size="sm" class="w-36 shrink-0">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="scroll">{{ t('settings.tabScroll') }}</SelectItem>
          <SelectItem value="wrap">{{ t('settings.tabWrap') }}</SelectItem>
        </SelectContent>
      </Select>
    </div>

    <div class="flex items-center justify-between gap-4">
      <Label for="settings-page-size" class="min-w-0">{{ t('settings.pageSize') }}</Label>
      <Select :key="locale" :model-value="String(settings.defaultPageSize)" @update:model-value="onPageSize">
        <SelectTrigger id="settings-page-size" size="sm" class="w-36 shrink-0">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem v-for="size in PAGE_SIZES" :key="size" :value="String(size)">
            {{ t('common.perPage', { n: size }) }}
          </SelectItem>
        </SelectContent>
      </Select>
    </div>

    <div class="flex items-center justify-between gap-4">
      <div class="min-w-0">
        <Label for="settings-query-timeout">{{ t('settings.queryTimeout') }}</Label>
        <p class="text-muted-foreground mt-1 text-xs">{{ t('settings.queryTimeoutHelp') }}</p>
      </div>
      <Select :key="locale" :model-value="String(settings.queryTimeoutSeconds)" @update:model-value="onQueryTimeout">
        <SelectTrigger id="settings-query-timeout" size="sm" class="w-36 shrink-0">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem v-for="seconds in QUERY_TIMEOUTS" :key="seconds" :value="String(seconds)">
            {{ timeoutLabel(seconds) }}
          </SelectItem>
        </SelectContent>
      </Select>
    </div>

    <div class="flex items-center justify-between gap-4">
      <div class="min-w-0">
        <Label for="settings-delete-on-abort">{{ t('settings.deleteOnAbort') }}</Label>
        <p class="text-muted-foreground mt-1 text-xs">
          {{ t('settings.deleteOnAbortHelp') }}
        </p>
      </div>
      <Switch
        id="settings-delete-on-abort"
        :model-value="settings.deleteOnAbortExport"
        @update:model-value="onDeleteOnAbort"
      />
    </div>

    <div class="flex flex-col gap-8 border-t pt-8">
      <div>
        <h2 class="text-sm font-medium">{{ t('settings.cells') }}</h2>
        <p class="text-muted-foreground mt-1 text-xs">
          {{ t('settings.cellsHelp') }}
        </p>
      </div>

      <div class="flex items-center justify-between gap-4">
        <Label for="settings-side-view-min-length" class="min-w-0">{{ t('settings.sideViewLength') }}</Label>
        <Input
          id="settings-side-view-min-length"
          :model-value="settings.sideViewMinTextLength"
          type="number"
          min="0"
          step="1"
          size="xs"
          class="w-20"
          @update:model-value="onSideViewMinTextLength"
        />
      </div>

      <div class="flex items-center justify-between gap-4">
        <div class="min-w-0">
          <Label for="settings-side-view-json">{{ t('settings.sideViewJson') }}</Label>
          <p class="text-muted-foreground mt-1 text-xs">
            {{ t('settings.sideViewJsonHelp') }}
          </p>
        </div>
        <Switch
          id="settings-side-view-json"
          :model-value="settings.applySideViewLengthToJson"
          @update:model-value="onApplySideViewLengthToJson"
        />
      </div>
    </div>

    <div class="flex flex-col gap-8 border-t pt-8">
      <div>
        <h2 class="text-sm font-medium">{{ t('settings.copy') }}</h2>
        <p class="text-muted-foreground mt-1 text-xs">{{ t('settings.copyHelp') }}</p>
      </div>

      <div class="flex items-center justify-between gap-4">
        <Label for="settings-cell-copy" class="min-w-0">{{ t('settings.selectedCells') }}</Label>
        <Select :key="locale" :model-value="settings.cellCopyLayout" @update:model-value="onCellCopyLayout">
          <SelectTrigger id="settings-cell-copy" size="sm" class="w-48 shrink-0">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="grid">{{ t('settings.copyGrid') }}</SelectItem>
            <SelectItem value="line">{{ t('settings.copyLine') }}</SelectItem>
            <SelectItem value="tab">{{ t('settings.copyTab') }}</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div v-if="settings.cellCopyLayout !== 'line'" class="flex items-center justify-between gap-4">
        <Label for="settings-custom-cell-delimiter" class="min-w-0">{{ t('settings.customCellDelimiter') }}</Label>
        <div class="flex shrink-0 items-center gap-2">
          <Input
            :model-value="settings.customCellDelimiter"
            :disabled="!settings.useCustomCellDelimiter"
            size="xs"
            class="w-20 font-mono"
            placeholder="\t"
            :aria-label="t('settings.cellDelimiter')"
            @update:model-value="onCustomCellDelimiter"
          />
          <Switch
            id="settings-custom-cell-delimiter"
            :model-value="settings.useCustomCellDelimiter"
            @update:model-value="onUseCustomCellDelimiter"
          />
        </div>
      </div>

      <div v-if="settings.cellCopyLayout !== 'tab'" class="flex items-center justify-between gap-4">
        <Label for="settings-custom-row-delimiter" class="min-w-0">{{ t('settings.customRowDelimiter') }}</Label>
        <div class="flex shrink-0 items-center gap-2">
          <Input
            :model-value="settings.customRowDelimiter"
            :disabled="!settings.useCustomRowDelimiter"
            size="xs"
            class="w-20 font-mono"
            placeholder="\n"
            :aria-label="t('settings.rowDelimiter')"
            @update:model-value="onCustomRowDelimiter"
          />
          <Switch
            id="settings-custom-row-delimiter"
            :model-value="settings.useCustomRowDelimiter"
            @update:model-value="onUseCustomRowDelimiter"
          />
        </div>
      </div>

      <p
        v-if="(settings.cellCopyLayout !== 'line' && settings.useCustomCellDelimiter) || (settings.cellCopyLayout !== 'tab' && settings.useCustomRowDelimiter)"
        class="text-muted-foreground -mt-4 text-xs"
      >
        {{ t('settings.delimiterHelp') }}
      </p>
    </div>
  </section>
  </div>
</template>
