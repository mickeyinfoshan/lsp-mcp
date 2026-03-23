# cmd/mcp-test-client

CLI test client for manual verification of the MCP-LSP bridge server.

## Overview

This program spawns the MCP server as a child process and communicates with it over stdin/stdout using JSON-RPC 2.0. It exercises the MCP protocol handshake and LSP tool calls to verify end-to-end functionality.

## Key Types

| Type | Description |
|------|-------------|
| `MCPClient` | Test client: manages server process, sends JSON-RPC requests, reads responses |
| `MCPRequest` | JSON-RPC 2.0 request structure |
| `MCPResponse` | JSON-RPC 2.0 response structure |

## Client API

```go
client, err := NewMCPClient("./bin/lsp-mcp", "./config/config.yaml")
defer client.Close()

client.Initialize(ctx)                          // MCP handshake
client.ListTools(ctx)                           // List available tools
client.CallTool(ctx, "lsp_definition", args)    // Call any MCP tool
client.TestListTools(ctx)                       // Test listing tools
client.TestLSPDefinition(ctx, lang, root, file, line, char)
client.TestLSPHover(ctx, lang, root, file, line, char)
```

## Usage

```bash
# Build the server first
make build

# Run the test client (edit main() for your test parameters)
go run ./cmd/mcp-test-client/
```

The default configuration in `main()` targets TypeScript — update `languageID`, `rootURI`, `fileURI`, `line`, and `character` to match your test project.
