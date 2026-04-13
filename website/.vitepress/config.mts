import { defineConfig } from "vitepress";

export default defineConfig({
  base: "/erdn-lang/",
  title: "erdn-lang",
  description: "Entity-Relationship Diagrams as Code",
  head: [["meta", { name: "theme-color", content: "#58a6ff" }]],
  themeConfig: {
    nav: [
      { text: "Home", link: "/" },
      { text: "Guide", link: "/guide" },
      { text: "Syntax Specification", link: "/syntax" },
      { text: "Playground", link: "/playground" },
    ],
    sidebar: [
      {
        text: "Getting Started",
        items: [
          { text: "Guide", link: "/guide" },
          { text: "Syntax Specification", link: "/syntax" },
        ],
      },
      {
        text: "Tools",
        items: [{ text: "Playground", link: "/playground" }],
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
    server: {
      headers: {
        "Cross-Origin-Opener-Policy": "same-origin",
        "Cross-Origin-Embedder-Policy": "require-corp",
      },
    },
  },
});
