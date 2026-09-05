import { onBeforeUnmount, ref } from "vue";

export type PointerResizeCursor = string;

export function usePointerResize<T extends Record<string, unknown> = Record<string, never>>(opts: {
  cursor?: PointerResizeCursor;
  onMove: (event: PointerEvent, drag: T & { startX: number; startY: number }) => void;
  onEnd?: () => void;
}) {
  let drag: (T & { startX: number; startY: number }) | null = null;
  const dragging = ref(false);

  function onMove(event: PointerEvent) {
    if (!drag) {
      return;
    }
    opts.onMove(event, drag);
  }

  function stop() {
    const wasDragging = dragging.value;
    drag = null;
    dragging.value = false;
    document.body.style.cursor = "";
    document.body.style.userSelect = "";
    window.removeEventListener("pointermove", onMove);
    window.removeEventListener("pointerup", stop);
    if (wasDragging) {
      opts.onEnd?.();
    }
  }

  function start(event: PointerEvent, extra: T) {
    event.preventDefault();
    event.stopPropagation();
    drag = { ...extra, startX: event.clientX, startY: event.clientY };
    dragging.value = true;
    document.body.style.cursor = opts.cursor ?? "col-resize";
    document.body.style.userSelect = "none";
    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup", stop);
  }

  onBeforeUnmount(stop);

  return { start, stop, dragging };
}
