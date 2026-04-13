import { defineConfig } from "vitepress";

export default defineConfig({
  title: "erdn-lang",
  description: "Entity-Relationship Diagrams as Code",
  head: [
    ["meta", { name: "theme-color", content: "#58a6ff" }],
  ],
  themeConfig: {
    nav: [
      { text: "Home", link: "/" },
      { text: "Playground", link: "/playground" },
      {
        text: "GitHub",
        link: "https://github.com/headercat/erdn-lang",
      },
    ],
    socialLinks: [
      { icon: "github", link: "https://github.com/headercat/erdn-lang" },
    ],
    footer: {
      message: "Released under the MIT License.",
      copyright: "Copyright © 2026 Headercat Inc.",
    },
  },
  vite: {
    // Allow loading .wasm files from public/
    server: {
      headers: {
        "Cross-Origin-Opener-Policy": "same-origin",
        "Cross-Origin-Embedder-Policy": "require-corp",
      },
    },
  },
});
