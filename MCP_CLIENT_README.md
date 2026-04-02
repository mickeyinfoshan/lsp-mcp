# MCP-LSP Bridge Service Client Testing

This document describes how to use the created MCP client to verify the availability of the MCP-LSP bridge service.

## Overview

The MCP-LSP bridge service exposes Language Server Protocol (LSP) functionality through the Model Context Protocol (MCP). It supports LSP features for multiple programming languages, including code completion, definition lookup, hover information, reference finding, and more.

## File Structure

```
lsp-mcp/
├── cmd/
│   ├── server/             # MCP server main program
│   └── mcp-test-client/    # MCP client test program
├── config/
│   └── config.yaml         # Service configuration file
├── MCP_CLIENT_README.md    # This document
├── README.md               # Main project documentation
└── ...
```

## Supported LSP Tools

The MCP-LSP bridge service provides the following LSP tools:

1. **lsp.definition** - Find symbol definitions
2. **lsp.hover** - Get hover information
3. **lsp.references** - Find symbol references
4. **lsp.completion** - Code completion
5. **lsp.shutdown** - Close LSP session

## Supported Programming Languages

According to the configuration file, the service supports the following languages:

- **Go** - Uses `gopls` language server
- **TypeScript/JavaScript** - Uses `typescript-language-server`
- **Python** - Uses `pylsp` (Python LSP Server)

## Manual Testing

### 1. Build MCP Server

```bash
# Build server
make build

```

### 2. Test with Client

```bash
# Build client
go run cmd/mcp-test-client/main.go
```

## MCP Client Features

The created MCP client (`cmd/mcp-test-client/main.go`) provides the following features:

### Core Features

- **JSON-RPC Communication** - Communicates with MCP server via stdin/stdout
- **MCP Protocol Support** - Implements MCP protocol
- **Tool Invocation** - Supports calling all MCP tools
- **Error Handling** - Comprehensive error handling and logging

### Test Cases

1. **MCP Initialization Test** - Verifies client can successfully initialize MCP connection
2. **Tool List Test** - Retrieves and displays all available LSP tools
3. **LSP Definition Lookup Test** - Tests code definition lookup functionality

## Configuration

Example `config/config.yaml` configuration file:

```yaml
lsp_servers:
  go:
    command: "gopls"
    args: ["serve"]
    initialization_options: {}
    env:
      GOPATH: "~/go"
      GOBIN: "~/go/bin"
      GO111MODULE: "on"
      GOPLSDEBUG: "all"
  typescript:
    command: "typescript-language-server"
    args: ["--stdio"]
    initialization_options: {}
    env: {}
  javascript:
    command: "typescript-language-server"
    args: ["--stdio"]
    initialization_options: {}
    env: {}
  python:
    command: "pylsp"
    args: []
    initialization_options: {}
    env: {}

mcp_server:
  name: "lsp-bridge"
  version: "1.0.0"
  description: "MCP-LSP Bridge Service for providing LSP capabilities through MCP protocol"

logging:
  level: "debug"
  format: "json"
  file_output: true
  file_path: "./logs/mcp-lsp-bridge.log"

session:
  timeout: 300
  max_sessions: 100
  cleanup_interval: 60
```

## Troubleshooting

### Common Issues

1. **Configuration file not found**
   - Ensure the configuration file path is correct: `config/config.yaml`
   - Check file permissions

2. **LSP server not installed**
   - Install the required LSP servers:
     ```bash
     # Go
     go install golang.org/x/tools/gopls@latest
     # TypeScript
     npm install -g typescript-language-server
     # Python
     pip install python-lsp-server
     ```

3. **Timeout errors**
   - Increase the `session.timeout` value in the configuration file
   - Check whether the LSP server is properly installed and running

4. **Logging and debugging**
   - See `logging.file_path` in config.yaml for the log file path
   - Set the log level to `debug` for detailed information

## Success Indicators

When you see the following output, MCP service verification is successful:

```
🎉 MCP service verification completed!
✅ MCP server can start and stop normally
✅ MCP server can load configuration file
✅ MCP server supports LSP tools: lsp.definition, lsp.hover, lsp.references, lsp.completion, lsp.shutdown
```

## Next Steps

After successful verification, you can:

1. Integrate the MCP service into your application
2. Extend support for more programming languages
3. Add more LSP features
4. Optimize performance and error handling

## Technical Details

- **Protocol Version**: See code implementation for details
- **Communication Method**: JSON-RPC over stdin/stdout
- **Supported LSP Version**: LSP 3.17
- **Concurrent Processing**: Supports multiple LSP sessions
- **Session Management**: Automatic session creation and cleanup