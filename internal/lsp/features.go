package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mickeyinfoshan/lsp-mcp/internal/logger"
	"github.com/mickeyinfoshan/lsp-mcp/pkg/types"
)

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
	logger.Debugf("[DEBUG] definition request params: %s", string(paramsJson))

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
