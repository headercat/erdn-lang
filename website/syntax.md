# Syntax Specification

This page is the complete reference for the **erdn-lang** schema language. It covers every construct, keyword, and rule recognized by the parser and semantic validator.

## File Structure

An `.erdn` file consists of three kinds of top-level constructs, which may appear in any order:

- **Comments** — `#` (hash) or `//` (line) comments.
- **Table definitions** — `table <name> ( … )`.
- **Link declarations** — `link <card> <table>.<column> to <card> <table>.<column>`.

Whitespace (spaces, tabs) is ignored between tokens. Newlines separate columns within a table body and delimit comments.

## Comments

Two comment styles are supported:

| Style        | Syntax    | Rendered in diagram? |
| ------------ | --------- | -------------------- |
| Hash comment | `# text`  | ✅ Yes — rendered as a subtitle row |
| Line comment | `// text` | ❌ No — ignored by the renderer |

### Hash Comments (`#`)

Hash comments are **semantic**: their text is captured into the AST and rendered as subtitle/annotation rows in the SVG output.

- A `#` comment immediately before a `table` declaration becomes the **table subtitle**.
- A `#` comment immediately before a column definition becomes the **column annotation**.
- A `#` comment immediately before a `link` declaration becomes the **link badge** displayed on the connector.

```erdn
# This text appears as a subtitle in the diagram
table users (
  # This text annotates the column below
  id bigint primary-key auto-increment
)
```

Multiple consecutive `#` comments are collected and associated with the next construct.

### Line Comments (`//`)

Line comments are purely for the developer. They are discarded during parsing and have no effect on the output.

```erdn
// This is a developer note — invisible in the diagram
table users (
  id bigint primary-key
)
```

## Tables

### Grammar

```
[# comment ...]
table <name> (
  [column ...]
)
```

Or equivalently with braces:

```
[# comment ...]
table <name> {
  [column ...]
}
```

Both `( )` and `{ }` delimiters are accepted and behave identically.

### Table Name

`<name>` is an **identifier**: a sequence of letters (Unicode), digits, and underscores (`_`), starting with a letter or underscore.

```
table users ( … )
table post_tags ( … )
table カテゴリ ( … )
```

### Semantic Rules

- Table names must be **unique** within a file. Duplicate table names produce a validation error.
- A table may contain **zero or more** columns.

## Columns

### Grammar

```
[# comment ...]
<name> <type>[(<param>, …)] [modifier ...]
```

Each column occupies a single line within a table body (delimited by newlines).

### Column Name

Same rules as table names: an identifier of letters, digits, and underscores.

### Column Type

`<type>` is an identifier, for example: `bigint`, `varchar`, `text`, `timestamp`, `bool`, `int`, `char`, `decimal`.

The type is followed by optional **type parameters** in parentheses:

```
varchar(255)
char(2)
decimal(10,2)
```

Type parameters can be:

- **Numbers** — e.g. `255`, `10`.
- **Identifiers** — e.g. `max`.
- **Strings** — e.g. `"utf8"`.

Multiple parameters are separated by commas.

### Semantic Rules

- Column names must be **unique** within the same table.

## Column Modifiers

Zero or more modifiers may follow the type (and type parameters) on the same line.

| Modifier         | Keyword            | Description                                              |
| ---------------- | ------------------ | -------------------------------------------------------- |
| Primary key      | `primary-key`      | Marks the column as the table's primary key.             |
| Auto-increment   | `auto-increment`   | Column value is auto-incremented by the database.        |
| Not null         | `not-null`         | Column must not contain NULL values.                     |
| Nullable         | `nullable`         | Column explicitly allows NULL values.                    |
| Indexed          | `indexed`          | Column has a database index.                             |
| Default value    | `default(<value>)` | Specifies a default value expression.                    |

### `default` Modifier

The `default` modifier requires a parenthesized value:

```
default("NOW()")
default("draft")
default(0)
default(active)
```

The value inside parentheses may be a **string literal** (double-quoted), a **number**, or an **identifier**.

### Modifier Ordering

Modifiers may appear in any order on the column line:

```
id bigint primary-key auto-increment
email varchar(255) not-null indexed
status varchar(32) not-null default("draft")
```

### Semantic Rules

- **At most one** `primary-key` modifier per table. Multiple primary keys in the same table produce a validation error.
- `nullable` and `not-null` are **mutually exclusive** on the same column. Using both produces a validation error.
- Duplicate modifiers of the same kind on a single column are allowed syntactically but redundant.

## Links

### Grammar

```
[# comment ...]
link <from-cardinality> <table>.<column> to <to-cardinality> <table>.<column>
```

### Cardinality

Each side of a link specifies a cardinality, which is one of two keywords:

| Keyword | Meaning                  |
| ------- | ------------------------ |
| `one`   | Exactly one (1)          |
| `many`  | Zero or more (many / N)  |

### Table.Column Reference

Each side of a link references a specific column in a specific table using dot notation:

```
<table_name>.<column_name>
```

### Examples

```erdn
# One author writes many posts
link one authors.id to many posts.author_id

# Self-referential: categories can have sub-categories
link one categories.id to many categories.parent_id
```

### Semantic Rules

- The referenced **table** must exist in the schema. An unknown table name produces a validation error.
- The referenced **column** must exist in the referenced table. An unknown column name produces a validation error.
- **Self-referential links** (a table linking to itself) are supported.
- Links may be declared **before or after** the tables they reference — declaration order does not matter.

## String Literals

String literals are delimited by double quotes (`"`). The following escape sequences are recognized:

| Escape   | Character        |
| -------- | ---------------- |
| `\\`     | Backslash (`\`)  |
| `\"`     | Double quote     |
| `\n`     | Newline          |
| `\t`     | Tab              |

An unterminated string literal (missing closing `"`) produces a parse error.

## Identifiers

An **identifier** is a sequence of:

- Unicode letters
- Digits (`0`–`9`)
- Underscores (`_`)

starting with a letter or underscore. Identifiers are used for table names, column names, and type names.

Identifiers are **case-sensitive**: `Users` and `users` are different names.

## Reserved Keywords

The following words are reserved and cannot be used as identifiers:

| Keyword          | Context           |
| ---------------- | ----------------- |
| `table`          | Table definition  |
| `link`           | Link declaration  |
| `one`            | Cardinality       |
| `many`           | Cardinality       |
| `to`             | Link syntax       |
| `primary-key`    | Column modifier   |
| `auto-increment` | Column modifier   |
| `not-null`       | Column modifier   |
| `nullable`       | Column modifier   |
| `indexed`        | Column modifier   |
| `default`        | Column modifier   |

## Full Example

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
  created_at timestamp   not-null default("NOW()")
)

# An author can write many posts
link one authors.id to many posts.author_id
```

## Validation Error Summary

| Error                                | Cause                                                  |
| ------------------------------------ | ------------------------------------------------------ |
| Duplicate table name                 | Two tables share the same name.                        |
| Duplicate column name                | Two columns in the same table share the same name.     |
| Multiple primary keys                | A table has `primary-key` on more than one column.     |
| Nullable and not-null conflict       | A column has both `nullable` and `not-null` modifiers. |
| Unknown table in link                | A link references a table that does not exist.         |
| Unknown column in link               | A link references a column not present in the table.   |
