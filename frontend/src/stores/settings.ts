import { defineStore } from "pinia";
import { ref } from "vue";
import { Events } from "@wailsio/runtime";

export const PAGE_SIZES = [50, 100, 250, 500, 1000, 5000, 10000] as const;
export const DEFAULT_PAGE_SIZE = 100;

const PAGE_SIZE_KEY = "lotustab.tablePageSize";
const TINT_KEY = "lotustab.connectionTint";
const SETTINGS_CHANGED = "settings:changed";

export function isPageSize(n: number): boolean {
  return (PAGE_SIZES as readonly number[]).includes(n);
}

function readPageSize(): number {
  const n = Number(localStorage.getItem(PAGE_SIZE_KEY));
  return isPageSize(n) ? n : DEFAULT_PAGE_SIZE;
}

function readTint(): boolean {
  return localStorage.getItem(TINT_KEY) === "true";
}

type SettingsPayload = {
  connectionTint: boolean;
  defaultPageSize: number;
};

export const useSettingsStore = defineStore("settings", () => {
  const connectionTint = ref(readTint());
  const defaultPageSize = ref(readPageSize());

  function snapshot(): SettingsPayload {
    return {
      connectionTint: connectionTint.value,
      defaultPageSize: defaultPageSize.value,
    };
  }

  function persist() {
    localStorage.setItem(TINT_KEY, String(connectionTint.value));
    localStorage.setItem(PAGE_SIZE_KEY, String(defaultPageSize.value));
  }

  function commit() {
    persist();
    void Events.Emit(SETTINGS_CHANGED, snapshot()).catch(() => {});
  }

  function apply(next: SettingsPayload) {
    const size = isPageSize(next.defaultPageSize)
      ? next.defaultPageSize
      : defaultPageSize.value;
    if (connectionTint.value === next.connectionTint && defaultPageSize.value === size) {
      return;
    }
    connectionTint.value = next.connectionTint;
    defaultPageSize.value = size;
    persist();
  }

  function setDefaultPageSize(size: number) {
    if (!isPageSize(size) || defaultPageSize.value === size) {
      return;
    }
    defaultPageSize.value = size;
    commit();
  }

  function setConnectionTint(on: boolean) {
    if (connectionTint.value === on) {
      return;
    }
    connectionTint.value = on;
    commit();
  }

  Events.On(SETTINGS_CHANGED, (ev) => {
    const data = ev.data as SettingsPayload | null;
    if (!data || typeof data !== "object") {
      return;
    }
    apply({
      connectionTint: Boolean(data.connectionTint),
      defaultPageSize: Number(data.defaultPageSize),
    });
  });

  return {
    connectionTint,
    defaultPageSize,
    setDefaultPageSize,
    setConnectionTint,
  };
});
