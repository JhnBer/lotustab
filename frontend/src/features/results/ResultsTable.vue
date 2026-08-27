<script setup lang="ts">
import { computed } from "vue";
import { ArrowDown, ArrowUp, ArrowUpDown } from "@lucide/vue";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { QueryResult } from "@/lib/wails";

const props = withDefaults(
  defineProps<{
    result: QueryResult | null;
    error: string;
    running: boolean;
    sortable?: boolean;
    sortColumn?: string;
    sortDir?: string;
    emptyHint?: string;
    showStatus?: boolean;
  }>(),
  {
    sortable: false,
    sortColumn: "",
    sortDir: "",
    emptyHint: "Run a query to see results.",
    showStatus: true,
  },
);

const emit = defineEmits<{
  sort: [column: string];
}>();

const columns = computed(() => props.result?.columns ?? []);
const rows = computed(() => props.result?.rows ?? []);

function cellText(value: unknown): string {
  if (value === null || value === undefined) {
    return "NULL";
  }
  if (typeof value === "object") {
    return JSON.stringify(value);
  }
  return String(value);
}

function isNull(value: unknown) {
  return value === null || value === undefined;
}

function onHeaderClick(name: string) {
  if (props.sortable) {
    emit("sort", name);
  }
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col">
    <div v-if="running" class="text-muted-foreground p-4 text-sm">
      Running…
    </div>
    <div v-else-if="error" class="text-destructive overflow-auto p-4 font-mono text-sm whitespace-pre-wrap">
      {{ error }}
    </div>
    <div v-else-if="!result" class="text-muted-foreground p-4 text-sm">
      {{ emptyHint }}
    </div>
    <template v-else>
      <div v-if="showStatus" class="text-muted-foreground flex items-center justify-between border-b px-3 py-2 text-xs">
        <span>
          {{ rows.length }} row{{ rows.length === 1 ? "" : "s" }}
          <span v-if="result.truncated"> (truncated)</span>
        </span>
        <span>{{ result.durationMs }} ms · {{ result.rowsAffected }} affected</span>
      </div>
      <div v-if="columns.length === 0" class="text-muted-foreground p-4 text-sm">
        Statement completed. {{ result.rowsAffected }} row{{ result.rowsAffected === 1 ? "" : "s" }} affected.
      </div>
      <div v-else class="min-h-0 flex-1 overflow-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead
                v-for="column in columns"
                :key="column.name"
                class="bg-background sticky top-0 whitespace-nowrap"
                :class="sortable ? 'cursor-pointer select-none hover:bg-muted/60' : ''"
                @click="onHeaderClick(column.name)"
              >
                <div class="flex items-center gap-1">
                  <span>{{ column.name }}</span>
                  <ArrowUp
                    v-if="sortable && sortColumn === column.name && sortDir === 'asc'"
                    class="size-3"
                  />
                  <ArrowDown
                    v-else-if="sortable && sortColumn === column.name && sortDir === 'desc'"
                    class="size-3"
                  />
                  <ArrowUpDown
                    v-else-if="sortable"
                    class="text-muted-foreground size-3 opacity-40"
                  />
                </div>
                <div class="text-muted-foreground font-normal">{{ column.type }}</div>
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="(row, rowIndex) in rows" :key="rowIndex">
              <TableCell
                v-for="(value, colIndex) in row ?? []"
                :key="colIndex"
                class="max-w-[28rem] truncate font-mono text-xs"
                :class="isNull(value) ? 'text-muted-foreground italic' : ''"
                :title="cellText(value)"
              >
                {{ cellText(value) }}
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </div>
    </template>
  </div>
</template>
