import { defineStore } from "pinia";
import { computed, ref } from "vue";
import { watchDebounced } from "@vueuse/core";
import { readJson, writeJson } from "@/lib/persist";
import {
  afterPaint,
  isQueryTab,
  isTableTab,
  tableTitle,
  type QueryTab,
  type TableTab,
  type WorkspaceSnapshot,
  type WorkspaceTab,
} from "@/features/editor/tabs";
import {
  hydrateTabs,
  parseActiveByProfile,
  parseWorkspace,
  persistTab,
} from "@/features/editor/workspacePersist";
import type { TableView } from "@/features/results/structure";
import { useExplorerStore } from "@/stores/explorer";
import { createQueryExec } from "@/stores/queryExec";
import { createTableBrowse } from "@/stores/tableBrowse";
import { createTableStructure } from "@/stores/tableStructure";
import type { QueryStoreCtx } from "@/stores/queryContext";
import { useSettingsStore } from "@/stores/settings";

const WORKSPACE_KEY = "lotustab.workspace";
const MOUNTED_TAB_LIMIT = 5;

function storedPageSize(): number {
  return useSettingsStore().defaultPageSize;
}

function defaultQueryDatabase(): string {
  if (useSettingsStore().showAllDatabases) {
    return "";
  }
  return useExplorerStore().activeDatabase || "";
}

function newQueryTab(profileId: string, title: string): QueryTab {
  return {
    kind: "query",
    id: crypto.randomUUID(),
    title,
    profileId,
    database: defaultQueryDatabase(),
    sql: "SELECT 1;",
    executedSql: "",
    page: 1,
    pageSize: storedPageSize(),
    running: false,
    error: "",
    result: null,
    results: [],
    selectedResult: 0,
    followResults: true,
    runId: "",
  };
}

export const useWorkspaceStore = defineStore("query", () => {
  const stored = parseWorkspace(readJson(WORKSPACE_KEY, {}));
  const tabs = ref<WorkspaceTab[]>(hydrateTabs(stored, storedPageSize()));
  const activeTabId = ref(
    stored.activeTabId && tabs.value.some((t) => t.id === stored.activeTabId)
      ? stored.activeTabId
      : (tabs.value[0]?.id ?? ""),
  );
  const activeByProfile = ref<Record<string, string>>(
    parseActiveByProfile(stored.activeByProfile, tabs.value, activeTabId.value),
  );
  const recentTabIds = ref<string[]>(activeTabId.value ? [activeTabId.value] : []);
  const coldMountId = ref("");
  const inflight = new Map<string, { cancel: () => void }>();

  watchDebounced(
    () => ({
      tabs: tabs.value.map(persistTab),
      activeTabId: activeTabId.value,
      activeByProfile: activeByProfile.value,
    }),
    (value) => writeJson(WORKSPACE_KEY, value),
    { debounce: 300, deep: true },
  );

  const activeTab = computed(() => tabs.value.find((t) => t.id === activeTabId.value));

  function tableTab(id: string): TableTab | undefined {
    const tab = tabs.value.find((t) => t.id === id);
    return isTableTab(tab) ? tab : undefined;
  }

  function queryTab(id: string): QueryTab | undefined {
    const tab = tabs.value.find((t) => t.id === id);
    return isQueryTab(tab) ? tab : undefined;
  }

  function cancel(id: string) {
    inflight.get(id)?.cancel();
    inflight.delete(id);
    const tab = tabs.value.find((t) => t.id === id);
    if (tab) {
      tab.running = false;
      if (tab.kind === "table") {
        tab.manuallyCancelled = true;
      }
    }
  }

  const ctx: QueryStoreCtx = {
    tabs,
    inflight,
    cancel,
    storedPageSize,
    tableTab,
    queryTab,
  };

  const exec = createQueryExec(ctx);
  const browse = createTableBrowse(ctx, exec);
  const structure = createTableStructure(ctx);

  function touchRecent(id: string) {
    if (!id) {
      return;
    }
    recentTabIds.value = [id, ...recentTabIds.value.filter((x) => x !== id)];
  }

  function forgetRecent(id: string) {
    if (!recentTabIds.value.includes(id)) {
      return;
    }
    recentTabIds.value = recentTabIds.value.filter((x) => x !== id);
  }

  function mountedTabIds(profileId: string): string[] {
    if (!profileId) {
      return [];
    }
    const inProfile = new Set(tabs.value.filter((t) => t.profileId === profileId).map((t) => t.id));
    const out: string[] = [];
    const active = activeTabId.value;
    if (active && inProfile.has(active)) {
      out.push(active);
    }
    for (const id of recentTabIds.value) {
      if (out.length >= MOUNTED_TAB_LIMIT) {
        break;
      }
      if (!inProfile.has(id) || out.includes(id)) {
        continue;
      }
      out.push(id);
    }
    return out;
  }

  function selectTab(id: string) {
    if (!id) {
      activeTabId.value = "";
      return;
    }
    const tab = tabs.value.find((t) => t.id === id);
    if (!tab) {
      activeTabId.value = "";
      return;
    }
    const isCold = !mountedTabIds(tab.profileId).includes(id);
    activeTabId.value = id;
    touchRecent(id);
    if (isCold) {
      coldMountId.value = id;
      afterPaint(() => {
        if (coldMountId.value === id) {
          coldMountId.value = "";
        }
      });
    }
    if (activeByProfile.value[tab.profileId] === id) {
      return;
    }
    activeByProfile.value = { ...activeByProfile.value, [tab.profileId]: id };
  }

  function activateProfile(profileId: string) {
    if (!profileId) {
      activeTabId.value = "";
      return;
    }
    const remembered = activeByProfile.value[profileId];
    const stillThere =
      Boolean(remembered) &&
      tabs.value.some((t) => t.id === remembered && t.profileId === profileId);
    selectTab(stillThere ? remembered! : (tabs.value.find((t) => t.profileId === profileId)?.id ?? ""));
  }

  function forgetProfile(profileId: string) {
    if (!(profileId in activeByProfile.value)) {
      return;
    }
    const next = { ...activeByProfile.value };
    delete next[profileId];
    activeByProfile.value = next;
  }

  function openForProfile(profileId: string, name: string) {
    const existing = tabs.value.find((t) => t.kind === "query" && t.profileId === profileId);
    if (existing) {
      selectTab(existing.id);
      return existing;
    }
    return add(profileId, name);
  }

  function add(profileId: string, name: string) {
    const count = tabs.value.filter((t) => t.kind === "query" && t.profileId === profileId).length;
    const tab = newQueryTab(profileId, count === 0 ? name : `${name} ${count + 1}`);
    tabs.value.push(tab);
    selectTab(tab.id);
    return tab;
  }

  function reloadTab(id: string) {
    const tab = tableTab(id);
    if (!tab) {
      return;
    }
    if (tab.view === "structure") {
      return structure.loadStructure(id);
    }
    return browse.loadTable(id);
  }

  function openTable(opts: {
    profileId: string;
    database: string;
    schema: string;
    name: string;
    view?: TableView;
  }) {
    const view = opts.view === "structure" ? "structure" : "data";
    const existing = tabs.value.find(
      (t): t is TableTab =>
        t.kind === "table" &&
        t.profileId === opts.profileId &&
        t.database === opts.database &&
        t.schema === opts.schema &&
        t.name === opts.name &&
        t.view === view,
    );
    if (existing) {
      selectTab(existing.id);
      if (existing.view === "structure") {
        if (existing.structure || existing.structureDraft) {
          return existing;
        }
        existing.running = true;
        afterPaint(() => void structure.loadStructure(existing.id));
      } else if (!existing.result && !existing.manuallyCancelled) {
        existing.running = true;
        afterPaint(() => void browse.loadTable(existing.id));
      }
      return existing;
    }
    const tab: TableTab = {
      kind: "table",
      id: crypto.randomUUID(),
      title: tableTitle(opts.schema, opts.name, opts.database),
      profileId: opts.profileId,
      database: opts.database,
      schema: opts.schema,
      name: opts.name,
      page: 1,
      pageSize: storedPageSize(),
      orderBy: "",
      orderDir: "",
      running: true,
      error: "",
      result: null,
      total: 0,
      primaryKey: [],
      edits: [],
      filtersEnabled: false,
      filtersOpen: false,
      filters: [],
      view,
      structure: null,
      structureDraft: null,
      structureError: "",
      manuallyCancelled: false,
    };
    tabs.value.push(tab);
    selectTab(tab.id);
    afterPaint(() => void reloadTab(tab.id));
    return tab;
  }

  function close(id: string) {
    cancel(id);
    const index = tabs.value.findIndex((t) => t.id === id);
    if (index < 0) {
      return;
    }
    const profileId = tabs.value[index]!.profileId;
    const groupIndex = tabs.value.filter((t) => t.profileId === profileId).findIndex((t) => t.id === id);
    tabs.value.splice(index, 1);
    forgetRecent(id);
    if (activeTabId.value !== id) {
      return;
    }
    const group = tabs.value.filter((t) => t.profileId === profileId);
    const next = group[groupIndex] ?? group[groupIndex - 1];
    if (next) {
      selectTab(next.id);
      return;
    }
    activeTabId.value = "";
    forgetProfile(profileId);
  }

  function snapshotForProfile(profileId: string): WorkspaceSnapshot {
    const group = tabs.value.filter((t) => t.profileId === profileId);
    const remembered = activeByProfile.value[profileId];
    return {
      tabs: group.map((t) => {
        if (t.kind === "table") {
          return { ...t, running: false, edits: [], structureDraft: t.structureDraft ? t.structureDraft.map((col) => ({ ...col })) : null };
        }
        return { ...t, running: false };
      }),
      activeTabId: group.some((t) => t.id === remembered) ? remembered : (group[0]?.id ?? ""),
    };
  }

  function removeForProfile(profileId: string) {
    const removingActive =
      tabs.value.find((t) => t.id === activeTabId.value)?.profileId === profileId;
    const removed = new Set(tabs.value.filter((t) => t.profileId === profileId).map((t) => t.id));
    tabs.value = tabs.value.filter((t) => t.profileId !== profileId);
    recentTabIds.value = recentTabIds.value.filter((id) => !removed.has(id));
    forgetProfile(profileId);
    if (removingActive) {
      selectTab(tabs.value[0]?.id ?? "");
    }
  }

  function restore(snapshot: WorkspaceSnapshot) {
    if (!snapshot.tabs.length) {
      return;
    }
    const profileId = snapshot.tabs[0]!.profileId;
    const group = tabs.value.filter((t) => t.profileId === profileId);
    const placeholder = group[0];
    if (
      group.length === 1 &&
      placeholder &&
      placeholder.kind === "query" &&
      placeholder.sql === "SELECT 1;" &&
      !placeholder.result
    ) {
      tabs.value = tabs.value.filter((t) => t.id !== placeholder.id);
      forgetRecent(placeholder.id);
    }
    const existing = new Set(tabs.value.map((t) => t.id));
    for (const tab of snapshot.tabs) {
      if (!existing.has(tab.id)) {
        if (tab.kind === "query") {
          tabs.value.push({
            ...tab,
            running: false,
            executedSql: tab.executedSql ?? "",
            results: tab.results ?? [],
            selectedResult: tab.selectedResult ?? 0,
            followResults: true,
            runId: "",
            page: tab.page >= 1 ? tab.page : 1,
            pageSize: tab.pageSize || storedPageSize(),
          });
        } else {
          tabs.value.push({
            ...tab,
            running: false,
            edits: [],
            primaryKey: tab.primaryKey ?? [],
            filtersEnabled: tab.filtersEnabled ?? false,
            filtersOpen: tab.filtersOpen ?? false,
            filters: tab.filters ?? [],
            view: tab.view === "structure" ? "structure" : "data",
            structure: tab.structure ?? null,
            structureDraft: null,
            structureError: "",
          });
        }
      }
    }
    selectTab(
      snapshot.activeTabId && tabs.value.some((t) => t.id === snapshot.activeTabId)
        ? snapshot.activeTabId
        : snapshot.tabs[0]!.id,
    );
  }

  function setOrder(profileId: string, ids: string[]) {
    const group = tabs.value.filter((t) => t.profileId === profileId);
    if (group.length !== ids.length) {
      return;
    }
    const byId = new Map(group.map((t) => [t.id, t]));
    const ordered: WorkspaceTab[] = [];
    for (const id of ids) {
      const tab = byId.get(id);
      if (!tab) {
        return;
      }
      ordered.push(tab);
    }
    let i = 0;
    tabs.value = tabs.value.map((t) => (t.profileId === profileId ? ordered[i++]! : t));
  }

  return {
    tabs,
    activeTabId,
    activeTab,
    mountedTabIds,
    coldMountId,
    selectTab,
    activateProfile,
    openForProfile,
    add,
    openTable,
    close,
    snapshotForProfile,
    removeForProfile,
    restore,
    setOrder,
    cancel,
    reloadTab,
    ...exec,
    ...browse,
    ...structure,
  };
});
