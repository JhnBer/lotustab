import { Events } from "@wailsio/runtime";
import { errorMessage, isCanceled } from "@/lib/errors";
import { QueryService } from "@/lib/wails";
import type { QueryBatch, QueryResult, StatementResult } from "@/lib/wails";
import type { QueryTab } from "@/features/editor/tabs";
import { useConnectionsStore } from "@/stores/connections";
import { useExplorerStore } from "@/stores/explorer";
import { useSettingsStore } from "@/stores/settings";
import type { QueryStoreCtx } from "@/stores/queryContext";

type QueryProgress = {
  runId?: string;
  index?: number;
  total?: number;
  sql?: string;
  result?: QueryResult | null;
  error?: string;
};

let listening = false;

function queryExecuteDatabase(tab: QueryTab): string {
  const settings = useSettingsStore();
  if (!settings.showAllDatabases) {
    return useExplorerStore().activeDatabase || "";
  }
  if (!settings.enableDatabaseChoice) {
    return "";
  }
  return tab.database ?? "";
}

function showSelected(tab: QueryTab) {
  const item = tab.results[tab.selectedResult];
  tab.result = item?.result ?? null;
  tab.error = item?.error ?? "";
}

function applyStatement(tab: QueryTab, item: StatementResult) {
  const next = tab.results.slice();
  while (next.length <= item.index) {
    next.push({ index: next.length, sql: "", result: null, error: "" });
  }
  next[item.index] = item;
  tab.results = next;
  if (tab.followResults) {
    tab.selectedResult = item.index;
  }
  showSelected(tab);
}

function applyBatch(tab: QueryTab, batch: QueryBatch | null) {
  const results = batch?.results ?? [];
  tab.results = results;
  if (!results.length) {
    tab.result = null;
    return;
  }
  if (tab.followResults) {
    tab.selectedResult = results.length - 1;
  } else if (tab.selectedResult >= results.length) {
    tab.selectedResult = results.length - 1;
  }
  showSelected(tab);
}

function bindProgress(ctx: QueryStoreCtx) {
  if (listening) {
    return;
  }
  listening = true;
  Events.On("query:progress", (ev) => {
    const data = ev.data as QueryProgress | QueryProgress[] | null;
    const next = Array.isArray(data) ? data[0] : data;
    if (!next || typeof next !== "object" || !next.runId) {
      return;
    }
    const tab = ctx.tabs.value.find(
      (t): t is QueryTab => t.kind === "query" && t.runId === next.runId,
    );
    if (!tab) {
      return;
    }
    applyStatement(tab, {
      index: Number(next.index) || 0,
      sql: typeof next.sql === "string" ? next.sql : "",
      result: next.result ?? null,
      error: typeof next.error === "string" ? next.error : "",
    });
  });
}

export function createQueryExec(ctx: QueryStoreCtx) {
  bindProgress(ctx);

  async function execute(id: string) {
    const tab = ctx.queryTab(id);
    if (!tab) {
      return;
    }
    if (!useConnectionsStore().isConnected(tab.profileId)) {
      return;
    }
    const sql = tab.executedSql || tab.sql;
    if (!sql.trim()) {
      tab.error = "Query is empty";
      return;
    }
    if (!tab.page || tab.page < 1) {
      tab.page = 1;
    }
    if (!tab.pageSize) {
      tab.pageSize = ctx.storedPageSize();
    }
    ctx.cancel(id);
    tab.running = true;
    tab.error = "";
    const runId = crypto.randomUUID();
    tab.runId = runId;
    const pending = QueryService.Execute(
      tab.profileId,
      sql,
      queryExecuteDatabase(tab),
      tab.pageSize,
      (tab.page - 1) * tab.pageSize,
      useSettingsStore().queryTimeoutSeconds,
      runId,
    );
    ctx.inflight.set(id, pending);
    try {
      const batch = await pending;
      if (ctx.inflight.get(id) !== pending) {
        return;
      }
      applyBatch(tab, batch);
    } catch (err) {
      if (isCanceled(err) || ctx.inflight.get(id) !== pending) {
        showSelected(tab);
        return;
      }
      if (tab.results.length) {
        showSelected(tab);
        if (!tab.error) {
          tab.error = errorMessage(err);
        }
        return;
      }
      tab.result = null;
      tab.error = errorMessage(err);
    } finally {
      if (ctx.inflight.get(id) === pending) {
        ctx.inflight.delete(id);
        tab.running = false;
      }
    }
  }

  async function run(id: string, sql?: string) {
    const tab = ctx.queryTab(id);
    if (!tab) {
      return;
    }
    tab.page = 1;
    tab.executedSql = sql ?? tab.sql;
    tab.followResults = true;
    tab.results = [];
    tab.selectedResult = 0;
    tab.result = null;
    tab.error = "";
    await execute(id);
  }

  function setSql(id: string, sql: string) {
    const tab = ctx.queryTab(id);
    if (tab) {
      tab.sql = sql;
    }
  }

  function setDatabase(id: string, database: string) {
    const tab = ctx.queryTab(id);
    if (tab) {
      tab.database = database;
    }
  }

  function setSelectedResult(id: string, index: number) {
    const tab = ctx.queryTab(id);
    if (!tab || index < 0 || index >= tab.results.length) {
      return;
    }
    tab.followResults = false;
    tab.selectedResult = index;
    showSelected(tab);
  }

  return { execute, run, setSql, setDatabase, setSelectedResult };
}

export type QueryExecApi = ReturnType<typeof createQueryExec>;
