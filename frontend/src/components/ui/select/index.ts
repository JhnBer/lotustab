import type { VariantProps } from 'class-variance-authority'
import { cva } from 'class-variance-authority'

export { default as Select } from './Select.vue'
export { default as SelectContent } from './SelectContent.vue'
export { default as SelectGroup } from './SelectGroup.vue'
export { default as SelectItem } from './SelectItem.vue'
export { default as SelectItemText } from './SelectItemText.vue'
export { default as SelectLabel } from './SelectLabel.vue'
export { default as SelectScrollDownButton } from './SelectScrollDownButton.vue'
export { default as SelectScrollUpButton } from './SelectScrollUpButton.vue'
export { default as SelectSeparator } from './SelectSeparator.vue'
export { default as SelectTrigger } from './SelectTrigger.vue'
export { default as SelectValue } from './SelectValue.vue'

export const selectTriggerVariants = cva(
  'border-input data-placeholder:text-muted-foreground dark:bg-input/30 dark:hover:bg-input/50 focus-visible:border-ring focus-visible:ring-ring/50 aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive dark:aria-invalid:border-destructive/50 gap-1.5 rounded-lg border bg-transparent py-2 pr-2 pl-2.5 text-sm transition-colors select-none focus-visible:ring-3 aria-invalid:ring-3 *:data-[slot=select-value]:gap-1.5 [&_svg:not([class*=size-])]:size-4 flex w-fit items-center justify-between whitespace-nowrap outline-none disabled:cursor-not-allowed disabled:opacity-50 *:data-[slot=select-value]:line-clamp-1 *:data-[slot=select-value]:flex *:data-[slot=select-value]:items-center [&_svg]:pointer-events-none [&_svg]:shrink-0',
  {
    variants: {
      size: {
        default: 'h-8',
        sm: 'h-7 rounded-[min(var(--radius-md),10px)]',
        xs: 'h-6 rounded-[min(var(--radius-md),10px)] text-xs [&_svg:not([class*=size-])]:size-3',
      },
    },
    defaultVariants: { size: 'default' },
  },
)
export type SelectTriggerVariants = VariantProps<typeof selectTriggerVariants>
