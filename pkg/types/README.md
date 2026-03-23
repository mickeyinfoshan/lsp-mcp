# pkg/types

Shared data types used across all packages in the MCP-LSP bridge.

## Overview

This package defines the request/response structures for MCP tool calls, LSP session state, and LSP client capabilities. It is imported by `internal/mcp`, `internal/lsp`, and `internal/session`.

## Files

### `mcp.go` — MCP Protocol Types

| Type | Description |
|------|-------------|
| `MCPToolRequest` | Incoming MCP tool call (name + arguments) |
| `MCPToolResponse` | Outgoing response (content array + error flag) |
| `MCPContent` | Single content item (type, text, data) |

### `lsp_session.go` — Session State

| Type | Description |
|------|-------------|
| `SessionKey` | Unique session identifier: `languageID + rootURI` |
| `LSPSession` | Session state: connection, process, timestamps, initialization params, opened documents |
| `LSPClientInfo` | Client name and version sent during initialization |
| `LSPWorkspaceFolder` | Workspace folder URI and name |

### `lsp_requests.go` — LSP Request/Response Types

Each LSP operation has a request, response, and agent-friendly result type:

| Operation | Request | Response | Agent Result |
|-----------|---------|----------|--------------|
| Definition | `FindDefinitionRequest` | `FindDefinitionResponse` | `AgentDefinitionResult` |
| References | `FindReferencesRequest` | `FindReferencesResponse` | `AgentReferenceResult` |
| Hover | `HoverRequest` | `HoverResponse` | `AgentHoverResult` |
| Completion | `CompletionRequest` | `CompletionResponse` | `AgentCompletionResult` |
| Open File | `OpenFileRequest` | — | — |

Agent result types provide structured, human-readable output: file path, 1-based line/character, summary text, and raw LSP range/location.

### `lsp_capabilities.go` — LSP Client Capabilities

Defines 60+ types for the full LSP client capabilities structure sent during initialization. Key types:

- `LSPClientCapabilities` — root capabilities
- `LSPTextDocumentClientCapabilities` — text document features (definition, references, hover, completion, etc.)
- `LSPWorkspaceClientCapabilities` — workspace features (applyEdit, configuration, workspaceFolders)
- `LSPWindowClientCapabilities` — window features (workDoneProgress, showMessage, showDocument)
- `LSPGeneralClientCapabilities` — general features (regex, markdown, position encodings)
