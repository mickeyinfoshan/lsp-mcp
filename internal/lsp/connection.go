package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/mickeyinfoshan/lsp-mcp/internal/logger"
)

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
		logger.Infof("[LSP-PROTO] workspace/configuration called, params: %s", string(params))
		return []interface{}{}, nil
	})

	// Register window/workDoneProgress/create handler, return empty response
	conn.RegisterServerRequestHandler("window/workDoneProgress/create", func(params json.RawMessage) (any, error) {
		logger.Infof("[LSP-PROTO] window/workDoneProgress/create called, params: %s", string(params))
		return nil, nil
	})

	// Register workspace/didChangeConfiguration handler, return empty response
	conn.RegisterServerRequestHandler("workspace/didChangeConfiguration", func(params json.RawMessage) (any, error) {
		logger.Infof("[LSP-PROTO] workspace/didChangeConfiguration called, params: %s", string(params))
		return nil, nil
	})

	// Register workspace/didChangeWorkspaceFolders handler, return empty response
	conn.RegisterServerRequestHandler("workspace/didChangeWorkspaceFolders", func(params json.RawMessage) (any, error) {
		logger.Infof("[LSP-PROTO] workspace/didChangeWorkspaceFolders called, params: %s", string(params))
		return nil, nil
	})

	// Register client/registerCapability handler, return empty response
	conn.RegisterServerRequestHandler("client/registerCapability", func(params json.RawMessage) (any, error) {
		logger.Infof("[LSP-PROTO] client/registerCapability called, params: %s", string(params))
		return nil, nil
	})

	// Register client/unregisterCapability handler, return empty response
	conn.RegisterServerRequestHandler("client/unregisterCapability", func(params json.RawMessage) (any, error) {
		logger.Infof("[LSP-PROTO] client/unregisterCapability called, params: %s", string(params))
		return nil, nil
	})

	// Register window/showMessageRequest handler, return first action or nil
	conn.RegisterServerRequestHandler("window/showMessageRequest", func(params json.RawMessage) (any, error) {
		logger.Infof("[LSP-PROTO] window/showMessageRequest called, params: %s", string(params))
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
		logger.Infof("[LSP-NOTIFY] $/progress: %s", string(params))
	})
	conn.RegisterNotificationHandler("window/logMessage", func(params json.RawMessage) {
		logger.Infof("[LSP-NOTIFY] window/logMessage: %s", string(params))
	})
	conn.RegisterNotificationHandler("textDocument/publishDiagnostics", func(params json.RawMessage) {
		logger.Infof("[LSP-NOTIFY] textDocument/publishDiagnostics: %s", string(params))
	})

	// Pipe stderr logs, consistent with test script
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			logger.Infof("[STDERR] %s", scanner.Text())
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
		logger.Infof("[LSP-IO] Read header line: %q, err: %v", line, err)
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
	logger.Infof("[LSP-IO] Read content bytes: %d, err: %v", readN, err)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}
	// Parse JSON
	var msg Message
	if err := json.Unmarshal(content, &msg); err != nil {
		logger.Infof("[LSP-IO] Unmarshal message error: %v", err)
		return nil, fmt.Errorf("failed to parse message: %w", err)
	}
	logger.Infof("[LSP-IO] Read message: %s", string(content))
	return &msg, nil
}

// handleMessages handles incoming messages in a loop
func (conn *LSPConnection) handleMessages() {
	for {
		msg, err := conn.ReadMessage()
		if err != nil {
			// Check if this is due to normal shutdown (EOF when closing connection)
			if strings.Contains(err.Error(), "EOF") {
				logger.Debugf("[DEBUG] LSP connection closed (EOF)")
			} else {
				logger.Debugf("[DEBUG] Error reading message: %v", err)
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
		logger.Debugf("[DEBUG] Received response for unknown request ID: %s", idStr)
		return
	}

	logger.Debugf("[DEBUG] Sending response for ID %v to handler", msg.ID)
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
		logger.Debugf("[DEBUG] Unhandled notification: %s", msg.Method)
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
		logger.Infof("[LSP-IO] Marshal message error: %v", err)
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Build LSP message with headers
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(msgBytes))
	fullMessage := header + string(msgBytes)

	logger.Infof("[LSP-IO] Write message: %s", string(msgBytes))
	// Send message
	written, err := conn.stdin.Write([]byte(fullMessage))
	logger.Infof("[LSP-IO] Write bytes: %d, err: %v", written, err)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	// Flush stdin if supported
	if flusher, ok := conn.stdin.(interface{ Flush() error }); ok {
		err := flusher.Flush()
		logger.Infof("[LSP-IO] Flush stdin: %v", err)
	}

	return nil
}

// Call sends a request and waits for a response
func (conn *LSPConnection) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	// Generate unique ID
	id := conn.nextID.Add(1)
	msgID := &MessageID{Value: id}

	logger.Debugf("[DEBUG] Making call: method=%s id=%v", method, id)

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

	logger.Debugf("[DEBUG] Waiting for response to request ID: %v", msgID)

	// Wait for response
	select {
	case resp := <-respCh:
		logger.Debugf("[DEBUG] Received response for request ID: %v", msgID)
		if resp == nil {
			return nil, fmt.Errorf("received nil response")
		}
		if resp.Error != nil {
			logger.Errorf("[ERROR] Request failed: %s (code: %d)", resp.Error.Message, resp.Error.Code)
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
	logger.Infof("[LSP-IO] Closing LSPConnection...")
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
		logger.Infof("[LSP-IO] Closing stdin...")
		if err := conn.stdin.Close(); err != nil {
			logger.Infof("[LSP-IO] Close stdin error: %v", err)
			errs = append(errs, err)
		}
	}
	if conn.stderr != nil {
		logger.Infof("[LSP-IO] Closing stderr...")
		if err := conn.stderr.Close(); err != nil {
			logger.Infof("[LSP-IO] Close stderr error: %v", err)
			errs = append(errs, err)
		}
	}

	// Wait for process to exit
	if conn.cmd != nil {
		logger.Infof("[LSP-IO] Waiting for process to exit...")
		if err := conn.cmd.Wait(); err != nil {
			logger.Infof("[LSP-IO] cmd.Wait error: %v", err)
			errs = append(errs, err)
		}
		if conn.cmd.ProcessState != nil {
			logger.Infof("[LSP-IO] ProcessState: %v", conn.cmd.ProcessState)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	logger.Infof("[LSP-IO] LSPConnection closed.")
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
