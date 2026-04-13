<template>
  <div class="pg">
    <div class="pg-toolbar">
      <div class="pg-status-wrap">
        <span class="pg-dot" :class="statusClass"></span>
        <span class="pg-status-text" :class="statusClass">{{ statusText }}</span>
      </div>
      <select class="pg-example-select" v-model="selectedExample" @change="onExampleChange">
        <option value="" disabled>Load Example…</option>
        <option v-for="ex in EXAMPLES" :key="ex.name" :value="ex.name">{{ ex.name }}</option>
      </select>
    </div>
    <div class="pg-body">
      <div class="pg-pane pg-editor-pane">
        <div class="pg-pane-header">
          <span>ERDN Source</span>
        </div>
        <textarea
          ref="editorRef"
          v-model="source"
          class="pg-editor"
          spellcheck="false"
          autocomplete="off"
          autocorrect="off"
          autocapitalize="off"
          placeholder="Write your ERDN schema here…"
          @input="onInput"
          @keydown="onKeyDown"
        ></textarea>
      </div>
      <div class="pg-pane pg-preview-pane">
        <div class="pg-pane-header">
          <span>ERD Preview</span>
          <div class="pg-preview-actions">
            <div class="pg-zoom" v-if="hasSvg">
              <button class="pg-zoom-btn" @click="zoomOut" title="Zoom out">−</button>
              <button class="pg-zoom-btn pg-zoom-label" @click="fitZoom" :title="`${Math.round(zoom * 100)}% — click to fit`">{{ Math.round(zoom * 100) }}%</button>
              <button class="pg-zoom-btn" @click="zoomIn" title="Zoom in">+</button>
            </div>
            <button
              v-if="hasSvg"
              class="pg-zoom-btn pg-download-btn"
              @click="downloadSvg"
              title="Download SVG"
            >↓ SVG</button>
          </div>
        </div>
        <div
          class="pg-canvas"
          ref="previewRef"
          :class="{ 'pg-canvas--panning': isPanning }"
          @pointerdown="onPanStart"
          @pointermove="onPanMove"
          @pointerup="onPanEnd"
          @pointercancel="onPanEnd"
        >
          <div class="pg-svg-wrap" :style="{ zoom: zoom }" v-html="previewHtml"></div>
        </div>
        <div v-if="errorText" class="pg-error">{{ errorText }}</div>
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

const EXAMPLES = [
  {
    name: "Blog",
    source: `# A simple blog schema

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
link one authors.id to many posts.author_id`,
  },
  {
    name: "E-Commerce",
    source: `# E-commerce schema

table users (
  id bigint primary-key auto-increment
  email varchar(255) not-null indexed
  name varchar(128) not-null
  created_at timestamp not-null default("NOW()")
)

table products (
  id bigint primary-key auto-increment
  name varchar(256) not-null
  description text nullable
  price decimal(10,2) not-null
  stock_qty int not-null default("0")
)

table orders (
  id bigint primary-key auto-increment
  user_id bigint not-null indexed
  # pending, paid, shipped, delivered
  status varchar(32) not-null default("pending")
  total decimal(10,2) not-null
  created_at timestamp not-null default("NOW()")
)

table order_items (
  id bigint primary-key auto-increment
  order_id bigint not-null indexed
  product_id bigint not-null indexed
  quantity int not-null
  unit_price decimal(10,2) not-null
)

# One user can place many orders
link one users.id to many orders.user_id
# One order contains many line items
link one orders.id to many order_items.order_id
# One product can appear in many order items
link one products.id to many order_items.product_id`,
  },
  {
    name: "Library",
    source: `# Library management schema

table members (
  id bigint primary-key auto-increment
  name varchar(128) not-null
  email varchar(255) not-null indexed
  joined_at timestamp not-null default("NOW()")
)

table authors (
  id bigint primary-key auto-increment
  name varchar(128) not-null
  bio text nullable
)

table books (
  id bigint primary-key auto-increment
  author_id bigint not-null indexed
  title varchar(512) not-null
  isbn varchar(20) not-null indexed
  published_year int nullable
)

table loans (
  id bigint primary-key auto-increment
  book_id bigint not-null indexed
  member_id bigint not-null indexed
  loaned_at timestamp not-null default("NOW()")
  due_at timestamp not-null
  returned_at timestamp nullable
)

# Each book is written by one author
link one authors.id to many books.author_id
# One book can be loaned many times
link one books.id to many loans.book_id
# One member can take many loans
link one members.id to many loans.member_id`,
  },
];

const source = ref(EXAMPLES[0].source);
const selectedExample = ref("");
const previewHtml = ref(
  '<p class="preview-placeholder">Your diagram will appear here.</p>'
);
const errorText = ref("");
const statusText = ref("Loading WASM…");
const statusClass = ref("");
const editorRef = ref<HTMLTextAreaElement | null>(null);
const previewRef = ref<HTMLDivElement | null>(null);
const zoom = ref(1);
const isPanning = ref(false);

const hasSvg = computed(() => previewHtml.value.includes("<svg"));

let wasmReady = false;
let debounceTimer: ReturnType<typeof setTimeout> | null = null;

// ── Zoom ─────────────────────────────────────────────────────────────────────

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
  const attrW = svgEl.getAttribute("width");
  const attrH = svgEl.getAttribute("height");
  const svgW = attrW ? parseFloat(attrW) : svgEl.viewBox.baseVal.width;
  const svgH = attrH ? parseFloat(attrH) : svgEl.viewBox.baseVal.height;
  if (!svgW || !svgH) return;
  const availW = container.clientWidth - 32;
  const availH = container.clientHeight - 32;
  const scale = Math.min(availW / svgW, availH / svgH);
  zoom.value = Math.max(ZOOM_MIN, Math.round(scale * 100) / 100);
}

// ── Drag-to-pan ───────────────────────────────────────────────────────────────

let panStartX = 0;
let panStartY = 0;
let panScrollLeft = 0;
let panScrollTop = 0;
let panMoved = false;

function onPanStart(e: PointerEvent) {
  if (e.button !== 0) return;
  const canvas = previewRef.value;
  if (!canvas) return;
  isPanning.value = true;
  panMoved = false;
  panStartX = e.clientX;
  panStartY = e.clientY;
  panScrollLeft = canvas.scrollLeft;
  panScrollTop = canvas.scrollTop;
  canvas.setPointerCapture(e.pointerId);
}

function onPanMove(e: PointerEvent) {
  if (!isPanning.value) return;
  const canvas = previewRef.value;
  if (!canvas) return;
  const dx = e.clientX - panStartX;
  const dy = e.clientY - panStartY;
  if (!panMoved && Math.abs(dx) < 4 && Math.abs(dy) < 4) return;
  panMoved = true;
  canvas.scrollLeft = panScrollLeft - dx;
  canvas.scrollTop = panScrollTop - dy;
}

function onPanEnd(e: PointerEvent) {
  if (!isPanning.value) return;
  isPanning.value = false;
  const canvas = previewRef.value;
  if (canvas) canvas.releasePointerCapture(e.pointerId);
}

// ── Compile ───────────────────────────────────────────────────────────────────

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
    requestAnimationFrame(() => {
      if (editorRef.value) {
        editorRef.value.selectionStart = editorRef.value.selectionEnd =
          start + 2;
      }
    });
    onInput();
  }
}

// ── Examples ──────────────────────────────────────────────────────────────────

function onExampleChange() {
  const ex = EXAMPLES.find((e) => e.name === selectedExample.value);
  selectedExample.value = "";
  if (!ex) return;
  source.value = ex.source;
  onInput();
}

// ── Download SVG ─────────────────────────────────────────────────────────────

function downloadSvg() {
  const svgEl = previewRef.value?.querySelector("svg");
  if (!svgEl) return;
  const svgContent = new XMLSerializer().serializeToString(svgEl);
  const blob = new Blob([svgContent], { type: "image/svg+xml" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = "diagram.svg";
  a.click();
  URL.revokeObjectURL(url);
}

// ── WASM bootstrap ────────────────────────────────────────────────────────────

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
/* ── Outer card ─────────────────────────────────────────────────────── */
.pg {
  display: flex;
  flex-direction: column;
  border: 1px solid var(--vp-c-divider);
  border-radius: 8px;
  overflow: hidden;
  /* Fill the viewport below the nav bar */
  height: calc(100vh - var(--vp-nav-height, 64px) - 32px);
  margin: 16px 24px;
}

/* ── Toolbar ─────────────────────────────────────────────────────────── */
.pg-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;
  background: var(--vp-c-bg-soft);
  border-bottom: 1px solid var(--vp-c-divider);
  gap: 8px;
  flex-shrink: 0;
}

.pg-status-wrap {
  display: flex;
  align-items: center;
  gap: 7px;
}

.pg-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--vp-c-text-3);
  flex-shrink: 0;
  transition: background 0.25s;
}

.pg-dot.ready {
  background: var(--vp-c-green-1);
}

.pg-dot.error {
  background: var(--vp-c-red-1);
}

.pg-status-text {
  font-size: 13px;
  color: var(--vp-c-text-2);
  transition: color 0.25s;
}

.pg-status-text.ready {
  color: var(--vp-c-green-1);
}

.pg-status-text.error {
  color: var(--vp-c-red-1);
}

.pg-example-select {
  font-size: 13px;
  padding: 4px 28px 4px 12px;
  border-radius: 20px;
  border: 1px solid var(--vp-c-divider);
  background: transparent;
  color: var(--vp-c-text-2);
  cursor: pointer;
  font-family: var(--vp-font-family-base);
  appearance: none;
  -webkit-appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='6' viewBox='0 0 10 6'%3E%3Cpath d='M0 0l5 6 5-6z' fill='%23888'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 10px center;
  transition: border-color 0.2s, color 0.2s;
}

.pg-example-select:hover {
  border-color: var(--vp-c-brand-1);
  color: var(--vp-c-brand-1);
}

.pg-example-select:focus {
  outline: none;
  border-color: var(--vp-c-brand-1);
}

/* ── Split panels ────────────────────────────────────────────────────── */
.pg-body {
  display: grid;
  /* minmax(0,1fr) overrides the default min-content minimum so SVG content
     cannot inflate the preview column at the editor's expense */
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  /* Explicit row prevents the implicit auto row from growing to SVG content height */
  grid-template-rows: 1fr;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.pg-pane {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.pg-editor-pane {
  border-right: 1px solid var(--vp-c-divider);
  /* Prevent the editor from collapsing when the SVG canvas is large */
  min-width: 280px;
}

/* ── Panel header (code-block tab style) ─────────────────────────────── */
.pg-pane-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 14px;
  background: var(--vp-code-block-bg);
  border-bottom: 1px solid var(--vp-c-divider);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--vp-c-text-3);
  font-family: var(--vp-font-family-base);
  flex-shrink: 0;
  overflow: hidden;
}

.pg-pane-header > span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.pg-preview-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  /* Never let zoom/download buttons be hidden when pane is narrow */
  flex-shrink: 0;
  margin-left: 8px;
}

/* ── Zoom controls ───────────────────────────────────────────────────── */
.pg-zoom {
  display: flex;
  align-items: center;
  gap: 2px;
}

.pg-zoom-btn {
  font-size: 12px;
  line-height: 1;
  padding: 2px 7px;
  border-radius: 4px;
  border: 1px solid var(--vp-c-divider);
  background: var(--vp-c-bg-mute);
  color: var(--vp-c-text-2);
  cursor: pointer;
  font-family: var(--vp-font-family-base);
  text-transform: none;
  letter-spacing: 0;
  font-weight: 500;
  transition: border-color 0.15s, color 0.15s;
}

.pg-zoom-btn:hover {
  border-color: var(--vp-c-brand-1);
  color: var(--vp-c-brand-1);
}

.pg-zoom-label {
  min-width: 46px;
  text-align: center;
}

.pg-download-btn {
  font-size: 11px;
  padding: 2px 8px;
}

/* ── Editor ──────────────────────────────────────────────────────────── */
.pg-editor {
  flex: 1;
  width: 100%;
  resize: none;
  border: none;
  outline: none;
  padding: 14px 16px;
  font-family: var(--vp-font-family-mono);
  font-size: 13.5px;
  line-height: 1.65;
  background: var(--vp-code-block-bg);
  color: var(--vp-c-text-1);
  tab-size: 2;
  min-height: 0;
}

.pg-editor::placeholder {
  color: var(--vp-c-text-3);
}

/* ── Preview canvas ──────────────────────────────────────────────────── */
.pg-canvas {
  flex: 1;
  overflow: auto;
  padding: 16px;
  /* White background is intentional: generated SVGs have a white canvas
     with dark strokes/text, so the preview pane must stay light. */
  background: #ffffff;
  min-height: 0;
  cursor: grab;
  user-select: none;
}

.pg-canvas--panning {
  cursor: grabbing;
}

.pg-svg-wrap {
  display: inline-block;
  line-height: 0;
  pointer-events: none;
}

.pg-canvas :deep(.preview-placeholder) {
  color: #9ca3af;
  font-size: 14px;
  line-height: 1.5;
  padding: 16px 0;
  cursor: default;
}

/* ── Error bar ───────────────────────────────────────────────────────── */
.pg-error {
  padding: 8px 14px;
  background: var(--vp-c-bg-soft);
  border-top: 2px solid var(--vp-c-red-1);
  color: var(--vp-c-red-1);
  font-family: var(--vp-font-family-mono);
  font-size: 13px;
  white-space: pre-wrap;
  max-height: 120px;
  overflow-y: auto;
  flex-shrink: 0;
}

/* ── Mobile ──────────────────────────────────────────────────────────── */
@media (max-width: 768px) {
  .pg {
    height: auto;
    margin: 8px 12px;
  }

  .pg-body {
    grid-template-columns: 1fr;
    /* Auto rows needed on mobile: height is auto so 1fr would be 0 */
    grid-template-rows: auto;
    overflow: visible;
  }

  .pg-editor-pane {
    border-right: none;
    border-bottom: 1px solid var(--vp-c-divider);
    min-height: 240px;
    max-height: 40vh;
  }

  .pg-preview-pane {
    min-height: 300px;
    max-height: 50vh;
  }
}
</style>
