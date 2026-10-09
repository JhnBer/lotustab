<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { X } from "@lucide/vue";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";
import JsonTree from "@/features/results/JsonTree.vue";
import {
  formatBinaryPreview,
  inspectorModeFor,
  tryParseJson,
  type ValueInspectorPayload,
} from "@/features/results/valueInspector";

const props = withDefaults(
  defineProps<{
    payload: ValueInspectorPayload;
    /** When true, the panel is hosted in a dedicated Wails window. */
    windowMode?: boolean;
    returnUnavailable?: boolean;
  }>(),
  {
    windowMode: false,
    returnUnavailable: false,
  },
);

const emit = defineEmits<{
  close: [];
  "update:text": [text: string];
  openWindow: [];
  returnPanel: [];
}>();

const text = ref(props.payload.text);
const view = ref<"tree" | "raw">("tree");
/** UI chrome for the current cell open; not recomputed from in-progress text. */
const mode = ref(inspectorModeFor(props.payload.columnType, props.payload.text));

function cellIdentity(p: ValueInspectorPayload): string {
  return `${p.token}\0${p.row}\0${p.col}\0${p.columnName}`;
}

watch(
  () => props.payload,
  (next, prev) => {
    const identityChanged = !prev || cellIdentity(next) !== cellIdentity(prev);
    if (identityChanged) {
      text.value = next.text;
      mode.value = inspectorModeFor(next.columnType, next.text);
      view.value = mode.value === "tree" ? "tree" : "raw";
      return;
    }
    // Same cell: parent may echo keystrokes or push an external text update.
    // Sync text when it diverges, but never reset Tree/Raw selection.
    if (next.text !== text.value) {
      text.value = next.text;
    }
  },
  { immediate: true },
);

const parsedJson = computed(() => tryParseJson(text.value));
const binaryPreview = computed(() => formatBinaryPreview(text.value));
const canEdit = computed(() => props.payload.editable && mode.value !== "binary");

function onTextInput(value: string | number) {
  const next = String(value);
  text.value = next;
  if (canEdit.value) {
    emit("update:text", next);
  }
}

const title = computed(() => {
  const name = props.payload.columnName || "value";
  const type = props.payload.columnType ? ` · ${props.payload.columnType}` : "";
  return `${name}${type}`;
});
</script>

<template>
  <div class="bg-background flex h-full min-h-0 flex-col border-l">
    <div class="flex items-center gap-2 border-b px-3 py-2">
      <div class="min-w-0 flex-1">
        <div class="truncate text-sm font-medium">{{ title }}</div>
        <div v-if="!payload.editable" class="text-muted-foreground text-[10px]">
          Read-only
        </div>
      </div>
      <Button
        v-if="windowMode"
        variant="outline"
        size="sm"
        class="shrink-0"
        @click="emit('returnPanel')"
      >
        Return to side panel
      </Button>
      <Button
        v-else
        variant="outline"
        size="sm"
        class="shrink-0"
        @click="emit('openWindow')"
      >
        Open in new window
      </Button>
      <Button
        v-if="!windowMode"
        variant="ghost"
        size="icon"
        class="size-8 shrink-0"
        @click="emit('close')"
      >
        <X class="size-4" />
        <span class="sr-only">Close</span>
      </Button>
    </div>
    <div
      v-if="returnUnavailable"
      class="text-destructive border-b px-3 py-2 text-xs"
    >
      Source window is closed. Nothing to return to.
    </div>
    <div class="flex min-h-0 flex-1 flex-col p-3">
      <template v-if="mode === 'binary'">
        <ScrollArea class="min-h-0 flex-1 rounded-md border">
          <pre class="p-3 font-mono text-xs break-all whitespace-pre-wrap">{{ binaryPreview }}</pre>
        </ScrollArea>
      </template>
      <template v-else-if="mode === 'tree'">
        <Tabs v-model="view" class="flex min-h-0 flex-1 flex-col gap-2">
          <TabsList class="w-fit">
            <TabsTrigger value="tree">Tree</TabsTrigger>
            <TabsTrigger value="raw">Raw</TabsTrigger>
          </TabsList>
          <TabsContent value="tree" class="min-h-0 flex-1 overflow-hidden rounded-md border">
            <JsonTree v-if="parsedJson !== undefined" :value="parsedJson" />
            <div v-else class="text-muted-foreground p-3 text-xs">
              Value is not valid JSON. Switch to Raw to inspect or edit.
            </div>
          </TabsContent>
          <TabsContent value="raw" class="min-h-0 flex-1">
            <Textarea
              class="h-full min-h-0 resize-none font-mono text-xs"
              :model-value="text"
              :readonly="!canEdit"
              @update:model-value="onTextInput"
            />
          </TabsContent>
        </Tabs>
      </template>
      <template v-else>
        <Textarea
          class="h-full min-h-0 resize-none font-mono text-xs"
          :model-value="text"
          :readonly="!canEdit"
          @update:model-value="onTextInput"
        />
      </template>
    </div>
  </div>
</template>
