// LSP 请求与响应相关类型定义
// Types for LSP requests and responses
package types

import (
	protocol "go.lsp.dev/protocol"
)

// LSPInitializeParams LSP初始化参数
type LSPInitializeParams struct {
	ProcessID             *int                  `json:"processId,omitempty"`
	ClientInfo            *LSPClientInfo        `json:"clientInfo,omitempty"`
	Locale                string                `json:"locale,omitempty"`
	RootPath              *string               `json:"rootPath,omitempty"`
	RootURI               *string               `json:"rootUri,omitempty"`
	InitializationOptions interface{}           `json:"initializationOptions,omitempty"`
	Capabilities          LSPClientCapabilities `json:"capabilities"`
	Trace                 string                `json:"trace,omitempty"`
	WorkspaceFolders      []LSPWorkspaceFolder  `json:"workspaceFolders,omitempty"`
}

// LSPClientInfo 客户端信息
type LSPClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// LSPWorkspaceFolder 工作区文件夹
type LSPWorkspaceFolder struct {
	URI  string `json:"uri"`
	Name string `json:"name"`
}

// FindDefinitionRequest 查找定义请求
type FindDefinitionRequest struct {
	LanguageID string            `json:"language_id"`
	RootURI    string            `json:"root_uri"`
	FileURI    string            `json:"file_uri"`
	Position   protocol.Position `json:"position"`
}

// FindDefinitionResponse 查找定义响应
type FindDefinitionResponse struct {
	AgentResults  []AgentDefinitionResult `json:"agent_results,omitempty"`
	Locations     []protocol.Location     `json:"locations,omitempty"`
	LocationLinks []protocol.LocationLink `json:"location_links,omitempty"`
	Error         string                  `json:"error,omitempty"`
	Message       string                  `json:"message,omitempty"`
}

// AgentDefinitionResult agent友好结构化跳转结果
type AgentDefinitionResult struct {
	Type      string             `json:"type"`
	File      string             `json:"file"`
	Line      int                `json:"line"`
	Character int                `json:"character"`
	Summary   string             `json:"summary"`
	Range     *protocol.Range    `json:"range,omitempty"`
	Location  *protocol.Location `json:"location,omitempty"`
}

// FindReferencesRequest 查找引用请求
type FindReferencesRequest struct {
	LanguageID         string            `json:"language_id"`
	RootURI            string            `json:"root_uri"`
	FileURI            string            `json:"file_uri"`
	Position           protocol.Position `json:"position"`
	IncludeDeclaration bool              `json:"include_declaration"`
}

// FindReferencesResponse 查找引用响应
type FindReferencesResponse struct {
	AgentResults []AgentReferenceResult `json:"agent_results,omitempty"`
	Locations    []protocol.Location    `json:"locations,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Message      string                 `json:"message,omitempty"`
}

// AgentReferenceResult agent友好结构化引用结果
type AgentReferenceResult struct {
	Type      string             `json:"type"`
	File      string             `json:"file"`
	Line      int                `json:"line"`
	Character int                `json:"character"`
	Summary   string             `json:"summary"`
	Range     *protocol.Range    `json:"range,omitempty"`
	Location  *protocol.Location `json:"location,omitempty"`
}

// HoverRequest 悬停信息请求
type HoverRequest struct {
	LanguageID string            `json:"language_id"`
	RootURI    string            `json:"root_uri"`
	FileURI    string            `json:"file_uri"`
	Position   protocol.Position `json:"position"`
}

// HoverResponse 悬停信息响应
type HoverResponse struct {
	AgentResults []AgentHoverResult `json:"agent_results,omitempty"`
	Contents     interface{}        `json:"contents,omitempty"`
	Range        *protocol.Range    `json:"range,omitempty"`
	Error        string             `json:"error,omitempty"`
	Message      string             `json:"message,omitempty"`
}

// AgentHoverResult agent友好结构化悬停结果
type AgentHoverResult struct {
	Summary         string             `json:"summary"`
	Doc             string             `json:"doc,omitempty"`
	Type            string             `json:"type"`
	File            string             `json:"file"`
	Line            int                `json:"line"`
	Character       int                `json:"character"`
	Range           *protocol.Range    `json:"range,omitempty"`
	Location        *protocol.Location `json:"location,omitempty"`
	TypeSignature   string             `json:"type_signature,omitempty"`
	ImportStatement string             `json:"import_statement,omitempty"`
	RawMarkdown     string             `json:"raw_markdown,omitempty"`
}

// CompletionRequest 代码补全请求
type CompletionRequest struct {
	LanguageID       string            `json:"language_id"`
	RootURI          string            `json:"root_uri"`
	FileURI          string            `json:"file_uri"`
	Position         protocol.Position `json:"position"`
	TriggerKind      int               `json:"trigger_kind,omitempty"`
	TriggerCharacter string            `json:"trigger_character,omitempty"`
}

// CompletionResponse 代码补全响应
type CompletionResponse struct {
	AgentResults []AgentCompletionResult   `json:"agent_results,omitempty"`
	Items        []protocol.CompletionItem `json:"items,omitempty"`
	IsIncomplete bool                      `json:"is_incomplete,omitempty"`
	Error        string                    `json:"error,omitempty"`
	Message      string                    `json:"message,omitempty"`
}

// AgentCompletionResult agent友好结构化补全结果
type AgentCompletionResult struct {
	Type           string                  `json:"type"`
	File           string                  `json:"file"`
	Line           int                     `json:"line"`
	Character      int                     `json:"character"`
	Summary        string                  `json:"summary"`
	TextEdit       *protocol.TextEdit      `json:"text_edit,omitempty"`
	CompletionItem protocol.CompletionItem `json:"completion_item"`
}

// OpenFileRequest 打开文件请求
type OpenFileRequest struct {
	LanguageID string `json:"language_id"`
	RootURI    string `json:"root_uri"`
	FileURI    string `json:"file_uri"`
}
