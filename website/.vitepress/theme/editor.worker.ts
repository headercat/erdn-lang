// Re-export Monaco's editor worker so it can be referenced via a
// relative-path new URL(), which Vite can statically analyze and bundle.
import "monaco-editor/esm/vs/editor/editor.worker";
