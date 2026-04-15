# Guide

This guide covers how to install **erdn-lang**, use the CLI, and integrate it into your workflow.

## Prerequisites

You need [Go 1.21](https://go.dev/dl/) or later installed on your system.

## Installation

### Install via `go install`

The fastest way to get started:

```sh
go install github.com/headercat/erdn-lang/cmd/erdn@latest
```

This places the `erdn` binary in your `$GOPATH/bin` (or `$HOME/go/bin` by default). Make sure this directory is in your `PATH`.

### Build from source

```sh
git clone https://github.com/headercat/erdn-lang.git
cd erdn-lang
go build -o erdn ./cmd/erdn
```

You can move the resulting `erdn` binary anywhere on your `PATH`.

## Usage

```
erdn render   <schema.erdn> [--out <file.svg>]
erdn validate <schema.erdn>
erdn sql      <schema.erdn> [--dbms <mysql|postgresql|mssql|oracle|sqlite>] [--out <file.sql>]
```

### `render`

Parses and validates the schema, then writes an SVG diagram.

```sh
# Output defaults to <schema>.svg
erdn render schema.erdn

# Specify a custom output path
erdn render schema.erdn --out diagrams/schema.svg
```

The command:

1. Reads and lexes the `.erdn` source file.
2. Parses it into an internal AST.
3. Runs semantic validation (catches unknown tables, duplicate columns, etc.).
4. Renders the validated AST to a self-contained SVG file.

### `validate`

Checks the schema for parse and semantic errors without producing any output file. Exits with a non-zero status code if errors are found.

```sh
erdn validate schema.erdn
# OK
```

Use `validate` in CI pipelines to catch schema errors early.

### `sql`

Generates SQL DDL from the schema — `CREATE TABLE` statements, indexes, and foreign key constraints. Use `--dbms` to target a specific database engine (default: `mysql`).

| DBMS | Flag value |
|------|-----------|
| MySQL | `mysql` |
| PostgreSQL | `postgresql` |
| Microsoft SQL Server | `mssql` |
| Oracle Database | `oracle` |
| SQLite | `sqlite` |

```sh
# Output defaults to <schema>.sql (MySQL)
erdn sql schema.erdn

# Target PostgreSQL
erdn sql schema.erdn --dbms postgresql

# Specify a custom output path
erdn sql schema.erdn --dbms mssql --out migrations/001_init.sql
```

The generated SQL includes:

- `CREATE TABLE` statements with DBMS-appropriate column types and constraints.
- `PRIMARY KEY` constraints.
- Auto-increment syntax suited to the target DBMS (`AUTO_INCREMENT`, `IDENTITY(1,1)`, `GENERATED ALWAYS AS IDENTITY`, or `AUTOINCREMENT`).
- `CREATE INDEX` statements for columns marked `indexed`.
- Foreign key constraints derived from `link` declarations — as `ALTER TABLE … ADD CONSTRAINT … FOREIGN KEY` for most databases, or as inline `FOREIGN KEY` table constraints for SQLite.

## Writing Your First Schema

Create a file called `blog.erdn`:

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

## Workflow Tips

- **Version control** — commit `.erdn` files alongside your application code. Diffs are human-readable.
- **CI integration** — add `erdn validate` to your CI pipeline to prevent broken schemas from being merged.
- **Automated rendering** — use `erdn render` in CI to produce SVG artifacts for every commit.
- **Playground** — use the online [Playground](/playground) to experiment without installing anything.

## Next Steps

- Read the full [Syntax Specification](/syntax) for every language construct.
- Try the live [Playground](/playground) in your browser.
- Explore ready-to-use [Recipes](/recipes) for GitHub Actions and other integrations.
