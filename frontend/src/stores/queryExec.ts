import { errorMessage, isCanceled } from "@/lib/errors";
import { QueryService } from "@/lib/wails";
import type { QueryTab } from "@/features/editor/tabs";
import { useConnectionsStore } from "@/stores/connections";
import { useExplorerStore } from "@/stores/explorer";
import { useSettingsStore } from "@/stores/settings";
import type { QueryStoreCtx } from "@/stores/queryContext";

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

export function createQueryExec(ctx: QueryStoreCtx) {
  async function execute(id: string) {
    const tab = ctx.queryTab(id);
    if (!tab) {
      return;
    }
    if (!useConnectionsStore().isConnected(tab.profileId)) {
      return;
    }
    if (!tab.sql.trim()) {
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
    const pending = QueryService.Execute(
      tab.profileId,
      tab.sql,
      queryExecuteDatabase(tab),
      tab.pageSize,
      (tab.page - 1) * tab.pageSize,
      useSettingsStore().queryTimeoutSeconds,
    );
    ctx.inflight.set(id, pending);
    try {
      const result = await pending;
      if (ctx.inflight.get(id) !== pending) {
        return;
      }
      tab.result = result;
    } catch (err) {
      if (isCanceled(err) || ctx.inflight.get(id) !== pending) {
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

  async function run(id: string) {
    const tab = ctx.queryTab(id);
    if (!tab) {
      return;
    }
    tab.page = 1;
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

  return { execute, run, setSql, setDatabase };
}

export type QueryExecApi = ReturnType<typeof createQueryExec>;
