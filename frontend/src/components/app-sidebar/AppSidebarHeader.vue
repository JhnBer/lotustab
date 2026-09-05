<script setup lang="ts">
import { Pin, SquareArrowOutUpRight } from "@lucide/vue";
import { toast } from "vue-sonner";
import { storeToRefs } from "pinia";
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { Window } from "@wailsio/runtime";
import { SidebarHeader } from "@/components/ui/sidebar";
import { Button } from "@/components/ui/button";
import { errorMessage } from "@/lib/errors";
import { connBg, connTint } from "@/lib/colors";
import { useConnectionsStore } from "@/stores/connections";
import { useSettingsStore } from "@/stores/settings";

const router = useRouter();
const connections = useConnectionsStore();
const settings = useSettingsStore();
const { selected } = storeToRefs(connections);
const isMainWindow = ref(true);

const headerTint = computed(() =>
  settings.connectionTint && selected.value ? connTint(selected.value.color) : "",
);

onMounted(async () => {
  try {
    isMainWindow.value = (await Window.Name()) === "main";
  } catch {
    // Preview / non-Wails: keep the pop-out control visible.
  }
});

async function togglePin() {
  if (!selected.value) {
    return;
  }
  try {
    await connections.setPinned(selected.value.id, !connections.isPinned(selected.value.id));
  } catch (err) {
    toast.error(errorMessage(err));
  }
}

async function openInNewWindow() {
  if (!selected.value) {
    return;
  }
  try {
    await connections.openInNewWindow(selected.value);
    await router.push("/connections");
  } catch (err) {
    toast.error(errorMessage(err));
  }
}
</script>

<template>
  <SidebarHeader :class="headerTint">
    <div class="h-12 -m-2 flex items-center p-2">
      <div v-if="selected" class="flex items-start gap-2 px-1 py-0.5">
        <span
          class="mt-1.5 size-2.5 shrink-0 rounded-full"
          :class="connBg(selected.color)"
        />
        <div class="min-w-0 flex-1">
          <div class="flex items-center gap-1">
            <span class="truncate text-sm font-medium">{{ selected.name }}</span>
            <Button
              size="icon-xs"
              variant="ghost"
              class="shrink-0"
              :title="connections.isPinned(selected.id) ? 'Unpin' : 'Pin'"
              @click="togglePin"
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
              @click="openInNewWindow"
            >
              <SquareArrowOutUpRight class="size-3.5" />
            </Button>
          </div>
        </div>
      </div>
      <div v-else class="text-sm font-medium px-2 py-1">
        Connections
      </div>
    </div>
  </SidebarHeader>
</template>
