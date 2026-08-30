<script setup lang="ts">
import ResultsTable from "@/features/results/ResultsTable.vue";
import TablePager from "@/features/results/TablePager.vue";
import { type TableTab, useQueryStore } from "@/stores/query";

defineProps<{
  tab: TableTab;
}>();

const queries = useQueryStore();
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
      empty-hint="Loading table…"
      @sort="queries.cycleSort(tab.id, $event)"
    />
    <TablePager
      :page="tab.page"
      :page-size="tab.pageSize"
      :row-count="tab.result?.rows?.length ?? 0"
      :running="tab.running"
      :total="tab.total"
      @page="queries.setPage(tab.id, $event)"
      @page-size="queries.setPageSize(tab.id, $event)"
    />
  </div>
</template>
