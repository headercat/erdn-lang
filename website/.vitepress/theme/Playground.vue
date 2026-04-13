<template>
  <div class="playground-container">
    <div class="playground-toolbar">
      <span class="playground-status" :class="statusClass">{{ statusText }}</span>
      <button class="playground-btn" @click="loadExample">Load Example</button>
    </div>
    <div class="playground-panels">
      <div class="playground-editor-panel">
        <div class="panel-label">ERDN Source</div>
        <textarea
          ref="editorRef"
          v-model="source"
          class="playground-editor"
          spellcheck="false"
          autocomplete="off"
          autocorrect="off"
          autocapitalize="off"
          placeholder="Write your ERDN schema here…"
          @input="onInput"
          @keydown="onKeyDown"
        ></textarea>
      </div>
      <div class="playground-preview-panel">
        <div class="panel-label">
          SVG Preview
          <div class="zoom-controls" v-if="hasSvg">
            <button class="zoom-btn" @click="zoomOut" title="Zoom out">−</button>
            <button class="zoom-btn zoom-level" @click="fitZoom" :title="`${Math.round(zoom * 100)}% — click to fit`">{{ Math.round(zoom * 100) }}%</button>
            <button class="zoom-btn" @click="zoomIn" title="Zoom in">+</button>
          </div>
        </div>
        <div class="playground-preview" ref="previewRef">
          <div class="svg-scale-wrapper" :style="{ zoom: zoom }" v-html="previewHtml"></div>
        </div>
        <div v-if="errorText" class="playground-error">{{ errorText }}</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, onMounted, onUnmounted } from "vue";
import { withBase } from "vitepress";

const DEBOUNCE_MS = 300;
const ZOOM_STEP = 0.25;
const ZOOM_MIN = 0.1;
const ZOOM_MAX = 4;
const WASM_PATH = withBase("/erdn.wasm");
const WASM_EXEC_PATH = withBase("/wasm_exec.js");

const EXAMPLE = `# A simple blog schema

table authors (
  # Unique identifier
  id bigint primary-key auto-increment
  username varchar(64) not-null indexed
  email varchar(255) not-null indexed
  bio text nullable
)

table posts (
  # One row per article
  id bigint primary-key auto-increment
  author_id bigint not-null indexed
  title varchar(512) not-null
  body text not-null
  # draft, published, archived
  status varchar(32) not-null default("draft")
  created_at timestamp not-null default("NOW()")
)

# An author can write many posts
link one authors.id to many posts.author_id`;

const source = ref(EXAMPLE);
const previewHtml = ref(
  '<p class="preview-placeholder">Your diagram will appear here.</p>'
);
const errorText = ref("");
const statusText = ref("Loading WASM…");
const statusClass = ref("");
const editorRef = ref<HTMLTextAreaElement | null>(null);
const previewRef = ref<HTMLDivElement | null>(null);
const zoom = ref(1);

const hasSvg = computed(() => previewHtml.value.includes("<svg"));

let wasmReady = false;
let debounceTimer: ReturnType<typeof setTimeout> | null = null;

function zoomIn() {
  zoom.value = Math.min(Math.round((zoom.value + ZOOM_STEP) * 100) / 100, ZOOM_MAX);
}

function zoomOut() {
  zoom.value = Math.max(Math.round((zoom.value - ZOOM_STEP) * 100) / 100, ZOOM_MIN);
}

function fitZoom() {
  const container = previewRef.value;
  if (!container) return;
  const svgEl = container.querySelector("svg");
  if (!svgEl) {
    zoom.value = 1;
    return;
  }
  // Read natural SVG dimensions from attributes or viewBox
  const attrW = svgEl.getAttribute("width");
  const attrH = svgEl.getAttribute("height");
  const svgW = attrW ? parseFloat(attrW) : svgEl.viewBox.baseVal.width;
  const svgH = attrH ? parseFloat(attrH) : svgEl.viewBox.baseVal.height;
  if (!svgW || !svgH) return;
  // Available area (subtract 32px for 16px padding on each side)
  const availW = container.clientWidth - 32;
  const availH = container.clientHeight - 32;
  const scale = Math.min(availW / svgW, availH / svgH);
  zoom.value = Math.max(ZOOM_MIN, Math.round(scale * 100) / 100);
}

function compile() {
  if (!wasmReady) return;
  const src = source.value;
  if (!src.trim()) {
    previewHtml.value =
      '<p class="preview-placeholder">Your diagram will appear here.</p>';
    errorText.value = "";
    return;
  }
  try {
    const result = (window as any).compileToSVG(src);
    if (result.error) {
      errorText.value = result.error;
    } else {
      errorText.value = "";
      if (result.svg) {
        previewHtml.value = result.svg;
        nextTick(() => fitZoom());
      } else {
        previewHtml.value =
          '<p class="preview-placeholder">Your diagram will appear here.</p>';
      }
    }
  } catch (err: any) {
    errorText.value = "Compilation error: " + err.message;
  }
}

function onInput() {
  if (debounceTimer !== null) {
    clearTimeout(debounceTimer);
  }
  debounceTimer = setTimeout(() => {
    debounceTimer = null;
    compile();
  }, DEBOUNCE_MS);
}

function onKeyDown(e: KeyboardEvent) {
  if (e.key === "Tab") {
    e.preventDefault();
    const el = editorRef.value;
    if (!el) return;
    const start = el.selectionStart;
    const end = el.selectionEnd;
    const val = el.value;
    source.value = val.substring(0, start) + "  " + val.substring(end);
    // Restore cursor after Vue updates the textarea
    requestAnimationFrame(() => {
      if (editorRef.value) {
        editorRef.value.selectionStart = editorRef.value.selectionEnd =
          start + 2;
      }
    });
    onInput();
  }
}

function loadExample() {
  source.value = EXAMPLE;
  onInput();
}

function loadScript(src: string): Promise<void> {
  return new Promise((resolve, reject) => {
    const script = document.createElement("script");
    script.src = src;
    script.onload = () => resolve();
    script.onerror = () => reject(new Error("Failed to load " + src));
    document.head.appendChild(script);
  });
}

onMounted(async () => {
  try {
    await loadScript(WASM_EXEC_PATH);
    const go = new (window as any).Go();
    const result = await WebAssembly.instantiateStreaming(
      fetch(WASM_PATH),
      go.importObject
    );
    go.run(result.instance);
    wasmReady = true;
    statusText.value = "Ready";
    statusClass.value = "ready";
    compile();
  } catch (err: any) {
    statusText.value = "WASM load failed";
    statusClass.value = "error";
    errorText.value = "Failed to load WASM module: " + err.message;
  }
});

onUnmounted(() => {
  if (debounceTimer !== null) {
    clearTimeout(debounceTimer);
  }
});
</script>

<style scoped>
.playground-container {
  display: flex;
  flex-direction: column;
  margin: 0 -24px; /* break out of VitePress content padding */
}

.playground-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;
  border-bottom: 1px solid var(--vp-c-divider);
  background: var(--vp-c-bg-soft);
}

.playground-status {
  font-size: 13px;
  color: var(--vp-c-text-2);
  padding: 2px 10px;
  border-radius: 12px;
  background: var(--vp-c-bg-mute);
}

.playground-status.ready {
  color: var(--vp-c-green-1);
}

.playground-status.error {
  color: var(--vp-c-red-1);
}

.playground-btn {
  font-size: 13px;
  padding: 4px 12px;
  border-radius: 4px;
  border: 1px solid var(--vp-c-divider);
  background: var(--vp-c-bg-mute);
  color: var(--vp-c-text-2);
  cursor: pointer;
  font-family: var(--vp-font-family-base);
}

.playground-btn:hover {
  border-color: var(--vp-c-text-3);
  color: var(--vp-c-text-1);
}

.playground-panels {
  display: grid;
  grid-template-columns: 1fr 1fr;
  height: clamp(400px, calc(100vh - 220px), 720px);
}

.playground-editor-panel {
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--vp-c-divider);
  min-height: 0;
}

.playground-preview-panel {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.panel-label {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--vp-c-text-3);
  font-weight: 600;
  padding: 4px 12px;
  border-bottom: 1px solid var(--vp-c-divider);
  background: var(--vp-c-bg-soft);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.zoom-controls {
  display: flex;
  align-items: center;
  gap: 2px;
}

.zoom-btn {
  font-size: 13px;
  line-height: 1;
  padding: 2px 6px;
  border-radius: 3px;
  border: 1px solid var(--vp-c-divider);
  background: var(--vp-c-bg-mute);
  color: var(--vp-c-text-2);
  cursor: pointer;
  font-family: var(--vp-font-family-base);
  text-transform: none;
  letter-spacing: 0;
  font-weight: 400;
}

.zoom-btn:hover {
  border-color: var(--vp-c-text-3);
  color: var(--vp-c-text-1);
}

.zoom-level {
  min-width: 44px;
  text-align: center;
}

.playground-editor {
  flex: 1;
  width: 100%;
  resize: none;
  border: none;
  outline: none;
  padding: 12px;
  font-family: var(--vp-font-family-mono);
  font-size: 14px;
  line-height: 1.5;
  background: var(--vp-c-bg);
  color: var(--vp-c-text-1);
  tab-size: 2;
  min-height: 0;
}

.playground-editor::placeholder {
  color: var(--vp-c-text-3);
}

.playground-preview {
  flex: 1;
  overflow: auto;
  padding: 16px;
  /* White background is intentional: SVG diagrams render with dark
     strokes/text on a white canvas, so the preview must stay light
     regardless of VitePress dark/light mode. */
  background: #ffffff;
  min-height: 0;
}

.svg-scale-wrapper {
  display: inline-block;
  line-height: 0;
}

.playground-preview :deep(.preview-placeholder) {
  color: #6b7280;
  font-size: 14px;
  line-height: 1.5;
  padding: 16px 0;
}

.playground-error {
  padding: 8px 12px;
  background: var(--vp-c-bg-soft);
  border-top: 2px solid var(--vp-c-red-1);
  color: var(--vp-c-red-1);
  font-family: var(--vp-font-family-mono);
  font-size: 13px;
  white-space: pre-wrap;
  max-height: 120px;
  overflow-y: auto;
}

@media (max-width: 768px) {
  .playground-panels {
    grid-template-columns: 1fr;
    height: auto;
  }

  .playground-editor-panel {
    border-right: none;
    border-bottom: 1px solid var(--vp-c-divider);
    min-height: 280px;
    max-height: 45vh;
  }

  .playground-preview-panel {
    min-height: 280px;
    max-height: 45vh;
  }

  .playground-editor {
    min-height: 0;
  }

  .playground-preview {
    min-height: 0;
  }
}
</style>
