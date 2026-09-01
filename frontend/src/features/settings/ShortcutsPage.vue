<script setup lang="ts">
import { WORKSPACE_SHORTCUTS } from "@/lib/shortcuts";
import ShortcutHint from "@/features/settings/ShortcutHint.vue";
</script>

<template>
  <section class="mx-auto flex h-full w-full max-w-lg flex-col gap-8 overflow-auto py-2">
    <h1 class="text-lg font-medium">Shortcuts</h1>

    <div class="flex flex-col gap-4">
      <div
        v-for="shortcut in WORKSPACE_SHORTCUTS"
        :key="shortcut.id"
        class="flex items-center justify-between gap-4"
      >
        <span class="text-sm">{{ shortcut.label }}</span>
        <ShortcutHint :shortcut="shortcut" />
      </div>
    </div>
  </section>
</template>
