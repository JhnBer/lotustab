<script setup lang="ts">
import { computed } from "vue";
import { ScrollArea } from "@/components/ui/scroll-area";
import JsonTreeNode, { type JsonTreeNodeModel } from "@/features/results/JsonTreeNode.vue";

const props = defineProps<{
  value: unknown;
}>();

function nodeLabel(key: string | undefined, preview: string): string {
  return key == null || key === "" ? preview : `${key}: ${preview}`;
}

function buildNode(value: unknown, key?: string): JsonTreeNodeModel {
  if (value == null) {
    return { kind: "scalar", label: nodeLabel(key, "null") };
  }
  if (Array.isArray(value)) {
    return {
      kind: "branch",
      label: nodeLabel(key, `Array(${value.length})`),
      children: value.map((item, index) => buildNode(item, String(index))),
    };
  }
  if (typeof value === "object") {
    const entries = Object.entries(value as Record<string, unknown>);
    return {
      kind: "branch",
      label: nodeLabel(key, `Object(${entries.length})`),
      children: entries.map(([childKey, child]) => buildNode(child, childKey)),
    };
  }
  if (typeof value === "string") {
    return { kind: "scalar", label: nodeLabel(key, JSON.stringify(value)) };
  }
  return { kind: "scalar", label: nodeLabel(key, String(value)) };
}

const root = computed(() => buildNode(props.value));
</script>

<template>
  <ScrollArea class="h-full">
    <div class="p-2">
      <JsonTreeNode :node="root" :depth="0" />
    </div>
  </ScrollArea>
</template>
