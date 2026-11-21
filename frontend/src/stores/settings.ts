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

export const CELL_COPY_LAYOUTS = ["grid", "line", "tab"] as const;
export type CellCopyLayout = (typeof CELL_COPY_LAYOUTS)[number];
export const DEFAULT_CELL_COPY_LAYOUT: CellCopyLayout = "grid";
export const DEFAULT_CUSTOM_CELL_DELIMITER = "\\t";
export const DEFAULT_CUSTOM_ROW_DELIMITER = "\\n";
export const DEFAULT_SIDE_VIEW_MIN_TEXT_LENGTH = 64;

export const UPDATE_CHECK_SOURCES = ["manual", "automatic"] as const;
export type UpdateCheckSource = (typeof UPDATE_CHECK_SOURCES)[number];

const PAGE_SIZE_KEY = "lotustab.tablePageSize";
const TINT_KEY = "lotustab.connectionTint";
const SHOW_ALL_DBS_KEY = "lotustab.showAllDatabases";
const ENABLE_DB_CHOICE_KEY = "lotustab.enableDatabaseChoice";
const TAB_LAYOUT_KEY = "lotustab.tabLayout";
const QUERY_TIMEOUT_KEY = "lotustab.queryTimeout";
const DELETE_ON_ABORT_KEY = "lotustab.deleteOnAbortExport";
const CELL_COPY_LAYOUT_KEY = "lotustab.cellCopyLayout";
const USE_CUSTOM_CELL_DELIMITER_KEY = "lotustab.useCustomCellDelimiter";
const USE_CUSTOM_ROW_DELIMITER_KEY = "lotustab.useCustomRowDelimiter";
const CUSTOM_CELL_DELIMITER_KEY = "lotustab.customCellDelimiter";
const CUSTOM_ROW_DELIMITER_KEY = "lotustab.customRowDelimiter";
const SIDE_VIEW_MIN_TEXT_LENGTH_KEY = "lotustab.sideViewMinTextLength";
const APPLY_SIDE_VIEW_LENGTH_TO_JSON_KEY = "lotustab.applySideViewLengthToJson";
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

export function isCellCopyLayout(v: unknown): v is CellCopyLayout {
  return (CELL_COPY_LAYOUTS as readonly string[]).includes(v as string);
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

function readCellCopyLayout(): CellCopyLayout {
  const v = localStorage.getItem(CELL_COPY_LAYOUT_KEY);
  return isCellCopyLayout(v) ? v : DEFAULT_CELL_COPY_LAYOUT;
}

function readUseCustomCellDelimiter(): boolean {
  return localStorage.getItem(USE_CUSTOM_CELL_DELIMITER_KEY) === "true";
}

function readUseCustomRowDelimiter(): boolean {
  return localStorage.getItem(USE_CUSTOM_ROW_DELIMITER_KEY) === "true";
}

function readCustomDelimiter(key: string, fallback: string): string {
  const v = localStorage.getItem(key);
  return v == null ? fallback : v;
}

export function isSideViewMinTextLength(n: number): boolean {
  return Number.isInteger(n) && n >= 0;
}

function readSideViewMinTextLength(): number {
  const raw = localStorage.getItem(SIDE_VIEW_MIN_TEXT_LENGTH_KEY);
  if (raw == null || raw === "") {
    return DEFAULT_SIDE_VIEW_MIN_TEXT_LENGTH;
  }
  const n = Number(raw);
  return isSideViewMinTextLength(n) ? n : DEFAULT_SIDE_VIEW_MIN_TEXT_LENGTH;
}

function readApplySideViewLengthToJson(): boolean {
  return localStorage.getItem(APPLY_SIDE_VIEW_LENGTH_TO_JSON_KEY) === "true";
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
  cellCopyLayout: CellCopyLayout;
  useCustomCellDelimiter: boolean;
  useCustomRowDelimiter: boolean;
  customCellDelimiter: string;
  customRowDelimiter: string;
  sideViewMinTextLength: number;
  applySideViewLengthToJson: boolean;
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
  const cellCopyLayout = ref(readCellCopyLayout());
  const useCustomCellDelimiter = ref(readUseCustomCellDelimiter());
  const useCustomRowDelimiter = ref(readUseCustomRowDelimiter());
  const customCellDelimiter = ref(readCustomDelimiter(CUSTOM_CELL_DELIMITER_KEY, DEFAULT_CUSTOM_CELL_DELIMITER));
  const customRowDelimiter = ref(readCustomDelimiter(CUSTOM_ROW_DELIMITER_KEY, DEFAULT_CUSTOM_ROW_DELIMITER));
  const sideViewMinTextLength = ref(readSideViewMinTextLength());
  const applySideViewLengthToJson = ref(readApplySideViewLengthToJson());
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
      cellCopyLayout: cellCopyLayout.value,
      useCustomCellDelimiter: useCustomCellDelimiter.value,
      useCustomRowDelimiter: useCustomRowDelimiter.value,
      customCellDelimiter: customCellDelimiter.value,
      customRowDelimiter: customRowDelimiter.value,
      sideViewMinTextLength: sideViewMinTextLength.value,
      applySideViewLengthToJson: applySideViewLengthToJson.value,
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
    localStorage.setItem(CELL_COPY_LAYOUT_KEY, cellCopyLayout.value);
    localStorage.setItem(USE_CUSTOM_CELL_DELIMITER_KEY, String(useCustomCellDelimiter.value));
    localStorage.setItem(USE_CUSTOM_ROW_DELIMITER_KEY, String(useCustomRowDelimiter.value));
    localStorage.setItem(CUSTOM_CELL_DELIMITER_KEY, customCellDelimiter.value);
    localStorage.setItem(CUSTOM_ROW_DELIMITER_KEY, customRowDelimiter.value);
    localStorage.setItem(SIDE_VIEW_MIN_TEXT_LENGTH_KEY, String(sideViewMinTextLength.value));
    localStorage.setItem(APPLY_SIDE_VIEW_LENGTH_TO_JSON_KEY, String(applySideViewLengthToJson.value));
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
      cellCopyLayout: isCellCopyLayout(next.cellCopyLayout) ? next.cellCopyLayout : cellCopyLayout.value,
      useCustomCellDelimiter:
        typeof next.useCustomCellDelimiter === "boolean"
          ? next.useCustomCellDelimiter
          : useCustomCellDelimiter.value,
      useCustomRowDelimiter:
        typeof next.useCustomRowDelimiter === "boolean"
          ? next.useCustomRowDelimiter
          : useCustomRowDelimiter.value,
      customCellDelimiter:
        typeof next.customCellDelimiter === "string" ? next.customCellDelimiter : customCellDelimiter.value,
      customRowDelimiter:
        typeof next.customRowDelimiter === "string" ? next.customRowDelimiter : customRowDelimiter.value,
      sideViewMinTextLength: isSideViewMinTextLength(Number(next.sideViewMinTextLength))
        ? Number(next.sideViewMinTextLength)
        : sideViewMinTextLength.value,
      applySideViewLengthToJson:
        typeof next.applySideViewLengthToJson === "boolean"
          ? next.applySideViewLengthToJson
          : applySideViewLengthToJson.value,
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
    const copyLayout = isCellCopyLayout(next.cellCopyLayout) ? next.cellCopyLayout : cellCopyLayout.value;
    const customCell =
      typeof next.useCustomCellDelimiter === "boolean"
        ? next.useCustomCellDelimiter
        : useCustomCellDelimiter.value;
    const customRow =
      typeof next.useCustomRowDelimiter === "boolean"
        ? next.useCustomRowDelimiter
        : useCustomRowDelimiter.value;
    const cellDelimiter =
      typeof next.customCellDelimiter === "string" ? next.customCellDelimiter : customCellDelimiter.value;
    const rowDelimiter =
      typeof next.customRowDelimiter === "string" ? next.customRowDelimiter : customRowDelimiter.value;
    const minTextLength = isSideViewMinTextLength(next.sideViewMinTextLength)
      ? next.sideViewMinTextLength
      : sideViewMinTextLength.value;
    const applyLengthToJson =
      typeof next.applySideViewLengthToJson === "boolean"
        ? next.applySideViewLengthToJson
        : applySideViewLengthToJson.value;
    if (
      connectionTint.value === next.connectionTint &&
      defaultPageSize.value === size &&
      showAllDatabases.value === next.showAllDatabases &&
      enableDatabaseChoice.value === next.enableDatabaseChoice &&
      tabLayout.value === layout &&
      queryTimeoutSeconds.value === timeout &&
      deleteOnAbortExport.value === deleteOnAbort &&
      cellCopyLayout.value === copyLayout &&
      useCustomCellDelimiter.value === customCell &&
      useCustomRowDelimiter.value === customRow &&
      customCellDelimiter.value === cellDelimiter &&
      customRowDelimiter.value === rowDelimiter &&
      sideViewMinTextLength.value === minTextLength &&
      applySideViewLengthToJson.value === applyLengthToJson &&
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
    cellCopyLayout.value = copyLayout;
    useCustomCellDelimiter.value = customCell;
    useCustomRowDelimiter.value = customRow;
    customCellDelimiter.value = cellDelimiter;
    customRowDelimiter.value = rowDelimiter;
    sideViewMinTextLength.value = minTextLength;
    applySideViewLengthToJson.value = applyLengthToJson;
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

  function setCellCopyLayout(layout: CellCopyLayout) {
    if (cellCopyLayout.value === layout) {
      return;
    }
    cellCopyLayout.value = layout;
    commit();
  }

  function setUseCustomCellDelimiter(on: boolean) {
    if (useCustomCellDelimiter.value === on) {
      return;
    }
    useCustomCellDelimiter.value = on;
    commit();
  }

  function setUseCustomRowDelimiter(on: boolean) {
    if (useCustomRowDelimiter.value === on) {
      return;
    }
    useCustomRowDelimiter.value = on;
    commit();
  }

  function setCustomCellDelimiter(value: string) {
    if (customCellDelimiter.value === value) {
      return;
    }
    customCellDelimiter.value = value;
    commit();
  }

  function setCustomRowDelimiter(value: string) {
    if (customRowDelimiter.value === value) {
      return;
    }
    customRowDelimiter.value = value;
    commit();
  }

  function setSideViewMinTextLength(length: number) {
    if (!isSideViewMinTextLength(length) || sideViewMinTextLength.value === length) {
      return;
    }
    sideViewMinTextLength.value = length;
    commit();
  }

  function setApplySideViewLengthToJson(on: boolean) {
    if (applySideViewLengthToJson.value === on) {
      return;
    }
    applySideViewLengthToJson.value = on;
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
    cellCopyLayout,
    useCustomCellDelimiter,
    useCustomRowDelimiter,
    customCellDelimiter,
    customRowDelimiter,
    sideViewMinTextLength,
    applySideViewLengthToJson,
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
    setCellCopyLayout,
    setUseCustomCellDelimiter,
    setUseCustomRowDelimiter,
    setCustomCellDelimiter,
    setCustomRowDelimiter,
    setSideViewMinTextLength,
    setApplySideViewLengthToJson,
    setAutoInstallUpdates,
    recordUpdateCheck,
  };
});
