import { ref } from "vue";
import { Events } from "@wailsio/runtime";
import { WindowsService } from "@/lib/wails";

const heightPx = ref(0);

const resyncEvents = [
  Events.Types.Common.WindowFullscreen,
  Events.Types.Common.WindowUnFullscreen,
  Events.Types.Common.WindowMaximise,
  Events.Types.Common.WindowUnMaximise,
  Events.Types.Common.WindowDidResize,
  Events.Types.Common.WindowRuntimeReady,
] as const;

function apply(px: number) {
  heightPx.value = Math.max(0, px);
  document.documentElement.style.setProperty(
    "--titlebar-height",
    `${heightPx.value}px`,
  );
}

function asHeight(value: unknown): number {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  return 0;
}

export async function syncTitleBarHeight() {
  try {
    apply(asHeight(await WindowsService.TitleBarHeight()));
  } catch {
    apply(0);
  }
}

let bound = false;
let scheduled = false;

function scheduleSync() {
  if (scheduled) {
    return;
  }
  scheduled = true;
  requestAnimationFrame(() => {
    scheduled = false;
    void syncTitleBarHeight();
  });
}

export function bindTitleBarHeight() {
  if (bound) {
    return;
  }
  bound = true;
  for (const name of resyncEvents) {
    Events.On(name, scheduleSync);
  }
}

export function useTitleBarHeight() {
  return heightPx;
}
