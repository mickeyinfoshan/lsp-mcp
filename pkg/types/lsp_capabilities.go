// lsp_capabilities.go
// Types for LSP client capabilities (Workspace, TextDocument, Window, General)
package types

// LSPClientCapabilities Client capabilities
// Represents the set of capabilities supported by the LSP client
type LSPClientCapabilities struct {
	Workspace    *LSPWorkspaceClientCapabilities    `json:"workspace,omitempty"`
	TextDocument *LSPTextDocumentClientCapabilities `json:"textDocument,omitempty"`
	Window       *LSPWindowClientCapabilities       `json:"window,omitempty"`
	General      *LSPGeneralClientCapabilities      `json:"general,omitempty"`
	Experimental interface{}                        `json:"experimental,omitempty"`
}

// LSPWorkspaceClientCapabilities Workspace client capabilities
// Represents the set of capabilities supported by the LSP client for Workspace
type LSPWorkspaceClientCapabilities struct {
	ApplyEdit              bool                                         `json:"applyEdit,omitempty"`
	WorkspaceEdit          *LSPWorkspaceEditClientCapabilities          `json:"workspaceEdit,omitempty"`
	DidChangeConfiguration *LSPDidChangeConfigurationClientCapabilities `json:"didChangeConfiguration,omitempty"`
	DidChangeWatchedFiles  *LSPDidChangeWatchedFilesClientCapabilities  `json:"didChangeWatchedFiles,omitempty"`
	Symbol                 *LSPWorkspaceSymbolClientCapabilities        `json:"symbol,omitempty"`
	ExecuteCommand         *LSPExecuteCommandClientCapabilities         `json:"executeCommand,omitempty"`
	WorkspaceFolders       bool                                         `json:"workspaceFolders,omitempty"`
	Configuration          bool                                         `json:"configuration,omitempty"`
}

// LSPTextDocumentClientCapabilities Text document client capabilities
// Represents the set of capabilities supported by the LSP client for TextDocument
type LSPTextDocumentClientCapabilities struct {
	Synchronization    *LSPTextDocumentSyncClientCapabilities         `json:"synchronization,omitempty"`
	Completion         *LSPCompletionClientCapabilities               `json:"completion,omitempty"`
	Hover              *LSPHoverClientCapabilities                    `json:"hover,omitempty"`
	SignatureHelp      *LSPSignatureHelpClientCapabilities            `json:"signatureHelp,omitempty"`
	Declaration        *LSPDeclarationClientCapabilities              `json:"declaration,omitempty"`
	Definition         *LSPDefinitionClientCapabilities               `json:"definition,omitempty"`
	TypeDefinition     *LSPTypeDefinitionClientCapabilities           `json:"typeDefinition,omitempty"`
	Implementation     *LSPImplementationClientCapabilities           `json:"implementation,omitempty"`
	References         *LSPReferencesClientCapabilities               `json:"references,omitempty"`
	DocumentHighlight  *LSPDocumentHighlightClientCapabilities        `json:"documentHighlight,omitempty"`
	DocumentSymbol     *LSPDocumentSymbolClientCapabilities           `json:"documentSymbol,omitempty"`
	CodeAction         *LSPCodeActionClientCapabilities               `json:"codeAction,omitempty"`
	CodeLens           *LSPCodeLensClientCapabilities                 `json:"codeLens,omitempty"`
	DocumentLink       *LSPDocumentLinkClientCapabilities             `json:"documentLink,omitempty"`
	ColorProvider      *LSPDocumentColorClientCapabilities            `json:"colorProvider,omitempty"`
	Formatting         *LSPDocumentFormattingClientCapabilities       `json:"formatting,omitempty"`
	RangeFormatting    *LSPDocumentRangeFormattingClientCapabilities  `json:"rangeFormatting,omitempty"`
	OnTypeFormatting   *LSPDocumentOnTypeFormattingClientCapabilities `json:"onTypeFormatting,omitempty"`
	Rename             *LSPRenameClientCapabilities                   `json:"rename,omitempty"`
	PublishDiagnostics *LSPPublishDiagnosticsClientCapabilities       `json:"publishDiagnostics,omitempty"`
	FoldingRange       *LSPFoldingRangeClientCapabilities             `json:"foldingRange,omitempty"`
	SelectionRange     *LSPSelectionRangeClientCapabilities           `json:"selectionRange,omitempty"`
	LinkedEditingRange *LSPLinkedEditingRangeClientCapabilities       `json:"linkedEditingRange,omitempty"`
	CallHierarchy      *LSPCallHierarchyClientCapabilities            `json:"callHierarchy,omitempty"`
	SemanticTokens     *LSPSemanticTokensClientCapabilities           `json:"semanticTokens,omitempty"`
	Moniker            *LSPMonikerClientCapabilities                  `json:"moniker,omitempty"`
	TypeHierarchy      *LSPTypeHierarchyClientCapabilities            `json:"typeHierarchy,omitempty"`
	InlineValue        *LSPInlineValueClientCapabilities              `json:"inlineValue,omitempty"`
	InlayHint          *LSPInlayHintClientCapabilities                `json:"inlayHint,omitempty"`
	Diagnostic         *LSPDiagnosticClientCapabilities               `json:"diagnostic,omitempty"`
}

// LSPWindowClientCapabilities Window client capabilities
// Represents the set of capabilities supported by the LSP client for Window
type LSPWindowClientCapabilities struct {
	WorkDoneProgress bool                                     `json:"workDoneProgress,omitempty"`
	ShowMessage      *LSPShowMessageRequestClientCapabilities `json:"showMessage,omitempty"`
	ShowDocument     *LSPShowDocumentClientCapabilities       `json:"showDocument,omitempty"`
}

// LSPGeneralClientCapabilities General client capabilities
// Represents the set of capabilities supported by the LSP client for General
type LSPGeneralClientCapabilities struct {
	StaleRequestSupport *LSPStaleRequestSupportOptions           `json:"staleRequestSupport,omitempty"`
	RegularExpressions  *LSPRegularExpressionsClientCapabilities `json:"regularExpressions,omitempty"`
	Markdown            *LSPMarkdownClientCapabilities           `json:"markdown,omitempty"`
	PositionEncodings   []string                                 `json:"positionEncodings,omitempty"`
}

// LSPWorkspaceEditClientCapabilities Workspace edit client capabilities
// Represents the set of capabilities supported by the LSP client for WorkspaceEdit
type LSPWorkspaceEditClientCapabilities struct {
	DocumentChanges bool `json:"documentChanges,omitempty"`
}

// LSPDidChangeConfigurationClientCapabilities Did change configuration client capabilities
// Represents the set of capabilities supported by the LSP client for DidChangeConfiguration
type LSPDidChangeConfigurationClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPDidChangeWatchedFilesClientCapabilities Did change watched files client capabilities
// Represents the set of capabilities supported by the LSP client for DidChangeWatchedFiles
type LSPDidChangeWatchedFilesClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPWorkspaceSymbolClientCapabilities Workspace symbol client capabilities
// Represents the set of capabilities supported by the LSP client for WorkspaceSymbol
type LSPWorkspaceSymbolClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPExecuteCommandClientCapabilities Execute command client capabilities
// Represents the set of capabilities supported by the LSP client for ExecuteCommand
type LSPExecuteCommandClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPTextDocumentSyncClientCapabilities Text document sync client capabilities
// Represents the set of capabilities supported by the LSP client for TextDocumentSync
type LSPTextDocumentSyncClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPCompletionClientCapabilities Completion client capabilities
// Represents the set of capabilities supported by the LSP client for Completion
type LSPCompletionClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPHoverClientCapabilities Hover client capabilities
// Represents the set of capabilities supported by the LSP client for Hover
type LSPHoverClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPSignatureHelpClientCapabilities Signature help client capabilities
// Represents the set of capabilities supported by the LSP client for SignatureHelp
type LSPSignatureHelpClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPDeclarationClientCapabilities Declaration client capabilities
// Represents the set of capabilities supported by the LSP client for Declaration
type LSPDeclarationClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	LinkSupport         bool `json:"linkSupport,omitempty"`
}

// LSPDefinitionClientCapabilities Definition client capabilities
// Represents the set of capabilities supported by the LSP client for Definition
type LSPDefinitionClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	LinkSupport         bool `json:"linkSupport,omitempty"`
}

// LSPTypeDefinitionClientCapabilities Type definition client capabilities
// Represents the set of capabilities supported by the LSP client for TypeDefinition
type LSPTypeDefinitionClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	LinkSupport         bool `json:"linkSupport,omitempty"`
}

// LSPImplementationClientCapabilities Implementation client capabilities
// Represents the set of capabilities supported by the LSP client for Implementation
type LSPImplementationClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	LinkSupport         bool `json:"linkSupport,omitempty"`
}

// LSPReferencesClientCapabilities References client capabilities
// Represents the set of capabilities supported by the LSP client for References
type LSPReferencesClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPDocumentHighlightClientCapabilities Document highlight client capabilities
// Represents the set of capabilities supported by the LSP client for DocumentHighlight
type LSPDocumentHighlightClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPDocumentSymbolClientCapabilities Document symbol client capabilities
// Represents the set of capabilities supported by the LSP client for DocumentSymbol
type LSPDocumentSymbolClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPCodeActionClientCapabilities Code action client capabilities
// Represents the set of capabilities supported by the LSP client for CodeAction
type LSPCodeActionClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPCodeLensClientCapabilities Code lens client capabilities
// Represents the set of capabilities supported by the LSP client for CodeLens
type LSPCodeLensClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPDocumentLinkClientCapabilities Document link client capabilities
// Represents the set of capabilities supported by the LSP client for DocumentLink
type LSPDocumentLinkClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPDocumentColorClientCapabilities Document color client capabilities
// Represents the set of capabilities supported by the LSP client for DocumentColor
type LSPDocumentColorClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPDocumentFormattingClientCapabilities Document formatting client capabilities
// Represents the set of capabilities supported by the LSP client for DocumentFormatting
type LSPDocumentFormattingClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPDocumentRangeFormattingClientCapabilities Document range formatting client capabilities
// Represents the set of capabilities supported by the LSP client for DocumentRangeFormatting
type LSPDocumentRangeFormattingClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPDocumentOnTypeFormattingClientCapabilities Document on type formatting client capabilities
// Represents the set of capabilities supported by the LSP client for DocumentOnTypeFormatting
type LSPDocumentOnTypeFormattingClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPRenameClientCapabilities Rename client capabilities
// Represents the set of capabilities supported by the LSP client for Rename
type LSPRenameClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPPublishDiagnosticsClientCapabilities Publish diagnostics client capabilities
// Represents the set of capabilities supported by the LSP client for PublishDiagnostics
type LSPPublishDiagnosticsClientCapabilities struct {
	RelatedInformation bool `json:"relatedInformation,omitempty"`
}

// LSPFoldingRangeClientCapabilities Folding range client capabilities
// Represents the set of capabilities supported by the LSP client for FoldingRange
type LSPFoldingRangeClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPSelectionRangeClientCapabilities Selection range client capabilities
// Represents the set of capabilities supported by the LSP client for SelectionRange
type LSPSelectionRangeClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPLinkedEditingRangeClientCapabilities Linked editing range client capabilities
// Represents the set of capabilities supported by the LSP client for LinkedEditingRange
type LSPLinkedEditingRangeClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPCallHierarchyClientCapabilities Call hierarchy client capabilities
// Represents the set of capabilities supported by the LSP client for CallHierarchy
type LSPCallHierarchyClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPSemanticTokensClientCapabilities Semantic tokens client capabilities
// Represents the set of capabilities supported by the LSP client for SemanticTokens
type LSPSemanticTokensClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPMonikerClientCapabilities Moniker client capabilities
// Represents the set of capabilities supported by the LSP client for Moniker
type LSPMonikerClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPTypeHierarchyClientCapabilities Type hierarchy client capabilities
// Represents the set of capabilities supported by the LSP client for TypeHierarchy
type LSPTypeHierarchyClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPInlineValueClientCapabilities Inline value client capabilities
// Represents the set of capabilities supported by the LSP client for InlineValue
type LSPInlineValueClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPInlayHintClientCapabilities Inlay hint client capabilities
// Represents the set of capabilities supported by the LSP client for InlayHint
type LSPInlayHintClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPDiagnosticClientCapabilities Diagnostic client capabilities
// Represents the set of capabilities supported by the LSP client for Diagnostic
type LSPDiagnosticClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPShowMessageRequestClientCapabilities Show message request client capabilities
// Represents the set of capabilities supported by the LSP client for ShowMessageRequest
type LSPShowMessageRequestClientCapabilities struct {
	MessageActionItem *LSPMessageActionItemCapabilities `json:"messageActionItem,omitempty"`
}

// LSPMessageActionItemCapabilities Message action item capabilities
// Represents the set of capabilities supported by the LSP client for MessageActionItem
type LSPMessageActionItemCapabilities struct {
	AdditionalPropertiesSupport bool `json:"additionalPropertiesSupport,omitempty"`
}

// LSPShowDocumentClientCapabilities Show document client capabilities
// Represents the set of capabilities supported by the LSP client for ShowDocument
type LSPShowDocumentClientCapabilities struct {
	Support bool `json:"support"`
}

// LSPStaleRequestSupportOptions Stale request support options
// Represents the set of capabilities supported by the LSP client for StaleRequestSupport
type LSPStaleRequestSupportOptions struct {
	Cancel                 bool     `json:"cancel"`
	RetryOnContentModified []string `json:"retryOnContentModified"`
}

// LSPRegularExpressionsClientCapabilities Regular expressions client capabilities
// Represents the set of capabilities supported by the LSP client for RegularExpressions
type LSPRegularExpressionsClientCapabilities struct {
	Engine  string `json:"engine"`
	Version string `json:"version,omitempty"`
}

// LSPMarkdownClientCapabilities Markdown client capabilities
// Represents the set of capabilities supported by the LSP client for Markdown
type LSPMarkdownClientCapabilities struct {
	Parser      string   `json:"parser"`
	Version     string   `json:"version,omitempty"`
	AllowedTags []string `json:"allowedTags,omitempty"`
}
