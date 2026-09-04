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
import { PAGE_SIZES } from "@/stores/settings";

const props = defineProps<{
  page: number;
  pageSize: number;
  rowCount: number;
  running?: boolean;
  total?: number;
  truncated?: boolean;
  durationMs?: number;
  dirty?: boolean;
  switchLabel?: string;
  compact?: boolean;
  label?: string;
}>();

const emit = defineEmits<{
  page: [page: number];
  pageSize: [size: number];
  apply: [];
  cancel: [];
  switch: [];
}>();

const pageCount = computed(() => {
  if (props.total == null) {
    return null;
  }
  return Math.max(1, Math.ceil(props.total / props.pageSize) || 1);
});

const from = computed(() => {
  if (props.rowCount === 0 && (props.total == null || props.total === 0)) {
    return 0;
  }
  return (props.page - 1) * props.pageSize + 1;
});

const to = computed(() => {
  if (from.value === 0) {
    return 0;
  }
  if (props.total != null) {
    return Math.min(props.page * props.pageSize, props.total);
  }
  return from.value + props.rowCount - 1;
});

const hasPrev = computed(() => props.page > 1);
const hasNext = computed(() => {
  if (pageCount.value != null) {
    return props.page < pageCount.value;
  }
  return Boolean(props.truncated);
});

const rangeLabel = computed(() => {
  if (props.total != null) {
    return `${from.value}–${to.value} of ${props.total}`;
  }
  if (props.truncated) {
    return `${from.value}–${to.value} (truncated)`;
  }
  return `${from.value}–${to.value}`;
});

function onPageSize(value: unknown) {
  const size = Number(value);
  if (!Number.isFinite(size)) {
    return;
  }
  emit("pageSize", size);
}
</script>

<template>
  <div
    class="flex h-11 shrink-0 items-center justify-end gap-2 border-t px-3"
  >
    <Button
      v-if="switchLabel"
      size="xs"
      variant="outline"
      class="shrink-0"
      @click="emit('switch')"
    >
      {{ switchLabel }}
    </Button>
    <span class="text-muted-foreground mr-auto min-w-0 truncate text-xs">
      {{ label ?? rangeLabel }}
      <template v-if="durationMs != null && !label"> · {{ durationMs }} ms</template>
    </span>
    <div v-if="dirty" class="flex items-center gap-1">
      <Button size="xs" :disabled="running" @click="emit('apply')">Apply</Button>
      <Button size="xs" variant="outline" :disabled="running" @click="emit('cancel')">Cancel</Button>
    </div>
    <template v-if="!compact">
      <Select :model-value="String(pageSize)" @update:model-value="onPageSize">
        <SelectTrigger size="xs" class="w-36">
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
          :disabled="running || !hasPrev"
          @click="emit('page', page - 1)"
        >
          <ChevronLeft class="size-3.5" />
        </Button>
        <span class="text-muted-foreground min-w-18 text-center text-xs">
          <template v-if="pageCount != null">{{ page }} / {{ pageCount }}</template>
          <template v-else>{{ page }}</template>
        </span>
        <Button
          size="icon-xs"
          variant="outline"
          :disabled="running || !hasNext"
          @click="emit('page', page + 1)"
        >
          <ChevronRight class="size-3.5" />
        </Button>
      </div>
    </template>
  </div>
</template>
