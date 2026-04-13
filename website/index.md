---
layout: home

hero:
  name: erdn-lang
  text: Entity-Relationship Diagrams as Code
  tagline: Write plain-text .erdn schemas, render clean SVG diagrams. Deterministic, version-control friendly, and validated.
  actions:
    - theme: brand
      text: Try the Playground
      link: /playground
    - theme: alt
      text: View on GitHub
      link: https://github.com/headercat/erdn-lang

features:
  - icon: 📝
    title: Text-Based
    details: Store .erdn files alongside your source code and track changes in version control — no more opaque binary diagram files.
  - icon: 🔒
    title: Deterministic
    details: The same input always produces the same SVG output. Diffs are meaningful, reviews are easy.
  - icon: ✅
    title: Validated
    details: Catch semantic errors — unknown tables, duplicate keys, conflicting modifiers — before you render.
  - icon: 💬
    title: Readable
    details: "# comments on tables, columns, and links are rendered as subtitle rows directly in the diagram."
---

## Quick Start

```sh
# Install
go install github.com/headercat/erdn-lang/cmd/erdn@latest

# Render a schema
erdn render schema.erdn
```

## Example

```erdn
table users (
  id bigint primary-key auto-increment
  username varchar(255) not-null indexed
  email varchar(255) not-null indexed
)

table posts (
  id bigint primary-key auto-increment
  title varchar(255) not-null
  body text nullable
  author_id bigint not-null indexed
)

# User has many posts
link one users.id to many posts.author_id
```

Run `erdn render schema.erdn` to produce a clean SVG diagram, or try it live in the [Playground](/playground).

## Syntax Overview

### Tables

```
table name (
  column type [modifiers...]
)
```

### Links

```
link one table.col to many table.col
```

### Comments

| Style        | Syntax    | Rendered in diagram? |
| ------------ | --------- | -------------------- |
| Hash comment | `# text`  | ✅ Yes               |
| Line comment | `// text` | ❌ No                |

### Column Modifiers

| Modifier         | Description                                     |
| ---------------- | ----------------------------------------------- |
| `primary-key`    | Marks the column as the primary key             |
| `auto-increment` | Column value is auto-incremented by the database |
| `not-null`       | Column must not contain NULL values             |
| `nullable`       | Column explicitly allows NULL values            |
| `indexed`        | Column has a database index                     |
| `default("val")` | Specifies a default value expression            |
