<template>
  <div ref="containerRef" class="monaco-editor-container"></div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from "vue";
import type { editor as MonacoEditor } from "monaco-editor";

const props = defineProps<{
  modelValue: string;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: string];
}>();

const containerRef = ref<HTMLDivElement | null>(null);
let editorInstance: MonacoEditor.IStandaloneCodeEditor | null = null;
// When true, content-change events emitted by Monaco are ignored to avoid
// feeding our own programmatic setValue() back to the parent.
let suppressEvent = false;
let themeObserver: MutationObserver | null = null;

function currentTheme(): string {
  return document.documentElement.classList.contains("dark")
    ? "erdn-dark"
    : "erdn-light";
}

onMounted(async () => {
  // ── Worker environment ──────────────────────────────────────────────────
  // Must be set before Monaco is imported so that it can locate the editor
  // worker.  We reference a local shim file (relative path) so that Vite
  // can statically analyze and bundle the worker asset.
  if (!(window as any).MonacoEnvironment) {
    (window as any).MonacoEnvironment = {
      getWorker() {
        return new Worker(
          new URL("./editor.worker.ts", import.meta.url),
          { type: "module" }
        );
      },
    };
  }

  // ── Dynamic import (keeps Monaco out of the SSR bundle) ─────────────────
  const [monaco, { registerErdnLanguage }] = await Promise.all([
    import("monaco-editor"),
    import("./erdn-language"),
  ]);

  registerErdnLanguage(monaco);

  if (!containerRef.value) return;

  // ── Create editor ────────────────────────────────────────────────────────
  editorInstance = monaco.editor.create(containerRef.value, {
    value: props.modelValue,
    language: "erdn",
    theme: currentTheme(),
    minimap: { enabled: false },
    scrollBeyondLastLine: false,
    fontSize: 13.5,
    lineHeight: 22,
    // Use the same broad mono stack VitePress ships with.
    fontFamily:
      "ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace",
    wordWrap: "on",
    overviewRulerLanes: 0,
    lineNumbers: "on",
    glyphMargin: false,
    folding: false,
    lineNumbersMinChars: 3,
    scrollbar: {
      verticalScrollbarSize: 8,
      horizontalScrollbarSize: 8,
    },
    // Reflows automatically when the container is resized (e.g. mobile
    // layout, window resize).
    automaticLayout: true,
    padding: { top: 14, bottom: 14 },
    renderLineHighlight: "none",
  });

  // ── Sync editor → parent ─────────────────────────────────────────────────
  editorInstance.onDidChangeModelContent(() => {
    if (suppressEvent) return;
    emit("update:modelValue", editorInstance!.getValue());
  });

  // ── Sync VitePress dark-mode toggle → Monaco theme ───────────────────────
  themeObserver = new MutationObserver(() => {
    monaco.editor.setTheme(currentTheme());
  });
  themeObserver.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ["class"],
  });
});

// ── Sync parent → editor ────────────────────────────────────────────────────
watch(
  () => props.modelValue,
  (newVal) => {
    if (editorInstance && editorInstance.getValue() !== newVal) {
      suppressEvent = true;
      editorInstance.setValue(newVal);
      suppressEvent = false;
    }
  }
);

onUnmounted(() => {
  themeObserver?.disconnect();
  editorInstance?.dispose();
});
</script>

<style scoped>
.monaco-editor-container {
  width: 100%;
  height: 100%;
}
</style>
