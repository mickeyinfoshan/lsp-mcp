// Package lsp 提供LSP客户端功能
// LSP 客户端通信与会话管理实现
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

// MessageID 表示 JSON-RPC 的 ID，可为字符串、数字或 null
// MessageID represents a JSON-RPC ID which can be a string, number, or null
type MessageID struct {
	// Value ID 的实际值，可以是 int32、string 或 nil
	// The actual value of the ID, can be int32, string, or nil
	Value any
}

// MarshalJSON 实现自定义 JSON 序列化
// MarshalJSON implements custom JSON marshaling for MessageID
// 返回值: JSON 字节数组和错误信息
func (id *MessageID) MarshalJSON() ([]byte, error) {
	if id == nil || id.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(id.Value)
}

// UnmarshalJSON 实现自定义 JSON 反序列化
// UnmarshalJSON implements custom JSON unmarshaling for MessageID
// 参数: data - JSON 字节数组
// 返回值: 错误信息
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

// String 返回 ID 的字符串表示
// String returns a string representation of the ID
// 返回值: 字符串形式的 ID
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

// Message 表示 JSON-RPC 2.0 消息
// Message represents a JSON-RPC 2.0 message
type Message struct {
	// JSONRPC 协议版本号，固定为 "2.0"
	// JSONRPC version, always "2.0"
	JSONRPC string `json:"jsonrpc"`
	// ID 消息 ID，可选
	// ID of the message, optional
	ID *MessageID `json:"id,omitempty"`
	// Method 方法名
	// Name of the method
	Method string `json:"method,omitempty"`
	// Params 参数
	// Parameters of the method
	Params json.RawMessage `json:"params,omitempty"`
	// Result 结果
	// Result of the method call
	Result json.RawMessage `json:"result,omitempty"`
	// Error 错误信息
	// Error information
	Error *ResponseError `json:"error,omitempty"`
}

// ResponseError 表示 JSON-RPC 2.0 错误
// ResponseError represents a JSON-RPC 2.0 error
type ResponseError struct {
	// Code 错误码
	// Error code
	Code int `json:"code"`
	// Message 错误描述
	// Error message
	Message string `json:"message"`
}

// LSPConnection 手动实现的 LSP 连接
// LSPConnection is a manually implemented LSP connection
// 用于管理与 LSP 服务器的进程、输入输出流、请求响应、通知等
// Manages process, IO streams, request/response, and notifications with LSP server
type LSPConnection struct {
	// stdin LSP 进程的标准输入
	stdin io.WriteCloser
	// stdout LSP 进程的标准输出
	stdout *bufio.Reader
	// stderr LSP 进程的标准错误输出
	stderr io.ReadCloser
	// cmd LSP 进程对象
	cmd *exec.Cmd

	// nextID 请求 ID 计数器
	// Request ID counter
	nextID atomic.Int32

	// handlers 响应处理通道映射
	// Map of response handlers
	handlers   map[string]chan *Message
	handlersMu sync.RWMutex

	// notificationHandlers 通知处理器映射
	// Map of notification handlers
	notificationHandlers map[string]NotificationHandler
	notificationMu       sync.RWMutex

	// serverRequestHandlers 服务器请求处理器映射
	// Map of server request handlers
	serverRequestHandlers map[string]ServerRequestHandler
	serverHandlersMu      sync.RWMutex
}

// NotificationHandler 通知处理函数类型
// NotificationHandler is a function type for handling notifications
type NotificationHandler func(params json.RawMessage)

// ServerRequestHandler 服务器请求处理函数类型
// ServerRequestHandler is a function type for handling server requests
type ServerRequestHandler func(params json.RawMessage) (any, error)

// NewLSPConnection 创建新的 LSP 连接
// NewLSPConnection creates a new LSP connection
// 参数:
//
//	cmd - LSP 进程对象
//	stdin - 标准输入流
//	stdout - 标准输出流
//	stderr - 标准错误输出流
//
// 返回值: LSPConnection 实例
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

	// 注册 workspace/configuration 响应，返回空配置数组
	conn.RegisterServerRequestHandler("workspace/configuration", func(params json.RawMessage) (any, error) {
		log.Printf("[LSP-PROTO] workspace/configuration called, params: %s", string(params))
		return []interface{}{}, nil
	})

	// 注册 window/workDoneProgress/create 响应，返回空响应
	conn.RegisterServerRequestHandler("window/workDoneProgress/create", func(params json.RawMessage) (any, error) {
		log.Printf("[LSP-PROTO] window/workDoneProgress/create called, params: %s", string(params))
		return nil, nil
	})

	// 注册 workspace/didChangeConfiguration 响应，返回空响应
	conn.RegisterServerRequestHandler("workspace/didChangeConfiguration", func(params json.RawMessage) (any, error) {
		log.Printf("[LSP-PROTO] workspace/didChangeConfiguration called, params: %s", string(params))
		return nil, nil
	})

	// 注册 workspace/didChangeWorkspaceFolders 响应，返回空响应
	conn.RegisterServerRequestHandler("workspace/didChangeWorkspaceFolders", func(params json.RawMessage) (any, error) {
		log.Printf("[LSP-PROTO] workspace/didChangeWorkspaceFolders called, params: %s", string(params))
		return nil, nil
	})

	// 注册 client/registerCapability 响应，返回空响应
	conn.RegisterServerRequestHandler("client/registerCapability", func(params json.RawMessage) (any, error) {
		log.Printf("[LSP-PROTO] client/registerCapability called, params: %s", string(params))
		return nil, nil
	})

	// 注册 client/unregisterCapability 响应，返回空响应
	conn.RegisterServerRequestHandler("client/unregisterCapability", func(params json.RawMessage) (any, error) {
		log.Printf("[LSP-PROTO] client/unregisterCapability called, params: %s", string(params))
		return nil, nil
	})

	// 注册 window/showMessageRequest 响应，返回第一个选项或空
	conn.RegisterServerRequestHandler("window/showMessageRequest", func(params json.RawMessage) (any, error) {
		log.Printf("[LSP-PROTO] window/showMessageRequest called, params: %s", string(params))
		// 默认返回第一个action
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

	// 注册通用通知处理
	conn.RegisterNotificationHandler("$/progress", func(params json.RawMessage) {
		log.Printf("[LSP-NOTIFY] $/progress: %s", string(params))
	})
	conn.RegisterNotificationHandler("window/logMessage", func(params json.RawMessage) {
		log.Printf("[LSP-NOTIFY] window/logMessage: %s", string(params))
	})
	conn.RegisterNotificationHandler("textDocument/publishDiagnostics", func(params json.RawMessage) {
		log.Printf("[LSP-NOTIFY] textDocument/publishDiagnostics: %s", string(params))
	})

	// stderr日志输出，和测试脚本一致
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Printf("[STDERR] %s", scanner.Text())
		}
	}()

	// 启动消息处理协程
	go conn.handleMessages()

	return conn
}

// ReadMessage严格按LSP协议读取一条完整消息
func (conn *LSPConnection) ReadMessage() (*Message, error) {
	// 读取header
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
	// 读取内容
	content := make([]byte, contentLength)
	readN, err := io.ReadFull(conn.stdout, content)
	log.Printf("[LSP-IO] Read content bytes: %d, err: %v", readN, err)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}
	// 解析JSON
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

	// 如果stdin支持Flush，主动flush
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

// GetServerInfo 获取服务器信息
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

// Client LSP客户端
type Client struct {
	// config 配置信息
	config *config.Config
	// sessions 会话管理
	sessions map[string]*types.LSPSession
	// sessionsMutex 会话互斥锁
	sessionsMutex sync.RWMutex
	// openedFiles 跟踪已打开的文件 (sessionKey -> map[fileURI]bool)
	openedFiles map[string]map[string]bool
	// openedFilesMutex 已打开文件的互斥锁
	openedFilesMutex sync.RWMutex
	// ctx 上下文
	ctx context.Context
	// cancel 取消函数
	cancel context.CancelFunc
}

// NewClient 创建新的LSP客户端
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

// GetOrCreateSession 获取或创建LSP会话
func (c *Client) GetOrCreateSession(languageID, rootURI string) (*types.LSPSession, error) {
	sessionKey := types.SessionKey{
		LanguageID: languageID,
		RootURI:    rootURI,
	}

	c.sessionsMutex.Lock()
	defer c.sessionsMutex.Unlock()

	// 检查是否已存在会话
	if session, exists := c.sessions[sessionKey.String()]; exists {
		// 更新最后使用时间
		session.UpdateLastUsed()
		return session, nil
	}

	// 检查会话数量限制
	if len(c.sessions) >= c.config.Session.MaxSessions {
		return nil, fmt.Errorf("已达到最大会话数限制: %d", c.config.Session.MaxSessions)
	}

	// 创建新会话
	session, err := c.createSession(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("创建LSP会话失败: %w", err)
	}

	c.sessions[sessionKey.String()] = session
	return session, nil
}

// createSession 创建新的LSP会话
func (c *Client) createSession(sessionKey types.SessionKey) (*types.LSPSession, error) {
	// 获取LSP服务器配置
	serverConfig, exists := c.config.GetLSPServerConfig(sessionKey.LanguageID)
	if !exists {
		return nil, fmt.Errorf("不支持的语言: %s", sessionKey.LanguageID)
	}

	log.Printf("[DEBUG] 启动LSP服务器: 命令=%s, 参数=%v", serverConfig.Command, serverConfig.Args)

	// 启动LSP服务器进程
	cmdArgs := serverConfig.Args
	cmd := exec.CommandContext(c.ctx, serverConfig.Command, cmdArgs...)

	// 新增：设置gopls进程工作目录为go.mod所在目录
	if strings.HasPrefix(sessionKey.RootURI, "file://") {
		cmd.Dir = strings.TrimPrefix(sessionKey.RootURI, "file://")
	}

	// 新增：合并环境变量
	cmd.Env = os.Environ()
	for k, v := range serverConfig.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// 创建管道
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("创建stdin管道失败: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("创建stdout管道失败: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("创建stderr管道失败: %w", err)
	}

	// 启动进程
	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		stderr.Close()
		return nil, fmt.Errorf("启动LSP服务器失败: %w", err)
	}

	// 创建手动实现的LSP连接
	conn := NewLSPConnection(cmd, stdin, stdout, stderr)

	// 创建会话对象
	session := &types.LSPSession{
		Key:           sessionKey,
		Conn:          conn,
		Process:       cmd,
		CreatedAt:     time.Now(),
		LastUsedAt:    time.Now(),
		IsInitialized: false,
	}

	// 初始化LSP服务器
	if err := c.initializeSession(session, serverConfig); err != nil {
		// 清理资源
		conn.Close()
		cmd.Process.Kill()
		return nil, fmt.Errorf("初始化LSP会话失败: %w", err)
	}

	return session, nil
}

// initializeSession 初始化LSP会话
func (c *Client) initializeSession(session *types.LSPSession, serverConfig *config.LSPServerConfig) error {
	// 构建工作区文件夹
	workspaceFolders := []types.LSPWorkspaceFolder{
		{
			URI:  session.Key.RootURI,
			Name: "lsp-mcp",
		},
	}

	// 构建初始化参数
	initParams := &types.LSPInitializeParams{
		ProcessID: func() *int { pid := os.Getpid(); return &pid }(),
		ClientInfo: &types.LSPClientInfo{
			Name:    c.config.MCPServer.Name,
			Version: c.config.MCPServer.Version,
		},
		RootURI:               &session.Key.RootURI, // 注释掉RootURI，仅传workspaceFolders
		WorkspaceFolders:      workspaceFolders,
		InitializationOptions: serverConfig.InitializationOptions,
		Capabilities:          c.buildClientCapabilities(),
		Trace:                 "off",
	}

	// 新增：打印initialize参数
	initParamsJson, _ := json.MarshalIndent(initParams, "", "  ")
	log.Printf("[DEBUG] initialize参数: %s", string(initParamsJson))

	// 保存初始化参数
	session.InitializeParams = initParams

	// 添加调试日志
	log.Printf("[DEBUG] 开始初始化LSP会话: 语言=%s, 根目录=%s", session.Key.LanguageID, session.Key.RootURI)

	// 发送初始化请求 - 增加超时时间到60秒，因为某些LSP服务器启动较慢
	ctx, cancel := context.WithTimeout(c.ctx, 60*time.Second)
	defer cancel()

	log.Printf("[DEBUG] 发送initialize请求...")
	conn := session.Conn.(*LSPConnection)
	result, err := conn.Call(ctx, "initialize", initParams)
	if err != nil {
		log.Printf("[ERROR] 发送初始化请求失败: %v", err)
		return fmt.Errorf("发送初始化请求失败: %w", err)
	}

	// 解析服务器能力
	var initResult struct {
		Capabilities json.RawMessage `json:"capabilities"`
		ServerInfo   struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"serverInfo"`
	}

	if err := json.Unmarshal(result, &initResult); err != nil {
		log.Printf("[ERROR] 解析初始化结果失败: %v", err)
		// 即使解析失败也继续，因为某些LSP服务器可能返回非标准格式
	} else {
		log.Printf("[DEBUG] 服务器信息: %s %s", initResult.ServerInfo.Name, initResult.ServerInfo.Version)
	}

	log.Printf("[DEBUG] initialize请求成功，响应: %s", string(result))
	log.Printf("[DEBUG] 发送initialized通知...")

	// 发送initialized通知
	err = conn.Notify(ctx, "initialized", map[string]interface{}{})
	if err != nil {
		log.Printf("[ERROR] 发送initialized通知失败: %v", err)
		return fmt.Errorf("发送initialized通知失败: %w", err)
	}

	log.Printf("[DEBUG] LSP会话初始化完成")
	session.IsInitialized = true
	return nil
}

// buildClientCapabilities 构建客户端能力
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

// FindDefinition 查找定义
func (c *Client) FindDefinition(ctx context.Context, req *types.FindDefinitionRequest) (*types.FindDefinitionResponse, error) {
	// 获取或创建会话
	session, err := c.GetOrCreateSession(req.LanguageID, req.RootURI)
	if err != nil {
		return &types.FindDefinitionResponse{
			Error: fmt.Sprintf("获取LSP会话失败: %v", err),
		}, nil
	}

	// 检查会话是否已初始化
	if !session.IsInitialized {
		return &types.FindDefinitionResponse{
			Error: "LSP会话未初始化",
		}, nil
	}

	// 发送didOpen通知（如果文件尚未打开）
	conn := session.Conn.(*LSPConnection)
	err = c.ensureFileOpen(ctx, conn, req.FileURI, req.LanguageID, req.RootURI)
	if err != nil {
		return &types.FindDefinitionResponse{
			Error: fmt.Sprintf("打开文件失败: %v", err),
		}, nil
	}

	// 构建LSP请求参数
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": req.FileURI,
		},
		"position": map[string]interface{}{
			"line":      req.Position.Line,
			"character": req.Position.Character,
		},
	}

	// 新增：打印definition请求参数
	paramsJson, _ := json.MarshalIndent(params, "", "  ")
	log.Printf("[DEBUG] definition请求参数: %s", string(paramsJson))

	// 设置超时 - 增加到30秒，因为gopls可能需要更多时间来分析代码
	requestCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 发送定义查找请求
	result, err := conn.Call(requestCtx, "textDocument/definition", params)
	if err != nil {
		return &types.FindDefinitionResponse{
			Error: fmt.Sprintf("发送定义查找请求失败: %v", err),
		}, nil
	}

	// 解析响应
	response, err := c.parseDefinitionResponse(result)
	if err != nil {
		return &types.FindDefinitionResponse{
			Error: fmt.Sprintf("解析定义查找响应失败: %v", err),
		}, nil
	}

	// 更新会话最后使用时间
	session.UpdateLastUsed()

	return response, nil
}

// FindReferences 查找引用
func (c *Client) FindReferences(ctx context.Context, req *types.FindReferencesRequest) (*types.FindReferencesResponse, error) {
	// 获取或创建会话
	session, err := c.GetOrCreateSession(req.LanguageID, req.RootURI)
	if err != nil {
		return &types.FindReferencesResponse{
			Error: fmt.Sprintf("获取LSP会话失败: %v", err),
		}, nil
	}

	// 检查会话是否已初始化
	if !session.IsInitialized {
		return &types.FindReferencesResponse{
			Error: "LSP会话未初始化",
		}, nil
	}

	// 发送didOpen通知（如果文件尚未打开）
	conn := session.Conn.(*LSPConnection)
	err = c.ensureFileOpen(ctx, conn, req.FileURI, req.LanguageID, req.RootURI)
	if err != nil {
		return &types.FindReferencesResponse{
			Error: fmt.Sprintf("打开文件失败: %v", err),
		}, nil
	}

	// 构建LSP请求参数
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

	// 设置超时 - 增加到30秒，因为gopls可能需要更多时间来分析代码
	requestCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 发送引用查找请求
	result, err := conn.Call(requestCtx, "textDocument/references", params)
	if err != nil {
		return &types.FindReferencesResponse{
			Error: fmt.Sprintf("发送引用查找请求失败: %v", err),
		}, nil
	}

	// 解析响应
	response, err := c.parseReferencesResponse(result)
	if err != nil {
		return &types.FindReferencesResponse{
			Error: fmt.Sprintf("解析引用查找响应失败: %v", err),
		}, nil
	}

	// 更新会话最后使用时间
	session.UpdateLastUsed()

	return response, nil
}

// GetHover 获取悬停信息
func (c *Client) GetHover(ctx context.Context, req *types.HoverRequest) (*types.HoverResponse, error) {
	// 获取或创建会话
	session, err := c.GetOrCreateSession(req.LanguageID, req.RootURI)
	if err != nil {
		return &types.HoverResponse{
			Error: fmt.Sprintf("获取LSP会话失败: %v", err),
		}, nil
	}

	// 检查会话是否已初始化
	if !session.IsInitialized {
		return &types.HoverResponse{
			Error: "LSP会话未初始化",
		}, nil
	}

	// 发送didOpen通知（如果文件尚未打开）
	conn := session.Conn.(*LSPConnection)
	err = c.ensureFileOpen(ctx, conn, req.FileURI, req.LanguageID, req.RootURI)
	if err != nil {
		return &types.HoverResponse{
			Error: fmt.Sprintf("打开文件失败: %v", err),
		}, nil
	}

	// 构建LSP请求参数
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": req.FileURI,
		},
		"position": map[string]interface{}{
			"line":      req.Position.Line,
			"character": req.Position.Character,
		},
	}

	// 设置超时 - 增加到30秒，因为gopls可能需要更多时间来分析代码
	requestCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 发送悬停请求
	result, err := conn.Call(requestCtx, "textDocument/hover", params)
	if err != nil {
		return &types.HoverResponse{
			Error: fmt.Sprintf("发送悬停请求失败: %v", err),
		}, nil
	}

	// 解析响应
	response, err := c.parseHoverResponse(result)
	if err != nil {
		return &types.HoverResponse{
			Error: fmt.Sprintf("解析悬停响应失败: %v", err),
		}, nil
	}

	// 更新会话最后使用时间
	session.UpdateLastUsed()

	return response, nil
}

// GetCompletion 获取代码补全
func (c *Client) GetCompletion(ctx context.Context, req *types.CompletionRequest) (*types.CompletionResponse, error) {
	// 获取或创建会话
	session, err := c.GetOrCreateSession(req.LanguageID, req.RootURI)
	if err != nil {
		return &types.CompletionResponse{
			Error: fmt.Sprintf("获取LSP会话失败: %v", err),
		}, nil
	}

	// 检查会话是否已初始化
	if !session.IsInitialized {
		return &types.CompletionResponse{
			Error: "LSP会话未初始化",
		}, nil
	}

	// 发送didOpen通知（如果文件尚未打开）
	conn := session.Conn.(*LSPConnection)
	err = c.ensureFileOpen(ctx, conn, req.FileURI, req.LanguageID, req.RootURI)
	if err != nil {
		return &types.CompletionResponse{
			Error: fmt.Sprintf("打开文件失败: %v", err),
		}, nil
	}

	// 构建LSP请求参数
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": req.FileURI,
		},
		"position": map[string]interface{}{
			"line":      req.Position.Line,
			"character": req.Position.Character,
		},
	}

	// 添加上下文信息（如果有）
	if req.TriggerKind > 0 {
		params["context"] = map[string]interface{}{
			"triggerKind": req.TriggerKind,
		}
		if req.TriggerCharacter != "" {
			params["context"].(map[string]interface{})["triggerCharacter"] = req.TriggerCharacter
		}
	}

	// 设置超时 - 增加到30秒，因为gopls可能需要更多时间来分析代码
	requestCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 发送代码补全请求
	result, err := conn.Call(requestCtx, "textDocument/completion", params)
	if err != nil {
		return &types.CompletionResponse{
			Error: fmt.Sprintf("发送代码补全请求失败: %v", err),
		}, nil
	}

	// 解析响应
	response, err := c.parseCompletionResponse(result)
	if err != nil {
		return &types.CompletionResponse{
			Error: fmt.Sprintf("解析代码补全响应失败: %v", err),
		}, nil
	}

	// 更新会话最后使用时间
	session.UpdateLastUsed()

	return response, nil
}

// parseDefinitionResponse 解析定义查找响应
func (c *Client) parseDefinitionResponse(result json.RawMessage) (*types.FindDefinitionResponse, error) {
	response := &types.FindDefinitionResponse{}

	// 先尝试解析为数组
	var rawArr []json.RawMessage
	if err := json.Unmarshal(result, &rawArr); err == nil {
		if len(rawArr) == 0 {
			// 空数组，合法，直接返回空响应
			response.Message = "未找到定义。"
			return response, nil
		}
		var probe map[string]interface{}
		if err := json.Unmarshal(rawArr[0], &probe); err == nil {
			if _, ok := probe["targetUri"]; ok {
				// 是 LocationLink
				var locationLinks []protocol.LocationLink
				if err := json.Unmarshal(result, &locationLinks); err == nil {
					response.LocationLinks = locationLinks
					// agent友好结构化输出
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
					response.Message = formatAgentMessage(len(response.AgentResults), "定义")
					return response, nil
				}
			} else if _, ok := probe["uri"]; ok {
				// 是 Location
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
					response.Message = formatAgentMessage(len(response.AgentResults), "定义")
					return response, nil
				}
			}
		}
	}

	// 尝试解析为单个Location
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
		response.Message = formatAgentMessage(1, "定义")
		return response, nil
	}

	// 尝试解析为单个LocationLink
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
		response.Message = formatAgentMessage(1, "定义")
		return response, nil
	}

	// 如果都解析失败，检查是否为null
	if string(result) == "null" {
		response.Message = "未找到定义。"
		return response, nil
	}

	return nil, fmt.Errorf("无法解析定义查找响应: %s", string(result))
}

// parseFileLineChar 解析 uri 和 range，返回完整路径、行号（1-based）、字符（1-based）
func parseFileLineChar(uri protocol.DocumentURI, rng protocol.Range) (string, int, int) {
	file := strings.TrimPrefix(string(uri), "file://")
	line := int(rng.Start.Line) + 1
	char := int(rng.Start.Character) + 1
	return file, line, char
}

// formatSummary 生成 agent 友好的 summary（完整路径）
func formatSummary(file string, line, char int) string {
	return fmt.Sprintf("跳转到 %s 第%d行第%d列", file, line, char)
}

// formatAgentMessage 支持类型参数
func formatAgentMessage(count int, typ string) string {
	if count == 0 {
		return fmt.Sprintf("未找到%s。", typ)
	}
	return fmt.Sprintf("共找到%d个%s。", count, typ)
}

// parseReferencesResponse 解析引用查找响应
func (c *Client) parseReferencesResponse(result json.RawMessage) (*types.FindReferencesResponse, error) {
	response := &types.FindReferencesResponse{}

	// 尝试解析为Location数组
	var locations []protocol.Location
	if err := json.Unmarshal(result, &locations); err == nil {
		response.Locations = locations
		// agent友好结构化输出
		for _, loc := range locations {
			file, line, char := parseFileLineChar(loc.URI, loc.Range)
			summary := fmt.Sprintf("引用于 %s 第%d行第%d列", file, line, char)
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
		response.Message = formatAgentMessage(len(response.AgentResults), "引用")
		return response, nil
	}

	// 如果解析失败，检查是否为null
	if string(result) == "null" {
		response.Message = "未找到引用。"
		return response, nil
	}

	return nil, fmt.Errorf("无法解析引用查找响应: %s", string(result))
}

// parseHoverResponse 解析悬停响应
func (c *Client) parseHoverResponse(result json.RawMessage) (*types.HoverResponse, error) {
	response := &types.HoverResponse{}

	// 尝试解析为Hover对象
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
		// 处理contents字段
		if hover.Contents != nil {
			// 1. 解析contents为kind/value结构
			var value string
			switch v := hover.Contents.(type) {
			case map[string]interface{}:
				if val, ok := v["value"].(string); ok {
					value = val
				}
			case []interface{}:
				// 多个MarkedString，拼接
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

			// 2. 还原转义字符
			value = htmlUnescapeString(value)
			rawMarkdown = value

			// 3. 提取代码块内容
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

			// 4. 生成summary和doc
			if typeSignature != "" && importStatement != "" {
				summary = "类型定义：" + typeSignature + "，可通过 " + importStatement + " 导入。"
			} else if typeSignature != "" {
				summary = "类型定义：" + typeSignature
			} else {
				summary = value
			}
			// doc 优先用注释，否则用 typeSignature
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

	// 如果解析失败，检查是否为null
	if string(result) == "null" {
		response.Message = "无悬停信息。"
		return response, nil
	}

	return nil, fmt.Errorf("无法解析悬停响应: %s", string(result))
}

// htmlUnescapeString 还原html转义字符
func htmlUnescapeString(s string) string {
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\\u003c", "<")
	s = strings.ReplaceAll(s, "\\u003e", ">")
	return s
}

// parseCompletionResponse 解析代码补全响应
func (c *Client) parseCompletionResponse(result json.RawMessage) (*types.CompletionResponse, error) {
	response := &types.CompletionResponse{}

	// 尝试解析为CompletionList
	var completionList struct {
		IsIncomplete bool                      `json:"isIncomplete"`
		Items        []protocol.CompletionItem `json:"items"`
	}

	if err := json.Unmarshal(result, &completionList); err == nil {
		response.IsIncomplete = completionList.IsIncomplete
		response.Items = completionList.Items
		// agent友好结构化输出
		for _, item := range completionList.Items {
			summary := fmt.Sprintf("补全项：%s %s", item.Label, item.Detail)
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
		response.Message = formatAgentMessage(len(response.AgentResults), "补全项")
		return response, nil
	}

	// 尝试解析为CompletionItem数组
	var items []protocol.CompletionItem
	if err := json.Unmarshal(result, &items); err == nil {
		response.Items = items
		for _, item := range items {
			summary := fmt.Sprintf("补全项：%s %s", item.Label, item.Detail)
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
		response.Message = formatAgentMessage(len(response.AgentResults), "补全项")
		return response, nil
	}

	// 如果解析失败，检查是否为null
	if string(result) == "null" {
		response.Message = "无补全项。"
		return response, nil
	}

	return nil, fmt.Errorf("无法解析代码补全响应: %s", string(result))
}

// closeSession 关闭会话
func (c *Client) closeSession(session *types.LSPSession) {
	if session.Conn != nil {
		// 发送shutdown请求
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn := session.Conn.(*LSPConnection)

		// 尝试发送shutdown请求，忽略错误
		_, _ = conn.Call(ctx, "shutdown", nil)

		// 尝试发送exit通知，忽略错误
		_ = conn.Notify(ctx, "exit", nil)

		// 关闭连接
		conn.Close()
	}

	// 终止进程
	if cmd, ok := session.Process.(*exec.Cmd); ok && cmd.Process != nil {
		cmd.Process.Kill()
		cmd.Wait()
	}
}

// OpenFile 打开文件
func (c *Client) OpenFile(ctx context.Context, req *types.OpenFileRequest) error {
	// 获取或创建会话
	session, err := c.GetOrCreateSession(req.LanguageID, req.RootURI)
	if err != nil {
		return fmt.Errorf("failed to get or create session: %w", err)
	}

	// 确保文件已打开
	conn := session.Conn.(*LSPConnection)
	return c.ensureFileOpen(ctx, conn, req.FileURI, req.LanguageID, req.RootURI)
}

// Close 关闭LSP客户端
func (c *Client) Close() error {
	c.cancel()

	c.sessionsMutex.Lock()
	defer c.sessionsMutex.Unlock()

	// 关闭所有会话
	for _, session := range c.sessions {
		c.closeSession(session)
	}

	c.sessions = make(map[string]*types.LSPSession)
	return nil
}

// GetSessionCount 获取当前会话数量
func (c *Client) GetSessionCount() int {
	c.sessionsMutex.RLock()
	defer c.sessionsMutex.RUnlock()
	return len(c.sessions)
}

// GetSessionInfo 获取会话信息
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

// ensureFileOpen 确保文件已通过didOpen通知发送给LSP服务器
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

	// 只从本地文件读取内容
	filePath := strings.TrimPrefix(fileURI, "file://")
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
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
	log.Printf("[DEBUG] didOpen参数: %s", string(didOpenParamsJson))

	log.Printf("[DEBUG] 发送didOpen通知: %s", fileURI)
	err = conn.Notify(ctx, "textDocument/didOpen", didOpenParams)
	if err != nil {
		return fmt.Errorf("发送didOpen通知失败: %w", err)
	}

	c.openedFiles[sessionKey][fileURI] = true
	log.Printf("[DEBUG] 文件已标记为打开: %s", fileURI)

	time.Sleep(1 * time.Second)

	return nil
}
