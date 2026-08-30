<script setup lang="ts">
import { toast } from "vue-sonner";
import ResultsTable from "@/features/results/ResultsTable.vue";
import TablePager from "@/features/results/TablePager.vue";
import { errorMessage } from "@/lib/errors";
import { type TableTab, useQueryStore } from "@/stores/query";

const props = defineProps<{
  tab: TableTab;
}>();

const queries = useQueryStore();

function onEdit(row: number, column: string, value: unknown) {
  queries.setCellDraft(props.tab.id, row, column, value);
}

async function onApply() {
  try {
    await queries.applyEdits(props.tab.id);
  } catch (err) {
    toast.error(errorMessage(err));
  }
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col">
    <div class="flex items-center border-b px-3 py-2 text-xs">
      <div class="text-muted-foreground min-w-0 truncate">
        <span class="text-foreground font-medium">{{ tab.title }}</span>
        <span v-if="tab.database" class="ml-1">· {{ tab.database }}</span>
        <span v-if="tab.result && !tab.running"> · {{ tab.result.durationMs }} ms</span>
      </div>
    </div>
    <ResultsTable
      class="min-h-0 flex-1"
      :result="tab.result"
      :error="tab.error"
      :running="tab.running"
      sortable
      :sort-column="tab.orderBy"
      :sort-dir="tab.orderDir"
      :show-status="false"
      :row-offset="(tab.page - 1) * tab.pageSize"
      :editable="(tab.primaryKey?.length ?? 0) > 0"
      :edits="tab.edits ?? []"
      :primary-key="tab.primaryKey ?? []"
      empty-hint="Loading table…"
      @sort="queries.cycleSort(tab.id, $event)"
      @edit="onEdit"
    />
    <TablePager
      :page="tab.page"
      :page-size="tab.pageSize"
      :row-count="tab.result?.rows?.length ?? 0"
      :running="tab.running"
      :total="tab.total"
      :dirty="(tab.edits?.length ?? 0) > 0"
      @page="queries.setPage(tab.id, $event)"
      @page-size="queries.setPageSize(tab.id, $event)"
      @apply="onApply"
      @cancel="queries.clearEdits(tab.id)"
    />
  </div>
</template>
