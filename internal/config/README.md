# internal/config

YAML configuration loading and validation for the MCP-LSP bridge service.

## Overview

This package defines the configuration structure and provides functions to load, parse, and validate `config.yaml`. All sections — LSP servers, MCP server metadata, logging, and session limits — are validated on load.

## Key Types

| Type | Description |
|------|-------------|
| `Config` | Root configuration struct; holds all sections |
| `LSPServerConfig` | Per-language LSP server: command, args, env, initialization options |
| `MCPServerConfig` | MCP server metadata: name, version, description |
| `LoggingConfig` | Log level (`debug`/`info`/`warn`/`error`), format (`json`/`text`), file output |
| `SessionConfig` | Session limits: `max_sessions` |

## Key Functions

```go
// Load and validate config from a specific path
cfg, err := config.LoadConfig("/path/to/config.yaml")

// Load from default location (next to the executable)
cfg, err := config.LoadConfigFromDefault()

// Validate an already-loaded config
err := cfg.Validate()

// Get LSP server config by language ID
serverCfg, ok := cfg.GetLSPServerConfig("go")
```

## Validation Rules

- At least one LSP server must be configured
- Each LSP server must have a non-empty `command`
- MCP server `name` and `version` are required
- Log level must be one of: `debug`, `info`, `warn`, `error`
- Log format must be one of: `json`, `text`
- `max_sessions` must be > 0

## Config File Format

See [`config/config.yaml`](../../config/config.yaml) for a full example.
