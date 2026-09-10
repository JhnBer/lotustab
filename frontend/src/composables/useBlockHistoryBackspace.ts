import { useEventListener } from "@vueuse/core";

const NON_TEXT_INPUT_TYPES = new Set([
  "button",
  "checkbox",
  "radio",
  "submit",
  "reset",
  "file",
  "hidden",
  "range",
  "color",
  "image",
]);

function isTypingTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) {
    return false;
  }
  if (target.isContentEditable) {
    return true;
  }
  if (target instanceof HTMLTextAreaElement) {
    return true;
  }
  if (target instanceof HTMLInputElement) {
    return !NON_TEXT_INPUT_TYPES.has(target.type);
  }
  return false;
}

export function useBlockHistoryBackspace() {
  useEventListener(
    "keydown",
    (event: KeyboardEvent) => {
      if (event.key !== "Backspace" || isTypingTarget(event.target)) {
        return;
      }
      event.preventDefault();
    },
    { capture: true },
  );
}
