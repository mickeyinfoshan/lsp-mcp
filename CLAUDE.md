# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`lsp-mcp` is a bridge service that connects the **Model Context Protocol (MCP)** with the **Language Server Protocol (LSP)**. It enables AI assistants to access multi-language code intelligence (go-to-definition, references, hover, completions) by translating MCP tool calls into LSP requests dispatched to language server processes (gopls, typescript-language-server, pylsp, etc.).

## Commands

```bash
make build          # Build for current platform (output: ./bin/lsp-mcp-bridge)
make run            # Build and run
make dev            # Run with config at ./config/config.yaml
make test           # Run tests
make test-coverage  # Generate coverage report
make lint           # Run fmt + vet
make fmt            # Format code
make ci             # Full CI pipeline: mod-tidy → lint → test → build
```

To run a single test:
```bash
go test ./internal/lsp/... -run TestFunctionName
go test ./internal/session/... -run TestFunctionName
```

## Architecture

```
AI Agent (MCP Client)
    │ MCP JSON-RPC
    ▼
internal/mcp/server.go     — MCP server, tool registration (lsp_definition, lsp_references, etc.)
internal/mcp/handlers.go   — Tool handlers, converts MCP params → LSP requests
    │
    ▼
internal/session/manager.go — Session lifecycle keyed by (language_id + root_uri)
    │
    ▼
internal/lsp/client.go     — Spawns LSP server process, speaks JSON-RPC 2.0 via stdio
    │ stdio JSON-RPC
    ▼
Language Servers (child processes: gopls, typescript-language-server, pylsp, ...)
```

**Request flow**: MCP tool call → handler validates params → session manager resolves/creates LSP session → LSP client sends textDocument/* request → response formatted and returned to agent.

**Session keying**: Sessions are identified by `language_id + root_uri`. Multiple concurrent sessions (different languages or workspaces) are supported.

**Document sync**: LSP client tracks open files, sending `textDocument/didOpen` / `didChange` / `didClose` notifications to maintain server state.

## Key Packages

| Package | Responsibility |
|---|---|
| `internal/mcp/` | MCP server, tool definitions, request handlers |
| `internal/lsp/` | LSP client, process management, JSON-RPC 2.0, message ID tracking |
| `internal/session/` | Session lifecycle, metrics (request counts, response times) |
| `internal/config/` | YAML config loading/validation for LSP server commands, args, env |
| `pkg/types/` | Shared types: MCP tool params, LSP requests/responses, session metrics |
| `cmd/server/` | Entry point (`-config`, `-version` flags, graceful shutdown) |
| `cmd/mcp-test-client/` | CLI test client for the MCP server |

## Configuration

Runtime config at `config/config.yaml`:
```yaml
lsp_servers:
  go:
    command: "/path/to/gopls"
    args: ["serve"]
    env: {GOPATH, GOBIN, ...}
  typescript:
    command: "typescript-language-server"
    args: ["--stdio"]
mcp_server:
  name: "lsp-bridge"
session:
  max_sessions: 100
  timeout: 300        # seconds
logging:
  level: "debug"
  file_path: "./logs/mcp-lsp-bridge.log"
```

## MCP Tools Exposed

All tools require `language_id`, `root_uri`, `file_uri`, `line`, `character`:
- `lsp_initialize` — start/reuse LSP session
- `lsp_shutdown` — terminate session
- `lsp_definition` — go to definition
- `lsp_references` — find all references
- `lsp_hover` — hover documentation
- `lsp_completion` — code completions
