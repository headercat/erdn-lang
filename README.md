# erdn-lang

**erdn-lang** is a lightweight, deterministic domain-specific language for describing Entity-Relationship Diagrams (ERDs). Write a plain-text `.erdn` schema file and render it into a clean, self-contained SVG diagram — no GUI required.

Maintaining ERD diagrams is tedious: graphical tools produce binary files that are hard to diff, and hand-rolled diagrams drift out of sync with real schemas. **erdn-lang** solves this by letting you describe your schema as code:

- **Text-based** — store `.erdn` files alongside your source code and track changes in version control.
- **Deterministic** — the same input always produces the same SVG output.
- **Validated** — the `validate` command catches semantic errors (unknown tables, duplicate keys, conflicting modifiers) before you render.
- **Readable** — `#` comments on tables, columns, and links are rendered as subtitle rows directly in the diagram.

---

## Table of Contents

- [Installation](#installation)
- [Usage](#usage)
- [Syntax](#syntax)
  - [Comments](#comments)
  - [Tables](#tables)
  - [Columns](#columns)
  - [Column Modifiers](#column-modifiers)
  - [Links](#links)
  - [Full Example](#full-example)
- [Contributing](#contributing)
- [License](#license)

---

## Installation

You need [Go 1.21](https://go.dev/dl/) or later.

### Install via `go install`

```sh
go install github.com/headercat/erdn-lang/cmd/erdn@latest
```

### Build from source

```sh
git clone https://github.com/headercat/erdn-lang.git
cd erdn-lang
go build -o erdn ./cmd/erdn
```

---

## Usage

```
erdn render   <schema.erdn> [--out <file.svg>]
erdn validate <schema.erdn>
```

### `render`

Parses and validates the schema, then writes an SVG diagram.

```sh
# Output written to schema.svg by default
erdn render schema.erdn

# Specify a custom output path
erdn render schema.erdn --out diagrams/schema.svg
```

### `validate`

Checks the schema for parse and semantic errors without producing any output file. Exits with a non-zero status code if errors are found.

```sh
erdn validate schema.erdn
# OK
```

---

## Syntax

An erdn-lang file is made up of **comments**, **table** definitions, and **link** declarations. Order does not matter — links may be declared before the tables they reference.

### Comments

Two comment styles are supported:

| Style | Syntax | Rendered in diagram? |
|-------|--------|----------------------|
| Hash comment | `# text` | ✅ Yes — shown as a subtitle row |
| Line comment | `// text` | ❌ No — ignored by the renderer |

```erdn
# This comment appears in the SVG as a subtitle
// This comment is invisible in the diagram
```

### Tables

```
table <name> (
  <columns…>
)
```

A `#` comment placed immediately before the opening parenthesis (or as the first line inside the body) becomes the table's subtitle in the diagram.

```erdn
# Stores registered user accounts
table users (
  id   bigint primary-key auto-increment
  name varchar(255) not-null
)
```

### Columns

```
<name> <type>[(<param>, …)] [modifiers…]
```

- **name** — any identifier (letters, digits, `_`).
- **type** — any identifier, e.g. `bigint`, `varchar`, `text`, `timestamp`, `bool`, `int`, `char`.
- **type params** — optional comma-separated values in parentheses, e.g. `varchar(255)`, `char(2)`.

```erdn
id         bigint       primary-key auto-increment
username   varchar(64)  not-null indexed
bio        text         nullable
created_at timestamp    not-null default(NOW())
```

A `#` comment on the line immediately above a column is rendered as the column's annotation in the diagram.

### Column Modifiers

| Modifier | Description |
|----------|-------------|
| `primary-key` | Marks the column as the primary key (at most one per table). |
| `auto-increment` | Column value is auto-incremented by the database. |
| `not-null` | Column must not contain NULL values. |
| `nullable` | Column explicitly allows NULL values. |
| `indexed` | Column has a database index. |
| `default(<value>)` | Specifies a default value expression. |

> `nullable` and `not-null` are mutually exclusive on the same column.

### Links

Links declare a directed relationship between two columns in (possibly different) tables.

```
[# comment]
link <from-cardinality> <table>.<column> to <to-cardinality> <table>.<column>
```

- **Cardinality** is either `one` or `many`.
- The source side (`from`) is typically `one`; the destination side (`to`) is typically `many`.
- Self-referential links (a table linking to itself) are supported.
- Links may be placed anywhere in the file — before or after the tables they reference.

```erdn
# One author writes many posts
link one authors.id to many posts.author_id

# Self-referential: a category can have sub-categories
link one categories.id to many categories.parent_id
```

### Full Example

```erdn
# A simple blog schema

table authors (
  # Unique identifier for each author
  id       bigint      primary-key auto-increment
  username varchar(64) not-null indexed
  email    varchar(255) not-null indexed
  bio      text        nullable
)

table posts (
  # One row per published article
  id        bigint       primary-key auto-increment
  author_id bigint       not-null indexed
  title     varchar(512) not-null
  body      text         not-null
  # draft, published, archived
  status    varchar(32)  not-null default("draft")
  created_at timestamp   not-null default(NOW())
)

# An author can write many posts
link one authors.id to many posts.author_id
```

Render it:

```sh
erdn render blog.erdn
# rendered blog.svg
```

More examples are available in the [`examples/`](examples/) directory.

---

## Contributing

Contributions are welcome! To get started:

1. Fork the repository and create a feature branch.
2. Make your changes and add or update tests where appropriate.
3. Run `go test ./...` to verify everything passes.
4. Open a pull request describing what you changed and why.

Please keep pull requests focused — one feature or fix per PR.

---

## License

This project is licensed under the [MIT License](LICENSE).

Copyright (c) 2026 Headercat Inc.
