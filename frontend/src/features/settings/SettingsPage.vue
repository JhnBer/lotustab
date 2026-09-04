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
import { isTabLayout, PAGE_SIZES, QUERY_TIMEOUTS, queryTimeoutLabel, useSettingsStore } from "@/stores/settings";

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
</script>

<template>
  <section class="mx-auto flex h-full w-full max-w-lg flex-col gap-8 overflow-auto py-2">
    <h1 class="text-lg font-medium">Settings</h1>

    <div class="flex items-center justify-between gap-4">
      <Label for="settings-theme">Theme</Label>
      <Select :model-value="mode" @update:model-value="onTheme">
        <SelectTrigger id="settings-theme" size="sm" class="w-36">
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
      <Label for="settings-tint">Colored connection headers</Label>
      <Switch
        id="settings-tint"
        :model-value="settings.connectionTint"
        @update:model-value="onTint"
      />
    </div>

    <div class="flex items-center justify-between gap-4">
      <Label for="settings-show-all-dbs">Show all databases</Label>
      <Switch
        id="settings-show-all-dbs"
        :model-value="settings.showAllDatabases"
        @update:model-value="onShowAllDatabases"
      />
    </div>

    <div class="flex items-center justify-between gap-4">
      <Label for="settings-tab-layout">Tab layout</Label>
      <Select :model-value="settings.tabLayout" @update:model-value="onTabLayout">
        <SelectTrigger id="settings-tab-layout" size="sm" class="w-36">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="scroll">Scroll</SelectItem>
          <SelectItem value="wrap">Wrap</SelectItem>
        </SelectContent>
      </Select>
    </div>

    <div class="flex items-center justify-between gap-4">
      <Label for="settings-page-size">Default rows per page</Label>
      <Select :model-value="String(settings.defaultPageSize)" @update:model-value="onPageSize">
        <SelectTrigger id="settings-page-size" size="sm" class="w-36">
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
        <SelectTrigger id="settings-query-timeout" size="sm" class="w-36">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem v-for="seconds in QUERY_TIMEOUTS" :key="seconds" :value="String(seconds)">
            {{ queryTimeoutLabel(seconds) }}
          </SelectItem>
        </SelectContent>
      </Select>
    </div>
  </section>
</template>
