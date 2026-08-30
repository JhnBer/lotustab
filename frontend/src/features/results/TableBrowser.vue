<script setup lang="ts">
import { ChevronDown, ChevronRight, Filter, Plus } from "@lucide/vue";
import { toast } from "vue-sonner";
import { Button } from "@/components/ui/button";
import { Toggle } from "@/components/ui/toggle";
import ResultsTable from "@/features/results/ResultsTable.vue";
import TableFilters from "@/features/results/TableFilters.vue";
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
    <div class="flex items-center gap-2 border-b px-3 py-2 text-xs">
      <div class="text-muted-foreground min-w-0 flex-1 truncate">
        <span class="text-foreground font-medium">{{ tab.title }}</span>
        <span v-if="tab.database" class="ml-1">· {{ tab.database }}</span>
        <span v-if="tab.result && !tab.running"> · {{ tab.result.durationMs }} ms</span>
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
          :disabled="!tab.filtersEnabled || tab.running"
          @click="queries.applyFilters(tab.id)"
        >
          Apply
        </Button>
      </div>
    </div>
    <TableFilters
      v-show="tab.filtersOpen"
      :filters="tab.filters ?? []"
      :columns="tab.result?.columns ?? []"
      :disabled="!tab.filtersEnabled"
      @update:filters="queries.setFilters(tab.id, $event)"
      @apply="queries.applyFilters(tab.id)"
    />
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
