<script setup lang="ts">
import type { ScrollAreaRootProps } from 'reka-ui'
import type { HTMLAttributes } from 'vue'
import { computed } from 'vue'
import { reactiveOmit } from '@vueuse/core'
import {
  ScrollAreaCorner,
  ScrollAreaRoot,
  ScrollAreaViewport,
} from 'reka-ui'
import { cn } from '@/lib/utils'
import ScrollBar from './ScrollBar.vue'

const props = withDefaults(
  defineProps<ScrollAreaRootProps & {
    class?: HTMLAttributes['class']
    orientation?: 'vertical' | 'horizontal' | 'both'
  }>(),
  { orientation: 'both' },
)

const delegatedProps = reactiveOmit(props, 'class', 'orientation')
const showVertical = computed(() => props.orientation !== 'horizontal')
const showHorizontal = computed(() => props.orientation !== 'vertical')
</script>

<template>
  <ScrollAreaRoot
    data-slot="scroll-area"
    v-bind="delegatedProps"
    :class="cn('relative', props.class)"
  >
    <ScrollAreaViewport
      data-slot="scroll-area-viewport"
      class="size-full rounded-[inherit] transition-[color,box-shadow] outline-none focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-1 scrollbar-none"
    >
      <slot />
    </ScrollAreaViewport>
    <ScrollBar v-if="showVertical" />
    <ScrollBar v-if="showHorizontal" orientation="horizontal" />
    <ScrollAreaCorner />
  </ScrollAreaRoot>
</template>
