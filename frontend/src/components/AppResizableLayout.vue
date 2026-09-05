<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from "vue";
import { useRoute } from "vue-router";
import AppSidebar from "@/components/app-sidebar/AppSidebar.vue";
import { SidebarInset, useSidebar } from "@/components/ui/sidebar";
import SettingsSidebar from "@/features/settings/SettingsSidebar.vue";

const route = useRoute();
const isSettings = computed(() => route.path.startsWith("/settings"));

const DEFAULT_WIDTH = 280;
const MIN_WIDTH = 220;

const { open } = useSidebar();
const explorerPx = ref(DEFAULT_WIDTH);
const dragging = ref(false);

const clipWidth = computed(() => (open.value ? explorerPx.value : 0));

let drag: { startX: number; startWidth: number } | null = null;

function clampWidth(value: number) {
  const max = Math.max(MIN_WIDTH, Math.round(window.innerWidth * 0.4));
  return Math.min(Math.max(value, MIN_WIDTH), max);
}

function onResizeMove(event: PointerEvent) {
  if (!drag) {
    return;
  }
  explorerPx.value = clampWidth(drag.startWidth + (event.clientX - drag.startX));
}

function onResizeUp() {
  drag = null;
  dragging.value = false;
  document.body.style.cursor = "";
  document.body.style.userSelect = "";
  window.removeEventListener("pointermove", onResizeMove);
  window.removeEventListener("pointerup", onResizeUp);
}

function startResize(event: PointerEvent) {
  if (!open.value) {
    return;
  }
  event.preventDefault();
  event.stopPropagation();
  dragging.value = true;
  drag = {
    startX: event.clientX,
    startWidth: explorerPx.value,
  };
  document.body.style.cursor = "col-resize";
  document.body.style.userSelect = "none";
  window.addEventListener("pointermove", onResizeMove);
  window.addEventListener("pointerup", onResizeUp);
}

onBeforeUnmount(() => {
  onResizeUp();
});
</script>

<template>
  <div class="flex min-h-0 min-w-0 flex-1">
    <div
      class="relative shrink-0 overflow-hidden"
      :class="[
        dragging ? 'transition-none' : 'transition-[width] duration-200 ease-linear',
        open ? '' : 'pointer-events-none',
      ]"
      :style="{ width: `${clipWidth}px` }"
    >
      <div class="flex h-full" :style="{ width: `${explorerPx}px` }">
        <AppSidebar v-if="!isSettings" class="min-w-0 flex-1" />
        <SettingsSidebar v-else class="min-w-0 flex-1" />
        <div
          class="bg-border relative w-px shrink-0 cursor-col-resize after:absolute after:inset-y-0 after:-left-1 after:w-2.5 after:content-['']"
          @pointerdown="startResize"
        />
      </div>
    </div>
    <SidebarInset class="h-full min-h-0 min-w-0 flex-1 overflow-hidden">
      <slot />
    </SidebarInset>
  </div>
</template>
