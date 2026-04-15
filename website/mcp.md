# MCP Server

**erdn-lang** ships a built-in [Model Context Protocol](https://modelcontextprotocol.io/) (MCP) server so AI assistants and MCP-compatible editors can convert SQL schemas to ERDN and generate diagrams — no GUI, no clipboard, no copy-paste.

## Installation

Install the MCP server binary with `go install`:

```sh
go install github.com/headercat/erdn-lang/cmd/erdn-mcp@latest
```

You need [Go 1.21](https://go.dev/dl/) or later. The binary is placed in `$GOPATH/bin` (usually `$HOME/go/bin`). Make sure that directory is on your `PATH`.

## Client Configuration

### Using the installed binary

Add the following to your MCP client's configuration file (e.g. `claude_desktop_config.json`, `.cursor/mcp.json`, or VS Code's `settings.json`):

```json
{
  "mcpServers": {
    "erdn-lang": {
      "type": "stdio",
      "command": "erdn-mcp"
    }
  }
}
```

### Without installing (run directly from the module proxy)

If you prefer not to install a binary, you can run the server on demand via `go run`. No local clone is needed — Go fetches the package automatically:

```json
{
  "mcpServers": {
    "erdn-lang": {
      "type": "stdio",
      "command": "go",
      "args": ["run", "github.com/headercat/erdn-lang/cmd/erdn-mcp@latest"]
    }
  }
}
```

### Auto-discovery

The repository root contains a ready-to-use `.mcp.json` file. MCP clients that support auto-discovery (such as recent versions of Claude Desktop and Cursor) will pick it up automatically when you open the repository folder.

## Available Tools

### `convert_sql_to_erdn`

Converts SQL `CREATE TABLE` and `FOREIGN KEY` statements into ERDN source text.

**Input**

| Parameter | Type   | Description                         |
|-----------|--------|-------------------------------------|
| `sql`     | string | One or more SQL DDL statements      |

**Output** — The equivalent `.erdn` schema as a string.

**Example prompt**

> Convert this SQL schema to ERDN:
>
> ```sql
> CREATE TABLE users (id BIGINT PRIMARY KEY, username VARCHAR(64) NOT NULL);
> CREATE TABLE posts (id BIGINT PRIMARY KEY, author_id BIGINT, FOREIGN KEY (author_id) REFERENCES users(id));
> ```

---

### `generate_svg_from_erdn`

Validates ERDN source and returns the rendered SVG diagram as a string.

**Input**

| Parameter | Type   | Description          |
|-----------|--------|----------------------|
| `erdn`    | string | ERDN schema source   |

**Output** — A self-contained SVG string that can be saved to a file or embedded in HTML.

**Example prompt**

> Generate an SVG diagram for this ERDN schema:
>
> ```erdn
> table users (
>   id       bigint primary-key auto-increment
>   username varchar(64) not-null indexed
> )
> ```

## Running the Server Manually

If you have a local clone of the repository you can also launch the server directly:

```sh
go run ./cmd/erdn-mcp
```

The server communicates over **stdio** using JSON-RPC 2.0, which is the standard MCP transport.
