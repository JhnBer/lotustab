<script setup lang="ts">
import { useColorMode } from "@vueuse/core";
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
import {
  isCellCopyLayout,
  isSideViewMinTextLength,
  isTabLayout,
  PAGE_SIZES,
  QUERY_TIMEOUTS,
  queryTimeoutLabel,
  useSettingsStore,
} from "@/stores/settings";

const settings = useSettingsStore();
const mode = useColorMode({ initialValue: "auto" });

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
    <h1 class="text-lg font-medium">Settings</h1>

    <div class="flex items-center justify-between gap-4">
      <Label for="settings-theme" class="min-w-0">Theme</Label>
      <Select :model-value="mode" @update:model-value="onTheme">
        <SelectTrigger id="settings-theme" size="sm" class="w-36 shrink-0">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="light">Light</SelectItem>
          <SelectItem value="dark">Dark</SelectItem>
          <SelectItem value="auto">System</SelectItem>
        </SelectContent>
      </Select>
    </div>

    <div class="flex items-center justify-between gap-4">
      <Label for="settings-tint" class="min-w-0">Colored connection headers</Label>
      <Switch
        id="settings-tint"
        :model-value="settings.connectionTint"
        @update:model-value="onTint"
      />
    </div>

    <div class="flex items-center justify-between gap-4">
      <Label for="settings-show-all-dbs" class="min-w-0">Show all databases</Label>
      <Switch
        id="settings-show-all-dbs"
        :model-value="settings.showAllDatabases"
        @update:model-value="onShowAllDatabases"
      />
    </div>

    <div v-if="settings.showAllDatabases" class="flex items-center justify-between gap-4">
      <Label for="settings-enable-db-choice" class="min-w-0">Enable database choice</Label>
      <Switch
        id="settings-enable-db-choice"
        :model-value="settings.enableDatabaseChoice"
        @update:model-value="onEnableDatabaseChoice"
      />
    </div>

    <div class="flex items-center justify-between gap-4">
      <Label for="settings-tab-layout" class="min-w-0">Tab layout</Label>
      <Select :model-value="settings.tabLayout" @update:model-value="onTabLayout">
        <SelectTrigger id="settings-tab-layout" size="sm" class="w-36 shrink-0">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="scroll">Scroll</SelectItem>
          <SelectItem value="wrap">Wrap</SelectItem>
        </SelectContent>
      </Select>
    </div>

    <div class="flex items-center justify-between gap-4">
      <Label for="settings-page-size" class="min-w-0">Default rows per page</Label>
      <Select :model-value="String(settings.defaultPageSize)" @update:model-value="onPageSize">
        <SelectTrigger id="settings-page-size" size="sm" class="w-36 shrink-0">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem v-for="size in PAGE_SIZES" :key="size" :value="String(size)">
            {{ size }} / page
          </SelectItem>
        </SelectContent>
      </Select>
    </div>

    <div class="flex items-center justify-between gap-4">
      <div class="min-w-0">
        <Label for="settings-query-timeout">Query timeout</Label>
        <p class="text-muted-foreground mt-1 text-xs">Cancels a running query after this time. Off waits until you cancel.</p>
      </div>
      <Select :model-value="String(settings.queryTimeoutSeconds)" @update:model-value="onQueryTimeout">
        <SelectTrigger id="settings-query-timeout" size="sm" class="w-36 shrink-0">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem v-for="seconds in QUERY_TIMEOUTS" :key="seconds" :value="String(seconds)">
            {{ queryTimeoutLabel(seconds) }}
          </SelectItem>
        </SelectContent>
      </Select>
    </div>

    <div class="flex items-center justify-between gap-4">
      <div class="min-w-0">
        <Label for="settings-delete-on-abort">Delete partial files on abort/error</Label>
        <p class="text-muted-foreground mt-1 text-xs">
          Removes export files if a dump is canceled or fails.
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
        <h2 class="text-sm font-medium">Cells</h2>
        <p class="text-muted-foreground mt-1 text-xs">
          Double-click edits a cell when its text is this long or shorter. Longer text opens the side view.
        </p>
      </div>

      <div class="flex items-center justify-between gap-4">
        <Label for="settings-side-view-min-length" class="min-w-0">Min text length to open side view</Label>
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
          <Label for="settings-side-view-json">Also apply to JSON fields</Label>
          <p class="text-muted-foreground mt-1 text-xs">
            JSON columns always open the side view unless this is on.
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
        <h2 class="text-sm font-medium">Copy</h2>
        <p class="text-muted-foreground mt-1 text-xs">Separators used when copying selected cells.</p>
      </div>

      <div class="flex items-center justify-between gap-4">
        <Label for="settings-cell-copy" class="min-w-0">Selected cells</Label>
        <Select :model-value="settings.cellCopyLayout" @update:model-value="onCellCopyLayout">
          <SelectTrigger id="settings-cell-copy" size="sm" class="w-48 shrink-0">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="grid">Per column and row</SelectItem>
            <SelectItem value="line">Per row</SelectItem>
            <SelectItem value="tab">Per column</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div v-if="settings.cellCopyLayout !== 'line'" class="flex items-center justify-between gap-4">
        <Label for="settings-custom-cell-delimiter" class="min-w-0">Enable custom cell delimiter</Label>
        <div class="flex shrink-0 items-center gap-2">
          <Input
            :model-value="settings.customCellDelimiter"
            :disabled="!settings.useCustomCellDelimiter"
            size="xs"
            class="w-20 font-mono"
            placeholder="\t"
            aria-label="Cell delimiter"
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
        <Label for="settings-custom-row-delimiter" class="min-w-0">Enable custom row delimiter</Label>
        <div class="flex shrink-0 items-center gap-2">
          <Input
            :model-value="settings.customRowDelimiter"
            :disabled="!settings.useCustomRowDelimiter"
            size="xs"
            class="w-20 font-mono"
            placeholder="\n"
            aria-label="Row delimiter"
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
        Use \t for a tab, \n for a new line, and \r for a carriage return.
      </p>
    </div>
  </section>
  </div>
</template>
