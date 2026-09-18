<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { ChevronDown } from "@lucide/vue";
import {
  PopoverContent,
  PopoverPortal,
  PopoverRoot,
  PopoverTrigger,
} from "reka-ui";
import { cn } from "@/lib/utils";
import { Input } from "@/components/ui/input";

const CAP = 200;

const props = withDefaults(
  defineProps<{
    modelValue?: string;
    options: string[];
    placeholder?: string;
    noneValue?: string;
    noneLabel?: string;
    disabled?: boolean;
    class?: string;
  }>(),
  {
    placeholder: "Select database",
    noneLabel: "None",
    disabled: false,
  },
);

const emit = defineEmits<{
  "update:modelValue": [value: string];
}>();

const open = ref(false);
const filter = ref("");

watch(open, (value) => {
  if (!value) {
    filter.value = "";
  }
});

watch(
  () => props.disabled,
  (disabled) => {
    if (disabled) {
      open.value = false;
    }
  },
);

const filtered = computed(() => {
  const q = filter.value.trim().toLowerCase();
  const names = q
    ? props.options.filter((name) => name.toLowerCase().includes(q))
    : props.options;
  return { items: names.slice(0, CAP), extra: Math.max(0, names.length - CAP) };
});

const display = computed(() => {
  if (props.noneValue && (!props.modelValue || props.modelValue === props.noneValue)) {
    return props.noneLabel;
  }
  return props.modelValue || "";
});

function select(value: string) {
  if (props.disabled) {
    return;
  }
  emit("update:modelValue", value);
  open.value = false;
}

function onOpen(value: boolean) {
  if (props.disabled) {
    open.value = false;
    return;
  }
  open.value = value;
}
</script>

<template>
  <PopoverRoot :open="open" @update:open="onOpen">
    <PopoverTrigger as-child>
      <button
        type="button"
        data-slot="select-trigger"
        data-size="sm"
        :disabled="disabled"
        :class="cn(
          'border-input data-placeholder:text-muted-foreground dark:bg-input/30 dark:hover:bg-input/50 focus-visible:border-ring focus-visible:ring-ring/50 gap-1.5 border bg-transparent py-2 pr-2 pl-2.5 text-sm transition-colors select-none focus-visible:ring-3 h-7 rounded-[min(var(--radius-md),10px)] flex w-fit items-center justify-between whitespace-nowrap outline-none disabled:cursor-not-allowed disabled:opacity-50',
          props.class,
        )"
      >
        <span class="min-w-0 flex-1 truncate text-left" :class="display ? '' : 'text-muted-foreground'">
          {{ display || placeholder }}
        </span>
        <ChevronDown class="text-muted-foreground size-4 shrink-0 pointer-events-none" />
      </button>
    </PopoverTrigger>
    <PopoverPortal>
      <PopoverContent
        side="bottom"
        align="start"
        :side-offset="4"
        :collision-padding="8"
        class="bg-popover text-popover-foreground z-50 overflow-hidden rounded-lg shadow-md ring-1 ring-foreground/10 outline-none"
        :style="{ width: 'var(--reka-popper-anchor-width)', minWidth: '12rem' }"
      >
        <div class="p-1">
          <Input
            v-model="filter"
            placeholder="Filter databases…"
            class="h-7"
            @keydown.esc="open = false"
          />
        </div>
        <div class="max-h-64 overflow-auto p-1">
          <button
            v-if="noneValue"
            type="button"
            class="focus:bg-accent focus:text-accent-foreground gap-1.5 rounded-md py-1 pr-2 pl-1.5 text-sm relative flex w-full cursor-default items-center outline-hidden select-none"
            @click="select(noneValue)"
          >
            {{ noneLabel }}
          </button>
          <button
            v-for="name in filtered.items"
            :key="name"
            type="button"
            class="focus:bg-accent focus:text-accent-foreground gap-1.5 rounded-md py-1 pr-2 pl-1.5 text-sm relative flex w-full cursor-default items-center outline-hidden select-none"
            :class="name === modelValue ? 'bg-accent/60' : ''"
            @click="select(name)"
          >
            <span class="truncate">{{ name }}</span>
          </button>
          <p v-if="filtered.items.length === 0" class="text-muted-foreground px-1.5 py-1 text-xs">
            No matching databases.
          </p>
          <p v-else-if="filtered.extra > 0" class="text-muted-foreground px-1.5 py-1 text-xs">
            {{ filtered.extra }} more. Use search.
          </p>
        </div>
      </PopoverContent>
    </PopoverPortal>
  </PopoverRoot>
</template>
