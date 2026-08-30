<script setup lang="ts">
import { X } from "@lucide/vue";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { Column } from "@/lib/wails";
import {
  FILTER_OPS,
  opNeedsValue,
  valuePlaceholder,
  type TableFilter,
} from "@/features/results/filters";

const props = defineProps<{
  filters: TableFilter[];
  columns: Column[];
  disabled?: boolean;
}>();

const emit = defineEmits<{
  "update:filters": [filters: TableFilter[]];
  apply: [];
}>();

function patch(index: number, part: Partial<TableFilter>) {
  emit(
    "update:filters",
    props.filters.map((filter, i) => (i === index ? { ...filter, ...part } : filter)),
  );
}

function remove(index: number) {
  emit(
    "update:filters",
    props.filters.filter((_, i) => i !== index),
  );
}
</script>

<template>
  <div class="flex flex-col gap-1.5 border-b px-3 py-2">
    <div
      v-for="(filter, index) in filters"
      :key="index"
      class="flex min-w-0 items-center gap-1.5"
    >
      <div class="w-20 shrink-0">
        <Select
          v-if="index > 0"
          :model-value="filter.join"
          :disabled="disabled"
          @update:model-value="patch(index, { join: String($event) })"
        >
          <SelectTrigger size="sm" class="h-7 w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="and">AND</SelectItem>
            <SelectItem value="or">OR</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <Select
        :model-value="filter.column || undefined"
        :disabled="disabled"
        @update:model-value="patch(index, { column: String($event ?? '') })"
      >
        <SelectTrigger size="sm" class="h-7 w-36">
          <SelectValue placeholder="Column" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem v-for="column in columns" :key="column.name" :value="column.name">
            {{ column.name }}
          </SelectItem>
        </SelectContent>
      </Select>
      <Select
        :model-value="filter.op"
        :disabled="disabled"
        @update:model-value="patch(index, { op: String($event) })"
      >
        <SelectTrigger size="sm" class="h-7 w-32">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem v-for="op in FILTER_OPS" :key="op.id" :value="op.id">
            {{ op.label }}
          </SelectItem>
        </SelectContent>
      </Select>
      <Input
        class="h-7 min-w-0 flex-1 text-xs"
        :model-value="filter.value"
        :disabled="disabled || !opNeedsValue(filter.op)"
        :placeholder="valuePlaceholder(filter.op)"
        @update:model-value="patch(index, { value: String($event) })"
        @keydown.enter.prevent="emit('apply')"
      />
      <Button
        size="icon-xs"
        variant="ghost"
        :disabled="disabled"
        aria-label="Remove filter"
        @click="remove(index)"
      >
        <X />
      </Button>
    </div>
  </div>
</template>
