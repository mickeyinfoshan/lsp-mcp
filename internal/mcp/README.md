# internal/mcp

MCP server: tool registration, request handling, and response formatting.

## Overview

This package implements the MCP server that exposes LSP functionality as MCP tools. It uses [mcp-go](https://github.com/mark3labs/mcp-go) for the MCP protocol layer and delegates actual LSP operations to the session manager.

## Key Types

| Type | Description |
|------|-------------|
| `Server` | Main MCP server wrapping `mcp-go/MCPServer`, session manager, and config |

## Server Lifecycle

```go
server, err := mcp.NewServer(cfg)  // Creates MCP server + session manager, registers tools
err := server.Serve()               // Starts stdio transport (blocking)
err := server.Shutdown(ctx)         // Graceful shutdown
```

## Registered Tools

| Tool | Handler | Description |
|------|---------|-------------|
| `lsp_definition` | `handleDefinition` | Go to definition at a file position |
| `lsp_references` | `handleReferences` | Find all references at a file position |
| `lsp_hover` | `handleHover` | Get hover info (type signature, docs) |
| `lsp_completion` | `handleCompletion` | Get code completion suggestions |
| `lsp_shutdown` | `handleShutdown` | Close an LSP session |

All tools (except `lsp_shutdown`) accept: `language_id`, `root_uri`, `file_uri`, `line`, `character`, and an optional `symbol` parameter for fuzzy position correction.

## Handler Flow

1. Extract and validate parameters from `mcp.CallToolRequest`
2. Auto-detect project root via `findProjectRoot()` (walks up looking for `go.mod`, `tsconfig.json`, etc.)
3. If `symbol` is provided, search nearby lines to find the best matching position via `findSymbolPositionInFile()`
4. Delegate to `session.Manager` (e.g. `FindDefinition`, `FindReferences`)
5. Format response as `MCPToolResponse` JSON with agent-friendly summaries

## Helper Functions

| Function | Description |
|----------|-------------|
| `findProjectRoot()` | Walks up from file to find language-specific project markers |
| `findSymbolPositionInFile()` | Searches +/-10 then +/-50 lines for a symbol name, returns nearest match |
| `adjustCharacterBySymbol()` | Extracts symbol at position and adjusts character offset if needed |
| `extractSymbolSmart()` | Extracts identifier token at a given column in a line |
| `validateURI()` / `validatePosition()` | Input validation helpers |
