<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { ChevronDown, ChevronRight, Filter, Plus, RotateCw, X } from "@lucide/vue";
import { toast } from "vue-sonner";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Toggle } from "@/components/ui/toggle";
import ResultsTable from "@/features/results/ResultsTable.vue";
import TableFilters from "@/features/results/TableFilters.vue";
import TablePager from "@/features/results/TablePager.vue";
import TableStructure from "@/features/results/TableStructure.vue";
import { isStructureDirty, structureOps } from "@/features/results/structure";
import { errorMessage } from "@/lib/errors";
import { useConnectionsStore } from "@/stores/connections";
import { type TableTab, useQueryStore } from "@/stores/query";

const props = defineProps<{
  tab: TableTab;
  active: boolean;
}>();

const connections = useConnectionsStore();
const queries = useQueryStore();
const connected = computed(() => connections.isConnected(props.tab.profileId));
const loading = computed(() => props.tab.running || connections.isConnecting(props.tab.profileId));
const structureView = computed(() => props.tab.view === "structure");
const structureDirty = computed(() =>
  isStructureDirty(props.tab.structure?.columns, props.tab.structureDraft, Boolean(props.tab.structure?.canReorder)),
);
const columnCount = computed(
  () => props.tab.structureDraft?.filter((col) => !col.dropped).length ?? props.tab.structure?.columns?.length ?? 0,
);
const confirmDrop = ref(false);
const dropCount = computed(
  () => structureOps(props.tab.structure?.columns, props.tab.structureDraft, Boolean(props.tab.structure?.canReorder)).filter((op) => op.kind === "drop").length,
);

watch(
  () => Boolean(props.active) && connected.value,
  (shouldLoad) => {
    if (!shouldLoad) {
      return;
    }
    if (structureView.value) {
      if (!props.tab.structure && !props.tab.structureDraft && !props.tab.running) {
        void queries.loadStructure(props.tab.id);
      }
      return;
    }
    if (!props.tab.result && !props.tab.running && !props.tab.manuallyCancelled) {
      void queries.loadTable(props.tab.id);
    }
  },
  { immediate: true },
);

function onEdit(row: number, column: string, value: unknown) {
  queries.setCellDraft(props.tab.id, row, column, value);
}

function openCompanion() {
  queries.openTable({
    profileId: props.tab.profileId,
    database: props.tab.database,
    schema: props.tab.schema,
    name: props.tab.name,
    view: structureView.value ? "data" : "structure",
  });
}

async function onApply() {
  if (structureView.value) {
    if (dropCount.value > 0) {
      confirmDrop.value = true;
      return;
    }
    await applyStructure();
    return;
  }
  try {
    await queries.applyEdits(props.tab.id);
  } catch (err) {
    toast.error(errorMessage(err));
  }
}

function onCancel() {
  if (structureView.value) {
    queries.cancelStructure(props.tab.id);
    return;
  }
  queries.clearEdits(props.tab.id);
}

async function applyStructure() {
  confirmDrop.value = false;
  try {
    await queries.applyStructure(props.tab.id);
  } catch (err) {
    toast.error(errorMessage(err));
  }
}
</script>

<template>
  <div class="flex min-h-0 flex-1 flex-col bg-background">
    <!--  control header   -->
    <div v-if="!structureView" class="relative z-20 flex items-center gap-2 border-b bg-background px-3 py-2 text-xs">
      <Button
        size="icon-xs"
        variant="outline"
        :disabled="!tab.running && loading"
        :aria-label="tab.running ? 'Cancel' : 'Refresh table'"
        @click="tab.running ? queries.cancel(tab.id) : queries.loadTable(tab.id)"
      >
        <X v-if="tab.running" class="size-3.5" />
        <RotateCw v-else class="size-3.5" />
      </Button>
      <div class="text-muted-foreground min-w-0 flex-1 truncate">
        <span class="text-foreground font-medium">{{ tab.title }}</span>
        <span v-if="tab.database" class="ml-1">· {{ tab.database }}</span>
        <Spinner v-if="tab.running" class="size-3 ml-1 inline-block" />
        <span v-else-if="tab.result"> · {{ tab.result.durationMs }} ms</span>
      </div>
      <div class="flex shrink-0 items-center gap-1">
        <Button
          size="icon-xs"
          variant="outline"
          class="aria-expanded:bg-muted"
          :aria-label="tab.filtersOpen ? 'Hide filters' : 'Show filters'"
          :aria-expanded="Boolean(tab.filtersOpen)"
          @click="queries.toggleFiltersOpen(tab.id)"
        >
          <ChevronDown v-if="tab.filtersOpen" class="size-3.5" />
          <ChevronRight v-else class="size-3.5" />
        </Button>
        <Toggle
          size="icon-xs"
          variant="outline"
          :model-value="Boolean(tab.filtersEnabled)"
          aria-label="Filters"
          @update:model-value="queries.setFiltersEnabled(tab.id, $event)"
        >
          <Filter class="size-3.5" />
        </Toggle>
        <Button
          size="icon-xs"
          variant="outline"
          :disabled="!tab.filtersEnabled"
          aria-label="Add filter"
          @click="queries.addFilter(tab.id)"
        >
          <Plus class="size-3.5" />
        </Button>
        <Button
          size="xs"
          :disabled="!tab.filtersEnabled || loading"
          @click="queries.applyFilters(tab.id)"
        >
          Apply
        </Button>
      </div>
    </div>
    <template v-if="structureView">
      <TableStructure :tab="tab" :running="loading" />
    </template>
    <template v-else>
      <TableFilters
        v-show="tab.filtersOpen"
        :filters="tab.filters ?? []"
        :columns="tab.result?.columns ?? []"
        :disabled="!tab.filtersEnabled"
        @update:filters="queries.setFilters(tab.id, $event)"
        @apply="queries.applyFilters(tab.id)"
      />
      <div
        v-if="!loading && !tab.result && !tab.error && tab.manuallyCancelled"
        class="text-muted-foreground flex flex-1 flex-col items-center justify-center gap-3 p-4"
      >
        <span class="text-sm">Load cancelled.</span>
        <Button size="sm" variant="outline" @click="queries.loadTable(tab.id)">
          <RotateCw class="mr-1.5 size-3.5" />
          Click to reload
        </Button>
      </div>
      <ResultsTable
        v-else
        class="relative z-0 -mt-px min-h-0 flex-1"
        :result="tab.result"
        :error="tab.error"
        :running="loading"
        sortable
        :sort-column="tab.orderBy"
        :sort-dir="tab.orderDir"
        :show-status="false"
        :row-offset="(tab.page - 1) * tab.pageSize"
        :editable="(tab.primaryKey?.length ?? 0) > 0"
        :edits="tab.edits ?? []"
        :primary-key="tab.primaryKey ?? []"
        :width-scope="{
          database: tab.database,
          schema: tab.schema,
          table: tab.name,
        }"
        :empty-hint="connected ? 'Loading table…' : 'Connect to view this table.'"
        @sort="queries.cycleSort(tab.id, $event)"
        @edit="onEdit"
        @edit-insert="(insertId, column, value) => queries.setInsertCell(tab.id, insertId, column, value)"
        @delete-row="queries.setRowDeleted(tab.id, $event)"
        @clone-row="(row, after) => queries.cloneRow(tab.id, row, after)"
      />
    </template>
    <TablePager
      :page="tab.page"
      :page-size="tab.pageSize"
      :row-count="tab.result?.rows?.length ?? 0"
      :running="loading"
      :total="tab.total"
      :dirty="structureView ? structureDirty : (tab.edits?.length ?? 0) > 0"
      :switch-label="structureView ? 'Table' : 'Structure'"
      :compact="structureView"
      :label="structureView ? `${columnCount} columns` : undefined"
      @page="queries.setPage(tab.id, $event)"
      @page-size="queries.setPageSize(tab.id, $event)"
      @apply="onApply"
      @cancel="onCancel"
      @switch="openCompanion"
    />

    <Dialog :open="confirmDrop" @update:open="confirmDrop = $event">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Drop {{ dropCount }} {{ dropCount === 1 ? "column" : "columns" }}?</DialogTitle>
          <DialogDescription>
            Dropping columns cannot be undone and will permanently delete their data.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" @click="confirmDrop = false">Cancel</Button>
          <Button variant="destructive" @click="applyStructure">Apply</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
