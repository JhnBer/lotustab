<script setup lang="ts">
import { computed } from "vue";
import { GripVertical, Undo2, X } from "@lucide/vue";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  COMMON_COLUMN_TYPES,
  defaultsEqual,
  originalFor,
  type DraftColumn,
} from "@/features/results/structure";
import type { ColumnInfo } from "@/lib/wails";
import { useQueryStore } from "@/stores/query";

const props = defineProps<{
  tabId: string;
  col: DraftColumn;
  columns?: ColumnInfo[] | null;
  running?: boolean;
  canDrag?: boolean;
}>();

const emit = defineEmits<{
  reorderstart: [event: DragEvent];
  reorderend: [];
  reorderdrop: [event: DragEvent];
}>();

const queries = useQueryStore();

const typeChoices = computed(() => {
  const current = props.col.type.trim();
  if (current && !COMMON_COLUMN_TYPES.includes(current)) {
    return [current, ...COMMON_COLUMN_TYPES];
  }
  return COMMON_COLUMN_TYPES;
});

const defaultPlaceholder = computed(() => {
  if (props.col.defaultValue === null) {
    return "NULL";
  }
  if (props.col.defaultValue === "") {
    return "empty string";
  }
  return undefined;
});

const disabled = computed(() => props.col.dropped || Boolean(props.running));

const orig = computed(() => originalFor(props.col, props.columns));
const extra = computed(() => orig.value?.extra ?? "");

const rowClass = computed(() => {
  if (props.col.dropped) {
    return "bg-destructive/10 text-destructive";
  }
  if (props.col.originalName === null) {
    return "bg-green-500/10";
  }
  return "";
});

function fieldDirty(field: "name" | "type" | "nullable" | "defaultValue" | "comment") {
  if (props.col.originalName === null || props.col.dropped) {
    return false;
  }
  const source = orig.value;
  if (!source) {
    return true;
  }
  switch (field) {
    case "name":
      return source.name !== props.col.name.trim();
    case "type":
      return source.type !== props.col.type.trim();
    case "nullable":
      return Boolean(source.nullable) !== props.col.nullable;
    case "defaultValue":
      return !defaultsEqual(source.default, props.col.defaultValue);
    case "comment":
      return (source.comment ?? "") !== props.col.comment.trim();
  }
}

function cellClass(active: boolean) {
  return active ? "bg-yellow-500/15 ring-1 ring-inset ring-yellow-500" : "";
}

function patch(next: Partial<DraftColumn>) {
  queries.updateStructureColumn(props.tabId, props.col.key, next);
}

function onHandleDragStart(event: DragEvent) {
  if (!props.canDrag || props.col.dropped) {
    event.preventDefault();
    return;
  }
  emit("reorderstart", event);
}
</script>

<template>
  <div
    class="flex border-b"
    :class="rowClass"
    role="row"
    @dragover.prevent
    @drop.prevent="emit('reorderdrop', $event)"
  >
    <div class="flex w-8 shrink-0 items-center px-1">
      <button
        v-if="canDrag"
        type="button"
        draggable="true"
        class="drag-handle text-muted-foreground hover:text-foreground cursor-grab px-1 active:cursor-grabbing"
        aria-label="Reorder column"
        :disabled="col.dropped"
        @dragstart="onHandleDragStart"
        @dragend="emit('reorderend')"
      >
        <GripVertical class="size-3.5" />
      </button>
      <span v-else class="text-muted-foreground/40 inline-flex px-1">
        <GripVertical class="size-3.5" />
      </span>
    </div>
    <div class="flex min-w-40 flex-1 items-center px-1 py-1" :class="cellClass(fieldDirty('name'))">
      <Input
        class="h-7"
        :model-value="col.name"
        :disabled="disabled"
        :class="col.dropped ? 'line-through' : ''"
        @update:model-value="patch({ name: String($event) })"
      />
    </div>
    <div class="flex w-48 shrink-0 items-center px-1 py-1" :class="cellClass(fieldDirty('type'))">
      <Select
        :model-value="col.type"
        :disabled="disabled"
        @update:model-value="patch({ type: String($event ?? '') })"
      >
        <SelectTrigger size="sm" class="h-7 w-full">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem v-for="type in typeChoices" :key="type" :value="type">
            {{ type }}
          </SelectItem>
        </SelectContent>
      </Select>
    </div>
    <div class="flex w-20 shrink-0 items-center justify-center px-1 py-1" :class="cellClass(fieldDirty('nullable'))">
      <Checkbox
        :checked="col.nullable"
        :disabled="disabled"
        @update:checked="patch({ nullable: Boolean($event) })"
      />
    </div>
    <div class="flex min-w-36 flex-1 items-center px-1 py-1" :class="cellClass(fieldDirty('defaultValue'))">
      <ContextMenu>
        <ContextMenuTrigger as-child>
          <div class="w-full">
            <Input
              class="h-7 placeholder:italic"
              :model-value="col.defaultValue ?? ''"
              :placeholder="defaultPlaceholder"
              :disabled="disabled"
              @update:model-value="patch({ defaultValue: String($event) })"
            />
          </div>
        </ContextMenuTrigger>
        <ContextMenuContent>
          <ContextMenuItem :disabled="disabled" @select="patch({ defaultValue: null })">
            Set NULL
          </ContextMenuItem>
        </ContextMenuContent>
      </ContextMenu>
    </div>
    <div class="flex w-20 shrink-0 items-center justify-center px-1 py-1">
      <Checkbox :checked="col.isPrimaryKey" disabled />
    </div>
    <div
      class="text-muted-foreground flex w-56 min-w-40 shrink-0 items-center truncate px-2 py-1"
      :title="extra || undefined"
    >
      {{ extra }}
    </div>
    <div class="flex min-w-36 flex-1 items-center px-1 py-1" :class="cellClass(fieldDirty('comment'))">
      <Input
        class="h-7"
        :model-value="col.comment"
        :disabled="disabled"
        @update:model-value="patch({ comment: String($event) })"
      />
    </div>
    <div class="flex w-10 shrink-0 items-center px-1 py-1">
      <Button
        size="icon-xs"
        variant="ghost"
        :aria-label="col.dropped ? 'Restore column' : 'Drop column'"
        :disabled="running"
        @click="queries.toggleDropStructureColumn(tabId, col.key)"
      >
        <Undo2 v-if="col.dropped" class="size-3.5" />
        <X v-else class="size-3.5" />
      </Button>
    </div>
  </div>
</template>
