// Package lsp provides LSP client functionality.
// Implements LSP client communication, session management, and JSON-RPC protocol handling.
package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"html"

	"github.com/mickeyinfoshan/lsp-mcp/internal/config"
	"github.com/mickeyinfoshan/lsp-mcp/pkg/types"
	protocol "go.lsp.dev/protocol"
)

// MessageID represents a JSON-RPC ID which can be a string, number, or null
type MessageID struct {
	// The actual value of the ID, can be int32, string, or nil
	Value any
}

// MarshalJSON implements custom JSON marshaling for MessageID
// Returns: JSON bytes and error.
func (id *MessageID) MarshalJSON() ([]byte, error) {
	if id == nil || id.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(id.Value)
}

// UnmarshalJSON implements custom JSON unmarshaling for MessageID
// Parameters: data - JSON bytes.
// Returns: error.
func (id *MessageID) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		id.Value = nil
		return nil
	}

	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	// Convert float64 (default JSON number type) to int32 for backward compatibility
	if num, ok := value.(float64); ok {
		id.Value = int32(num)
	} else {
		id.Value = value
	}

	return nil
}

// String returns a string representation of the ID
// Returns: ID in string form.
func (id *MessageID) String() string {
	if id == nil || id.Value == nil {
		return "<null>"
	}

	switch v := id.Value.(type) {
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Message represents a JSON-RPC 2.0 message
type Message struct {
	// JSONRPC version, always "2.0"
	JSONRPC string `json:"jsonrpc"`
	// ID of the message, optional
	ID *MessageID `json:"id,omitempty"`
	// Name of the method
	Method string `json:"method,omitempty"`
	// Parameters of the method
	Params json.RawMessage `json:"params,omitempty"`
	// Result of the method call
	Result json.RawMessage `json:"result,omitempty"`
	// Error information
	Error *ResponseError `json:"error,omitempty"`
}

// ResponseError represents a JSON-RPC 2.0 error
type ResponseError struct {
	// Error code
	Code int `json:"code"`
	// Error message
	Message string `json:"message"`
}

// LSPConnection is a manually implemented LSP connection
// Manages process, IO streams, request/response, and notifications with LSP server
type LSPConnection struct {
	// stdin is the LSP process standard input
	stdin io.WriteCloser
	// stdout is the LSP process standard output
	stdout *bufio.Reader
	// stderr is the LSP process standard error
	stderr io.ReadCloser
	// cmd is the LSP process
	cmd *exec.Cmd

	// Request ID counter
	nextID atomic.Int32

	// Map of response handlers
	handlers   map[string]chan *Message
	handlersMu sync.RWMutex

	// Map of notification handlers
	notificationHandlers map[string]NotificationHandler
	notificationMu       sync.RWMutex

	// Map of server request handlers
	serverRequestHandlers map[string]ServerRequestHandler
	serverHandlersMu      sync.RWMutex
}

// NotificationHandler is a function type for handling notifications
type NotificationHandler func(params json.RawMessage)

// ServerRequestHandler is a function type for handling server requests
type ServerRequestHandler func(params json.RawMessage) (any, error)

// NewLSPConnection creates a new LSP connection
//
//	cmd - LSP process
//	stdin - standard input stream
//	stdout - standard output stream
//
// Returns: LSPConnection instance.
func NewLSPConnection(cmd *exec.Cmd, stdin io.WriteCloser, stdout io.ReadCloser, stderr io.ReadCloser) *LSPConnection {
	conn := &LSPConnection{
		cmd:                   cmd,
		stdin:                 stdin,
		stdout:                bufio.NewReader(stdout),
		stderr:                stderr,
		handlers:              make(map[string]chan *Message),
		notificationHandlers:  make(map[string]NotificationHandler),
		serverRequestHandlers: make(map[string]ServerRequestHandler),
	}

	// Register workspace/configuration handler, return empty config array
	conn.RegisterServerRequestHandler("workspace/configuration", func(params json.RawMessage) (any, error) {
		log.Printf("[LSP-PROTO] workspace/configuration called, params: %s", string(params))
		return []interface{}{}, nil
	})

	// Register window/workDoneProgress/create handler, return empty response
	conn.RegisterServerRequestHandler("window/workDoneProgress/create", func(params json.RawMessage) (any, error) {
		log.Printf("[LSP-PROTO] window/workDoneProgress/create called, params: %s", string(params))
		return nil, nil
	})

	// Register workspace/didChangeConfiguration handler, return empty response
	conn.RegisterServerRequestHandler("workspace/didChangeConfiguration", func(params json.RawMessage) (any, error) {
		log.Printf("[LSP-PROTO] workspace/didChangeConfiguration called, params: %s", string(params))
		return nil, nil
	})

	// Register workspace/didChangeWorkspaceFolders handler, return empty response
	conn.RegisterServerRequestHandler("workspace/didChangeWorkspaceFolders", func(params json.RawMessage) (any, error) {
		log.Printf("[LSP-PROTO] workspace/didChangeWorkspaceFolders called, params: %s", string(params))
		return nil, nil
	})

	// Register client/registerCapability handler, return empty response
	conn.RegisterServerRequestHandler("client/registerCapability", func(params json.RawMessage) (any, error) {
		log.Printf("[LSP-PROTO] client/registerCapability called, params: %s", string(params))
		return nil, nil
	})

	// Register client/unregisterCapability handler, return empty response
	conn.RegisterServerRequestHandler("client/unregisterCapability", func(params json.RawMessage) (any, error) {
		log.Printf("[LSP-PROTO] client/unregisterCapability called, params: %s", string(params))
		return nil, nil
	})

	// Register window/showMessageRequest handler, return first action or nil
	conn.RegisterServerRequestHandler("window/showMessageRequest", func(params json.RawMessage) (any, error) {
		log.Printf("[LSP-PROTO] window/showMessageRequest called, params: %s", string(params))
		// Default to the first action
		var req struct {
			Actions []struct {
				Title string `json:"title"`
			} `json:"actions"`
		}
		_ = json.Unmarshal(params, &req)
		if len(req.Actions) > 0 {
			return map[string]interface{}{"title": req.Actions[0].Title}, nil
		}
		return nil, nil
	})

	// Register common notification handlers
	conn.RegisterNotificationHandler("$/progress", func(params json.RawMessage) {
		log.Printf("[LSP-NOTIFY] $/progress: %s", string(params))
	})
	conn.RegisterNotificationHandler("window/logMessage", func(params json.RawMessage) {
		log.Printf("[LSP-NOTIFY] window/logMessage: %s", string(params))
	})
	conn.RegisterNotificationHandler("textDocument/publishDiagnostics", func(params json.RawMessage) {
		log.Printf("[LSP-NOTIFY] textDocument/publishDiagnostics: %s", string(params))
	})

	// Pipe stderr logs, consistent with test script
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Printf("[STDERR] %s", scanner.Text())
		}
	}()

	// Start message handling goroutine
	go conn.handleMessages()

	return conn
}

// ReadMessage reads a complete message per the LSP protocol
func (conn *LSPConnection) ReadMessage() (*Message, error) {
	// Read header
	var contentLength int
	for {
		line, err := conn.stdout.ReadString('\n')
		log.Printf("[LSP-IO] Read header line: %q, err: %v", line, err)
		if err != nil {
			return nil, fmt.Errorf("failed to read header: %w", err)
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "Content-Length:") {
			lengthStr := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			contentLength, err = strconv.Atoi(lengthStr)
			if err != nil {
				return nil, fmt.Errorf("invalid Content-Length: %w", err)
			}
		}
	}
	if contentLength == 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}
	// Read content
	content := make([]byte, contentLength)
	readN, err := io.ReadFull(conn.stdout, content)
	log.Printf("[LSP-IO] Read content bytes: %d, err: %v", readN, err)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}
	// Parse JSON
	var msg Message
	if err := json.Unmarshal(content, &msg); err != nil {
		log.Printf("[LSP-IO] Unmarshal message error: %v", err)
		return nil, fmt.Errorf("failed to parse message: %w", err)
	}
	log.Printf("[LSP-IO] Read message: %s", string(content))
	return &msg, nil
}

// handleMessages handles incoming messages in a loop
func (conn *LSPConnection) handleMessages() {
	for {
		msg, err := conn.ReadMessage()
		if err != nil {
			// Check if this is due to normal shutdown (EOF when closing connection)
			if strings.Contains(err.Error(), "EOF") {
				log.Printf("[DEBUG] LSP connection closed (EOF)")
			} else {
				log.Printf("[DEBUG] Error reading message: %v", err)
			}
			// Clean up all pending handlers
			conn.handlersMu.Lock()
			for _, ch := range conn.handlers {
				close(ch)
			}
			conn.handlers = make(map[string]chan *Message)
			conn.handlersMu.Unlock()
			return
		}

		// Handle server->client request (has both Method and ID)
		if msg.Method != "" && msg.ID != nil && msg.ID.Value != nil {
			conn.handleServerRequest(msg)
			continue
		}

		// Handle notification (has Method but no ID)
		if msg.Method != "" && (msg.ID == nil || msg.ID.Value == nil) {
			conn.handleNotification(msg)
			continue
		}

		// Handle response to our request (has ID but no Method)
		if msg.ID != nil && msg.ID.Value != nil && msg.Method == "" {
			conn.handleResponse(msg)
		}
	}
}

// handleResponse handles responses to client requests
func (conn *LSPConnection) handleResponse(msg *Message) {
	if msg.ID == nil {
		return
	}

	idStr := msg.ID.String()
	conn.handlersMu.RLock()
	ch, exists := conn.handlers[idStr]
	conn.handlersMu.RUnlock()

	if !exists {
		log.Printf("[DEBUG] Received response for unknown request ID: %s", idStr)
		return
	}

	log.Printf("[DEBUG] Sending response for ID %v to handler", msg.ID)
	ch <- msg
	close(ch)
}

// handleNotification handles server-to-client notifications
func (conn *LSPConnection) handleNotification(msg *Message) {
	conn.notificationMu.RLock()
	handler, exists := conn.notificationHandlers[msg.Method]
	conn.notificationMu.RUnlock()

	if exists {
		go handler(msg.Params)
	} else {
		log.Printf("[DEBUG] Unhandled notification: %s", msg.Method)
	}
}

// handleServerRequest handles server-to-client requests
func (conn *LSPConnection) handleServerRequest(msg *Message) {
	conn.serverHandlersMu.RLock()
	handler, exists := conn.serverRequestHandlers[msg.Method]
	conn.serverHandlersMu.RUnlock()

	if !exists {
		// Send error response
		errorResp := &Message{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &ResponseError{
				Code:    -32601, // Method not found
				Message: fmt.Sprintf("Method not found: %s", msg.Method),
			},
		}
		conn.sendMessage(errorResp)
		return
	}

	// Handle request in goroutine
	go func() {
		result, err := handler(msg.Params)
		var resp *Message
		if err != nil {
			resp = &Message{
				JSONRPC: "2.0",
				ID:      msg.ID,
				Error: &ResponseError{
					Code:    -32603, // Internal error
					Message: err.Error(),
				},
			}
		} else {
			resultBytes, _ := json.Marshal(result)
			resp = &Message{
				JSONRPC: "2.0",
				ID:      msg.ID,
				Result:  resultBytes,
			}
		}
		conn.sendMessage(resp)
	}()
}

// sendMessage sends a message to the LSP server
func (conn *LSPConnection) sendMessage(msg *Message) error {
	// Serialize message
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[LSP-IO] Marshal message error: %v", err)
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Build LSP message with headers
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(msgBytes))
	fullMessage := header + string(msgBytes)

	log.Printf("[LSP-IO] Write message: %s", string(msgBytes))
	// Send message
	written, err := conn.stdin.Write([]byte(fullMessage))
	log.Printf("[LSP-IO] Write bytes: %d, err: %v", written, err)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	// Flush stdin if supported
	if flusher, ok := conn.stdin.(interface{ Flush() error }); ok {
		err := flusher.Flush()
		log.Printf("[LSP-IO] Flush stdin: %v", err)
	}

	return nil
}

// Call sends a request and waits for a response
func (conn *LSPConnection) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	// Generate unique ID
	id := conn.nextID.Add(1)
	msgID := &MessageID{Value: id}

	log.Printf("[DEBUG] Making call: method=%s id=%v", method, id)

	// Create response channel
	respCh := make(chan *Message, 1)
	idStr := msgID.String()

	// Register handler
	conn.handlersMu.Lock()
	conn.handlers[idStr] = respCh
	conn.handlersMu.Unlock()

	// Cleanup on exit
	defer func() {
		conn.handlersMu.Lock()
		delete(conn.handlers, idStr)
		conn.handlersMu.Unlock()
	}()

	// Marshal params
	var paramsBytes json.RawMessage
	if params != nil {
		var err error
		paramsBytes, err = json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
	}

	// Create request message
	req := &Message{
		JSONRPC: "2.0",
		ID:      msgID,
		Method:  method,
		Params:  paramsBytes,
	}

	// Send request
	if err := conn.sendMessage(req); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	log.Printf("[DEBUG] Waiting for response to request ID: %v", msgID)

	// Wait for response
	select {
	case resp := <-respCh:
		log.Printf("[DEBUG] Received response for request ID: %v", msgID)
		if resp == nil {
			return nil, fmt.Errorf("received nil response")
		}
		if resp.Error != nil {
			log.Printf("[ERROR] Request failed: %s (code: %d)", resp.Error.Message, resp.Error.Code)
			return nil, fmt.Errorf("request failed: %s (code: %d)", resp.Error.Message, resp.Error.Code)
		}
		return resp.Result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Notify sends a notification (no response expected)
func (conn *LSPConnection) Notify(ctx context.Context, method string, params any) error {
	// Marshal params
	var paramsBytes json.RawMessage
	if params != nil {
		var err error
		paramsBytes, err = json.Marshal(params)
		if err != nil {
			return fmt.Errorf("failed to marshal params: %w", err)
		}
	}

	// Create notification message
	notif := &Message{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsBytes,
	}

	// Send notification
	return conn.sendMessage(notif)
}

// RegisterNotificationHandler registers a handler for server-to-client notifications
func (conn *LSPConnection) RegisterNotificationHandler(method string, handler NotificationHandler) {
	conn.notificationMu.Lock()
	conn.notificationHandlers[method] = handler
	conn.notificationMu.Unlock()
}

// RegisterServerRequestHandler registers a handler for server-to-client requests
func (conn *LSPConnection) RegisterServerRequestHandler(method string, handler ServerRequestHandler) {
	conn.serverHandlersMu.Lock()
	conn.serverRequestHandlers[method] = handler
	conn.serverHandlersMu.Unlock()
}

// Close closes the connection
func (conn *LSPConnection) Close() error {
	log.Printf("[LSP-IO] Closing LSPConnection...")
	// Close all pending handlers
	conn.handlersMu.Lock()
	for _, ch := range conn.handlers {
		close(ch)
	}
	conn.handlers = make(map[string]chan *Message)
	conn.handlersMu.Unlock()

	// Close streams
	var errs []error
	if conn.stdin != nil {
		log.Printf("[LSP-IO] Closing stdin...")
		if err := conn.stdin.Close(); err != nil {
			log.Printf("[LSP-IO] Close stdin error: %v", err)
			errs = append(errs, err)
		}
	}
	if conn.stderr != nil {
		log.Printf("[LSP-IO] Closing stderr...")
		if err := conn.stderr.Close(); err != nil {
			log.Printf("[LSP-IO] Close stderr error: %v", err)
			errs = append(errs, err)
		}
	}

	// Wait for process to exit
	if conn.cmd != nil {
		log.Printf("[LSP-IO] Waiting for process to exit...")
		if err := conn.cmd.Wait(); err != nil {
			log.Printf("[LSP-IO] cmd.Wait error: %v", err)
			errs = append(errs, err)
		}
		if conn.cmd.ProcessState != nil {
			log.Printf("[LSP-IO] ProcessState: %v", conn.cmd.ProcessState)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	log.Printf("[LSP-IO] LSPConnection closed.")
	return nil
}

// GetServerInfo returns server info
func (conn *LSPConnection) GetServerInfo() map[string]interface{} {
	info := make(map[string]interface{})
	info["connection_type"] = "lsp"
	info["next_id"] = conn.nextID.Load()
	info["active_handlers"] = len(conn.handlers)
	info["notification_handlers"] = len(conn.notificationHandlers)
	info["server_request_handlers"] = len(conn.serverRequestHandlers)

	if conn.cmd != nil {
		info["command"] = conn.cmd.Path
		info["args"] = conn.cmd.Args
		if conn.cmd.Process != nil {
			info["pid"] = conn.cmd.Process.Pid
		}
	}

	return info
}

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

	log.Printf("[DEBUG] starting LSP server: command=%s, args=%v", serverConfig.Command, serverConfig.Args)

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
		RootURI:               &session.Key.RootURI, // RootURI is commented out; only pass workspaceFolders
		WorkspaceFolders:      workspaceFolders,
		InitializationOptions: serverConfig.InitializationOptions,
		Capabilities:          c.buildClientCapabilities(),
		Trace:                 "off",
	}

	// Log initialize params
	initParamsJson, _ := json.MarshalIndent(initParams, "", "  ")
	log.Printf("[DEBUG] initialize params: %s", string(initParamsJson))

	// Save initialize params
	session.InitializeParams = initParams

	// Add debug logs
	log.Printf("[DEBUG] starting LSP session initialization: language=%s, root=%s", session.Key.LanguageID, session.Key.RootURI)

	// Send initialize request - timeout 60s because some LSP servers start slowly
	ctx, cancel := context.WithTimeout(c.ctx, 60*time.Second)
	defer cancel()

	log.Printf("[DEBUG] sending initialize request...")
	conn := session.Conn.(*LSPConnection)
	result, err := conn.Call(ctx, "initialize", initParams)
	if err != nil {
		log.Printf("[ERROR] failed to send initialize request: %v", err)
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
		log.Printf("[ERROR] failed to parse initialize result: %v", err)
		// Continue even if parsing fails; some servers return non-standard format
	} else {
		log.Printf("[DEBUG] server info: %s %s", initResult.ServerInfo.Name, initResult.ServerInfo.Version)
	}

	log.Printf("[DEBUG] initialize request succeeded, response: %s", string(result))
	log.Printf("[DEBUG] sending initialized notification...")

	// Send initialized notification
	err = conn.Notify(ctx, "initialized", map[string]interface{}{})
	if err != nil {
		log.Printf("[ERROR] failed to send initialized notification: %v", err)
		return fmt.Errorf("failed to send initialized notification: %w", err)
	}

	log.Printf("[DEBUG] LSP session initialization complete")
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

// FindDefinition finds definitions
func (c *Client) FindDefinition(ctx context.Context, req *types.FindDefinitionRequest) (*types.FindDefinitionResponse, error) {
	// Get or create session
	session, err := c.GetOrCreateSession(req.LanguageID, req.RootURI)
	if err != nil {
		return &types.FindDefinitionResponse{
			Error: fmt.Sprintf("failed to get LSP session: %v", err),
		}, nil
	}

	// Check if session is initialized
	if !session.IsInitialized {
		return &types.FindDefinitionResponse{
			Error: "LSP session not initialized",
		}, nil
	}

	// Send didOpen notification (if file is not open)
	conn := session.Conn.(*LSPConnection)
	err = c.ensureFileOpen(ctx, conn, req.FileURI, req.LanguageID, req.RootURI)
	if err != nil {
		return &types.FindDefinitionResponse{
			Error: fmt.Sprintf("failed to open file: %v", err),
		}, nil
	}

	// Build LSP request params
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": req.FileURI,
		},
		"position": map[string]interface{}{
			"line":      req.Position.Line,
			"character": req.Position.Character,
		},
	}

	// Log definition request params
	paramsJson, _ := json.MarshalIndent(params, "", "  ")
	log.Printf("[DEBUG] definition request params: %s", string(paramsJson))

	// Set timeout to 30s because gopls may need more time
	requestCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Send definition request
	result, err := conn.Call(requestCtx, "textDocument/definition", params)
	if err != nil {
		return &types.FindDefinitionResponse{
			Error: fmt.Sprintf("failed to send definition request: %v", err),
		}, nil
	}

	// Parse response
	response, err := c.parseDefinitionResponse(result)
	if err != nil {
		return &types.FindDefinitionResponse{
			Error: fmt.Sprintf("failed to parse definition response: %v", err),
		}, nil
	}

	// Update session last used time
	session.UpdateLastUsed()

	return response, nil
}

// FindReferences finds references
func (c *Client) FindReferences(ctx context.Context, req *types.FindReferencesRequest) (*types.FindReferencesResponse, error) {
	// Get or create session
	session, err := c.GetOrCreateSession(req.LanguageID, req.RootURI)
	if err != nil {
		return &types.FindReferencesResponse{
			Error: fmt.Sprintf("failed to get LSP session: %v", err),
		}, nil
	}

	// Check if session is initialized
	if !session.IsInitialized {
		return &types.FindReferencesResponse{
			Error: "LSP session not initialized",
		}, nil
	}

	// Send didOpen notification (if file is not open)
	conn := session.Conn.(*LSPConnection)
	err = c.ensureFileOpen(ctx, conn, req.FileURI, req.LanguageID, req.RootURI)
	if err != nil {
		return &types.FindReferencesResponse{
			Error: fmt.Sprintf("failed to open file: %v", err),
		}, nil
	}

	// Build LSP request params
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": req.FileURI,
		},
		"position": map[string]interface{}{
			"line":      req.Position.Line,
			"character": req.Position.Character,
		},
		"context": map[string]interface{}{
			"includeDeclaration": req.IncludeDeclaration,
		},
	}

	// Set timeout to 30s because gopls may need more time
	requestCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Send references request
	result, err := conn.Call(requestCtx, "textDocument/references", params)
	if err != nil {
		return &types.FindReferencesResponse{
			Error: fmt.Sprintf("failed to send references request: %v", err),
		}, nil
	}

	// Parse response
	response, err := c.parseReferencesResponse(result)
	if err != nil {
		return &types.FindReferencesResponse{
			Error: fmt.Sprintf("failed to parse references response: %v", err),
		}, nil
	}

	// Update session last used time
	session.UpdateLastUsed()

	return response, nil
}

// GetHover gets hover info
func (c *Client) GetHover(ctx context.Context, req *types.HoverRequest) (*types.HoverResponse, error) {
	// Get or create session
	session, err := c.GetOrCreateSession(req.LanguageID, req.RootURI)
	if err != nil {
		return &types.HoverResponse{
			Error: fmt.Sprintf("failed to get LSP session: %v", err),
		}, nil
	}

	// Check if session is initialized
	if !session.IsInitialized {
		return &types.HoverResponse{
			Error: "LSP session not initialized",
		}, nil
	}

	// Send didOpen notification (if file is not open)
	conn := session.Conn.(*LSPConnection)
	err = c.ensureFileOpen(ctx, conn, req.FileURI, req.LanguageID, req.RootURI)
	if err != nil {
		return &types.HoverResponse{
			Error: fmt.Sprintf("failed to open file: %v", err),
		}, nil
	}

	// Build LSP request params
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": req.FileURI,
		},
		"position": map[string]interface{}{
			"line":      req.Position.Line,
			"character": req.Position.Character,
		},
	}

	// Set timeout to 30s because gopls may need more time
	requestCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Send hover request
	result, err := conn.Call(requestCtx, "textDocument/hover", params)
	if err != nil {
		return &types.HoverResponse{
			Error: fmt.Sprintf("failed to send hover request: %v", err),
		}, nil
	}

	// Parse response
	response, err := c.parseHoverResponse(result)
	if err != nil {
		return &types.HoverResponse{
			Error: fmt.Sprintf("failed to parse hover response: %v", err),
		}, nil
	}

	// Update session last used time
	session.UpdateLastUsed()

	return response, nil
}

// GetCompletion gets completions
func (c *Client) GetCompletion(ctx context.Context, req *types.CompletionRequest) (*types.CompletionResponse, error) {
	// Get or create session
	session, err := c.GetOrCreateSession(req.LanguageID, req.RootURI)
	if err != nil {
		return &types.CompletionResponse{
			Error: fmt.Sprintf("failed to get LSP session: %v", err),
		}, nil
	}

	// Check if session is initialized
	if !session.IsInitialized {
		return &types.CompletionResponse{
			Error: "LSP session not initialized",
		}, nil
	}

	// Send didOpen notification (if file is not open)
	conn := session.Conn.(*LSPConnection)
	err = c.ensureFileOpen(ctx, conn, req.FileURI, req.LanguageID, req.RootURI)
	if err != nil {
		return &types.CompletionResponse{
			Error: fmt.Sprintf("failed to open file: %v", err),
		}, nil
	}

	// Build LSP request params
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": req.FileURI,
		},
		"position": map[string]interface{}{
			"line":      req.Position.Line,
			"character": req.Position.Character,
		},
	}

	// Add context info (if any)
	if req.TriggerKind > 0 {
		params["context"] = map[string]interface{}{
			"triggerKind": req.TriggerKind,
		}
		if req.TriggerCharacter != "" {
			params["context"].(map[string]interface{})["triggerCharacter"] = req.TriggerCharacter
		}
	}

	// Set timeout to 30s because gopls may need more time
	requestCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Send completion request
	result, err := conn.Call(requestCtx, "textDocument/completion", params)
	if err != nil {
		return &types.CompletionResponse{
			Error: fmt.Sprintf("failed to send completion request: %v", err),
		}, nil
	}

	// Parse response
	response, err := c.parseCompletionResponse(result)
	if err != nil {
		return &types.CompletionResponse{
			Error: fmt.Sprintf("failed to parse completion response: %v", err),
		}, nil
	}

	// Update session last used time
	session.UpdateLastUsed()

	return response, nil
}

// parseDefinitionResponse parses definition response
func (c *Client) parseDefinitionResponse(result json.RawMessage) (*types.FindDefinitionResponse, error) {
	response := &types.FindDefinitionResponse{}

	// First try parsing as an array
	var rawArr []json.RawMessage
	if err := json.Unmarshal(result, &rawArr); err == nil {
		if len(rawArr) == 0 {
			// Empty array is valid; return empty response
			response.Message = "definition not found."
			return response, nil
		}
		var probe map[string]interface{}
		if err := json.Unmarshal(rawArr[0], &probe); err == nil {
			if _, ok := probe["targetUri"]; ok {
				// Is LocationLink
				var locationLinks []protocol.LocationLink
				if err := json.Unmarshal(result, &locationLinks); err == nil {
					response.LocationLinks = locationLinks
					// Agent-friendly structured output
					for _, link := range locationLinks {
						file, line, char := parseFileLineChar(link.TargetURI, link.TargetSelectionRange)
						summary := formatSummary(file, line, char)
						loc := protocol.Location{URI: link.TargetURI, Range: link.TargetSelectionRange}
						response.AgentResults = append(response.AgentResults, types.AgentDefinitionResult{
							Type:      "location_link",
							File:      file,
							Line:      line,
							Character: char,
							Summary:   summary,
							Range:     &link.TargetSelectionRange,
							Location:  &loc,
						})
					}
					response.Message = formatAgentMessage(len(response.AgentResults), "definition")
					return response, nil
				}
			} else if _, ok := probe["uri"]; ok {
				// Is Location
				var locations []protocol.Location
				if err := json.Unmarshal(result, &locations); err == nil {
					response.Locations = locations
					for _, loc := range locations {
						file, line, char := parseFileLineChar(loc.URI, loc.Range)
						summary := formatSummary(file, line, char)
						response.AgentResults = append(response.AgentResults, types.AgentDefinitionResult{
							Type:      "location",
							File:      file,
							Line:      line,
							Character: char,
							Summary:   summary,
							Range:     &loc.Range,
							Location:  &loc,
						})
					}
					response.Message = formatAgentMessage(len(response.AgentResults), "definition")
					return response, nil
				}
			}
		}
	}

	// Try parsing as a single Location
	var location protocol.Location
	if err := json.Unmarshal(result, &location); err == nil && location.URI != "" {
		response.Locations = []protocol.Location{location}
		file, line, char := parseFileLineChar(location.URI, location.Range)
		summary := formatSummary(file, line, char)
		response.AgentResults = []types.AgentDefinitionResult{{
			Type:      "location",
			File:      file,
			Line:      line,
			Character: char,
			Summary:   summary,
			Range:     &location.Range,
			Location:  &location,
		}}
		response.Message = formatAgentMessage(1, "definition")
		return response, nil
	}

	// Try parsing as a single LocationLink
	var locationLink protocol.LocationLink
	if err := json.Unmarshal(result, &locationLink); err == nil && locationLink.TargetURI != "" {
		response.LocationLinks = []protocol.LocationLink{locationLink}
		file, line, char := parseFileLineChar(locationLink.TargetURI, locationLink.TargetSelectionRange)
		summary := formatSummary(file, line, char)
		loc := protocol.Location{URI: locationLink.TargetURI, Range: locationLink.TargetSelectionRange}
		response.AgentResults = []types.AgentDefinitionResult{{
			Type:      "location_link",
			File:      file,
			Line:      line,
			Character: char,
			Summary:   summary,
			Range:     &locationLink.TargetSelectionRange,
			Location:  &loc,
		}}
		response.Message = formatAgentMessage(1, "definition")
		return response, nil
	}

	// If parsing fails, check for null
	if string(result) == "null" {
		response.Message = "definition not found."
		return response, nil
	}

	return nil, fmt.Errorf("failed to parse definition response: %s", string(result))
}

// parseFileLineChar parses URI and range, returns full path, line (1-based), char (1-based)
func parseFileLineChar(uri protocol.DocumentURI, rng protocol.Range) (string, int, int) {
	file := strings.TrimPrefix(string(uri), "file://")
	line := int(rng.Start.Line) + 1
	char := int(rng.Start.Character) + 1
	return file, line, char
}

// formatSummary builds an agent-friendly summary (full path)
func formatSummary(file string, line, char int) string {
	return fmt.Sprintf("Jump to %s line %d column %d", file, line, char)
}

// formatAgentMessage supports typed labels
func formatAgentMessage(count int, typ string) string {
	if count == 0 {
		return fmt.Sprintf("No %s found.", typ)
	}
	if count == 1 {
		return fmt.Sprintf("Found 1 %s.", typ)
	}
	return fmt.Sprintf("Found %d %ss.", count, typ)
}

// parseReferencesResponse parses references response
func (c *Client) parseReferencesResponse(result json.RawMessage) (*types.FindReferencesResponse, error) {
	response := &types.FindReferencesResponse{}

	// Try parsing as Location array
	var locations []protocol.Location
	if err := json.Unmarshal(result, &locations); err == nil {
		response.Locations = locations
		// Agent-friendly structured output
		for _, loc := range locations {
			file, line, char := parseFileLineChar(loc.URI, loc.Range)
			summary := fmt.Sprintf("Referenced at %s line %d column %d", file, line, char)
			response.AgentResults = append(response.AgentResults, types.AgentReferenceResult{
				Type:      "reference",
				File:      file,
				Line:      line,
				Character: char,
				Summary:   summary,
				Range:     &loc.Range,
				Location:  &loc,
			})
		}
		response.Message = formatAgentMessage(len(response.AgentResults), "reference")
		return response, nil
	}

	// If parsing fails, check for null
	if string(result) == "null" {
		response.Message = "No references found."
		return response, nil
	}

	return nil, fmt.Errorf("failed to parse references response: %s", string(result))
}

// parseHoverResponse parses hover response
func (c *Client) parseHoverResponse(result json.RawMessage) (*types.HoverResponse, error) {
	response := &types.HoverResponse{}

	// Try parsing as Hover object
	var hover struct {
		Contents interface{}     `json:"contents"`
		Range    *protocol.Range `json:"range,omitempty"`
	}

	if err := json.Unmarshal(result, &hover); err == nil {
		var (
			rawMarkdown     string
			typeSignature   string
			importStatement string
			doc             string
			summary         string
		)
		// Process contents field
		if hover.Contents != nil {
			// 1. Parse contents as kind/value
			var value string
			switch v := hover.Contents.(type) {
			case map[string]interface{}:
				if val, ok := v["value"].(string); ok {
					value = val
				}
			case []interface{}:
				// Multiple MarkedString; concatenate
				for _, item := range v {
					if m, ok := item.(map[string]interface{}); ok {
						if val, ok := m["value"].(string); ok {
							value += val + "\n"
						}
					}
				}
			case string:
				value = v
			}

			// 2. Unescape characters
			value = htmlUnescapeString(value)
			rawMarkdown = value

			// 3. Extract code block content
			re := regexp.MustCompile("```[a-zA-Z]*\\n([\\s\\S]+?)```")
			matches := re.FindStringSubmatch(value)
			if len(matches) > 1 {
				block := matches[1]
				lines := strings.Split(block, "\n")
				if len(lines) > 0 {
					typeSignature = strings.TrimSpace(lines[0])
				}
				if len(lines) > 1 {
					for _, l := range lines[1:] {
						l = strings.TrimSpace(l)
						if strings.HasPrefix(l, "import ") {
							importStatement = l
						} else if l != "" && doc == "" {
							doc = l
						}
					}
				}
			}

			// 4. Build summary and doc
			if typeSignature != "" && importStatement != "" {
				summary = "Type definition: " + typeSignature + ", can be imported via " + importStatement + "."
			} else if typeSignature != "" {
				summary = "Type definition: " + typeSignature
			} else {
				summary = value
			}
			// Prefer doc comment; fallback to typeSignature
			if doc == "" {
				doc = typeSignature
			}

			file := ""
			line, char := 0, 0
			var rng *protocol.Range
			if hover.Range != nil {
				file = "(unknown)"
				line = int(hover.Range.Start.Line) + 1
				char = int(hover.Range.Start.Character) + 1
				rng = hover.Range
			}
			response.AgentResults = []types.AgentHoverResult{{
				Summary:         summary,
				Doc:             doc,
				Type:            "hover",
				File:            file,
				Line:            line,
				Character:       char,
				Range:           rng,
				Location:        nil,
				TypeSignature:   typeSignature,
				ImportStatement: importStatement,
				RawMarkdown:     rawMarkdown,
			}}
			response.Message = summary
			response.Contents = hover.Contents
		}
		response.Range = hover.Range
		return response, nil
	}

	// If parsing fails, check for null
	if string(result) == "null" {
		response.Message = "No hover info."
		return response, nil
	}

	return nil, fmt.Errorf("failed to parse hover response: %s", string(result))
}

// htmlUnescapeString unescapes HTML entities
func htmlUnescapeString(s string) string {
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\\u003c", "<")
	s = strings.ReplaceAll(s, "\\u003e", ">")
	return s
}

// parseCompletionResponse parses completion response
func (c *Client) parseCompletionResponse(result json.RawMessage) (*types.CompletionResponse, error) {
	response := &types.CompletionResponse{}

	// Try parsing as CompletionList
	var completionList struct {
		IsIncomplete bool                      `json:"isIncomplete"`
		Items        []protocol.CompletionItem `json:"items"`
	}

	if err := json.Unmarshal(result, &completionList); err == nil {
		response.IsIncomplete = completionList.IsIncomplete
		response.Items = completionList.Items
		// Agent-friendly structured output
		for _, item := range completionList.Items {
			summary := fmt.Sprintf("Completion item: %s %s", item.Label, item.Detail)
			var textEditPtr *protocol.TextEdit
			if item.TextEdit != nil {
				te := *item.TextEdit
				textEditPtr = &te
			}
			response.AgentResults = append(response.AgentResults, types.AgentCompletionResult{
				Type:           "completion",
				File:           "",
				Line:           0,
				Character:      0,
				Summary:        summary,
				TextEdit:       textEditPtr,
				CompletionItem: item,
			})
		}
		response.Message = formatAgentMessage(len(response.AgentResults), "completion item")
		return response, nil
	}

	// Try parsing as CompletionItem array
	var items []protocol.CompletionItem
	if err := json.Unmarshal(result, &items); err == nil {
		response.Items = items
		for _, item := range items {
			summary := fmt.Sprintf("Completion item: %s %s", item.Label, item.Detail)
			var textEditPtr *protocol.TextEdit
			if item.TextEdit != nil {
				te := *item.TextEdit
				textEditPtr = &te
			}
			response.AgentResults = append(response.AgentResults, types.AgentCompletionResult{
				Type:           "completion",
				File:           "",
				Line:           0,
				Character:      0,
				Summary:        summary,
				TextEdit:       textEditPtr,
				CompletionItem: item,
			})
		}
		response.Message = formatAgentMessage(len(response.AgentResults), "completion item")
		return response, nil
	}

	// If parsing fails, check for null
	if string(result) == "null" {
		response.Message = "No completion items."
		return response, nil
	}

	return nil, fmt.Errorf("failed to parse completion response: %s", string(result))
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
	log.Printf("[DEBUG] didOpen params: %s", string(didOpenParamsJson))

	log.Printf("[DEBUG] sending didOpen notification: %s", fileURI)
	err = conn.Notify(ctx, "textDocument/didOpen", didOpenParams)
	if err != nil {
		return fmt.Errorf("failed to send didOpen notification: %w", err)
	}

	c.openedFiles[sessionKey][fileURI] = true
	log.Printf("[DEBUG] file marked as open: %s", fileURI)

	time.Sleep(1 * time.Second)

	return nil
}
