<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { Copy } from "@lucide/vue";
import { toast } from "vue-sonner";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { writeClipboard } from "@/composables/useClipboardText";
import { errorMessage } from "@/lib/errors";
import { QueryService, type TableRef } from "@/lib/wails";

const { t } = useI18n();

const props = defineProps<{
  open: boolean;
  profileId: string;
  table: TableRef | null;
}>();

const emit = defineEmits<{
  "update:open": [value: boolean];
  dropped: [];
}>();

const typed = ref("");
const dropping = ref(false);

const name = computed(() => props.table?.name ?? "");
const confirmed = computed(() => name.value !== "" && typed.value === name.value);

watch(
  () => props.open,
  (open) => {
    if (open) {
      typed.value = "";
    }
  },
);

async function copyName() {
  if (!name.value) {
    return;
  }
  if (!(await writeClipboard(name.value))) {
    toast.error(t("common.copyFailed"));
  }
}

async function drop() {
  if (!props.table || !props.profileId || !confirmed.value || dropping.value) {
    return;
  }
  dropping.value = true;
  try {
    await QueryService.DropTable(props.profileId, props.table);
    emit("update:open", false);
    emit("dropped");
  } catch (err) {
    toast.error(errorMessage(err));
  } finally {
    dropping.value = false;
  }
}
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent>
      <DialogHeader>
        <DialogTitle>{{ t('connections.dropTable') }}</DialogTitle>
        <DialogDescription>
          {{ t('connections.dropTableHelp') }}
        </DialogDescription>
      </DialogHeader>
      <div class="flex items-center gap-1">
        <span class="min-w-0 flex-1 truncate font-mono text-sm" :title="name">{{ name }}</span>
        <Button
          size="icon-xs"
          variant="ghost"
          class="shrink-0"
          :title="t('common.copy')"
          :aria-label="t('common.copy')"
          @click="copyName"
        >
          <Copy class="size-3.5" />
        </Button>
      </div>
      <Input
        v-model="typed"
        autofocus
        :placeholder="t('connections.typeTableName')"
        @keydown.enter.prevent="drop"
      />
      <DialogFooter>
        <Button variant="outline" :disabled="dropping" @click="emit('update:open', false)">
          {{ t('common.cancel') }}
        </Button>
        <Button variant="destructive" :disabled="!confirmed || dropping" @click="drop">
          {{ t('common.delete') }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
