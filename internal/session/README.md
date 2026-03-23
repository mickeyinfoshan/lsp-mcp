# internal/session

Session lifecycle management, request dispatching, and metrics tracking.

## Overview

The session manager sits between the MCP handlers and the LSP client. It validates request parameters, dispatches calls to the LSP client, and tracks per-request metrics (counts, response times, success rates).

## Key Types

| Type | Description |
|------|-------------|
| `Manager` | Coordinates LSP client access, parameter validation, and metrics |
| `SessionMetrics` | Thread-safe counters for requests, response times, and session lifecycle |

## Manager API

```go
manager, err := session.NewManager(cfg)

// Code intelligence (validates params, then delegates to LSP client)
defResp, err   := manager.FindDefinition(ctx, req)
refResp, err   := manager.FindReferences(ctx, req)
hoverResp, err := manager.GetHover(ctx, req)
compResp, err  := manager.GetCompletion(ctx, req)

// Introspection
client := manager.GetLSPClient()
info   := manager.GetSessionInfo()
metrics := manager.GetMetrics()
langs  := manager.GetSupportedLanguages()
cfg, ok := manager.GetLSPServerConfig("go")

// Lifecycle
manager.Shutdown(ctx)
manager.Close()
```

## Request Validation

Each request type has a dedicated validator that checks:
- Non-nil request
- Non-empty `language_id` and `file_uri`
- `root_uri` is required for definition requests (other requests may not enforce it)
- Non-negative `line` and `character`
- Language is configured in `config.yaml`

## Metrics

`SessionMetrics` tracks:

| Metric | Description |
|--------|-------------|
| `TotalRequests` | Total number of requests processed |
| `SuccessfulRequests` | Requests that returned without error |
| `FailedRequests` | Requests that failed validation or LSP call |
| `AverageResponseTime` | Exponential moving average (weight 0.1) in milliseconds |
| `SessionsCreated` / `SessionsClosed` | Session lifecycle counts |
| `LastRequestTime` | Timestamp of the most recent request |

All metrics are thread-safe (guarded by `sync.RWMutex`). Call `metrics.Reset()` to zero out all counters.
