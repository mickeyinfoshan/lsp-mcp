package mcp

import (
	"context"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/mickeyinfoshan/lsp-mcp/internal/config"
	"github.com/mickeyinfoshan/lsp-mcp/internal/session"
)

// Server MCP服务器
type Server struct {
	// mcpServer MCP服务器实例
	mcpServer *server.MCPServer
	// sessionManager 会话管理器
	sessionManager *session.Manager
	// config 配置信息
	config *config.Config
}

// NewServer 创建新的MCP服务器
func NewServer(cfg *config.Config) (*Server, error) {
	if cfg == nil {
		return nil, fmt.Errorf("配置不能为空")
	}

	// 创建MCP服务器实例
	mcpServer := server.NewMCPServer(
		cfg.MCPServer.Name,
		cfg.MCPServer.Version,
		server.WithToolCapabilities(true), // 启用工具功能
		server.WithLogging(),              // 启用日志
		server.WithRecovery(),             // 启用恢复机制
	)

	// 创建会话管理器
	sessionManager, err := session.NewManager(cfg)
	if err != nil {
		return nil, fmt.Errorf("创建会话管理器失败: %w", err)
	}

	s := &Server{
		mcpServer:      mcpServer,
		sessionManager: sessionManager,
		config:         cfg,
	}

	// 注册LSP工具
	if err := s.registerLSPTools(); err != nil {
		return nil, fmt.Errorf("注册LSP工具失败: %w", err)
	}

	return s, nil
}

// registerLSPTools 注册LSP相关的MCP工具
func (s *Server) registerLSPTools() error {
	// 注册lsp.initialize工具
	// if err := s.registerInitializeTool(); err != nil {
	// 	return fmt.Errorf("注册initialize工具失败: %w", err)
	// }

	// 注册lsp.shutdown工具
	if err := s.registerShutdownTool(); err != nil {
		return fmt.Errorf("注册shutdown工具失败: %w", err)
	}

	// 注册lsp.definition工具
	if err := s.registerDefinitionTool(); err != nil {
		return fmt.Errorf("注册definition工具失败: %w", err)
	}

	// 注册lsp.references工具
	if err := s.registerReferencesTool(); err != nil {
		return fmt.Errorf("注册references工具失败: %w", err)
	}

	// 注册lsp.hover工具
	if err := s.registerHoverTool(); err != nil {
		return fmt.Errorf("注册hover工具失败: %w", err)
	}

	// 注册lsp.completion工具
	if err := s.registerCompletionTool(); err != nil {
		return fmt.Errorf("注册completion工具失败: %w", err)
	}

	return nil
}

// registerInitializeTool 注册lsp.initialize工具
func (s *Server) registerInitializeTool() error {
	initializeTool := mcp.NewTool("lsp_initialize",
		mcp.WithDescription("初始化（或复用）一个 LSP 会话。会自动为指定语言和项目根目录启动语言服务器进程，并建立通信。"),
		mcp.WithString("language_id",
			mcp.Required(),
			mcp.Description("编程语言标识符，例如 'go'、'python'、'typescript'。例：'go'"),
		),
		mcp.WithString("root_uri",
			mcp.Required(),
			mcp.Description("项目根目录的 URI，形如 'file:///path/to/project'。例：'file:///Users/foo/bar'"),
		),
	)

	s.mcpServer.AddTool(initializeTool, s.handleInitialize)
	return nil
}

// registerShutdownTool 注册lsp.shutdown工具
func (s *Server) registerShutdownTool() error {
	shutdownTool := mcp.NewTool("lsp_shutdown",
		mcp.WithDescription("关闭指定 LSP 会话，释放对应语言服务器进程和资源。"),
		mcp.WithString("language_id",
			mcp.Required(),
			mcp.Description("要关闭的编程语言标识符。例：'go'"),
		),
		mcp.WithString("root_uri",
			mcp.Required(),
			mcp.Description("要关闭的会话对应的项目根目录 URI。例：'file:///Users/foo/bar'"),
		),
	)

	s.mcpServer.AddTool(shutdownTool, s.handleShutdown)
	return nil
}

// registerDefinitionTool 注册lsp.definition工具
func (s *Server) registerDefinitionTool() error {
	definitionTool := mcp.NewTool("lsp_definition",
		mcp.WithDescription("查找指定文档位置符号的定义（Go to Definition），自动处理文件同步。\n【重要】请确保 character 参数尽量精确落在目标 symbol 的字母或下划线等内部字符上，避免落在空格、标点、点号、括号等非 symbol 区域，这样能获得更准确的跳转体验。可选 symbol 参数辅助精确定位。"),
		mcp.WithString("language_id",
			mcp.Required(),
			mcp.Description("编程语言标识符。例：'go'"),
		),
		mcp.WithString("root_uri",
			mcp.Required(),
			mcp.Description("项目根目录的 URI。例：'file:///Users/foo/bar'"),
		),
		mcp.WithString("file_uri",
			mcp.Required(),
			mcp.Description("目标文档的 URI。例：'file:///Users/foo/bar/main.go'"),
		),
		mcp.WithNumber("line",
			mcp.Required(),
			mcp.Description("目标位置的行号（从 0 开始）。例：10"),
		),
		mcp.WithNumber("character",
			mcp.Required(),
			mcp.Description("目标位置的字符偏移（从 0 开始）。例：5"),
		),
		mcp.WithString("symbol",
			mcp.Description("目标符号名称，辅助精确定位。例：'MyFunc'"),
		),
	)

	s.mcpServer.AddTool(definitionTool, s.handleDefinition)
	return nil
}

// registerReferencesTool 注册lsp.references工具
func (s *Server) registerReferencesTool() error {
	referencesTool := mcp.NewTool("lsp_references",
		mcp.WithDescription("查找指定文档位置符号的所有引用（Find References），可选包含声明。\n【重要】请确保 character 参数尽量精确落在目标 symbol 的字母或下划线等内部字符上，避免落在空格、标点、点号、括号等非 symbol 区域，这样能获得更准确的引用体验。可选 symbol 参数辅助精确定位。"),
		mcp.WithString("language_id",
			mcp.Required(),
			mcp.Description("编程语言标识符。例：'go'"),
		),
		mcp.WithString("root_uri",
			mcp.Required(),
			mcp.Description("项目根目录的 URI。例：'file:///Users/foo/bar'"),
		),
		mcp.WithString("file_uri",
			mcp.Required(),
			mcp.Description("目标文档的 URI。例：'file:///Users/foo/bar/main.go'"),
		),
		mcp.WithNumber("line",
			mcp.Required(),
			mcp.Description("目标位置的行号（从 0 开始）。例：10"),
		),
		mcp.WithNumber("character",
			mcp.Required(),
			mcp.Description("目标位置的字符偏移（从 0 开始）。例：5"),
		),
		mcp.WithBoolean("include_declaration",
			mcp.Description("结果中是否包含符号声明（默认为 false）。例：true"),
		),
		mcp.WithString("symbol",
			mcp.Description("目标符号名称，辅助精确定位。例：'MyFunc'"),
		),
	)

	s.mcpServer.AddTool(referencesTool, s.handleReferences)
	return nil
}

// registerHoverTool 注册lsp.hover工具
func (s *Server) registerHoverTool() error {
	hoverTool := mcp.NewTool("lsp_hover",
		mcp.WithDescription("获取指定文档位置符号的悬停信息（Hover），包括类型、文档等。\n【重要】请确保 character 参数尽量精确落在目标 symbol 的字母或下划线等内部字符上，避免落在空格、标点、点号、括号等非 symbol 区域，这样能获得更准确的悬停体验。可选 symbol 参数辅助精确定位。"),
		mcp.WithString("language_id",
			mcp.Required(),
			mcp.Description("编程语言标识符。例：'go'"),
		),
		mcp.WithString("root_uri",
			mcp.Required(),
			mcp.Description("项目根目录的 URI。例：'file:///Users/foo/bar'"),
		),
		mcp.WithString("file_uri",
			mcp.Required(),
			mcp.Description("目标文档的 URI。例：'file:///Users/foo/bar/main.go'"),
		),
		mcp.WithNumber("line",
			mcp.Required(),
			mcp.Description("目标位置的行号（从 0 开始）。例：10"),
		),
		mcp.WithNumber("character",
			mcp.Required(),
			mcp.Description("目标位置的字符偏移（从 0 开始）。例：5"),
		),
		mcp.WithString("symbol",
			mcp.Description("目标符号名称，辅助精确定位。例：'MyFunc'"),
		),
	)

	s.mcpServer.AddTool(hoverTool, s.handleHover)
	return nil
}

// registerCompletionTool 注册lsp.completion工具
func (s *Server) registerCompletionTool() error {
	completionTool := mcp.NewTool("lsp_completion",
		mcp.WithDescription("获取指定文档位置的代码补全建议（Completion），支持触发类型和字符。\n【重要】请确保 character 参数尽量精确落在目标 symbol 的字母或下划线等内部字符上，避免落在空格、标点、点号、括号等非 symbol 区域，这样能获得更准确的补全体验。"),
		mcp.WithString("language_id",
			mcp.Required(),
			mcp.Description("编程语言标识符。例：'go'"),
		),
		mcp.WithString("root_uri",
			mcp.Required(),
			mcp.Description("项目根目录的 URI。例：'file:///Users/foo/bar'"),
		),
		mcp.WithString("file_uri",
			mcp.Required(),
			mcp.Description("目标文档的 URI。例：'file:///Users/foo/bar/main.go'"),
		),
		mcp.WithNumber("line",
			mcp.Required(),
			mcp.Description("目标位置的行号（从 0 开始）。例：10"),
		),
		mcp.WithNumber("character",
			mcp.Required(),
			mcp.Description("目标位置的字符偏移（从 0 开始）。例：5"),
		),
		mcp.WithNumber("trigger_kind",
			mcp.Description("补全触发类型（1=手动，2=输入字符，3=命令等，参见 LSP 规范）。例：2"),
		),
		mcp.WithString("trigger_character",
			mcp.Description("触发补全的字符（如 '.'、'>' 等，部分语言服务器支持）。例：'.'"),
		),
	)

	s.mcpServer.AddTool(completionTool, s.handleCompletion)
	return nil
}

// Serve 启动MCP服务器
func (s *Server) Serve() error {
	log.Printf("启动MCP-LSP桥接服务器: %s v%s", s.config.MCPServer.Name, s.config.MCPServer.Version)
	return server.ServeStdio(s.mcpServer)
}

// Shutdown 关闭服务器
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("正在关闭MCP-LSP桥接服务器...")

	// 关闭会话管理器
	if err := s.sessionManager.Shutdown(ctx); err != nil {
		log.Printf("关闭会话管理器时出错: %v", err)
	}

	// MCPServer doesn't have a Shutdown method, it's managed by the transport layer
	// The server will be closed when the transport (stdio, http, etc.) is stopped

	log.Println("MCP-LSP桥接服务器已关闭")
	return nil
}
