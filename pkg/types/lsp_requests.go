// Types for LSP requests and responses
package types

import (
	protocol "go.lsp.dev/protocol"
)

// LSPInitializeParams LSP initialization parameters
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

// LSPClientInfo client info
type LSPClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// LSPWorkspaceFolder workspace folder
type LSPWorkspaceFolder struct {
	URI  string `json:"uri"`
	Name string `json:"name"`
}

// FindDefinitionRequest find definition request
type FindDefinitionRequest struct {
	LanguageID string            `json:"language_id"`
	RootURI    string            `json:"root_uri"`
	FileURI    string            `json:"file_uri"`
	Position   protocol.Position `json:"position"`
}

// FindDefinitionResponse find definition response
type FindDefinitionResponse struct {
	AgentResults  []AgentDefinitionResult `json:"agent_results,omitempty"`
	Locations     []protocol.Location     `json:"locations,omitempty"`
	LocationLinks []protocol.LocationLink `json:"location_links,omitempty"`
	Error         string                  `json:"error,omitempty"`
	Message       string                  `json:"message,omitempty"`
}

// AgentDefinitionResult agent-friendly structured definition result
type AgentDefinitionResult struct {
	Type      string             `json:"type"`
	File      string             `json:"file"`
	Line      int                `json:"line"`
	Character int                `json:"character"`
	Summary   string             `json:"summary"`
	Range     *protocol.Range    `json:"range,omitempty"`
	Location  *protocol.Location `json:"location,omitempty"`
}

// FindReferencesRequest find references request
type FindReferencesRequest struct {
	LanguageID         string            `json:"language_id"`
	RootURI            string            `json:"root_uri"`
	FileURI            string            `json:"file_uri"`
	Position           protocol.Position `json:"position"`
	IncludeDeclaration bool              `json:"include_declaration"`
}

// FindReferencesResponse find references response
type FindReferencesResponse struct {
	AgentResults []AgentReferenceResult `json:"agent_results,omitempty"`
	Locations    []protocol.Location    `json:"locations,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Message      string                 `json:"message,omitempty"`
}

// AgentReferenceResult agent-friendly structured reference result
type AgentReferenceResult struct {
	Type      string             `json:"type"`
	File      string             `json:"file"`
	Line      int                `json:"line"`
	Character int                `json:"character"`
	Summary   string             `json:"summary"`
	Range     *protocol.Range    `json:"range,omitempty"`
	Location  *protocol.Location `json:"location,omitempty"`
}

// HoverRequest hover request
type HoverRequest struct {
	LanguageID string            `json:"language_id"`
	RootURI    string            `json:"root_uri"`
	FileURI    string            `json:"file_uri"`
	Position   protocol.Position `json:"position"`
}

// HoverResponse hover response
type HoverResponse struct {
	AgentResults []AgentHoverResult `json:"agent_results,omitempty"`
	Contents     interface{}        `json:"contents,omitempty"`
	Range        *protocol.Range    `json:"range,omitempty"`
	Error        string             `json:"error,omitempty"`
	Message      string             `json:"message,omitempty"`
}

// AgentHoverResult agent-friendly structured hover result
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

// CompletionRequest completion request
type CompletionRequest struct {
	LanguageID       string            `json:"language_id"`
	RootURI          string            `json:"root_uri"`
	FileURI          string            `json:"file_uri"`
	Position         protocol.Position `json:"position"`
	TriggerKind      int               `json:"trigger_kind,omitempty"`
	TriggerCharacter string            `json:"trigger_character,omitempty"`
}

// CompletionResponse completion response
type CompletionResponse struct {
	AgentResults []AgentCompletionResult   `json:"agent_results,omitempty"`
	Items        []protocol.CompletionItem `json:"items,omitempty"`
	IsIncomplete bool                      `json:"is_incomplete,omitempty"`
	Error        string                    `json:"error,omitempty"`
	Message      string                    `json:"message,omitempty"`
}

// AgentCompletionResult agent-friendly structured completion result
type AgentCompletionResult struct {
	Type           string                  `json:"type"`
	File           string                  `json:"file"`
	Line           int                     `json:"line"`
	Character      int                     `json:"character"`
	Summary        string                  `json:"summary"`
	TextEdit       *protocol.TextEdit      `json:"text_edit,omitempty"`
	CompletionItem protocol.CompletionItem `json:"completion_item"`
}

// OpenFileRequest open file request
type OpenFileRequest struct {
	LanguageID string `json:"language_id"`
	RootURI    string `json:"root_uri"`
	FileURI    string `json:"file_uri"`
}
