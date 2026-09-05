import { errorMessage, isCanceled } from "@/lib/errors";
import { QueryService } from "@/lib/wails";
import {
  columnsToDraft,
  emptyDraftColumn,
  isStructureDirty,
  nextColumnName,
  structureOps,
  type DraftColumn,
} from "@/features/results/structure";
import { useConnectionsStore } from "@/stores/connections";
import type { QueryStoreCtx } from "@/stores/queryContext";
import type { TableTab } from "@/features/editor/tabs";

export function createTableStructure(ctx: QueryStoreCtx) {
  function clearBrowse(tab: TableTab) {
    tab.edits = [];
    tab.result = null;
    tab.total = 0;
    tab.primaryKey = [];
  }

  async function loadStructure(id: string, opts?: { force?: boolean }) {
    const tab = ctx.tableTab(id);
    if (!tab) {
      return;
    }
    if (!useConnectionsStore().isConnected(tab.profileId)) {
      tab.running = false;
      return;
    }
    const keepDraft =
      !opts?.force &&
      Boolean(tab.structureDraft) &&
      isStructureDirty(tab.structure?.columns, tab.structureDraft, tab.structure?.canReorder ?? false);
    ctx.cancel(id);
    tab.running = true;
    tab.structureError = "";
    const pending = QueryService.DescribeTable(tab.profileId, {
      database: tab.database,
      schema: tab.schema,
      name: tab.name,
    });
    ctx.inflight.set(id, pending);
    try {
      const result = await pending;
      if (ctx.inflight.get(id) !== pending) {
        return;
      }
      tab.structure = result
        ? { columns: result.columns ?? [], canReorder: Boolean(result.canReorder) }
        : { columns: [], canReorder: false };
      if (!keepDraft) {
        tab.structureDraft = columnsToDraft(tab.structure.columns);
      }
    } catch (err) {
      if (isCanceled(err) || ctx.inflight.get(id) !== pending) {
        return;
      }
      tab.structure = null;
      if (!keepDraft) {
        tab.structureDraft = null;
      }
      tab.structureError = errorMessage(err);
    } finally {
      if (ctx.inflight.get(id) === pending) {
        ctx.inflight.delete(id);
        tab.running = false;
      }
    }
  }

  function ensureStructureDraft(tab: TableTab): DraftColumn[] {
    if (!tab.structureDraft) {
      tab.structureDraft = columnsToDraft(tab.structure?.columns ?? []);
    }
    return tab.structureDraft;
  }

  function updateStructureColumn(id: string, key: string, patch: Partial<DraftColumn>) {
    const tab = ctx.tableTab(id);
    if (!tab) {
      return;
    }
    const draft = ensureStructureDraft(tab);
    const index = draft.findIndex((col) => col.key === key);
    if (index < 0) {
      return;
    }
    tab.structureDraft = draft.map((col, i) => (i === index ? { ...col, ...patch, key: col.key } : col));
  }

  function addStructureColumn(id: string, driverId = "") {
    const tab = ctx.tableTab(id);
    if (!tab) {
      return;
    }
    const draft = ensureStructureDraft(tab);
    tab.structureDraft = [...draft, emptyDraftColumn(nextColumnName(draft), driverId)];
  }

  function toggleDropStructureColumn(id: string, key: string) {
    const tab = ctx.tableTab(id);
    if (!tab) {
      return;
    }
    const draft = ensureStructureDraft(tab);
    const index = draft.findIndex((col) => col.key === key);
    if (index < 0) {
      return;
    }
    const col = draft[index]!;
    if (col.originalName === null) {
      tab.structureDraft = draft.filter((item) => item.key !== key);
      return;
    }
    tab.structureDraft = draft.map((item) =>
      item.key === key ? { ...item, dropped: !item.dropped } : item,
    );
  }

  function moveStructureColumns(id: string, next: DraftColumn[]) {
    const tab = ctx.tableTab(id);
    if (!tab) {
      return;
    }
    tab.structureDraft = next;
  }

  function cancelStructure(id: string) {
    const tab = ctx.tableTab(id);
    if (!tab) {
      return;
    }
    tab.structureDraft = columnsToDraft(tab.structure?.columns ?? []);
  }

  async function applyStructure(id: string) {
    const tab = ctx.tableTab(id);
    if (!tab) {
      return;
    }
    const ops = structureOps(tab.structure?.columns, tab.structureDraft, tab.structure?.canReorder ?? false);
    if (ops.length === 0) {
      return;
    }
    await QueryService.AlterTable(tab.profileId, {
      database: tab.database,
      schema: tab.schema,
      name: tab.name,
      ops,
    });
    clearBrowse(tab);
    const dataTab = ctx.tabs.value.find(
      (t): t is TableTab =>
        t.kind === "table" &&
        t.view === "data" &&
        t.id !== tab.id &&
        t.profileId === tab.profileId &&
        t.database === tab.database &&
        t.schema === tab.schema &&
        t.name === tab.name,
    );
    if (dataTab) {
      clearBrowse(dataTab);
    }
    await loadStructure(id, { force: true });
  }

  return {
    loadStructure,
    updateStructureColumn,
    addStructureColumn,
    toggleDropStructureColumn,
    moveStructureColumns,
    cancelStructure,
    applyStructure,
  };
}

export type TableStructureApi = ReturnType<typeof createTableStructure>;
