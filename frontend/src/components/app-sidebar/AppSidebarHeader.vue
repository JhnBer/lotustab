<script setup lang="ts">
import { Pencil, Pin, Plus, SquareArrowOutUpRight } from "@lucide/vue";
import { storeToRefs } from "pinia";
import { onMounted, ref } from "vue";
import { Window } from "@wailsio/runtime";
import { SidebarHeader } from "@/components/ui/sidebar";
import { Button } from "@/components/ui/button";
import ConnectionDialog from "@/features/connections/ConnectionDialog.vue";
import { useConnectionDialog } from "@/composables/useConnectionDialog";
import { useConnectionTint } from "@/composables/useConnectionTint";
import { useProfileActions } from "@/composables/useProfileActions";
import { connBg } from "@/lib/colors";
import { useConnectionsStore } from "@/stores/connections";

const connections = useConnectionsStore();
const { selected } = storeToRefs(connections);
const { togglePin, openInNewWindow } = useProfileActions();
const { dialogOpen, editing, openCreate, openEdit } = useConnectionDialog();
const { tintClass: headerTint } = useConnectionTint();
const isMainWindow = ref(true);

onMounted(async () => {
  try {
    isMainWindow.value = (await Window.Name()) === "main";
  } catch {
    // Preview / non-Wails: keep the pop-out control visible.
  }
});
</script>

<template>
  <SidebarHeader :class="headerTint">
      <div class="flex h-12 w-full min-w-0 items-center p-2 -m-2">
      <div v-if="selected" class="flex w-full min-w-0 items-center gap-1 px-1">
        <span
          class="size-2.5 shrink-0 rounded-full"
          :class="connBg(selected.color)"
        />
        <span class="min-w-0 flex-1 truncate text-sm font-medium" :title="selected.name">{{ selected.name }}</span>
        <Button
          size="icon-xs"
          variant="ghost"
          class="shrink-0"
          title="Edit"
          aria-label="Edit"
          @click="openEdit(selected)"
        >
          <Pencil class="size-3.5" />
        </Button>
        <Button
          size="icon-xs"
          variant="ghost"
          class="shrink-0"
          :title="connections.isPinned(selected.id) ? 'Unpin' : 'Pin'"
          @click="togglePin(selected)"
        >
          <Pin
            class="size-3.5"
            :class="connections.isPinned(selected.id) ? 'fill-current' : ''"
          />
        </Button>
        <Button
          v-if="isMainWindow"
          size="icon-xs"
          variant="ghost"
          class="shrink-0"
          title="Open in new window"
          @click="openInNewWindow(selected)"
        >
          <SquareArrowOutUpRight class="size-3.5" />
        </Button>
      </div>
      <div v-else class="flex w-full items-center gap-1 px-2 py-1">
        <span class="min-w-0 flex-1 text-sm font-medium">Connections</span>
        <Button
          size="icon-xs"
          variant="ghost"
          class="shrink-0"
          title="New connection"
          aria-label="New connection"
          @click="openCreate"
        >
          <Plus class="size-3.5" />
        </Button>
      </div>
    </div>
    <ConnectionDialog v-model:open="dialogOpen" :profile="editing" />
  </SidebarHeader>
</template>
