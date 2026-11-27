<script setup lang="ts">
import { useI18n } from "vue-i18n";
import { WORKSPACE_SHORTCUTS } from "@/lib/shortcuts";
import ShortcutHint from "@/features/settings/ShortcutHint.vue";

const { t } = useI18n();
</script>

<template>
  <div class="h-full overflow-y-auto">
  <section class="mx-auto flex w-full max-w-lg flex-col gap-8 px-4 py-4">
    <h1 class="text-lg font-medium">{{ t('shortcuts.title') }}</h1>

    <div class="flex flex-col gap-4">
      <div
        v-for="shortcut in WORKSPACE_SHORTCUTS"
        :key="shortcut.id"
        class="flex items-center justify-between gap-4"
      >
        <span class="text-sm">{{ t(shortcut.labelKey) }}</span>
        <ShortcutHint :shortcut="shortcut" />
      </div>
    </div>
  </section>
  </div>
</template>
