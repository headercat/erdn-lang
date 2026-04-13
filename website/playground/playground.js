// playground.js — ERDN playground with WASM-powered live preview.
// Uses debounced input handling to avoid compiling on every keystroke.

(function () {
  "use strict";

  var DEBOUNCE_MS = 300;
  var WASM_PATH = "../erdn.wasm";

  // Default example loaded into the editor.
  var EXAMPLE = [
    "# A simple blog schema",
    "",
    "table authors (",
    "  # Unique identifier",
    "  id bigint primary-key auto-increment",
    "  username varchar(64) not-null indexed",
    "  email varchar(255) not-null indexed",
    "  bio text nullable",
    ")",
    "",
    "table posts (",
    "  # One row per article",
    "  id bigint primary-key auto-increment",
    "  author_id bigint not-null indexed",
    "  title varchar(512) not-null",
    "  body text not-null",
    "  # draft, published, archived",
    "  status varchar(32) not-null default(\"draft\")",
    "  created_at timestamp not-null default(\"NOW()\")",
    ")",
    "",
    "# An author can write many posts",
    "link one authors.id to many posts.author_id",
  ].join("\n");

  // DOM elements.
  var editor = document.getElementById("editor");
  var preview = document.getElementById("preview");
  var errorBox = document.getElementById("error-box");
  var status = document.getElementById("status");
  var btnExample = document.getElementById("btn-example");

  var wasmReady = false;
  var debounceTimer = null;

  // ---- WASM loading ----

  function setStatus(text, className) {
    status.textContent = text;
    status.className = "playground-status" + (className ? " " + className : "");
  }

  async function loadWasm() {
    try {
      // wasm_exec.js is expected to set the global Go constructor.
      if (typeof Go === "undefined") {
        setStatus("Error: wasm_exec.js not loaded", "error");
        return;
      }
      var go = new Go();
      var result = await WebAssembly.instantiateStreaming(
        fetch(WASM_PATH),
        go.importObject
      );
      go.run(result.instance);
      wasmReady = true;
      setStatus("Ready", "ready");

      // Run initial compilation if editor has content.
      compile();
    } catch (err) {
      setStatus("WASM load failed", "error");
      showError("Failed to load WASM module: " + err.message);
    }
  }

  // ---- Compilation ----

  function compile() {
    if (!wasmReady) return;

    var src = editor.value;
    if (!src.trim()) {
      preview.innerHTML =
        '<p class="preview-placeholder">Your diagram will appear here.</p>';
      hideError();
      return;
    }

    try {
      var result = compileToSVG(src);
      if (result.error) {
        showError(result.error);
        // Keep the last good SVG visible; only clear if there's nothing yet.
        if (
          preview.querySelector("svg") === null &&
          !preview.querySelector(".preview-placeholder")
        ) {
          preview.innerHTML =
            '<p class="preview-placeholder">Fix errors to see preview.</p>';
        }
      } else {
        hideError();
        if (result.svg) {
          preview.innerHTML = result.svg;
        } else {
          preview.innerHTML =
            '<p class="preview-placeholder">Your diagram will appear here.</p>';
        }
      }
    } catch (err) {
      showError("Compilation error: " + err.message);
    }
  }

  // ---- Debounced input ----

  function onInput() {
    if (debounceTimer !== null) {
      clearTimeout(debounceTimer);
    }
    debounceTimer = setTimeout(function () {
      debounceTimer = null;
      compile();
    }, DEBOUNCE_MS);
  }

  // ---- Error display ----

  function showError(msg) {
    errorBox.textContent = msg;
    errorBox.hidden = false;
  }

  function hideError() {
    errorBox.textContent = "";
    errorBox.hidden = true;
  }

  // ---- Tab key support ----

  function onKeyDown(e) {
    if (e.key === "Tab") {
      e.preventDefault();
      var start = editor.selectionStart;
      var end = editor.selectionEnd;
      var value = editor.value;
      editor.value = value.substring(0, start) + "  " + value.substring(end);
      editor.selectionStart = editor.selectionEnd = start + 2;
      onInput();
    }
  }

  // ---- Load example ----

  function loadExample() {
    editor.value = EXAMPLE;
    onInput();
  }

  // ---- Initialize ----

  editor.addEventListener("input", onInput);
  editor.addEventListener("keydown", onKeyDown);
  btnExample.addEventListener("click", loadExample);

  // Load example on startup.
  editor.value = EXAMPLE;

  // Load wasm_exec.js dynamically, then initialize WASM.
  var script = document.createElement("script");
  script.src = "../wasm_exec.js";
  script.onload = function () {
    loadWasm();
  };
  script.onerror = function () {
    setStatus("Error: wasm_exec.js not found", "error");
  };
  document.head.appendChild(script);
})();
