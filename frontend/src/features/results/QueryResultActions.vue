<script setup lang="ts">
import { computed, ref } from "vue";
import { ChevronDown, Copy, Save } from "@lucide/vue";
import { toast } from "vue-sonner";
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
    toast.error("Could not copy to clipboard");
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
        title="Copy"
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
            title="Copy options"
            :disabled="!canCopyCurrent && !canCopyAll"
          >
            <ChevronDown />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" class="w-44">
          <DropdownMenuItem :disabled="!canCopyCurrent" @select="copyCurrent(false)">
            Copy
          </DropdownMenuItem>
          <DropdownMenuItem :disabled="!canCopyCurrent" @select="copyCurrent(true)">
            Copy as JSON
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem :disabled="!canCopyAll" @select="copyAll(false)">
            Copy all tabs
          </DropdownMenuItem>
          <DropdownMenuItem :disabled="!canCopyAll" @select="copyAll(true)">
            Copy all as JSON
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
    <Button
      size="icon-xs"
      variant="outline"
      title="Save"
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
