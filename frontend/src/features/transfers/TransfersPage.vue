<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useNow } from "@vueuse/core";
import { Badge } from "@/components/ui/badge";
import { Spinner } from "@/components/ui/spinner";
import { formatBytes, formatDuration, formatRows } from "@/features/export/format";
import {
  useTransfersStore,
  type TransferRecord,
  type TransferStatus,
} from "@/stores/transfers";

const transfers = useTransfersStore();
const { t } = useI18n();
const now = useNow({ interval: 1000 });

const records = computed(() => transfers.records);

function elapsed(rec: TransferRecord): string {
  if (rec.durationMs != null) {
    return formatDuration(rec.durationMs);
  }
  return formatDuration(now.value.getTime() - rec.startedAt);
}

function rowsLabel(rec: TransferRecord): string {
  const rows = formatRows(rec.rows);
  if (rec.totalRows != null) {
    return t("common.rowsOf", { rows, total: formatRows(rec.totalRows) });
  }
  return t("common.rows", { n: rows });
}

function statusLabel(status: TransferStatus): string {
  switch (status) {
    case "running":
      return t("transfers.running");
    case "done":
      return t("transfers.done");
    case "failed":
      return t("transfers.failed");
    case "canceled":
      return t("transfers.canceled");
  }
}

function statusVariant(status: TransferStatus): "default" | "secondary" | "destructive" | "outline" {
  switch (status) {
    case "running":
      return "default";
    case "done":
      return "secondary";
    case "failed":
      return "destructive";
    case "canceled":
      return "outline";
  }
}
</script>

<template>
  <section class="mx-auto flex h-full w-full max-w-2xl flex-col gap-6 overflow-auto py-2">
    <div>
      <h1 class="text-lg font-medium">{{ t('transfers.exports') }}</h1>
      <p class="text-muted-foreground mt-1 text-sm">
        {{ t('transfers.subtitle') }}
      </p>
    </div>

    <p v-if="records.length === 0" class="text-muted-foreground text-sm">
      {{ t('transfers.empty') }}
    </p>

    <ul v-else class="flex flex-col gap-2">
      <li
        v-for="rec in records"
        :key="rec.id"
        class="flex items-start justify-between gap-3 rounded-lg border p-3"
      >
        <div class="min-w-0 flex-1">
          <div class="flex items-center gap-2">
            <Spinner v-if="rec.status === 'running'" class="size-3.5 shrink-0" />
            <p class="truncate text-sm font-medium">{{ rec.table }}</p>
            <Badge :variant="statusVariant(rec.status)">
              {{ statusLabel(rec.status) }}
            </Badge>
          </div>
          <p class="text-muted-foreground mt-1 text-xs">
            {{ rowsLabel(rec) }}
            · {{ formatBytes(rec.bytes) }}
            · {{ elapsed(rec) }}
          </p>
          <p v-if="rec.error" class="text-destructive mt-1 truncate text-xs">
            {{ rec.error }}
          </p>
        </div>
      </li>
    </ul>

    <p class="text-muted-foreground mt-auto text-xs">
      {{ t('transfers.cleared') }}
    </p>
  </section>
</template>
