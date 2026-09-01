import { defineStore } from "pinia";
import { ref } from "vue";
import { Events } from "@wailsio/runtime";

export const PAGE_SIZES = [50, 100, 250, 500, 1000, 5000, 10000] as const;
export const DEFAULT_PAGE_SIZE = 100;

export const TAB_LAYOUTS = ["scroll", "wrap"] as const;
export type TabLayout = (typeof TAB_LAYOUTS)[number];
export const DEFAULT_TAB_LAYOUT: TabLayout = "scroll";

const PAGE_SIZE_KEY = "lotustab.tablePageSize";
const TINT_KEY = "lotustab.connectionTint";
const SHOW_ALL_DBS_KEY = "lotustab.showAllDatabases";
const TAB_LAYOUT_KEY = "lotustab.tabLayout";
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

function readShowAllDatabases(): boolean {
  return localStorage.getItem(SHOW_ALL_DBS_KEY) !== "false";
}

export function isTabLayout(v: unknown): v is TabLayout {
  return (TAB_LAYOUTS as readonly string[]).includes(v as string);
}

function readTabLayout(): TabLayout {
  const v = localStorage.getItem(TAB_LAYOUT_KEY);
  return isTabLayout(v) ? v : DEFAULT_TAB_LAYOUT;
}

type SettingsPayload = {
  connectionTint: boolean;
  defaultPageSize: number;
  showAllDatabases: boolean;
  tabLayout: TabLayout;
};

export const useSettingsStore = defineStore("settings", () => {
  const connectionTint = ref(readTint());
  const defaultPageSize = ref(readPageSize());
  const showAllDatabases = ref(readShowAllDatabases());
  const tabLayout = ref(readTabLayout());

  function snapshot(): SettingsPayload {
    return {
      connectionTint: connectionTint.value,
      defaultPageSize: defaultPageSize.value,
      showAllDatabases: showAllDatabases.value,
      tabLayout: tabLayout.value,
    };
  }

  function persist() {
    localStorage.setItem(TINT_KEY, String(connectionTint.value));
    localStorage.setItem(PAGE_SIZE_KEY, String(defaultPageSize.value));
    localStorage.setItem(SHOW_ALL_DBS_KEY, String(showAllDatabases.value));
    localStorage.setItem(TAB_LAYOUT_KEY, tabLayout.value);
  }

  function commit() {
    persist();
    void Events.Emit(SETTINGS_CHANGED, snapshot()).catch(() => {});
  }

  function apply(next: SettingsPayload) {
    const size = isPageSize(next.defaultPageSize)
      ? next.defaultPageSize
      : defaultPageSize.value;
    const layout = isTabLayout(next.tabLayout) ? next.tabLayout : tabLayout.value;
    if (
      connectionTint.value === next.connectionTint &&
      defaultPageSize.value === size &&
      showAllDatabases.value === next.showAllDatabases &&
      tabLayout.value === layout
    ) {
      return;
    }
    connectionTint.value = next.connectionTint;
    defaultPageSize.value = size;
    showAllDatabases.value = next.showAllDatabases;
    tabLayout.value = layout;
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

  function setShowAllDatabases(on: boolean) {
    if (showAllDatabases.value === on) {
      return;
    }
    showAllDatabases.value = on;
    commit();
  }

  function setTabLayout(layout: TabLayout) {
    if (tabLayout.value === layout) {
      return;
    }
    tabLayout.value = layout;
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
      showAllDatabases: data.showAllDatabases !== false,
      tabLayout: isTabLayout(data.tabLayout) ? data.tabLayout : DEFAULT_TAB_LAYOUT,
    });
  });

  return {
    connectionTint,
    defaultPageSize,
    showAllDatabases,
    tabLayout,
    setDefaultPageSize,
    setConnectionTint,
    setShowAllDatabases,
    setTabLayout,
  };
});
