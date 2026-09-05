import { errorMessage, isCanceled } from "@/lib/errors";
import { QueryService } from "@/lib/wails";
import { draftLookup, rowKey, valuesEqual } from "@/features/results/edits";
import { completeFilters, emptyFilter, type TableFilter } from "@/features/results/filters";
import { asQueryResult } from "@/features/results/browse";
import { useConnectionsStore } from "@/stores/connections";
import { useSettingsStore } from "@/stores/settings";
import type { QueryStoreCtx } from "@/stores/queryContext";
import type { QueryExecApi } from "@/stores/queryExec";

export function createTableBrowse(ctx: QueryStoreCtx, exec: QueryExecApi) {
  async function loadTable(id: string) {
    const tab = ctx.tableTab(id);
    if (!tab) {
      return;
    }
    if (!useConnectionsStore().isConnected(tab.profileId)) {
      tab.running = false;
      return;
    }
    if (!tab.edits) {
      tab.edits = [];
    }
    if (!tab.filters) {
      tab.filters = [];
    }
    ctx.cancel(id);
    tab.running = true;
    tab.manuallyCancelled = false;
    tab.error = "";
    const pending = QueryService.BrowseTable(tab.profileId, {
      database: tab.database,
      schema: tab.schema,
      name: tab.name,
      limit: tab.pageSize,
      offset: (tab.page - 1) * tab.pageSize,
      orderBy: tab.orderBy,
      orderDir: tab.orderDir,
      filters: tab.filtersEnabled ? completeFilters(tab.filters) : [],
    });
    ctx.inflight.set(id, pending);
    try {
      const result = await pending;
      if (ctx.inflight.get(id) !== pending) {
        return;
      }
      tab.result = asQueryResult(result);
      tab.total = result?.total ?? 0;
      tab.primaryKey = result?.primaryKey ?? [];
    } catch (err) {
      if (isCanceled(err) || ctx.inflight.get(id) !== pending) {
        return;
      }
      tab.result = null;
      tab.total = 0;
      tab.primaryKey = [];
      tab.error = errorMessage(err);
    } finally {
      if (ctx.inflight.get(id) === pending) {
        ctx.inflight.delete(id);
        tab.running = false;
      }
    }
  }

  function dropTableData(profileId: string) {
    for (const tab of ctx.tabs.value) {
      if (tab.profileId !== profileId || tab.kind !== "table") {
        continue;
      }
      tab.result = null;
      tab.total = 0;
      tab.primaryKey = [];
      tab.edits = [];
      tab.error = "";
      tab.running = false;
      tab.structure = null;
      tab.structureDraft = null;
      tab.structureError = "";
    }
  }

  function setPage(id: string, page: number) {
    const tab = ctx.tabs.value.find((t) => t.id === id);
    if (!tab) {
      return;
    }
    if (tab.kind === "table") {
      const maxPage = Math.max(1, Math.ceil(tab.total / tab.pageSize) || 1);
      tab.page = Math.min(Math.max(1, page), maxPage);
      void loadTable(id);
      return;
    }
    if (tab.kind === "query") {
      tab.page = Math.max(1, page);
      void exec.execute(id);
    }
  }

  function setPageSize(id: string, pageSize: number) {
    const tab = ctx.tabs.value.find((t) => t.id === id);
    if (!tab || (tab.kind !== "table" && tab.kind !== "query")) {
      return;
    }
    tab.pageSize = pageSize;
    tab.page = 1;
    useSettingsStore().setDefaultPageSize(pageSize);
    if (tab.kind === "table") {
      void loadTable(id);
      return;
    }
    void exec.execute(id);
  }

  function cycleSort(id: string, column: string) {
    const tab = ctx.tableTab(id);
    if (!tab) {
      return;
    }
    if (tab.orderBy !== column) {
      tab.orderBy = column;
      tab.orderDir = "asc";
    } else if (tab.orderDir === "asc") {
      tab.orderDir = "desc";
    } else {
      tab.orderBy = "";
      tab.orderDir = "";
    }
    tab.page = 1;
    void loadTable(id);
  }

  function setFilters(id: string, filters: TableFilter[]) {
    const tab = ctx.tableTab(id);
    if (!tab) {
      return;
    }
    tab.filters = filters;
  }

  function addFilter(id: string) {
    const tab = ctx.tableTab(id);
    if (!tab || !tab.filtersEnabled) {
      return;
    }
    tab.filters = [...(tab.filters ?? []), emptyFilter()];
    tab.filtersOpen = true;
  }

  function setFiltersEnabled(id: string, enabled: boolean) {
    const tab = ctx.tableTab(id);
    if (!tab) {
      return;
    }
    tab.filtersEnabled = enabled;
    if (enabled) {
      tab.filtersOpen = true;
      if (!tab.filters?.length) {
        tab.filters = [emptyFilter()];
      }
    }
    tab.page = 1;
    void loadTable(id);
  }

  function toggleFiltersOpen(id: string) {
    const tab = ctx.tableTab(id);
    if (!tab) {
      return;
    }
    tab.filtersOpen = !tab.filtersOpen;
    if (tab.filtersOpen && !tab.filters?.length) {
      tab.filters = [emptyFilter()];
    }
  }

  function applyFilters(id: string) {
    const tab = ctx.tableTab(id);
    if (!tab || !tab.filtersEnabled) {
      return;
    }
    tab.page = 1;
    void loadTable(id);
  }

  function setCellDraft(id: string, rowIndex: number, column: string, value: unknown) {
    const tab = ctx.tableTab(id);
    if (!tab || !tab.primaryKey?.length) {
      return;
    }
    if (!tab.edits) {
      tab.edits = [];
    }
    const columns = tab.result?.columns ?? [];
    const row = tab.result?.rows?.[rowIndex];
    const key = rowKey(row, columns, tab.primaryKey);
    if (!key) {
      return;
    }
    const colIndex = columns.findIndex((c) => c.name === column);
    if (colIndex < 0) {
      return;
    }
    const original = row?.[colIndex];
    const lookup = draftLookup(key);
    const edits = tab.edits.slice();
    const index = edits.findIndex((e) => draftLookup(e.key) === lookup);
    const cells = { ...(index >= 0 ? edits[index]!.cells : {}) };
    if (valuesEqual(original, value)) {
      delete cells[column];
    } else {
      cells[column] = value;
    }
    if (Object.keys(cells).length === 0) {
      if (index >= 0) {
        edits.splice(index, 1);
      }
    } else if (index >= 0) {
      edits[index] = { key, cells };
    } else {
      edits.push({ key, cells });
    }
    tab.edits = edits;
  }

  function clearEdits(id: string) {
    const tab = ctx.tableTab(id);
    if (!tab) {
      return;
    }
    tab.edits = [];
  }

  async function applyEdits(id: string) {
    const tab = ctx.tableTab(id);
    if (!tab || !tab.edits?.length) {
      return;
    }
    await QueryService.ApplyEdits(tab.profileId, {
      database: tab.database,
      schema: tab.schema,
      name: tab.name,
      rows: tab.edits,
    });
    tab.edits = [];
    await loadTable(id);
  }

  return {
    loadTable,
    dropTableData,
    setPage,
    setPageSize,
    cycleSort,
    setFilters,
    addFilter,
    setFiltersEnabled,
    toggleFiltersOpen,
    applyFilters,
    setCellDraft,
    clearEdits,
    applyEdits,
  };
}

export type TableBrowseApi = ReturnType<typeof createTableBrowse>;
