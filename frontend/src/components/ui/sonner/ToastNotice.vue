<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  CircleCheckIcon,
  CopyIcon,
  InfoIcon,
  Loader2Icon,
  OctagonXIcon,
  TriangleAlertIcon,
  XIcon,
} from '@lucide/vue'
import { writeClipboard } from '@/composables/useClipboardText'

const props = withDefaults(defineProps<{
  message: string
  description?: string
  tone?: 'default' | 'success' | 'info' | 'warning' | 'error' | 'loading'
  showClose?: boolean
  copyText?: string
  actionLabel?: string
  onAction?: (event: MouseEvent) => void
  onCloseToast?: () => void
}>(), {
  tone: 'default',
  showClose: true,
})

const { t } = useI18n()

const icons = {
  success: CircleCheckIcon,
  info: InfoIcon,
  warning: TriangleAlertIcon,
  error: OctagonXIcon,
  loading: Loader2Icon,
} as const

const icon = computed(() => (props.tone === 'default' ? undefined : icons[props.tone]))
const fullText = computed(() => [props.message, props.description].filter(Boolean).join('\n'))
const showActions = computed(() => props.showClose || !!props.copyText)

function dismiss() {
  props.onCloseToast?.()
}

function onCopy() {
  if (props.copyText) {
    void writeClipboard(props.copyText)
  }
  dismiss()
}

function onActionClick(event: MouseEvent) {
  props.onAction?.(event)
  if (!event.defaultPrevented) {
    dismiss()
  }
}
</script>

<template>
  <div class="flex w-full items-start gap-2.5 px-3.5 py-3">
    <component
      :is="icon"
      v-if="icon"
      class="mt-0.5 size-4 shrink-0"
      :class="tone === 'loading' ? 'animate-spin' : ''"
    />
    <div class="min-w-0 flex-1">
      <p class="line-clamp-3 break-words text-[13px] leading-5 whitespace-pre-wrap" :title="fullText">
        <span class="font-medium">{{ message }}</span><template v-if="description"><span class="block font-normal">{{ description }}</span></template>
      </p>
      <button
        v-if="actionLabel"
        type="button"
        class="mt-1 block text-left text-xs font-medium underline-offset-2 hover:underline"
        @pointerdown.stop
        @click="onActionClick"
      >
        {{ actionLabel }}
      </button>
    </div>
    <div v-if="showActions" class="flex shrink-0 flex-col gap-1">
      <button
        v-if="showClose"
        type="button"
        class="inline-flex size-6 items-center justify-center rounded-md text-inherit hover:bg-black/10 dark:hover:bg-white/15"
        :aria-label="t('common.close')"
        @pointerdown.stop
        @click="dismiss"
      >
        <XIcon class="size-3.5" />
      </button>
      <button
        v-if="copyText"
        type="button"
        class="inline-flex size-6 items-center justify-center rounded-md text-inherit hover:bg-black/10 dark:hover:bg-white/15"
        :aria-label="t('common.copy')"
        @pointerdown.stop
        @click="onCopy"
      >
        <CopyIcon class="size-3.5" />
      </button>
    </div>
  </div>
</template>
