import { defineStore } from "pinia";
import { ref } from "vue";
import { Events } from "@wailsio/runtime";

export const PAGE_SIZES = [50, 100, 250, 500, 1000, 5000, 10000] as const;
export const DEFAULT_PAGE_SIZE = 100;

export const QUERY_TIMEOUTS = [0, 15, 30, 60, 300, 900] as const;
export const DEFAULT_QUERY_TIMEOUT = 0;

export const TAB_LAYOUTS = ["scroll", "wrap"] as const;
export type TabLayout = (typeof TAB_LAYOUTS)[number];
export const DEFAULT_TAB_LAYOUT: TabLayout = "scroll";

const PAGE_SIZE_KEY = "lotustab.tablePageSize";
const TINT_KEY = "lotustab.connectionTint";
const SHOW_ALL_DBS_KEY = "lotustab.showAllDatabases";
const TAB_LAYOUT_KEY = "lotustab.tabLayout";
const QUERY_TIMEOUT_KEY = "lotustab.queryTimeout";
const DELETE_ON_ABORT_KEY = "lotustab.deleteOnAbortExport";
const SETTINGS_CHANGED = "settings:changed";

export function isPageSize(n: number): boolean {
  return (PAGE_SIZES as readonly number[]).includes(n);
}

export function isQueryTimeout(n: number): boolean {
  return (QUERY_TIMEOUTS as readonly number[]).includes(n);
}

export function queryTimeoutLabel(seconds: number): string {
  if (seconds <= 0) {
    return "Off";
  }
  if (seconds < 60) {
    return `${seconds}s`;
  }
  return `${seconds / 60} min`;
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

function readQueryTimeout(): number {
  const n = Number(localStorage.getItem(QUERY_TIMEOUT_KEY));
  return isQueryTimeout(n) ? n : DEFAULT_QUERY_TIMEOUT;
}

function readDeleteOnAbortExport(): boolean {
  return localStorage.getItem(DELETE_ON_ABORT_KEY) !== "false";
}

type SettingsPayload = {
  connectionTint: boolean;
  defaultPageSize: number;
  showAllDatabases: boolean;
  tabLayout: TabLayout;
  queryTimeoutSeconds: number;
  deleteOnAbortExport: boolean;
};

export const useSettingsStore = defineStore("settings", () => {
  const connectionTint = ref(readTint());
  const defaultPageSize = ref(readPageSize());
  const showAllDatabases = ref(readShowAllDatabases());
  const tabLayout = ref(readTabLayout());
  const queryTimeoutSeconds = ref(readQueryTimeout());
  const deleteOnAbortExport = ref(readDeleteOnAbortExport());

  function snapshot(): SettingsPayload {
    return {
      connectionTint: connectionTint.value,
      defaultPageSize: defaultPageSize.value,
      showAllDatabases: showAllDatabases.value,
      tabLayout: tabLayout.value,
      queryTimeoutSeconds: queryTimeoutSeconds.value,
      deleteOnAbortExport: deleteOnAbortExport.value,
    };
  }

  function persist() {
    localStorage.setItem(TINT_KEY, String(connectionTint.value));
    localStorage.setItem(PAGE_SIZE_KEY, String(defaultPageSize.value));
    localStorage.setItem(SHOW_ALL_DBS_KEY, String(showAllDatabases.value));
    localStorage.setItem(TAB_LAYOUT_KEY, tabLayout.value);
    localStorage.setItem(QUERY_TIMEOUT_KEY, String(queryTimeoutSeconds.value));
    localStorage.setItem(DELETE_ON_ABORT_KEY, String(deleteOnAbortExport.value));
  }

  function commit() {
    persist();
    void Events.Emit(SETTINGS_CHANGED, snapshot()).catch(() => {});
  }

  function payloadFromEvent(raw: unknown): SettingsPayload | null {
    const data = Array.isArray(raw) ? raw[0] : raw;
    if (!data || typeof data !== "object") {
      return null;
    }
    const next = data as Partial<SettingsPayload>;
    return {
      connectionTint: Boolean(next.connectionTint),
      defaultPageSize: Number(next.defaultPageSize),
      showAllDatabases: next.showAllDatabases !== false,
      tabLayout: isTabLayout(next.tabLayout) ? next.tabLayout : DEFAULT_TAB_LAYOUT,
      queryTimeoutSeconds: Number(next.queryTimeoutSeconds),
      deleteOnAbortExport:
        typeof next.deleteOnAbortExport === "boolean"
          ? next.deleteOnAbortExport
          : deleteOnAbortExport.value,
    };
  }

  function apply(next: SettingsPayload) {
    const size = isPageSize(next.defaultPageSize)
      ? next.defaultPageSize
      : defaultPageSize.value;
    const layout = isTabLayout(next.tabLayout) ? next.tabLayout : tabLayout.value;
    const timeout = isQueryTimeout(next.queryTimeoutSeconds)
      ? next.queryTimeoutSeconds
      : queryTimeoutSeconds.value;
    const deleteOnAbort =
      typeof next.deleteOnAbortExport === "boolean"
        ? next.deleteOnAbortExport
        : deleteOnAbortExport.value;
    if (
      connectionTint.value === next.connectionTint &&
      defaultPageSize.value === size &&
      showAllDatabases.value === next.showAllDatabases &&
      tabLayout.value === layout &&
      queryTimeoutSeconds.value === timeout &&
      deleteOnAbortExport.value === deleteOnAbort
    ) {
      return;
    }
    connectionTint.value = next.connectionTint;
    defaultPageSize.value = size;
    showAllDatabases.value = next.showAllDatabases;
    tabLayout.value = layout;
    queryTimeoutSeconds.value = timeout;
    deleteOnAbortExport.value = deleteOnAbort;
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

  function setQueryTimeout(seconds: number) {
    if (!isQueryTimeout(seconds) || queryTimeoutSeconds.value === seconds) {
      return;
    }
    queryTimeoutSeconds.value = seconds;
    commit();
  }

  function setDeleteOnAbortExport(on: boolean) {
    if (deleteOnAbortExport.value === on) {
      return;
    }
    deleteOnAbortExport.value = on;
    commit();
  }

  Events.On(SETTINGS_CHANGED, (ev) => {
    const data = payloadFromEvent(ev.data);
    if (!data) {
      return;
    }
    apply(data);
  });

  return {
    connectionTint,
    defaultPageSize,
    showAllDatabases,
    tabLayout,
    queryTimeoutSeconds,
    deleteOnAbortExport,
    setDefaultPageSize,
    setConnectionTint,
    setShowAllDatabases,
    setTabLayout,
    setQueryTimeout,
    setDeleteOnAbortExport,
  };
});
