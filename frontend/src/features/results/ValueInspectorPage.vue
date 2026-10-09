<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref } from "vue";
import { useRoute } from "vue-router";
import { Events } from "@wailsio/runtime";
import { toast } from "vue-sonner";
import ValueInspectorPanel from "@/features/results/ValueInspectorPanel.vue";
import type { ValueInspectorPayload } from "@/features/results/valueInspector";
import { errorMessage } from "@/lib/errors";
import { WindowsService } from "@/lib/wails";

const route = useRoute();
const payload = ref<ValueInspectorPayload | null>(null);
const returnUnavailable = ref(false);
const missing = ref(false);

function parsePayload(raw: string | null | undefined): ValueInspectorPayload | null {
  if (!raw) {
    return null;
  }
  try {
    return JSON.parse(raw) as ValueInspectorPayload;
  } catch {
    return null;
  }
}

async function loadHandoff() {
  const token = String(route.query.id ?? "").trim();
  if (!token) {
    missing.value = true;
    return;
  }
  const next = parsePayload(await WindowsService.TakeValueHandoff(token));
  if (!next) {
    missing.value = true;
    return;
  }
  payload.value = next;
  missing.value = false;
}

function onTextUpdate(text: string) {
  if (!payload.value) {
    return;
  }
  payload.value = { ...payload.value, text };
}

async function returnToPanel() {
  const current = payload.value;
  if (!current) {
    return;
  }
  try {
    await WindowsService.ReturnValueToPanel(
      current.token,
      current.sourceWindow,
      JSON.stringify(current),
    );
  } catch (err) {
    const message = errorMessage(err);
    if (message.toLowerCase().includes("closed")) {
      returnUnavailable.value = true;
      toast.error("Source window is closed. Nothing to return to.");
      return;
    }
    toast.error(message);
  }
}

onMounted(async () => {
  await loadHandoff();
  Events.On("value:handoff", () => {
    void loadHandoff();
  });
});

onBeforeUnmount(() => {
  Events.Off("value:handoff");
});
</script>

<template>
  <div class="bg-background flex h-full min-h-0 flex-col">
    <div
      class="drag-region border-b shrink-0"
      :style="{ height: 'var(--titlebar-height)' }"
    />
    <div class="min-h-0 flex-1" style="--wails-draggable: no-drag">
      <ValueInspectorPanel
        v-if="payload"
        :payload="payload"
        window-mode
        :return-unavailable="returnUnavailable"
        @update:text="onTextUpdate"
        @return-panel="returnToPanel"
      />
      <div
        v-else-if="missing"
        class="text-muted-foreground flex h-full items-center justify-center p-6 text-sm"
      >
        No value payload. Open a cell with Open in new window.
      </div>
    </div>
  </div>
</template>
