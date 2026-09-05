<script setup lang="ts">
import { computed } from "vue";
import { FileCode2, Plus, Table2, TableProperties, X } from "@lucide/vue";
import { VueDraggable } from "vue-draggable-plus";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import { useClipboardText } from "@/composables/useClipboardText";
import { useConnectionsStore } from "@/stores/connections";
import { useQueryStore, type WorkspaceTab } from "@/stores/query";
import { useSettingsStore } from "@/stores/settings";

const connections = useConnectionsStore();
const queries = useQueryStore();
const settings = useSettingsStore();
const { writeClipboard } = useClipboardText();

const wrapTabs = computed(() => settings.tabLayout === "wrap");

const visibleTabs = computed({
  get: () => queries.tabs.filter((t) => t.profileId === connections.selectedId),
  set: (next: WorkspaceTab[]) => {
    if (!connections.selectedId) {
      return;
    }
    queries.setOrder(
      connections.selectedId,
      next.map((t) => t.id),
    );
  },
});

function addTab() {
  const profile = connections.selected;
  if (!profile) {
    return;
  }
  queries.add(profile.id, profile.name);
}

async function copyName(title: string) {
  await writeClipboard(title);
}
</script>

<template>
  <div
    class="flex min-h-10 shrink-0 gap-1 border-b px-2 py-1"
    :class="wrapTabs ? 'items-start' : 'items-center'"
  >
    <component
      :is="wrapTabs ? 'div' : ScrollArea"
      :type="wrapTabs ? undefined : 'hover'"
      :orientation="wrapTabs ? undefined : 'horizontal'"
      class="min-w-0 flex-1"
      :class="wrapTabs ? '' : 'pb-2.5'"
      style="--wails-draggable: no-drag"
    >
      <VueDraggable
        v-model="visibleTabs"
        :animation="150"
        filter=".tab-close"
        :prevent-on-filter="true"
        :force-fallback="true"
        :fallback-on-body="true"
        :fallback-tolerance="4"
        :direction="wrapTabs ? undefined : 'horizontal'"
        class="flex gap-1"
        :class="wrapTabs ? 'flex-wrap' : 'w-max min-w-full flex-nowrap items-center'"
      >
        <ContextMenu v-for="tab in visibleTabs" :key="tab.id">
          <ContextMenuTrigger as-child>
            <div
              class="flex max-w-48 shrink-0 cursor-grab select-none items-center gap-1 rounded-md px-2 py-1 text-xs active:cursor-grabbing [-webkit-user-drag:none]"
              :class="tab.id === queries.activeTabId ? 'bg-muted text-foreground' : 'text-muted-foreground hover:bg-muted/60'"
              @click="queries.selectTab(tab.id)"
            >
              <TableProperties
                v-if="tab.kind === 'table' && tab.view === 'structure'"
                class="pointer-events-none size-3 shrink-0"
              />
              <Table2 v-else-if="tab.kind === 'table'" class="pointer-events-none size-3 shrink-0" />
              <FileCode2 v-else class="pointer-events-none size-3 shrink-0" />
              <span class="pointer-events-none truncate">{{ tab.title }}</span>
              <span
                class="tab-close hover:bg-background cursor-pointer rounded-sm p-0.5"
                @click.stop="queries.close(tab.id)"
                @mousedown.stop
              >
                <X class="size-3" />
              </span>
            </div>
          </ContextMenuTrigger>
          <ContextMenuContent>
            <ContextMenuItem @select="copyName(tab.title)">
              Copy name
            </ContextMenuItem>
            <ContextMenuItem @select="queries.close(tab.id)">
              Close
            </ContextMenuItem>
          </ContextMenuContent>
        </ContextMenu>
      </VueDraggable>
    </component>
    <Button
      size="icon-xs"
      variant="ghost"
      class="shrink-0"
      :class="wrapTabs ? 'self-start' : 'mb-2.5 self-center'"
      :disabled="!connections.selected"
      @click="addTab"
    >
      <Plus class="size-3.5" />
    </Button>
  </div>
</template>
