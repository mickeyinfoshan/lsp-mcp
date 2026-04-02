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

// Server MCP server
type Server struct {
	// mcpServer MCP server instance
	mcpServer *server.MCPServer
	// sessionManager session manager
	sessionManager *session.Manager
	// config configuration info
	config *config.Config
}

// NewServer creates a new MCP server
func NewServer(cfg *config.Config) (*Server, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Create MCP server instance
	mcpServer := server.NewMCPServer(
		cfg.MCPServer.Name,
		cfg.MCPServer.Version,
		server.WithToolCapabilities(true), // Enable tool capabilities
		server.WithLogging(),              // Enable logging
		server.WithRecovery(),             // Enable recovery
	)

	// Create session manager
	sessionManager, err := session.NewManager(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create session manager: %w", err)
	}

	s := &Server{
		mcpServer:      mcpServer,
		sessionManager: sessionManager,
		config:         cfg,
	}

	// Register LSP tools
	if err := s.registerLSPTools(); err != nil {
		return nil, fmt.Errorf("failed to register LSP tools: %w", err)
	}

	return s, nil
}

// registerLSPTools registers MCP tools related to LSP
func (s *Server) registerLSPTools() error {
	// Register lsp.initialize tool
	// if err := s.registerInitializeTool(); err != nil {
	// 	return fmt.Errorf("failed to register initialize tool: %w", err)
	// }

	// Register lsp.shutdown tool
	if err := s.registerShutdownTool(); err != nil {
		return fmt.Errorf("failed to register shutdown tool: %w", err)
	}

	// Register lsp.definition tool
	if err := s.registerDefinitionTool(); err != nil {
		return fmt.Errorf("failed to register definition tool: %w", err)
	}

	// Register lsp.references tool
	if err := s.registerReferencesTool(); err != nil {
		return fmt.Errorf("failed to register references tool: %w", err)
	}

	// Register lsp.hover tool
	if err := s.registerHoverTool(); err != nil {
		return fmt.Errorf("failed to register hover tool: %w", err)
	}

	// Register lsp.completion tool
	if err := s.registerCompletionTool(); err != nil {
		return fmt.Errorf("failed to register completion tool: %w", err)
	}

	return nil
}

// registerInitializeTool registers the lsp.initialize tool
func (s *Server) registerInitializeTool() error {
	initializeTool := mcp.NewTool("lsp_initialize",
		mcp.WithDescription("Initialize (or reuse) an LSP session. It automatically starts the language server for the specified language and project root and establishes communication."),
		mcp.WithString("language_id",
			mcp.Required(),
			mcp.Description("Programming language identifier, e.g. 'go', 'python', 'typescript'. Example: 'go'."),
		),
		mcp.WithString("root_uri",
			mcp.Required(),
			mcp.Description("Project root URI, e.g. 'file:///path/to/project'. Example: 'file:///Users/foo/bar'."),
		),
	)

	s.mcpServer.AddTool(initializeTool, s.handleInitialize)
	return nil
}

// registerShutdownTool registers the lsp.shutdown tool
func (s *Server) registerShutdownTool() error {
	shutdownTool := mcp.NewTool("lsp_shutdown",
		mcp.WithDescription("Close the specified LSP session and release the language server process and resources."),
		mcp.WithString("language_id",
			mcp.Required(),
			mcp.Description("Language identifier to close. Example: 'go'."),
		),
		mcp.WithString("root_uri",
			mcp.Required(),
			mcp.Description("Project root URI of the session to close. Example: 'file:///Users/foo/bar'."),
		),
	)

	s.mcpServer.AddTool(shutdownTool, s.handleShutdown)
	return nil
}

// registerDefinitionTool registers the lsp.definition tool
func (s *Server) registerDefinitionTool() error {
	definitionTool := mcp.NewTool("lsp_definition",
		mcp.WithDescription("Find the definition of the symbol at the specified document position (Go to Definition) and handle file sync automatically.\n[Important] Keep the character parameter inside the target symbol (letter/underscore) rather than whitespace, punctuation, dot, or parentheses for more accurate jumps. Optional symbol helps with precise positioning."),
		mcp.WithString("language_id",
			mcp.Required(),
			mcp.Description("Programming language identifier. Example: 'go'."),
		),
		mcp.WithString("root_uri",
			mcp.Required(),
			mcp.Description("Project root URI. Example: 'file:///Users/foo/bar'."),
		),
		mcp.WithString("file_uri",
			mcp.Required(),
			mcp.Description("Target document URI. Example: 'file:///Users/foo/bar/main.go'."),
		),
		mcp.WithNumber("line",
			mcp.Required(),
			mcp.Description("Zero-based line number. Example: 10."),
		),
		mcp.WithNumber("character",
			mcp.Required(),
			mcp.Description("Zero-based character offset. Example: 5."),
		),
		mcp.WithString("symbol",
			mcp.Description("Target symbol name (optional) to aid precise positioning. Example: 'MyFunc'."),
		),
	)

	s.mcpServer.AddTool(definitionTool, s.handleDefinition)
	return nil
}

// registerReferencesTool registers the lsp.references tool
func (s *Server) registerReferencesTool() error {
	referencesTool := mcp.NewTool("lsp_references",
		mcp.WithDescription("Find all references to the symbol at the specified document position (Find References), optionally including the declaration.\n[Important] Keep the character parameter inside the target symbol (letter/underscore) rather than whitespace, punctuation, dot, or parentheses for more accurate references. Optional symbol helps with precise positioning."),
		mcp.WithString("language_id",
			mcp.Required(),
			mcp.Description("Programming language identifier. Example: 'go'."),
		),
		mcp.WithString("root_uri",
			mcp.Required(),
			mcp.Description("Project root URI. Example: 'file:///Users/foo/bar'."),
		),
		mcp.WithString("file_uri",
			mcp.Required(),
			mcp.Description("Target document URI. Example: 'file:///Users/foo/bar/main.go'."),
		),
		mcp.WithNumber("line",
			mcp.Required(),
			mcp.Description("Zero-based line number. Example: 10."),
		),
		mcp.WithNumber("character",
			mcp.Required(),
			mcp.Description("Zero-based character offset. Example: 5."),
		),
		mcp.WithBoolean("include_declaration",
			mcp.Description("Whether to include the symbol declaration in results (default false). Example: true."),
		),
		mcp.WithString("symbol",
			mcp.Description("Target symbol name (optional) to aid precise positioning. Example: 'MyFunc'."),
		),
	)

	s.mcpServer.AddTool(referencesTool, s.handleReferences)
	return nil
}

// registerHoverTool registers the lsp.hover tool
func (s *Server) registerHoverTool() error {
	hoverTool := mcp.NewTool("lsp_hover",
		mcp.WithDescription("Get hover info for the symbol at the specified document position (Hover), including type and documentation.\n[Important] Keep the character parameter inside the target symbol (letter/underscore) rather than whitespace, punctuation, dot, or parentheses for more accurate hover results. Optional symbol helps with precise positioning."),
		mcp.WithString("language_id",
			mcp.Required(),
			mcp.Description("Programming language identifier. Example: 'go'."),
		),
		mcp.WithString("root_uri",
			mcp.Required(),
			mcp.Description("Project root URI. Example: 'file:///Users/foo/bar'."),
		),
		mcp.WithString("file_uri",
			mcp.Required(),
			mcp.Description("Target document URI. Example: 'file:///Users/foo/bar/main.go'."),
		),
		mcp.WithNumber("line",
			mcp.Required(),
			mcp.Description("Zero-based line number. Example: 10."),
		),
		mcp.WithNumber("character",
			mcp.Required(),
			mcp.Description("Zero-based character offset. Example: 5."),
		),
		mcp.WithString("symbol",
			mcp.Description("Target symbol name (optional) to aid precise positioning. Example: 'MyFunc'."),
		),
	)

	s.mcpServer.AddTool(hoverTool, s.handleHover)
	return nil
}

// registerCompletionTool registers the lsp.completion tool
func (s *Server) registerCompletionTool() error {
	completionTool := mcp.NewTool("lsp_completion",
		mcp.WithDescription("Get completion suggestions at the specified document position (Completion), supporting trigger kind and character.\n[Important] Keep the character parameter inside the target symbol (letter/underscore) rather than whitespace, punctuation, dot, or parentheses for more accurate completions."),
		mcp.WithString("language_id",
			mcp.Required(),
			mcp.Description("Programming language identifier. Example: 'go'."),
		),
		mcp.WithString("root_uri",
			mcp.Required(),
			mcp.Description("Project root URI. Example: 'file:///Users/foo/bar'."),
		),
		mcp.WithString("file_uri",
			mcp.Required(),
			mcp.Description("Target document URI. Example: 'file:///Users/foo/bar/main.go'."),
		),
		mcp.WithNumber("line",
			mcp.Required(),
			mcp.Description("Zero-based line number. Example: 10."),
		),
		mcp.WithNumber("character",
			mcp.Required(),
			mcp.Description("Zero-based character offset. Example: 5."),
		),
		mcp.WithNumber("trigger_kind",
			mcp.Description("Completion trigger kind (1=manual, 2=trigger character, 3=command, etc.; see LSP spec). Example: 2."),
		),
		mcp.WithString("trigger_character",
			mcp.Description("Trigger character for completion (e.g. '.', '>' ); supported by some language servers. Example: '.'."),
		),
	)

	s.mcpServer.AddTool(completionTool, s.handleCompletion)
	return nil
}

// Serve starts the MCP server
func (s *Server) Serve() error {
	log.Printf("Starting MCP-LSP bridge server: %s v%s", s.config.MCPServer.Name, s.config.MCPServer.Version)
	return server.ServeStdio(s.mcpServer)
}

// Shutdown shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down MCP-LSP bridge server...")

	// Shut down the session manager
	if err := s.sessionManager.Shutdown(ctx); err != nil {
		log.Printf("Error shutting down session manager: %v", err)
	}

	// MCPServer doesn't have a Shutdown method, it's managed by the transport layer
	// The server will be closed when the transport (stdio, http, etc.) is stopped

	log.Println("MCP-LSP bridge server stopped")
	return nil
}
