import type * as Monaco from "monaco-editor";

/**
 * Register the "erdn" language with Monaco.
 * Safe to call multiple times – re-registration is skipped after the first.
 */
export function registerErdnLanguage(monaco: typeof Monaco): void {
  if (monaco.languages.getLanguages().some((l) => l.id === "erdn")) {
    return;
  }

  monaco.languages.register({ id: "erdn", extensions: [".erdn"] });

  // ── Monarch tokenizer ───────────────────────────────────────────────────
  monaco.languages.setMonarchTokensProvider("erdn", {
    tokenizer: {
      root: [
        // Line comments (// — discarded by parser, not rendered in diagram)
        [/\/\/.*/, "comment"],

        // Hash comments (# — rendered as subtitle / annotation in diagram)
        [/#.*/, "comment.doc"],

        // Column modifiers – must be matched before bare identifiers because
        // some modifiers start with a common word (e.g. "not" in "not-null").
        [
          /\b(primary-key|auto-increment|not-null|nullable|indexed|default)\b/,
          "keyword.modifier",
        ],

        // Top-level keywords
        [/\b(table|link|one|many|to)\b/, "keyword"],

        // String literals (double-quoted, with escape sequences)
        [/"/, { token: "string.quote", bracket: "@open", next: "@string_lit" }],

        // Numeric literals (integers and decimals)
        [/\d+(\.\d+)?/, "number"],

        // Identifiers: table names, column names, SQL type names
        [/[a-zA-Z_]\w*/, "identifier"],

        // Structural punctuation
        [/[(){}]/, "delimiter"],
        [/\./, "delimiter.dot"],
        [/,/, "delimiter"],
      ],

      string_lit: [
        [/[^\\"]+/, "string"],
        [/\\./, "string.escape"],
        [/"/, { token: "string.quote", bracket: "@close", next: "@pop" }],
      ],
    },
  } as Monaco.languages.IMonarchLanguage);

  // ── Language configuration (brackets, auto-closing, comments) ──────────
  monaco.languages.setLanguageConfiguration("erdn", {
    comments: {
      lineComment: "//",
    },
    brackets: [
      ["(", ")"],
      ["{", "}"],
    ],
    autoClosingPairs: [
      { open: "(", close: ")" },
      { open: "{", close: "}" },
      { open: '"', close: '"', notIn: ["string"] },
    ],
    surroundingPairs: [
      { open: "(", close: ")" },
      { open: "{", close: "}" },
      { open: '"', close: '"' },
    ],
  });

  // ── Custom themes ───────────────────────────────────────────────────────

  monaco.editor.defineTheme("erdn-light", {
    base: "vs",
    inherit: true,
    rules: [
      { token: "keyword", foreground: "0070c1", fontStyle: "bold" },
      { token: "keyword.modifier", foreground: "811f3f", fontStyle: "bold" },
      { token: "comment", foreground: "6a9955" },
      { token: "comment.doc", foreground: "007a6e", fontStyle: "bold" },
      { token: "string", foreground: "a31515" },
      { token: "string.quote", foreground: "a31515" },
      { token: "string.escape", foreground: "ff0000" },
      { token: "number", foreground: "098658" },
      { token: "identifier", foreground: "1f1f1f" },
      { token: "delimiter", foreground: "808080" },
      { token: "delimiter.dot", foreground: "808080" },
    ],
    colors: {
      "editor.background": "#f6f8fa",
    },
  });

  monaco.editor.defineTheme("erdn-dark", {
    base: "vs-dark",
    inherit: true,
    rules: [
      { token: "keyword", foreground: "569cd6", fontStyle: "bold" },
      { token: "keyword.modifier", foreground: "c586c0", fontStyle: "bold" },
      { token: "comment", foreground: "6a9955" },
      { token: "comment.doc", foreground: "4ec9b0", fontStyle: "bold" },
      { token: "string", foreground: "ce9178" },
      { token: "string.quote", foreground: "ce9178" },
      { token: "string.escape", foreground: "d7ba7d" },
      { token: "number", foreground: "b5cea8" },
      { token: "identifier", foreground: "d4d4d4" },
      { token: "delimiter", foreground: "858585" },
      { token: "delimiter.dot", foreground: "858585" },
    ],
    colors: {
      "editor.background": "#161618",
    },
  });
}
