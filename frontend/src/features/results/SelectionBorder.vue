<script setup lang="ts">
import { computed } from "vue";

const props = defineProps<{
  top?: boolean;
  right?: boolean;
  bottom?: boolean;
  left?: boolean;
  row?: number;
  col?: number;
  selectedKeys?: Set<string>;
}>();

const edges = computed(() => {
  const keys = props.selectedKeys;
  if (keys && props.row != null && props.col != null) {
    if (!keys.has(`${props.row}:${props.col}`)) {
      return null;
    }
    return {
      top: !keys.has(`${props.row - 1}:${props.col}`),
      bottom: !keys.has(`${props.row + 1}:${props.col}`),
      left: !keys.has(`${props.row}:${props.col - 1}`),
      right: !keys.has(`${props.row}:${props.col + 1}`),
    };
  }
  if (props.top || props.right || props.bottom || props.left) {
    return {
      top: Boolean(props.top),
      right: Boolean(props.right),
      bottom: Boolean(props.bottom),
      left: Boolean(props.left),
    };
  }
  return null;
});
</script>

<template>
  <div v-if="edges" class="pointer-events-none absolute inset-0 z-10">
    <div v-if="edges.top" class="bg-primary absolute inset-x-0 top-0 h-px" />
    <div v-if="edges.bottom" class="bg-primary absolute inset-x-0 bottom-0 h-px" />
    <div v-if="edges.left" class="bg-primary absolute inset-y-0 left-0 w-px" />
    <div v-if="edges.right" class="bg-primary absolute inset-y-0 right-0 w-px" />
  </div>
</template>
