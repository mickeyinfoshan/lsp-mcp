# lsp-mcp

A bridge between the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) and the [Language Server Protocol (LSP)](https://microsoft.github.io/language-server-protocol/), giving AI coding assistants access to real code intelligence — go-to-definition, find references, hover docs, and completions — across multiple languages.

## Why?

AI assistants understand code by reading text. Language servers understand code by parsing it. **lsp-mcp** connects the two: it translates MCP tool calls into LSP requests, dispatching them to standard language servers (gopls, typescript-language-server, pylsp, etc.) running as child processes. The result is precise, compiler-grade code navigation available to any MCP-compatible agent.

## Features

- **Multi-language** — Go, TypeScript, JavaScript, Python out of the box; add any LSP-compliant server via config
- **Standard MCP protocol** — works with Claude Code, Cursor, Windsurf, and any MCP client
- **Session management** — isolated sessions keyed by (language, project root), with automatic lifecycle and cleanup
- **Document sync** — tracks open files and sends `didOpen`/`didChange`/`didClose` to keep language servers in sync
- **Cross-platform** — builds for Linux, macOS (Intel + Apple Silicon), and Windows

## Quick Start

### 1. Install language servers

```bash
# Go
go install golang.org/x/tools/gopls@latest

# TypeScript / JavaScript
npm install -g typescript-language-server typescript

# Python
pip install python-lsp-server
```

### 2. Build

```bash
git clone https://github.com/mickeyinfoshan/lsp-mcp.git
cd lsp-mcp
make build
```

The binary is output to `./bin/lsp-mcp`.

### 3. Configure

Edit `config/config.yaml` to match your environment (the defaults work for most setups):

```yaml
lsp_servers:
  go:
    command: "gopls"
    args: ["serve"]
    env:
      GOPATH: "~/go"
      GO111MODULE: "on"
  typescript:
    command: "typescript-language-server"
    args: ["--stdio"]
  python:
    command: "pylsp"
    args: []

mcp_server:
  name: "lsp-bridge"
  version: "1.0.0"

session:
  timeout: 300          # seconds before idle session cleanup
  max_sessions: 100

logging:
  level: "info"         # debug | info | warn | error
  format: "json"        # json | text
  file_output: true
  file_path: "./logs/lsp-mcp.log"
```

### 4. Add to your MCP client

**Claude Code** (`~/.claude/settings.json`):

```json
{
  "mcpServers": {
    "lsp": {
      "command": "/absolute/path/to/lsp-mcp",
      "args": ["-config", "/absolute/path/to/config.yaml"]
    }
  }
}
```

**Cursor / Windsurf / Generic MCP client** — point to the binary with the `-config` flag in your MCP server configuration, following the same pattern above.

## MCP Tools

| Tool | Description | Required Parameters | Optional |
|------|-------------|-------------------|----------|
| `lsp_definition` | Go to definition | `language_id`, `root_uri`, `file_uri`, `line`, `character` | `symbol` |
| `lsp_references` | Find all references | `language_id`, `root_uri`, `file_uri`, `line`, `character` | `include_declaration`, `symbol` |
| `lsp_hover` | Hover documentation / type info | `language_id`, `root_uri`, `file_uri`, `line`, `character` | `symbol` |
| `lsp_completion` | Code completions | `language_id`, `root_uri`, `file_uri`, `line`, `character` | `trigger_kind`, `trigger_character` |
| `lsp_shutdown` | Close an LSP session | `language_id`, `root_uri` | — |

**Parameter notes:**
- `line` and `character` are 0-based
- `root_uri` and `file_uri` use the `file://` scheme (e.g., `file:///Users/me/project`)
- `language_id` matches the config key: `go`, `typescript`, `javascript`, `python`
- Sessions are created automatically on the first tool call — no explicit initialize step needed

### Example

```json
{
  "language_id": "go",
  "root_uri": "file:///Users/me/myproject",
  "file_uri": "file:///Users/me/myproject/main.go",
  "line": 42,
  "character": 10,
  "symbol": "HandleRequest"
}
```

## Architecture

```
AI Assistant (MCP Client)
    │
    │  MCP JSON-RPC (stdio)
    ▼
┌─────────────────────────────────┐
│  MCP Server (internal/mcp/)     │  Tool registration & request routing
│    ├─ server.go                 │
│    └─ handlers.go               │  Param validation, response formatting
├─────────────────────────────────┤
│  Session Manager                │  Session lifecycle keyed by
│  (internal/session/)            │  (language_id + root_uri)
├─────────────────────────────────┤
│  LSP Client (internal/lsp/)     │  Spawns & manages language server
│    ├─ JSON-RPC 2.0 over stdio   │  processes, document sync,
│    └─ Request/response tracking │  concurrent session support
└─────────────┬───────────────────┘
              │  stdio
              ▼
    Language Servers (child processes)
    ├─ gopls
    ├─ typescript-language-server
    └─ pylsp
```

## Development

### Make targets

```bash
make build          # Build for current platform
make build-all      # Build for Linux, macOS, Windows (amd64 + arm64)
make dev            # Run in dev mode with ./config/config.yaml
make test           # Run all tests
make test-coverage  # Generate HTML coverage report
make lint           # Format + vet
make ci             # Full CI: mod-tidy → lint → test → build
make install        # Install to /usr/local/bin
make clean          # Remove build artifacts
```

### Running a single test

```bash
go test ./internal/lsp/... -run TestFunctionName
go test ./internal/session/... -run TestFunctionName
```

### Project structure

```
cmd/
  server/              Entry point, CLI flags (-config, -version)
  mcp-test-client/     CLI test client for manual verification
internal/
  mcp/                 MCP server, tool definitions, handlers
  lsp/                 LSP client, process management, JSON-RPC 2.0
  session/             Session lifecycle, metrics, request validation
  config/              YAML config loading and validation
pkg/
  types/               Shared types (MCP, LSP, session)
config/
  config.yaml          Default configuration
```

## Troubleshooting

**Check the logs** — default path: `./logs/lsp-mcp.log`. Set `logging.level: "debug"` for verbose output.

| Problem | Things to check |
|---------|----------------|
| LSP server won't start | Is the command installed and in PATH? Run it manually to verify. |
| Definition/references returns empty | Ensure `line`/`character` point to the symbol itself, not whitespace or punctuation. Use the optional `symbol` parameter. |
| TypeScript navigation fails | Verify `tsconfig.json` exists in `root_uri`. Compare parameters with what VS Code sends. |
| Session timeout | Increase `session.timeout` in config. Default is 300s. |
| macOS quarantine error | Run: `xattr -d com.apple.quarantine ./bin/lsp-mcp` |

## Contributing

```bash
make fmt        # Format code
make vet        # Static analysis
make test       # Run tests
make ci         # Full CI pipeline
```

Issues and pull requests are welcome.

## License

[MIT](LICENSE)
