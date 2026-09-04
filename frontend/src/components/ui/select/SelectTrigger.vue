<script setup lang="ts">
import type { SelectTriggerProps } from 'reka-ui'
import type { HTMLAttributes } from 'vue'
import type { SelectTriggerVariants } from '.'
import { ChevronDownIcon } from '@lucide/vue'
import { reactiveOmit } from '@vueuse/core'
import { SelectIcon, SelectTrigger, useForwardProps } from 'reka-ui'
import { cn } from '@/lib/utils'
import { selectTriggerVariants } from '.'

const props = withDefaults(
  defineProps<SelectTriggerProps & { class?: HTMLAttributes['class'], size?: SelectTriggerVariants['size'] }>(),
  { size: 'default' },
)

const delegatedProps = reactiveOmit(props, 'class', 'size')
const forwardedProps = useForwardProps(delegatedProps)
</script>

<template>
  <SelectTrigger
    data-slot="select-trigger"
    v-bind="forwardedProps"
    :class="cn(selectTriggerVariants({ size }), props.class)"
  >
    <slot />
    <SelectIcon as-child>
      <ChevronDownIcon class="text-muted-foreground size-4 pointer-events-none" />
    </SelectIcon>
  </SelectTrigger>
</template>
