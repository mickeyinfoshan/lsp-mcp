# internal/lsp

LSP client implementation: process management, JSON-RPC 2.0 over stdio, and document synchronization.

## Overview

This package spawns language server processes (gopls, typescript-language-server, pylsp, etc.) as child processes, communicates with them over stdin/stdout using JSON-RPC 2.0 with `Content-Length` headers, and tracks open documents to keep language servers in sync.

## Key Types

| Type | Description |
|------|-------------|
| `Client` | High-level LSP client managing sessions and code intelligence requests |
| `LSPConnection` | Low-level connection: stdio pipes, message routing, request/response tracking |
| `MessageID` | JSON-RPC message ID (string, number, or null) with custom marshaling |
| `Message` | JSON-RPC 2.0 message (request, response, or notification) |
| `NotificationHandler` | Callback for server-to-client notifications |
| `ServerRequestHandler` | Callback for server-to-client requests (e.g. `workspace/configuration`) |

## Client API

```go
client := lsp.NewClient(cfg)

// Sessions are created lazily on first use, keyed by (languageID, rootURI)
session, err := client.GetOrCreateSession("go", "file:///path/to/project")

// Code intelligence
defResp, err  := client.FindDefinition(ctx, req)
refResp, err  := client.FindReferences(ctx, req)
hoverResp, err := client.GetHover(ctx, req)
compResp, err := client.GetCompletion(ctx, req)

// File management
err := client.OpenFile(ctx, req)

// Lifecycle
client.Close()
```

## Session Lifecycle

1. **Create** — `GetOrCreateSession` spawns the language server process and sends `initialize` / `initialized`
2. **Use** — Each request calls `ensureFileOpen` (sends `textDocument/didOpen` if needed), then the appropriate `textDocument/*` method
3. **Close** — `closeSession` sends `shutdown` + `exit`, then kills the process

## Document Sync

Open files are tracked per session (`openedFiles` map). On first access to a file, its content is read from disk and sent via `textDocument/didOpen`. Subsequent requests to the same file skip the notification.

## LSPConnection Internals

- Runs a background goroutine (`handleMessages`) that reads messages and routes them to response handlers, notification handlers, or server request handlers
- `Call()` — sends a request and blocks until a response arrives (or context expires)
- `Notify()` — sends a one-way notification
- Pre-registered handlers for common server requests: `workspace/configuration`, `window/workDoneProgress/create`, `client/registerCapability`, etc.
