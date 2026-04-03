package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/mickeyinfoshan/lsp-mcp/internal/config"
	"github.com/mickeyinfoshan/lsp-mcp/internal/logger"
	"github.com/mickeyinfoshan/lsp-mcp/pkg/types"
)

// Client LSP client
type Client struct {
	// config configuration info
	config *config.Config
	// sessions session map
	sessions map[string]*types.LSPSession
	// sessionsMutex protects sessions
	sessionsMutex sync.RWMutex
	// openedFiles tracks opened files (sessionKey -> map[fileURI]bool)
	openedFiles map[string]map[string]bool
	// openedFilesMutex protects opened files
	openedFilesMutex sync.RWMutex
	// ctx context
	ctx context.Context
	// cancel function
	cancel context.CancelFunc
}

// NewClient creates a new LSP client
func NewClient(cfg *config.Config) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		config:      cfg,
		sessions:    make(map[string]*types.LSPSession),
		openedFiles: make(map[string]map[string]bool),
		ctx:         ctx,
		cancel:      cancel,
	}

	return client
}

// GetOrCreateSession gets or creates an LSP session
func (c *Client) GetOrCreateSession(languageID, rootURI string) (*types.LSPSession, error) {
	sessionKey := types.SessionKey{
		LanguageID: languageID,
		RootURI:    rootURI,
	}

	c.sessionsMutex.Lock()
	defer c.sessionsMutex.Unlock()

	// Check if session already exists
	if session, exists := c.sessions[sessionKey.String()]; exists {
		// Update last used time
		session.UpdateLastUsed()
		return session, nil
	}

	// Check session count limit
	if len(c.sessions) >= c.config.Session.MaxSessions {
		return nil, fmt.Errorf("max sessions limit reached: %d", c.config.Session.MaxSessions)
	}

	// Create a new session
	session, err := c.createSession(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create LSP session: %w", err)
	}

	c.sessions[sessionKey.String()] = session
	return session, nil
}

// createSession creates a new LSP session
func (c *Client) createSession(sessionKey types.SessionKey) (*types.LSPSession, error) {
	// Get LSP server config
	serverConfig, exists := c.config.GetLSPServerConfig(sessionKey.LanguageID)
	if !exists {
		return nil, fmt.Errorf("unsupported language: %s", sessionKey.LanguageID)
	}

	logger.Debugf("[DEBUG] starting LSP server: command=%s, args=%v", serverConfig.Command, serverConfig.Args)

	// Start LSP server process
	cmdArgs := serverConfig.Args
	cmd := exec.CommandContext(c.ctx, serverConfig.Command, cmdArgs...)

	// Set gopls working directory to the go.mod directory
	if strings.HasPrefix(sessionKey.RootURI, "file://") {
		cmd.Dir = strings.TrimPrefix(sessionKey.RootURI, "file://")
	}

	// Merge environment variables
	cmd.Env = os.Environ()
	for k, v := range serverConfig.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Create pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start process
	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		stderr.Close()
		return nil, fmt.Errorf("failed to start LSP server: %w", err)
	}

	// Create manually implemented LSP connection
	conn := NewLSPConnection(cmd, stdin, stdout, stderr)

	// Create session object
	session := &types.LSPSession{
		Key:           sessionKey,
		Conn:          conn,
		Process:       cmd,
		CreatedAt:     time.Now(),
		LastUsedAt:    time.Now(),
		IsInitialized: false,
	}

	// Initialize LSP server
	if err := c.initializeSession(session, serverConfig); err != nil {
		// Cleanup resources
		conn.Close()
		cmd.Process.Kill()
		return nil, fmt.Errorf("failed to initialize LSP session: %w", err)
	}

	return session, nil
}

// initializeSession initializes an LSP session
func (c *Client) initializeSession(session *types.LSPSession, serverConfig *config.LSPServerConfig) error {
	// Build workspace folders
	workspaceFolders := []types.LSPWorkspaceFolder{
		{
			URI:  session.Key.RootURI,
			Name: "lsp-mcp",
		},
	}

	// Build initialize params
	initParams := &types.LSPInitializeParams{
		ProcessID: func() *int { pid := os.Getpid(); return &pid }(),
		ClientInfo: &types.LSPClientInfo{
			Name:    c.config.MCPServer.Name,
			Version: c.config.MCPServer.Version,
		},
		RootURI:               &session.Key.RootURI,
		WorkspaceFolders:      workspaceFolders,
		InitializationOptions: serverConfig.InitializationOptions,
		Capabilities:          c.buildClientCapabilities(),
		Trace:                 "off",
	}

	// Log initialize params
	initParamsJson, _ := json.MarshalIndent(initParams, "", "  ")
	logger.Debugf("[DEBUG] initialize params: %s", string(initParamsJson))

	// Save initialize params
	session.InitializeParams = initParams

	// Add debug logs
	logger.Debugf("[DEBUG] starting LSP session initialization: language=%s, root=%s", session.Key.LanguageID, session.Key.RootURI)

	// Send initialize request - timeout 60s because some LSP servers start slowly
	ctx, cancel := context.WithTimeout(c.ctx, 60*time.Second)
	defer cancel()

	logger.Debugf("[DEBUG] sending initialize request...")
	conn := session.Conn.(*LSPConnection)
	result, err := conn.Call(ctx, "initialize", initParams)
	if err != nil {
		logger.Errorf("[ERROR] failed to send initialize request: %v", err)
		return fmt.Errorf("failed to send initialize request: %w", err)
	}

	// Parse server capabilities
	var initResult struct {
		Capabilities json.RawMessage `json:"capabilities"`
		ServerInfo   struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"serverInfo"`
	}

	if err := json.Unmarshal(result, &initResult); err != nil {
		logger.Errorf("[ERROR] failed to parse initialize result: %v", err)
		// Continue even if parsing fails; some servers return non-standard format
	} else {
		logger.Debugf("[DEBUG] server info: %s %s", initResult.ServerInfo.Name, initResult.ServerInfo.Version)
	}

	logger.Debugf("[DEBUG] initialize request succeeded, response: %s", string(result))
	logger.Debugf("[DEBUG] sending initialized notification...")

	// Send initialized notification
	err = conn.Notify(ctx, "initialized", map[string]interface{}{})
	if err != nil {
		logger.Errorf("[ERROR] failed to send initialized notification: %v", err)
		return fmt.Errorf("failed to send initialized notification: %w", err)
	}

	logger.Debugf("[DEBUG] LSP session initialization complete")
	session.IsInitialized = true
	return nil
}

// buildClientCapabilities builds client capabilities
func (c *Client) buildClientCapabilities() types.LSPClientCapabilities {
	return types.LSPClientCapabilities{
		TextDocument: &types.LSPTextDocumentClientCapabilities{
			Definition: &types.LSPDefinitionClientCapabilities{
				DynamicRegistration: false,
				LinkSupport:         true,
			},
			Declaration: &types.LSPDeclarationClientCapabilities{
				DynamicRegistration: false,
				LinkSupport:         true,
			},
			TypeDefinition: &types.LSPTypeDefinitionClientCapabilities{
				DynamicRegistration: false,
				LinkSupport:         true,
			},
			Implementation: &types.LSPImplementationClientCapabilities{
				DynamicRegistration: false,
				LinkSupport:         true,
			},
			References: &types.LSPReferencesClientCapabilities{
				DynamicRegistration: false,
			},
			Hover: &types.LSPHoverClientCapabilities{
				DynamicRegistration: false,
			},
			Completion: &types.LSPCompletionClientCapabilities{
				DynamicRegistration: false,
			},
			SignatureHelp: &types.LSPSignatureHelpClientCapabilities{
				DynamicRegistration: false,
			},
			DocumentSymbol: &types.LSPDocumentSymbolClientCapabilities{
				DynamicRegistration: false,
			},
		},
		Workspace: &types.LSPWorkspaceClientCapabilities{
			ApplyEdit: true,
			WorkspaceEdit: &types.LSPWorkspaceEditClientCapabilities{
				DocumentChanges: true,
			},
			DidChangeConfiguration: &types.LSPDidChangeConfigurationClientCapabilities{
				DynamicRegistration: false,
			},
			DidChangeWatchedFiles: &types.LSPDidChangeWatchedFilesClientCapabilities{
				DynamicRegistration: false,
			},
			Symbol: &types.LSPWorkspaceSymbolClientCapabilities{
				DynamicRegistration: false,
			},
			ExecuteCommand: &types.LSPExecuteCommandClientCapabilities{
				DynamicRegistration: false,
			},
			WorkspaceFolders: true,
			Configuration:    true,
		},
		Window: &types.LSPWindowClientCapabilities{
			WorkDoneProgress: true,
			ShowMessage: &types.LSPShowMessageRequestClientCapabilities{
				MessageActionItem: &types.LSPMessageActionItemCapabilities{
					AdditionalPropertiesSupport: false,
				},
			},
			ShowDocument: &types.LSPShowDocumentClientCapabilities{
				Support: true,
			},
		},
		General: &types.LSPGeneralClientCapabilities{
			RegularExpressions: &types.LSPRegularExpressionsClientCapabilities{
				Engine:  "ECMAScript",
				Version: "ES2020",
			},
			Markdown: &types.LSPMarkdownClientCapabilities{
				Parser:  "marked",
				Version: "1.1.0",
			},
			PositionEncodings: []string{"utf-16"},
		},
	}
}

// closeSession closes a session
func (c *Client) closeSession(session *types.LSPSession) {
	if session.Conn != nil {
		// Send shutdown request
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn := session.Conn.(*LSPConnection)

		// Try sending shutdown request; ignore errors
		_, _ = conn.Call(ctx, "shutdown", nil)

		// Try sending exit notification; ignore errors
		_ = conn.Notify(ctx, "exit", nil)

		// Close connection
		conn.Close()
	}

	// Terminate process
	if cmd, ok := session.Process.(*exec.Cmd); ok && cmd.Process != nil {
		cmd.Process.Kill()
		cmd.Wait()
	}
}

// OpenFile opens a file
func (c *Client) OpenFile(ctx context.Context, req *types.OpenFileRequest) error {
	// Get or create session
	session, err := c.GetOrCreateSession(req.LanguageID, req.RootURI)
	if err != nil {
		return fmt.Errorf("failed to get or create session: %w", err)
	}

	// Ensure file is open
	conn := session.Conn.(*LSPConnection)
	return c.ensureFileOpen(ctx, conn, req.FileURI, req.LanguageID, req.RootURI)
}

// Close closes the LSP client
func (c *Client) Close() error {
	c.cancel()

	c.sessionsMutex.Lock()
	defer c.sessionsMutex.Unlock()

	// Close all sessions
	for _, session := range c.sessions {
		c.closeSession(session)
	}

	c.sessions = make(map[string]*types.LSPSession)
	return nil
}

// GetSessionCount returns current session count
func (c *Client) GetSessionCount() int {
	c.sessionsMutex.RLock()
	defer c.sessionsMutex.RUnlock()
	return len(c.sessions)
}

// GetSessionInfo returns session info
func (c *Client) GetSessionInfo() map[string]interface{} {
	c.sessionsMutex.RLock()
	defer c.sessionsMutex.RUnlock()

	info := make(map[string]interface{})
	info["total_sessions"] = len(c.sessions)
	info["max_sessions"] = c.config.Session.MaxSessions

	sessions := make([]map[string]interface{}, 0, len(c.sessions))
	for _, session := range c.sessions {
		sessionInfo := map[string]interface{}{
			"language_id":    session.Key.LanguageID,
			"root_uri":       session.Key.RootURI,
			"created_at":     session.CreatedAt,
			"last_used_at":   session.LastUsedAt,
			"is_initialized": session.IsInitialized,
		}
		sessions = append(sessions, sessionInfo)
	}
	info["sessions"] = sessions

	return info
}

// ensureFileOpen ensures the file was opened via didOpen
func (c *Client) ensureFileOpen(ctx context.Context, conn *LSPConnection, fileURI, languageID, rootURI string) error {
	sessionKey := types.SessionKey{
		LanguageID: languageID,
		RootURI:    rootURI,
	}.String()

	c.openedFilesMutex.Lock()
	defer c.openedFilesMutex.Unlock()

	if sessionFiles, exists := c.openedFiles[sessionKey]; exists {
		if sessionFiles[fileURI] {
			return nil
		}
	} else {
		c.openedFiles[sessionKey] = make(map[string]bool)
	}

	// Read content from local file only
	filePath := strings.TrimPrefix(fileURI, "file://")
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	didOpenParams := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":        fileURI,
			"languageId": languageID,
			"version":    1,
			"text":       string(content),
		},
	}

	didOpenParamsJson, _ := json.MarshalIndent(didOpenParams, "", "  ")
	logger.Debugf("[DEBUG] didOpen params: %s", string(didOpenParamsJson))

	logger.Debugf("[DEBUG] sending didOpen notification: %s", fileURI)
	err = conn.Notify(ctx, "textDocument/didOpen", didOpenParams)
	if err != nil {
		return fmt.Errorf("failed to send didOpen notification: %w", err)
	}

	c.openedFiles[sessionKey][fileURI] = true
	logger.Debugf("[DEBUG] file marked as open: %s", fileURI)

	time.Sleep(1 * time.Second)

	return nil
}
