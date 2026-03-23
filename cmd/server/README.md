# cmd/server

Main entry point for the MCP-LSP bridge service.

## Overview

This is the `main` package that wires together configuration, logging, and the MCP server. It handles CLI flags, signal-based graceful shutdown, and process lifecycle.

## Usage

```bash
# With explicit config
./bin/lsp-mcp -config /path/to/config.yaml

# Default config (./config/config.yaml)
./bin/lsp-mcp

# Version info
./bin/lsp-mcp -version

# Help
./bin/lsp-mcp -help
```

## CLI Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-config` | Path to YAML config file | `./config/config.yaml` |
| `-version` | Print version, build time, git commit | — |
| `-help` | Print usage information | — |

## Startup Flow

1. Parse CLI flags
2. Load and validate config via `config.LoadConfig()`
3. Configure logging (level, format, file output) via `setupLogging()`
4. Create MCP server via `mcp.NewServer()` (registers all LSP tools)
5. Start stdio transport in a goroutine (`server.Serve()`)
6. Wait for `SIGINT`/`SIGTERM` or server error
7. Graceful shutdown with 30-second timeout
