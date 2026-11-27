<script setup lang="ts">
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { ChevronDown, Copy, Save } from "@lucide/vue";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { writeClipboard } from "@/composables/useClipboardText";
import SaveResultsDialog from "@/features/results/SaveResultsDialog.vue";
import {
  copyableResults,
  currentCopyable,
  resultJson,
  resultTsv,
  resultsJson,
  resultsTsv,
} from "@/features/results/copyResults";
import type { QueryTab } from "@/features/editor/tabs";

const { t } = useI18n();

const props = defineProps<{
  tab: QueryTab;
}>();

const saveOpen = ref(false);

const allSets = computed(() => copyableResults(props.tab));
const current = computed(() => currentCopyable(props.tab));
const canCopyCurrent = computed(() => current.value != null);
const canCopyAll = computed(() => allSets.value.length > 0);

async function copyText(text: string) {
  if (!(await writeClipboard(text))) {
    toast.error(t("common.copyFailed"), { copy: true });
  }
}

async function copyCurrent(asJson: boolean) {
  const item = current.value;
  if (!item) {
    return;
  }
  await copyText(asJson ? resultJson(item) : resultTsv(item));
}

async function copyAll(asJson: boolean) {
  if (allSets.value.length === 0) {
    return;
  }
  await copyText(asJson ? resultsJson(allSets.value) : resultsTsv(allSets.value));
}
</script>

<template>
  <div class="flex shrink-0 items-center gap-1">
    <div class="flex">
      <Button
        size="icon-xs"
        variant="outline"
        class="rounded-r-none"
        :title="t('common.copy')"
        :disabled="!canCopyCurrent"
        @click="copyCurrent(false)"
      >
        <Copy />
      </Button>
      <DropdownMenu>
        <DropdownMenuTrigger as-child>
          <Button
            size="icon-xs"
            variant="outline"
            class="rounded-l-none border-l-0 px-1"
            :title="t('results.copyOptions')"
            :disabled="!canCopyCurrent && !canCopyAll"
          >
            <ChevronDown />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" class="w-44">
          <DropdownMenuItem :disabled="!canCopyCurrent" @select="copyCurrent(false)">
            {{ t('common.copy') }}
          </DropdownMenuItem>
          <DropdownMenuItem :disabled="!canCopyCurrent" @select="copyCurrent(true)">
            {{ t('results.copyJson') }}
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem :disabled="!canCopyAll" @select="copyAll(false)">
            {{ t('results.copyAll') }}
          </DropdownMenuItem>
          <DropdownMenuItem :disabled="!canCopyAll" @select="copyAll(true)">
            {{ t('results.copyAllJson') }}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
    <Button
      size="icon-xs"
      variant="outline"
      :title="t('results.save')"
      :disabled="!canCopyAll"
      @click="saveOpen = true"
    >
      <Save />
    </Button>
    <SaveResultsDialog
      :open="saveOpen"
      :profile-id="tab.profileId"
      :sets="allSets"
      @update:open="saveOpen = $event"
    />
  </div>
</template>
