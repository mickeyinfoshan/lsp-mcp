# MCP-LSP 桥接服务客户端测试

本文档介绍如何使用创建的 MCP 客户端来验证 MCP-LSP 桥接服务的可用性。

## 概述

MCP-LSP 桥接服务是一个将 Language Server Protocol (LSP) 功能通过 Model Context Protocol (MCP) 暴露的服务。它支持多种编程语言的 LSP 功能，包括代码补全、定义查找、悬停信息、引用查找等。

## 文件结构

```
lsp-mcp/
├── cmd/
│   ├── server/             # MCP服务器主程序
│   └── mcp-test-client/    # MCP客户端测试程序
├── config/
│   └── config.yaml         # 服务配置文件
├── MCP_CLIENT_README.md    # 本文档
├── README.md               # 主项目说明
└── ...
```

## 支持的 LSP 工具

MCP-LSP 桥接服务提供以下 LSP 工具：

1. **lsp.definition** - 查找符号定义
2. **lsp.hover** - 获取悬停信息
3. **lsp.references** - 查找符号引用
4. **lsp.completion** - 代码补全
5. **lsp.shutdown** - 关闭 LSP 会话

## 支持的编程语言

根据配置文件，服务支持以下语言：

- **Go** - 使用 `gopls` 语言服务器
- **TypeScript/JavaScript** - 使用 `typescript-language-server`
- **Python** - 使用 `pylsp` (Python LSP Server)

## 手动测试

### 1. 编译 MCP 服务器

```bash
# 编译服务器
make build

```

### 2. 使用客户端测试

```bash
# 编译客户端
go run cmd/mcp-test-client/main.go
```

## MCP 客户端功能

创建的 MCP 客户端 (`cmd/mcp-test-client/main.go`) 提供以下功能：

### 核心功能

- **JSON-RPC 通信** - 通过 stdin/stdout 与 MCP 服务器通信
- **MCP 协议支持** - 实现 MCP 协议
- **工具调用** - 支持调用所有 MCP 工具
- **错误处理** - 完善的错误处理和日志记录

### 测试用例

1. **MCP 初始化测试** - 验证客户端可以成功初始化 MCP 连接
2. **工具列表测试** - 获取并显示所有可用的 LSP 工具
3. **LSP 定义查找测试** - 测试代码定义查找功能

## 配置说明

配置文件 `config/config.yaml` 示例：

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

## 故障排除

### 常见问题

1. **配置文件未找到**
   - 确保配置文件路径正确：`config/config.yaml`
   - 检查文件权限

2. **LSP 服务器未安装**
   - 安装所需的 LSP 服务器：
     ```bash
     # Go
     go install golang.org/x/tools/gopls@latest
     # TypeScript
     npm install -g typescript-language-server
     # Python
     pip install python-lsp-server
     ```

3. **超时错误**
   - 增加配置文件中的 session timeout 值
   - 检查 LSP 服务器是否正常安装和运行

4. **日志与调试**
   - 日志文件路径见 config.yaml 的 logging.file_path
   - 日志级别可调为 debug 以获得详细信息

## 成功验证标志

当看到以下输出时，表示 MCP 服务验证成功：

```
🎉 MCP服务验证完成！
✅ MCP服务器可以正常启动和关闭
✅ MCP服务器可以加载配置文件
✅ MCP服务器支持LSP工具: lsp.definition, lsp.hover, lsp.references, lsp.completion, lsp.shutdown
```

## 下一步

验证成功后，你可以：

1. 将 MCP 服务集成到你的应用中
2. 扩展支持更多编程语言
3. 添加更多 LSP 功能
4. 优化性能和错误处理

## 技术细节

- **协议版本**: 详见代码实现
- **通信方式**: JSON-RPC over stdin/stdout
- **支持的 LSP 版本**: LSP 3.17
- **并发处理**: 支持多个 LSP 会话
- **会话管理**: 自动会话创建和清理