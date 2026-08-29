<script setup lang="ts">
import { computed } from "vue";
import { ChevronLeft, ChevronRight } from "@lucide/vue";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import ResultsTable from "@/features/results/ResultsTable.vue";
import { PAGE_SIZES, type TableTab, useQueryStore } from "@/stores/query";

const props = defineProps<{
  tab: TableTab;
}>();

const queries = useQueryStore();

const from = computed(() => {
  if (props.tab.total === 0) {
    return 0;
  }
  return (props.tab.page - 1) * props.tab.pageSize + 1;
});

const to = computed(() => Math.min(props.tab.page * props.tab.pageSize, props.tab.total));

const pageCount = computed(() => Math.max(1, Math.ceil(props.tab.total / props.tab.pageSize) || 1));

function onPageSize(value: unknown) {
  const size = Number(value);
  if (!Number.isFinite(size)) {
    return;
  }
  queries.setPageSize(props.tab.id, size);
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col">
    <div class="flex items-center border-b px-3 py-2 text-xs">
      <div class="text-muted-foreground min-w-0 truncate">
        <span class="text-foreground font-medium">{{ tab.title }}</span>
        <span v-if="tab.database" class="ml-1">· {{ tab.database }}</span>
        <span v-if="!tab.running && !tab.error">
          · {{ from }}–{{ to }} of {{ tab.total }}
        </span>
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
      empty-hint="Loading table…"
      @sort="queries.cycleSort(tab.id, $event)"
    />
    <div class="flex shrink-0 items-center justify-end gap-2 border-t px-3 py-2">
      <Select :model-value="String(tab.pageSize)" @update:model-value="onPageSize">
        <SelectTrigger size="sm" class="h-7 w-36">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem v-for="size in PAGE_SIZES" :key="size" :value="String(size)">
            {{ size }} / page
          </SelectItem>
        </SelectContent>
      </Select>
      <div class="flex items-center gap-1">
        <Button
          size="icon-xs"
          variant="outline"
          :disabled="tab.running || tab.page <= 1"
          @click="queries.setPage(tab.id, tab.page - 1)"
        >
          <ChevronLeft class="size-3.5" />
        </Button>
        <span class="text-muted-foreground min-w-18 text-center text-xs">
          {{ tab.page }} / {{ pageCount }}
        </span>
        <Button
          size="icon-xs"
          variant="outline"
          :disabled="tab.running || tab.page >= pageCount"
          @click="queries.setPage(tab.id, tab.page + 1)"
        >
          <ChevronRight class="size-3.5" />
        </Button>
      </div>
    </div>
  </div>
</template>
