<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import type { InputVariants } from '.'
import { useVModel } from '@vueuse/core'
import { cn } from '@/lib/utils'
import { inputVariants } from '.'

const props = withDefaults(
  defineProps<{
    defaultValue?: string | number
    modelValue?: string | number
    class?: HTMLAttributes['class']
    size?: InputVariants['size']
  }>(),
  { size: 'default' },
)

const emits = defineEmits<{
  (e: 'update:modelValue', payload: string | number): void
}>()

const modelValue = useVModel(props, 'modelValue', emits, {
  passive: true,
  defaultValue: props.defaultValue,
})
</script>

<template>
  <input
    v-model="modelValue"
    data-slot="input"
    autocapitalize="none"
    autocorrect="off"
    spellcheck="false"
    :class="cn(inputVariants({ size }), props.class)"
  >
</template>
