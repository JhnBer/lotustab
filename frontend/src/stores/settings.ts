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

export const UPDATE_CHECK_SOURCES = ["manual", "automatic"] as const;
export type UpdateCheckSource = (typeof UPDATE_CHECK_SOURCES)[number];

const PAGE_SIZE_KEY = "lotustab.tablePageSize";
const TINT_KEY = "lotustab.connectionTint";
const SHOW_ALL_DBS_KEY = "lotustab.showAllDatabases";
const ENABLE_DB_CHOICE_KEY = "lotustab.enableDatabaseChoice";
const TAB_LAYOUT_KEY = "lotustab.tabLayout";
const QUERY_TIMEOUT_KEY = "lotustab.queryTimeout";
const DELETE_ON_ABORT_KEY = "lotustab.deleteOnAbortExport";
const AUTO_INSTALL_KEY = "lotustab.autoInstallUpdates";
const LAST_CHECK_AT_KEY = "lotustab.lastUpdateCheckAt";
const LAST_CHECK_SOURCE_KEY = "lotustab.lastUpdateCheckSource";
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

function readEnableDatabaseChoice(): boolean {
  return localStorage.getItem(ENABLE_DB_CHOICE_KEY) !== "false";
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

export function isUpdateCheckSource(v: unknown): v is UpdateCheckSource {
  return (UPDATE_CHECK_SOURCES as readonly string[]).includes(v as string);
}

function readAutoInstallUpdates(): boolean {
  return localStorage.getItem(AUTO_INSTALL_KEY) === "true";
}

function readLastUpdateCheckAt(): string | null {
  const v = localStorage.getItem(LAST_CHECK_AT_KEY);
  return v && !Number.isNaN(Date.parse(v)) ? v : null;
}

function readLastUpdateCheckSource(): UpdateCheckSource | null {
  const v = localStorage.getItem(LAST_CHECK_SOURCE_KEY);
  return isUpdateCheckSource(v) ? v : null;
}

type SettingsPayload = {
  connectionTint: boolean;
  defaultPageSize: number;
  showAllDatabases: boolean;
  enableDatabaseChoice: boolean;
  tabLayout: TabLayout;
  queryTimeoutSeconds: number;
  deleteOnAbortExport: boolean;
  autoInstallUpdates: boolean;
  lastUpdateCheckAt: string | null;
  lastUpdateCheckSource: UpdateCheckSource | null;
};

export const useSettingsStore = defineStore("settings", () => {
  const connectionTint = ref(readTint());
  const defaultPageSize = ref(readPageSize());
  const showAllDatabases = ref(readShowAllDatabases());
  const enableDatabaseChoice = ref(readEnableDatabaseChoice());
  const tabLayout = ref(readTabLayout());
  const queryTimeoutSeconds = ref(readQueryTimeout());
  const deleteOnAbortExport = ref(readDeleteOnAbortExport());
  const autoInstallUpdates = ref(readAutoInstallUpdates());
  const lastUpdateCheckAt = ref<string | null>(readLastUpdateCheckAt());
  const lastUpdateCheckSource = ref<UpdateCheckSource | null>(readLastUpdateCheckSource());

  function snapshot(): SettingsPayload {
    return {
      connectionTint: connectionTint.value,
      defaultPageSize: defaultPageSize.value,
      showAllDatabases: showAllDatabases.value,
      enableDatabaseChoice: enableDatabaseChoice.value,
      tabLayout: tabLayout.value,
      queryTimeoutSeconds: queryTimeoutSeconds.value,
      deleteOnAbortExport: deleteOnAbortExport.value,
      autoInstallUpdates: autoInstallUpdates.value,
      lastUpdateCheckAt: lastUpdateCheckAt.value,
      lastUpdateCheckSource: lastUpdateCheckSource.value,
    };
  }

  function persist() {
    localStorage.setItem(TINT_KEY, String(connectionTint.value));
    localStorage.setItem(PAGE_SIZE_KEY, String(defaultPageSize.value));
    localStorage.setItem(SHOW_ALL_DBS_KEY, String(showAllDatabases.value));
    localStorage.setItem(ENABLE_DB_CHOICE_KEY, String(enableDatabaseChoice.value));
    localStorage.setItem(TAB_LAYOUT_KEY, tabLayout.value);
    localStorage.setItem(QUERY_TIMEOUT_KEY, String(queryTimeoutSeconds.value));
    localStorage.setItem(DELETE_ON_ABORT_KEY, String(deleteOnAbortExport.value));
    localStorage.setItem(AUTO_INSTALL_KEY, String(autoInstallUpdates.value));
    if (lastUpdateCheckAt.value) {
      localStorage.setItem(LAST_CHECK_AT_KEY, lastUpdateCheckAt.value);
    } else {
      localStorage.removeItem(LAST_CHECK_AT_KEY);
    }
    if (lastUpdateCheckSource.value) {
      localStorage.setItem(LAST_CHECK_SOURCE_KEY, lastUpdateCheckSource.value);
    } else {
      localStorage.removeItem(LAST_CHECK_SOURCE_KEY);
    }
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
      enableDatabaseChoice: next.enableDatabaseChoice !== false,
      tabLayout: isTabLayout(next.tabLayout) ? next.tabLayout : DEFAULT_TAB_LAYOUT,
      queryTimeoutSeconds: Number(next.queryTimeoutSeconds),
      deleteOnAbortExport:
        typeof next.deleteOnAbortExport === "boolean"
          ? next.deleteOnAbortExport
          : deleteOnAbortExport.value,
      autoInstallUpdates: Boolean(next.autoInstallUpdates),
      lastUpdateCheckAt:
        typeof next.lastUpdateCheckAt === "string" && !Number.isNaN(Date.parse(next.lastUpdateCheckAt))
          ? next.lastUpdateCheckAt
          : next.lastUpdateCheckAt === null
            ? null
            : lastUpdateCheckAt.value,
      lastUpdateCheckSource: isUpdateCheckSource(next.lastUpdateCheckSource)
        ? next.lastUpdateCheckSource
        : next.lastUpdateCheckSource === null
          ? null
          : lastUpdateCheckSource.value,
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
      enableDatabaseChoice.value === next.enableDatabaseChoice &&
      tabLayout.value === layout &&
      queryTimeoutSeconds.value === timeout &&
      deleteOnAbortExport.value === deleteOnAbort &&
      autoInstallUpdates.value === next.autoInstallUpdates &&
      lastUpdateCheckAt.value === next.lastUpdateCheckAt &&
      lastUpdateCheckSource.value === next.lastUpdateCheckSource
    ) {
      return;
    }
    connectionTint.value = next.connectionTint;
    defaultPageSize.value = size;
    showAllDatabases.value = next.showAllDatabases;
    enableDatabaseChoice.value = next.enableDatabaseChoice;
    tabLayout.value = layout;
    queryTimeoutSeconds.value = timeout;
    deleteOnAbortExport.value = deleteOnAbort;
    autoInstallUpdates.value = next.autoInstallUpdates;
    lastUpdateCheckAt.value = next.lastUpdateCheckAt;
    lastUpdateCheckSource.value = next.lastUpdateCheckSource;
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

  function setEnableDatabaseChoice(on: boolean) {
    if (enableDatabaseChoice.value === on) {
      return;
    }
    enableDatabaseChoice.value = on;
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

  function setAutoInstallUpdates(on: boolean) {
    if (autoInstallUpdates.value === on) {
      return;
    }
    autoInstallUpdates.value = on;
    commit();
  }

  function recordUpdateCheck(source: UpdateCheckSource) {
    lastUpdateCheckAt.value = new Date().toISOString();
    lastUpdateCheckSource.value = source;
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
    enableDatabaseChoice,
    tabLayout,
    queryTimeoutSeconds,
    deleteOnAbortExport,
    autoInstallUpdates,
    lastUpdateCheckAt,
    lastUpdateCheckSource,
    setDefaultPageSize,
    setConnectionTint,
    setShowAllDatabases,
    setEnableDatabaseChoice,
    setTabLayout,
    setQueryTimeout,
    setDeleteOnAbortExport,
    setAutoInstallUpdates,
    recordUpdateCheck,
  };
});
