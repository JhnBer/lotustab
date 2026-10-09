<script setup lang="ts">
import { ref } from "vue";
import { ChevronRight } from "@lucide/vue";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";

export type JsonTreeNodeModel =
  | { kind: "scalar"; label: string }
  | { kind: "branch"; label: string; children: JsonTreeNodeModel[] };

const props = withDefaults(
  defineProps<{
    node: JsonTreeNodeModel;
    depth?: number;
  }>(),
  { depth: 0 },
);

const open = ref(props.depth < 2);
</script>

<template>
  <div class="font-mono text-xs">
    <div
      v-if="node.kind === 'scalar'"
      class="text-foreground/90 truncate py-0.5"
      :style="{ paddingLeft: `${depth * 12}px` }"
    >
      {{ node.label }}
    </div>
    <Collapsible v-else v-model:open="open">
      <CollapsibleTrigger
        class="hover:bg-muted/60 flex w-full items-center gap-1 rounded-sm py-0.5 text-left"
        :style="{ paddingLeft: `${depth * 12}px` }"
      >
        <ChevronRight
          class="size-3 shrink-0 transition-transform"
          :class="open ? 'rotate-90' : ''"
        />
        <span class="truncate">{{ node.label }}</span>
      </CollapsibleTrigger>
      <CollapsibleContent>
        <JsonTreeNode
          v-for="(child, index) in node.children"
          :key="index"
          :node="child"
          :depth="depth + 1"
        />
      </CollapsibleContent>
    </Collapsible>
  </div>
</template>
