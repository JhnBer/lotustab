<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { ScrollArea } from "@/components/ui/scroll-area";
import JsonTreeNode, { type JsonTreeNodeModel } from "@/features/results/JsonTreeNode.vue";

const { t } = useI18n();

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
      label: nodeLabel(key, t("results.jsonArray", { n: value.length })),
      children: value.map((item, index) => buildNode(item, String(index))),
    };
  }
  if (typeof value === "object") {
    const entries = Object.entries(value as Record<string, unknown>);
    return {
      kind: "branch",
      label: nodeLabel(key, t("results.jsonObject", { n: entries.length })),
      children: entries.map(([childKey, child]) => buildNode(child, childKey)),
    };
  }
  if (typeof value === "string") {
    return { kind: "scalar", label: nodeLabel(key, JSON.stringify(value)) };
  }
  return { kind: "scalar", label: nodeLabel(key, String(value)) };
}

const root = computed(() => {
  void t("results.jsonArray", { n: 0 });
  return buildNode(props.value);
});
</script>

<template>
  <ScrollArea class="h-full">
    <div class="p-2">
      <JsonTreeNode :node="root" :depth="0" />
    </div>
  </ScrollArea>
</template>
