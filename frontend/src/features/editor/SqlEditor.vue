<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from "vue";
import { defaultKeymap, history, historyKeymap, indentWithTab } from "@codemirror/commands";
import { sql } from "@codemirror/lang-sql";
import { Compartment, EditorState } from "@codemirror/state";
import { oneDark } from "@codemirror/theme-one-dark";
import { EditorView, highlightActiveLine, keymap, lineNumbers, placeholder } from "@codemirror/view";
import { useColorMode } from "@vueuse/core";

const props = defineProps<{
  modelValue: string;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: string];
  run: [];
}>();

const host = ref<HTMLDivElement | null>(null);
const mode = useColorMode();
const themeComp = new Compartment();
let view: EditorView | undefined;

function themeExtensions() {
  return mode.value === "light" ? [] : [oneDark];
}

function createEditor() {
  if (!host.value) {
    return;
  }
  view?.destroy();
  view = new EditorView({
    parent: host.value,
    state: EditorState.create({
      doc: props.modelValue,
      extensions: [
        lineNumbers(),
        highlightActiveLine(),
        history(),
        sql(),
        placeholder("Write SQL and press ⌘/Ctrl+Enter"),
        keymap.of([
          ...defaultKeymap,
          ...historyKeymap,
          indentWithTab,
          {
            key: "Mod-Enter",
            run: () => {
              emit("run");
              return true;
            },
          },
        ]),
        EditorView.updateListener.of((update) => {
          if (update.docChanged) {
            emit("update:modelValue", update.state.doc.toString());
          }
        }),
        EditorView.theme({
          "&": { height: "100%", fontSize: "13px" },
          ".cm-scroller": { overflow: "auto" },
          ".cm-content": { fontFamily: "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace" },
        }),
        themeComp.of(themeExtensions()),
      ],
    }),
  });
}

onMounted(createEditor);

watch(
  () => props.modelValue,
  (value) => {
    if (!view) {
      return;
    }
    if (value !== view.state.doc.toString()) {
      view.dispatch({
        changes: { from: 0, to: view.state.doc.length, insert: value },
      });
    }
  },
);

watch(mode, () => {
  view?.dispatch({
    effects: themeComp.reconfigure(themeExtensions()),
  });
});

onBeforeUnmount(() => {
  view?.destroy();
  view = undefined;
});
</script>

<template>
  <div ref="host" class="h-full min-h-0 overflow-hidden" />
</template>
