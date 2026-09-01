<script setup lang="ts">
import { Kbd, KbdGroup } from "@/components/ui/kbd";
import type { ShortcutSpec } from "@/lib/shortcuts";

defineProps<{ shortcut: ShortcutSpec }>();
</script>

<template>
  <KbdGroup>
    <Kbd>Ctrl</Kbd>
    <span class="text-muted-foreground">/</span>
    <Kbd>⌘</Kbd>
    <span class="text-muted-foreground">+</span>
    <Kbd v-if="shortcut.shift">Shift</Kbd>
    <span v-if="shortcut.shift" class="text-muted-foreground">+</span>
    <Kbd>{{ shortcut.key }}</Kbd>
  </KbdGroup>
</template>
