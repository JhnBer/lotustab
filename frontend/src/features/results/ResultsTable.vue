<script setup lang="ts">
import { computed } from "vue";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { QueryResult } from "@/lib/wails";

const props = defineProps<{
  result: QueryResult | null;
  error: string;
  running: boolean;
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
      Run a query to see results.
    </div>
    <template v-else>
      <div class="text-muted-foreground flex items-center justify-between border-b px-3 py-2 text-xs">
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
              >
                <div>{{ column.name }}</div>
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
