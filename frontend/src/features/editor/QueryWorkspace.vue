<script setup lang="ts">
import { computed, watch } from "vue";
import { Play, Plug } from "@lucide/vue";
import { toast } from "vue-sonner";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import DatabaseCombobox from "@/components/DatabaseCombobox.vue";
import { useProfileActions } from "@/composables/useProfileActions";
import { errorMessage, isCanceled } from "@/lib/errors";
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from "@/components/ui/resizable";
import SqlEditor from "@/features/editor/SqlEditor.vue";
import WorkspaceTabBar from "@/features/editor/WorkspaceTabBar.vue";
import ResultsTable from "@/features/results/ResultsTable.vue";
import TableBrowser from "@/features/results/TableBrowser.vue";
import QueryResultActions from "@/features/results/QueryResultActions.vue";
import TablePager from "@/features/results/TablePager.vue";
import { useConnectionsStore } from "@/stores/connections";
import { useExplorerStore } from "@/stores/explorer";
import { isQueryTab, isTableTab, useQueryStore, type QueryTab, type TableTab } from "@/stores/query";
import { useSettingsStore } from "@/stores/settings";

const NONE = "__none__";

const connections = useConnectionsStore();
const queries = useQueryStore();
const explorer = useExplorerStore();
const settings = useSettingsStore();
const { connect } = useProfileActions();

type SqlEditorExpose = { runSql: () => string };
const editors = new Map<string, SqlEditorExpose>();

function bindEditor(id: string, el: unknown) {
  if (el && typeof el === "object" && "runSql" in el && typeof (el as SqlEditorExpose).runSql === "function") {
    editors.set(id, el as SqlEditorExpose);
  } else {
    editors.delete(id);
  }
}

function queryPager(tab: QueryTab) {
  const result = tab.result;
  const results = tab.results ?? [];
  const multi = results.length > 1;
  const statements = results.map((stmt) => ({ index: stmt.index, sql: stmt.sql }));
  if (!result) {
    return {
      compact: multi,
      label: undefined as string | undefined,
      durationMs: undefined as number | undefined,
      rowCount: 0,
      truncated: undefined as boolean | undefined,
      statements,
    };
  }
  if (!result.columns?.length) {
    if (result.affectedKnown) {
      return {
        compact: true,
        label: `${result.rowsAffected} affected · ${result.durationMs} ms`,
        durationMs: undefined as number | undefined,
        rowCount: 0,
        truncated: result.truncated,
        statements,
      };
    }
    return {
      compact: true,
      label: `${result.durationMs} ms`,
      durationMs: undefined as number | undefined,
      rowCount: 0,
      truncated: result.truncated,
      statements,
    };
  }
  if (!multi) {
    return {
      compact: false,
      label: undefined as string | undefined,
      durationMs: result.durationMs,
      rowCount: result.rows?.length ?? 0,
      truncated: result.truncated,
      statements,
    };
  }
  const n = result.rows?.length ?? 0;
  const trunc = result.truncated ? " (truncated)" : "";
  return {
    compact: true,
    label: `${n} row${n === 1 ? "" : "s"}${trunc} · ${result.durationMs} ms`,
    durationMs: undefined as number | undefined,
    rowCount: n,
    truncated: result.truncated,
    statements,
  };
}

const connecting = computed(() => {
  const id = connections.selectedId;
  return Boolean(id && connections.isConnecting(id));
});

const selectedOffline = computed(() => {
  const id = connections.selectedId;
  return Boolean(id && !connections.isConnected(id) && !connections.isConnecting(id));
});

const profileMountedIds = computed(() => new Set(queries.mountedTabIds(connections.selectedId ?? "")));
const profileTableTabs = computed(() =>
  queries.tabs.filter(
    (t): t is TableTab =>
      isTableTab(t) && t.profileId === connections.selectedId && profileMountedIds.value.has(t.id),
  ),
);
const profileQueryTabs = computed(() =>
  queries.tabs.filter(
    (t): t is QueryTab =>
      isQueryTab(t) && t.profileId === connections.selectedId && profileMountedIds.value.has(t.id),
  ),
);

watch(
  () => connections.selectedId,
  (id) => queries.activateProfile(id),
  { immediate: true },
);

async function run(sql?: string) {
  const tab = queries.activeTab;
  if (!tab || tab.kind !== "query") {
    return;
  }
  try {
    if (!connections.isConnected(tab.profileId)) {
      await connections.connect(tab.profileId);
      if (!connections.isConnected(tab.profileId)) {
        return;
      }
    }
    await queries.run(tab.id, typeof sql === "string" ? sql : (editors.get(tab.id)?.runSql() ?? tab.sql));
  } catch (err) {
    if (isCanceled(err)) {
      return;
    }
    toast.error(errorMessage(err));
  }
}

async function connectSelected() {
  const profile = connections.selected;
  if (!profile) {
    return;
  }
  await connect(profile);
}

function onDatabase(tabId: string, value: string) {
  if (!settings.enableDatabaseChoice) {
    return;
  }
  queries.setDatabase(tabId, value === NONE ? "" : value);
}
</script>

<template>
  <section class="flex h-full min-h-0 flex-1 flex-col -m-4 overflow-hidden">
    <WorkspaceTabBar />

    <div
      v-if="connecting"
      class="text-muted-foreground flex flex-1 flex-col items-center justify-center gap-2 p-8 text-sm"
    >
      <Spinner />
      <span>Connecting…</span>
    </div>

    <div
      v-else-if="!connections.selectedId"
      class="text-muted-foreground flex flex-1 flex-col items-center justify-center gap-2 p-8 text-sm"
    >
      <span>Select a connection to start querying.</span>
      <RouterLink to="/connections" class="text-foreground hover:underline">
        Open connections
      </RouterLink>
    </div>

    <div
      v-else-if="selectedOffline && !queries.activeTab"
      class="text-muted-foreground flex flex-1 flex-col items-center justify-center gap-3 p-8 text-sm"
    >
      <span>Connect this database to start querying.</span>
      <Button size="sm" @click="connectSelected">
        <Plug class="size-3.5" />
        Connect
      </Button>
    </div>

    <template v-else>
      <div class="relative min-h-0 flex-1">
        <TableBrowser
          v-for="tab in profileTableTabs"
          :key="tab.id"
          class="absolute inset-0"
          :class="tab.id === queries.activeTabId ? 'z-10' : 'invisible pointer-events-none'"
          :tab="tab"
          :active="tab.id === queries.activeTabId"
        />
        <ResizablePanelGroup
          v-for="tab in profileQueryTabs"
          :key="tab.id"
          direction="vertical"
          class="absolute inset-0"
          :class="tab.id === queries.activeTabId ? 'z-10 bg-background' : 'invisible pointer-events-none'"
        >
      <ResizablePanel :default-size="45" :min-size="20">
        <div class="flex h-full min-h-0 flex-col">
          <div class="flex items-center gap-2 border-b px-3 py-2 text-xs" style="--wails-draggable: no-drag">
            <template v-if="settings.showAllDatabases">
              <DatabaseCombobox
                class="h-7 w-44"
                :model-value="tab.database || NONE"
                :options="explorer.databases"
                :none-value="NONE"
                none-label="None"
                placeholder="Database"
                :disabled="!settings.enableDatabaseChoice"
                @update:model-value="onDatabase(tab.id, $event)"
              />
              <Label
                class="text-muted-foreground flex shrink-0 items-center gap-2 font-normal"
              >
                <Switch
                  :id="`query-enable-db-choice-${tab.id}`"
                  size="sm"
                  :model-value="settings.enableDatabaseChoice"
                  @update:model-value="settings.setEnableDatabaseChoice(Boolean($event))"
                />
                Enable database choice
              </Label>
              <span class="text-muted-foreground min-w-0 flex-1 truncate">
                <template v-if="settings.enableDatabaseChoice">
                  You can choose a database here or specify it in the query.
                </template>
                <template v-else>
                  Specify the database in the query.
                </template>
              </span>
            </template>
            <span v-else class="min-w-0 flex-1" />
            <Button
              v-if="tab.running"
              size="sm"
              variant="outline"
              class="shrink-0"
              @click="queries.cancel(tab.id)"
            >
              Cancel
            </Button>
            <Button size="sm" class="shrink-0" :disabled="tab.running" @click="run()">
              <Spinner v-if="tab.running" class="size-3.5" />
              <Play v-else class="size-3.5" />
              Run
            </Button>
          </div>
          <SqlEditor
            class="min-h-0 flex-1"
            :ref="(el) => bindEditor(tab.id, el)"
            :model-value="tab.sql"
            @update:model-value="queries.setSql(tab.id, $event)"
            @run="run"
          />
        </div>
      </ResizablePanel>
      <ResizableHandle />
      <ResizablePanel :default-size="55" :min-size="20">
        <div class="flex h-full min-h-0 flex-col">
          <ResultsTable
            class="min-h-0 flex-1"
            :result="tab.result"
            :error="tab.error"
            :running="tab.running"
            :show-status="false"
            :row-offset="(tab.results?.length ?? 0) > 1 ? 0 : (tab.page - 1) * tab.pageSize"
          />
          <TablePager
            :page="tab.page"
            :page-size="tab.pageSize"
            :row-count="queryPager(tab).rowCount"
            :running="tab.running"
            :truncated="queryPager(tab).truncated"
            :duration-ms="queryPager(tab).durationMs"
            :compact="queryPager(tab).compact"
            :label="queryPager(tab).label"
            :statements="queryPager(tab).statements"
            :selected-statement="tab.selectedResult"
            @page="queries.setPage(tab.id, $event)"
            @page-size="queries.setPageSize(tab.id, $event)"
            @statement="queries.setSelectedResult(tab.id, $event)"
          >
            <template #actions>
              <QueryResultActions :tab="tab" />
            </template>
          </TablePager>
        </div>
      </ResizablePanel>
    </ResizablePanelGroup>
        <div
          v-if="queries.coldMountId && queries.coldMountId === queries.activeTabId"
          class="absolute inset-0 z-10 flex items-center justify-center bg-background/60"
        >
          <Spinner />
        </div>
      </div>
    </template>
  </section>
</template>
