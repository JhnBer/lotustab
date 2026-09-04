import type { VariantProps } from 'class-variance-authority'
import { cva } from 'class-variance-authority'

export { default as Input } from './Input.vue'

export const inputVariants = cva(
  'dark:bg-input/30 border-input focus-visible:border-ring focus-visible:ring-ring/50 aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive dark:aria-invalid:border-destructive/50 disabled:bg-input/50 dark:disabled:bg-input/80 rounded-lg border bg-transparent transition-colors file:text-sm file:font-medium focus-visible:ring-3 aria-invalid:ring-3 w-full min-w-0 outline-none file:inline-flex file:border-0 file:bg-transparent file:text-foreground placeholder:text-muted-foreground disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50',
  {
    variants: {
      size: {
        default: 'h-8 px-2.5 py-1 text-base md:text-sm file:h-6',
        xs: 'h-6 px-2 py-0.5 text-xs file:h-4',
      },
    },
    defaultVariants: { size: 'default' },
  },
)
export type InputVariants = VariantProps<typeof inputVariants>
