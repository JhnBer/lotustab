import { markRaw, type Component } from "vue";
import { toast as sonnerToast, type ExternalToast } from "vue-sonner";
import ToastNotice from "@/components/ui/sonner/ToastNotice.vue";
import { writeClipboard } from "@/composables/useClipboardText";
import { translate } from "@/i18n";

const Notice = markRaw(ToastNotice);

type ToastTitle = string | Component | (() => string | Component);
type ToastKind = "default" | "success" | "info" | "warning" | "error" | "loading" | "message";
type ToastTone = "default" | "success" | "info" | "warning" | "error" | "loading";

export type ToastOptions = ExternalToast & {
  /** Renders a Copy button under Close. Omitted from the toast when false or unset. */
  copy?: boolean;
};

function plain(value: unknown): string | undefined {
  if (typeof value === "string") {
    return value;
  }
  if (typeof value === "function") {
    const next = value();
    return typeof next === "string" ? next : undefined;
  }
  return undefined;
}

function stringAction(
  value: ExternalToast["action"],
): { label: string; onClick: (event: MouseEvent) => void } | undefined {
  if (!value || typeof value !== "object" || !("label" in value) || !("onClick" in value)) {
    return undefined;
  }
  const action = value as { label: unknown; onClick: unknown };
  if (typeof action.label !== "string" || typeof action.onClick !== "function") {
    return undefined;
  }
  return {
    label: action.label,
    onClick: action.onClick as (event: MouseEvent) => void,
  };
}

function native(kind: ToastKind, message: ToastTitle, options?: ExternalToast) {
  if (kind === "default") {
    return sonnerToast(message, options);
  }
  if (kind === "message") {
    return sonnerToast.message(message, options);
  }
  return sonnerToast[kind](message, options);
}

function withNativeCopy(message: ToastTitle, options?: ToastOptions): ExternalToast | undefined {
  if (!options) {
    return undefined;
  }
  const { copy, ...rest } = options;
  if (!copy) {
    return rest;
  }
  const text = [plain(message), plain(rest.description)].filter((part): part is string => !!part).join("\n");
  const button = {
    label: translate("common.copy"),
    onClick: () => {
      void writeClipboard(text);
    },
  };
  if (!rest.action) {
    return { ...rest, action: button };
  }
  if (!rest.cancel) {
    return { ...rest, cancel: button };
  }
  return rest;
}

function show(kind: ToastKind, message: ToastTitle, options?: ToastOptions) {
  const { copy, ...rest } = options ?? {};
  const title = plain(message);
  const action = stringAction(rest.action);
  const descriptionIsCustom = rest.description != null && plain(rest.description) === undefined;
  if (title === undefined || descriptionIsCustom || (rest.action && !action)) {
    return native(kind, message, withNativeCopy(message, options));
  }

  const description = plain(rest.description);
  const copyText = copy
    ? [title, description].filter((part): part is string => !!part).join("\n")
    : undefined;
  const {
    action: _action,
    cancel: _cancel,
    classes: _classes,
    closeButton,
    description: _description,
    ...toastOptions
  } = rest;
  const tone: ToastTone = kind === "message" || kind === "default" ? "default" : kind;

  return sonnerToast.custom(Notice, {
    ...toastOptions,
    type: tone,
    componentProps: {
      message: title,
      description,
      tone,
      showClose: closeButton !== false && tone !== "loading",
      copyText,
      actionLabel: action?.label,
      onAction: action?.onClick,
    },
  } as ExternalToast);
}

function showDefault(message: ToastTitle, options?: ToastOptions) {
  return show("default", message, options);
}

export const toast = Object.assign(showDefault, {
  success: (message: ToastTitle, options?: ToastOptions) => show("success", message, options),
  info: (message: ToastTitle, options?: ToastOptions) => show("info", message, options),
  warning: (message: ToastTitle, options?: ToastOptions) => show("warning", message, options),
  error: (message: ToastTitle, options?: ToastOptions) => show("error", message, options),
  message: (message: ToastTitle, options?: ToastOptions) => show("message", message, options),
  loading: (message: ToastTitle, options?: ToastOptions) => show("loading", message, options),
  custom: sonnerToast.custom,
  promise: sonnerToast.promise,
  dismiss: sonnerToast.dismiss,
});
