// lsp_capabilities.go
// LSP 客户端能力相关类型定义
// Types for LSP client capabilities (Workspace, TextDocument, Window, General)
package types

// LSPClientCapabilities 客户端能力 / Client capabilities
// 代表 LSP 客户端支持的功能集合
// Represents the set of capabilities supported by the LSP client
type LSPClientCapabilities struct {
	Workspace    *LSPWorkspaceClientCapabilities    `json:"workspace,omitempty"`
	TextDocument *LSPTextDocumentClientCapabilities `json:"textDocument,omitempty"`
	Window       *LSPWindowClientCapabilities       `json:"window,omitempty"`
	General      *LSPGeneralClientCapabilities      `json:"general,omitempty"`
	Experimental interface{}                        `json:"experimental,omitempty"`
}

// LSPWorkspaceClientCapabilities 工作区客户端能力 / Workspace client capabilities
// 代表 LSP 客户端在 Workspace 相关的功能集合
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

// LSPTextDocumentClientCapabilities 文本文档客户端能力 / Text document client capabilities
// 代表 LSP 客户端在 TextDocument 相关的功能集合
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

// LSPWindowClientCapabilities 窗口客户端能力 / Window client capabilities
// 代表 LSP 客户端在 Window 相关的功能集合
// Represents the set of capabilities supported by the LSP client for Window
type LSPWindowClientCapabilities struct {
	WorkDoneProgress bool                                     `json:"workDoneProgress,omitempty"`
	ShowMessage      *LSPShowMessageRequestClientCapabilities `json:"showMessage,omitempty"`
	ShowDocument     *LSPShowDocumentClientCapabilities       `json:"showDocument,omitempty"`
}

// LSPGeneralClientCapabilities 通用客户端能力 / General client capabilities
// 代表 LSP 客户端在 General 相关的功能集合
// Represents the set of capabilities supported by the LSP client for General
type LSPGeneralClientCapabilities struct {
	StaleRequestSupport *LSPStaleRequestSupportOptions           `json:"staleRequestSupport,omitempty"`
	RegularExpressions  *LSPRegularExpressionsClientCapabilities `json:"regularExpressions,omitempty"`
	Markdown            *LSPMarkdownClientCapabilities           `json:"markdown,omitempty"`
	PositionEncodings   []string                                 `json:"positionEncodings,omitempty"`
}

// LSPWorkspaceEditClientCapabilities 工作区编辑客户端能力 / Workspace edit client capabilities
// 代表 LSP 客户端在 WorkspaceEdit 相关的功能集合
// Represents the set of capabilities supported by the LSP client for WorkspaceEdit
type LSPWorkspaceEditClientCapabilities struct {
	DocumentChanges bool `json:"documentChanges,omitempty"`
}

// LSPDidChangeConfigurationClientCapabilities 配置变更客户端能力 / Did change configuration client capabilities
// 代表 LSP 客户端在 DidChangeConfiguration 相关的功能集合
// Represents the set of capabilities supported by the LSP client for DidChangeConfiguration
type LSPDidChangeConfigurationClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPDidChangeWatchedFilesClientCapabilities 文件监视客户端能力 / Did change watched files client capabilities
// 代表 LSP 客户端在 DidChangeWatchedFiles 相关的功能集合
// Represents the set of capabilities supported by the LSP client for DidChangeWatchedFiles
type LSPDidChangeWatchedFilesClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPWorkspaceSymbolClientCapabilities 工作区符号客户端能力 / Workspace symbol client capabilities
// 代表 LSP 客户端在 WorkspaceSymbol 相关的功能集合
// Represents the set of capabilities supported by the LSP client for WorkspaceSymbol
type LSPWorkspaceSymbolClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPExecuteCommandClientCapabilities 执行命令客户端能力 / Execute command client capabilities
// 代表 LSP 客户端在 ExecuteCommand 相关的功能集合
// Represents the set of capabilities supported by the LSP client for ExecuteCommand
type LSPExecuteCommandClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPTextDocumentSyncClientCapabilities 文本文档同步客户端能力 / Text document sync client capabilities
// 代表 LSP 客户端在 TextDocumentSync 相关的功能集合
// Represents the set of capabilities supported by the LSP client for TextDocumentSync
type LSPTextDocumentSyncClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPCompletionClientCapabilities 补全客户端能力 / Completion client capabilities
// 代表 LSP 客户端在 Completion 相关的功能集合
// Represents the set of capabilities supported by the LSP client for Completion
type LSPCompletionClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPHoverClientCapabilities 悬停客户端能力 / Hover client capabilities
// 代表 LSP 客户端在 Hover 相关的功能集合
// Represents the set of capabilities supported by the LSP client for Hover
type LSPHoverClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPSignatureHelpClientCapabilities 签名帮助客户端能力 / Signature help client capabilities
// 代表 LSP 客户端在 SignatureHelp 相关的功能集合
// Represents the set of capabilities supported by the LSP client for SignatureHelp
type LSPSignatureHelpClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPDeclarationClientCapabilities 声明客户端能力 / Declaration client capabilities
// 代表 LSP 客户端在 Declaration 相关的功能集合
// Represents the set of capabilities supported by the LSP client for Declaration
type LSPDeclarationClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	LinkSupport         bool `json:"linkSupport,omitempty"`
}

// LSPDefinitionClientCapabilities 定义客户端能力 / Definition client capabilities
// 代表 LSP 客户端在 Definition 相关的功能集合
// Represents the set of capabilities supported by the LSP client for Definition
type LSPDefinitionClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	LinkSupport         bool `json:"linkSupport,omitempty"`
}

// LSPTypeDefinitionClientCapabilities 类型定义客户端能力 / Type definition client capabilities
// 代表 LSP 客户端在 TypeDefinition 相关的功能集合
// Represents the set of capabilities supported by the LSP client for TypeDefinition
type LSPTypeDefinitionClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	LinkSupport         bool `json:"linkSupport,omitempty"`
}

// LSPImplementationClientCapabilities 实现客户端能力 / Implementation client capabilities
// 代表 LSP 客户端在 Implementation 相关的功能集合
// Represents the set of capabilities supported by the LSP client for Implementation
type LSPImplementationClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	LinkSupport         bool `json:"linkSupport,omitempty"`
}

// LSPReferencesClientCapabilities 引用客户端能力 / References client capabilities
// 代表 LSP 客户端在 References 相关的功能集合
// Represents the set of capabilities supported by the LSP client for References
type LSPReferencesClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPDocumentHighlightClientCapabilities 文档高亮客户端能力 / Document highlight client capabilities
// 代表 LSP 客户端在 DocumentHighlight 相关的功能集合
// Represents the set of capabilities supported by the LSP client for DocumentHighlight
type LSPDocumentHighlightClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPDocumentSymbolClientCapabilities 文档符号客户端能力 / Document symbol client capabilities
// 代表 LSP 客户端在 DocumentSymbol 相关的功能集合
// Represents the set of capabilities supported by the LSP client for DocumentSymbol
type LSPDocumentSymbolClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPCodeActionClientCapabilities 代码操作客户端能力 / Code action client capabilities
// 代表 LSP 客户端在 CodeAction 相关的功能集合
// Represents the set of capabilities supported by the LSP client for CodeAction
type LSPCodeActionClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPCodeLensClientCapabilities 代码镜头客户端能力 / Code lens client capabilities
// 代表 LSP 客户端在 CodeLens 相关的功能集合
// Represents the set of capabilities supported by the LSP client for CodeLens
type LSPCodeLensClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPDocumentLinkClientCapabilities 文档链接客户端能力 / Document link client capabilities
// 代表 LSP 客户端在 DocumentLink 相关的功能集合
// Represents the set of capabilities supported by the LSP client for DocumentLink
type LSPDocumentLinkClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPDocumentColorClientCapabilities 文档颜色客户端能力 / Document color client capabilities
// 代表 LSP 客户端在 DocumentColor 相关的功能集合
// Represents the set of capabilities supported by the LSP client for DocumentColor
type LSPDocumentColorClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPDocumentFormattingClientCapabilities 文档格式化客户端能力 / Document formatting client capabilities
// 代表 LSP 客户端在 DocumentFormatting 相关的功能集合
// Represents the set of capabilities supported by the LSP client for DocumentFormatting
type LSPDocumentFormattingClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPDocumentRangeFormattingClientCapabilities 文档范围格式化客户端能力 / Document range formatting client capabilities
// 代表 LSP 客户端在 DocumentRangeFormatting 相关的功能集合
// Represents the set of capabilities supported by the LSP client for DocumentRangeFormatting
type LSPDocumentRangeFormattingClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPDocumentOnTypeFormattingClientCapabilities 文档输入时格式化客户端能力 / Document on type formatting client capabilities
// 代表 LSP 客户端在 DocumentOnTypeFormatting 相关的功能集合
// Represents the set of capabilities supported by the LSP client for DocumentOnTypeFormatting
type LSPDocumentOnTypeFormattingClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPRenameClientCapabilities 重命名客户端能力 / Rename client capabilities
// 代表 LSP 客户端在 Rename 相关的功能集合
// Represents the set of capabilities supported by the LSP client for Rename
type LSPRenameClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPPublishDiagnosticsClientCapabilities 发布诊断客户端能力 / Publish diagnostics client capabilities
// 代表 LSP 客户端在 PublishDiagnostics 相关的功能集合
// Represents the set of capabilities supported by the LSP client for PublishDiagnostics
type LSPPublishDiagnosticsClientCapabilities struct {
	RelatedInformation bool `json:"relatedInformation,omitempty"`
}

// LSPFoldingRangeClientCapabilities 折叠范围客户端能力 / Folding range client capabilities
// 代表 LSP 客户端在 FoldingRange 相关的功能集合
// Represents the set of capabilities supported by the LSP client for FoldingRange
type LSPFoldingRangeClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPSelectionRangeClientCapabilities 选择范围客户端能力 / Selection range client capabilities
// 代表 LSP 客户端在 SelectionRange 相关的功能集合
// Represents the set of capabilities supported by the LSP client for SelectionRange
type LSPSelectionRangeClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPLinkedEditingRangeClientCapabilities 链接编辑范围客户端能力 / Linked editing range client capabilities
// 代表 LSP 客户端在 LinkedEditingRange 相关的功能集合
// Represents the set of capabilities supported by the LSP client for LinkedEditingRange
type LSPLinkedEditingRangeClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPCallHierarchyClientCapabilities 调用层次客户端能力 / Call hierarchy client capabilities
// 代表 LSP 客户端在 CallHierarchy 相关的功能集合
// Represents the set of capabilities supported by the LSP client for CallHierarchy
type LSPCallHierarchyClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPSemanticTokensClientCapabilities 语义标记客户端能力 / Semantic tokens client capabilities
// 代表 LSP 客户端在 SemanticTokens 相关的功能集合
// Represents the set of capabilities supported by the LSP client for SemanticTokens
type LSPSemanticTokensClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPMonikerClientCapabilities 标记客户端能力 / Moniker client capabilities
// 代表 LSP 客户端在 Moniker 相关的功能集合
// Represents the set of capabilities supported by the LSP client for Moniker
type LSPMonikerClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPTypeHierarchyClientCapabilities 类型层次客户端能力 / Type hierarchy client capabilities
// 代表 LSP 客户端在 TypeHierarchy 相关的功能集合
// Represents the set of capabilities supported by the LSP client for TypeHierarchy
type LSPTypeHierarchyClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPInlineValueClientCapabilities 内联值客户端能力 / Inline value client capabilities
// 代表 LSP 客户端在 InlineValue 相关的功能集合
// Represents the set of capabilities supported by the LSP client for InlineValue
type LSPInlineValueClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPInlayHintClientCapabilities 内嵌提示客户端能力 / Inlay hint client capabilities
// 代表 LSP 客户端在 InlayHint 相关的功能集合
// Represents the set of capabilities supported by the LSP client for InlayHint
type LSPInlayHintClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPDiagnosticClientCapabilities 诊断客户端能力 / Diagnostic client capabilities
// 代表 LSP 客户端在 Diagnostic 相关的功能集合
// Represents the set of capabilities supported by the LSP client for Diagnostic
type LSPDiagnosticClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// LSPShowMessageRequestClientCapabilities 显示消息请求客户端能力 / Show message request client capabilities
// 代表 LSP 客户端在 ShowMessageRequest 相关的功能集合
// Represents the set of capabilities supported by the LSP client for ShowMessageRequest
type LSPShowMessageRequestClientCapabilities struct {
	MessageActionItem *LSPMessageActionItemCapabilities `json:"messageActionItem,omitempty"`
}

// LSPMessageActionItemCapabilities 消息操作项能力 / Message action item capabilities
// 代表 LSP 客户端在 MessageActionItem 相关的功能集合
// Represents the set of capabilities supported by the LSP client for MessageActionItem
type LSPMessageActionItemCapabilities struct {
	AdditionalPropertiesSupport bool `json:"additionalPropertiesSupport,omitempty"`
}

// LSPShowDocumentClientCapabilities 显示文档客户端能力 / Show document client capabilities
// 代表 LSP 客户端在 ShowDocument 相关的功能集合
// Represents the set of capabilities supported by the LSP client for ShowDocument
type LSPShowDocumentClientCapabilities struct {
	Support bool `json:"support"`
}

// LSPStaleRequestSupportOptions 过期请求支持选项 / Stale request support options
// 代表 LSP 客户端在 StaleRequestSupport 相关的功能集合
// Represents the set of capabilities supported by the LSP client for StaleRequestSupport
type LSPStaleRequestSupportOptions struct {
	Cancel                 bool     `json:"cancel"`
	RetryOnContentModified []string `json:"retryOnContentModified"`
}

// LSPRegularExpressionsClientCapabilities 正则表达式客户端能力 / Regular expressions client capabilities
// 代表 LSP 客户端在 RegularExpressions 相关的功能集合
// Represents the set of capabilities supported by the LSP client for RegularExpressions
type LSPRegularExpressionsClientCapabilities struct {
	Engine  string `json:"engine"`
	Version string `json:"version,omitempty"`
}

// LSPMarkdownClientCapabilities Markdown客户端能力 / Markdown client capabilities
// 代表 LSP 客户端在 Markdown 相关的功能集合
// Represents the set of capabilities supported by the LSP client for Markdown
type LSPMarkdownClientCapabilities struct {
	Parser      string   `json:"parser"`
	Version     string   `json:"version,omitempty"`
	AllowedTags []string `json:"allowedTags,omitempty"`
}
