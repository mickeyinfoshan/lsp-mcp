// handlers.go
// Handles MCP tool requests and adapts them to LSP operations.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/mickeyinfoshan/lsp-mcp/pkg/types"
	protocol "go.lsp.dev/protocol"
)

// handleLSPInitialize handles LSP initialize requests
// Parameters: ctx - context, req - MCP tool request
// Returns: MCP tool call result and error information
// Used to initialize LSP session, validate parameters and call LSP client
func (s *Server) handleLSPInitialize(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters from request
	languageId, err := req.RequireString("language_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	rootUri, err := req.RequireString("root_uri")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Verify if language ID is supported
	if _, exists := s.config.GetLSPServerConfig(languageId); !exists {
		return mcp.NewToolResultError(fmt.Sprintf("unsupported language: %s", languageId)), nil
	}

	// Directly call LSP client's GetOrCreateSession method to initialize session
	// This avoids the complex logic of triggering initialization through virtual files
	_, err = s.sessionManager.GetLSPClient().GetOrCreateSession(languageId, rootUri)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("LSP session initialization failed: %v", err)), nil
	}

	// Build success response
	response := map[string]interface{}{
		"success":     true,
		"language_id": languageId,
		"root_uri":    rootUri,
		"message":     "LSP session initialized successfully",
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to serialize response: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// handleLSPShutdown handles LSP shutdown requests
func (s *Server) handleLSPShutdown(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters from request
	languageId, err := req.RequireString("language_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	rootUri, err := req.RequireString("root_uri")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Build response
	response := map[string]interface{}{
		"success":     true,
		"language_id": languageId,
		"root_uri":    rootUri,
		"message":     "LSP session shutdown request received",
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to serialize response: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// handleInitialize handles lsp.initialize tool calls
func (s *Server) handleInitialize(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return s.handleLSPInitialize(ctx, req)
}

// handleShutdown handles lsp.shutdown tool calls
func (s *Server) handleShutdown(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return s.handleLSPShutdown(ctx, req)
}

// Wrapper: intelligently extract and adjust character based on symbol, and output logs
func adjustCharacterBySymbol(fileUri string, line, character int) (int, error) {
	filePath := strings.TrimPrefix(fileUri, "file://")
	lineContent, err := readFileLine(filePath, line)
	if err != nil {
		log.Printf("[symbol-extract] Failed to read file: %s line=%d err=%v", fileUri, line, err)
		return character, err
	}
	symbol, start := extractSymbolSmart(lineContent, character)
	if start >= 0 && len(symbol) > 0 {
		if start <= character && character < start+len(symbol) {
			// character is already inside symbol, no adjustment
			log.Printf("[symbol-extract] %s line=%d original char=%d already inside symbol '%s', keeping position", fileUri, line, character, symbol)
			return character, nil
		} else {
			// character is not inside symbol, adjust to symbol start
			log.Printf("[symbol-extract] %s line=%d original char=%d adjusted char=%d symbol='%s'", fileUri, line, character, start, symbol)
			return start, nil
		}
	}
	log.Printf("[symbol-extract] %s line=%d original char=%d no symbol extracted, keeping position", fileUri, line, character)
	return character, nil
}

// handleDefinition handles lsp.definition tool calls
func (s *Server) handleDefinition(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	languageId, err := req.RequireString("language_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	fileUri, err := req.RequireString("file_uri")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	rootUriInput, err := req.RequireString("root_uri")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	// Automatically find rootUri, use rootUriInput as workspaceRoot fallback if not found
	rootUri := findProjectRoot(fileUri, languageId, strings.TrimPrefix(rootUriInput, "file://"))
	line, err := req.RequireInt("line")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	character, err := req.RequireInt("character")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	symbol := req.GetString("symbol", "")
	pos := protocol.Position{Line: uint32(line), Character: uint32(character)}
	if symbol != "" {
		pos = findSymbolPositionInFile(fileUri, pos, symbol)
		line = int(pos.Line)
		character = int(pos.Character)
	}
	findDefReq := &types.FindDefinitionRequest{
		LanguageID: languageId,
		RootURI:    rootUri,
		FileURI:    fileUri,
		Position: protocol.Position{
			Line:      uint32(line),
			Character: uint32(character),
		},
	}
	response, err := s.sessionManager.FindDefinition(ctx, findDefReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to find definition: %v", err)), nil
	}
	text := response.Message
	if len(response.AgentResults) > 0 {
		for _, r := range response.AgentResults {
			text += "\n" + r.Summary
		}
	}
	content := types.MCPContent{
		Type: "text",
		Text: text,
		Data: response,
	}
	toolResp := types.MCPToolResponse{
		Content: []types.MCPContent{content},
	}
	jsonData, err := json.Marshal(toolResp)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to serialize response: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

// handleReferences handles lsp.references tool calls
func (s *Server) handleReferences(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	languageId, err := req.RequireString("language_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	fileUri, err := req.RequireString("file_uri")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	rootUriInput, err := req.RequireString("root_uri")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	rootUri := findProjectRoot(fileUri, languageId, strings.TrimPrefix(rootUriInput, "file://"))
	line, err := req.RequireInt("line")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	character, err := req.RequireInt("character")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	includeDeclaration := req.GetBool("include_declaration", false)
	symbol := req.GetString("symbol", "")
	pos := protocol.Position{Line: uint32(line), Character: uint32(character)}
	if symbol != "" {
		pos = findSymbolPositionInFile(fileUri, pos, symbol)
		line = int(pos.Line)
		character = int(pos.Character)
	}
	findRefReq := &types.FindReferencesRequest{
		LanguageID: languageId,
		RootURI:    rootUri,
		FileURI:    fileUri,
		Position: protocol.Position{
			Line:      uint32(line),
			Character: uint32(character),
		},
		IncludeDeclaration: includeDeclaration,
	}
	response, err := s.sessionManager.FindReferences(ctx, findRefReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to find references: %v", err)), nil
	}
	text := response.Message
	if len(response.AgentResults) > 0 {
		for _, r := range response.AgentResults {
			text += "\n" + r.Summary
		}
	}
	content := types.MCPContent{
		Type: "text",
		Text: text,
		Data: response,
	}
	toolResp := types.MCPToolResponse{
		Content: []types.MCPContent{content},
	}
	jsonData, err := json.Marshal(toolResp)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to serialize response: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

// handleHover handles lsp.hover tool calls
func (s *Server) handleHover(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	languageId, err := req.RequireString("language_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	fileUri, err := req.RequireString("file_uri")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	rootUriInput, err := req.RequireString("root_uri")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	rootUri := findProjectRoot(fileUri, languageId, strings.TrimPrefix(rootUriInput, "file://"))
	line, err := req.RequireInt("line")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	character, err := req.RequireInt("character")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	symbol := req.GetString("symbol", "")
	pos := protocol.Position{Line: uint32(line), Character: uint32(character)}
	if symbol != "" {
		pos = findSymbolPositionInFile(fileUri, pos, symbol)
		line = int(pos.Line)
		character = int(pos.Character)
	}
	hoverReq := &types.HoverRequest{
		LanguageID: languageId,
		RootURI:    rootUri,
		FileURI:    fileUri,
		Position: protocol.Position{
			Line:      uint32(line),
			Character: uint32(character),
		},
	}
	response, err := s.sessionManager.GetHover(ctx, hoverReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get hover information: %v", err)), nil
	}
	text := response.Message
	if len(response.AgentResults) > 0 {
		for _, r := range response.AgentResults {
			text += "\n" + r.Summary
		}
	}
	content := types.MCPContent{
		Type: "text",
		Text: text,
		Data: response,
	}
	toolResp := types.MCPToolResponse{
		Content: []types.MCPContent{content},
	}
	jsonData, err := json.Marshal(toolResp)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to serialize response: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

// handleCompletion handles lsp.completion tool calls
func (s *Server) handleCompletion(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	languageId, err := req.RequireString("language_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	fileUri, err := req.RequireString("file_uri")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	rootUriInput, err := req.RequireString("root_uri")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	rootUri := findProjectRoot(fileUri, languageId, strings.TrimPrefix(rootUriInput, "file://"))
	line, err := req.RequireInt("line")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	character, err := req.RequireInt("character")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	triggerKind := req.GetInt("trigger_kind", 1)
	triggerCharacter := req.GetString("trigger_character", "")
	if newChar, _ := adjustCharacterBySymbol(fileUri, line, character); newChar != character {
		character = newChar
	}
	completionReq := &types.CompletionRequest{
		LanguageID: languageId,
		RootURI:    rootUri,
		FileURI:    fileUri,
		Position: protocol.Position{
			Line:      uint32(line),
			Character: uint32(character),
		},
		TriggerKind:      triggerKind,
		TriggerCharacter: triggerCharacter,
	}
	response, err := s.sessionManager.GetCompletion(ctx, completionReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get code completion: %v", err)), nil
	}
	text := response.Message
	content := types.MCPContent{
		Type: "text",
		Text: text,
		Data: response,
	}
	toolResp := types.MCPToolResponse{
		Content: []types.MCPContent{content},
	}
	jsonData, err := json.Marshal(toolResp)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to serialize response: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

// getCompletionCount gets the number of completion items
func getCompletionCount(completions interface{}) int {
	switch v := completions.(type) {
	case []interface{}:
		return len(v)
	case map[string]interface{}:
		if items, ok := v["items"].([]interface{}); ok {
			return len(items)
		}
		return 1
	default:
		return 0
	}
}

// formatLSPError formats LSP error information
func formatLSPError(method string, err error) string {
	if err == nil {
		return ""
	}

	errorMsg := err.Error()
	if strings.Contains(errorMsg, "timeout") {
		return fmt.Sprintf("LSP request '%s' timed out: %s", method, errorMsg)
	}

	if strings.Contains(errorMsg, "connection") {
		return fmt.Sprintf("LSP connection error '%s': %s", method, errorMsg)
	}

	return fmt.Sprintf("LSP request '%s' failed: %s", method, errorMsg)
}

// validateURI validates URI format
func validateURI(uri string) error {
	if uri == "" {
		return fmt.Errorf("URI cannot be empty")
	}

	if !strings.HasPrefix(uri, "file://") {
		return fmt.Errorf("URI must start with 'file://'")
	}

	return nil
}

// validatePosition validates position parameters
func validatePosition(line, character float64) error {
	if line < 0 {
		return fmt.Errorf("line number cannot be negative")
	}

	if character < 0 {
		return fmt.Errorf("character position cannot be negative")
	}

	return nil
}

// convertToJSONString converts an object to a JSON string
func convertToJSONString(obj interface{}) string {
	if obj == nil {
		return "null"
	}

	data, err := json.Marshal(obj)
	if err != nil {
		return fmt.Sprintf("JSON serialization error: %v", err)
	}

	return string(data)
}

// Symbol intelligent extraction function, see design document
func extractSymbolSmart(line string, character int) (string, int) {
	pos := character
	for pos < len(line) && !isSymbolChar(line[pos]) {
		pos++
	}
	if pos == len(line) {
		return "", -1
	}
	start := pos
	for start > 0 && isSymbolChar(line[start-1]) {
		start--
	}
	end := pos
	for end < len(line) && isSymbolChar(line[end]) {
		end++
	}
	return line[start:end], start
}

func isSymbolChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '_'
}

// Read only the specified line content
func readFileLine(filePath string, lineNum int) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	current := 0
	for scanner.Scan() {
		if current == lineNum {
			return scanner.Text(), nil
		}
		current++
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("line %d out of range", lineNum)
}

// findSymbolPositionInFile searches for symbol in the specified file's [line-10, line+10) range, expands to [line-50, line+50) if not found, returns protocol.Position, returns original position if not found
func findSymbolPositionInFile(fileUri string, orig protocol.Position, symbol string) protocol.Position {
	if symbol == "" {
		return orig
	}
	filePath := strings.TrimPrefix(fileUri, "file://")
	// Try twice: first ±10 lines, if not found then ±50 lines
	for _, delta := range []int{10, 50} {
		startLine := int(orig.Line) - delta
		if startLine < 0 {
			startLine = 0
		}
		endLine := int(orig.Line) + delta

		lines, err := readLinesRange(filePath, startLine, endLine)
		if err != nil {
			return orig
		}
		positions := make([]protocol.Position, 0)
		inBlockComment := false
		for i, lineStr := range lines {
			trimmed := strings.TrimSpace(lineStr)
			// Skip single-line comments
			if strings.HasPrefix(trimmed, "//") {
				continue
			}
			// Handle multi-line comments
			if inBlockComment {
				if idx := strings.Index(trimmed, "*/"); idx != -1 {
					inBlockComment = false
					// Continue processing content after comment
					trimmed = trimmed[idx+2:]
					lineStr = lineStr[idx+2:]
				} else {
					continue
				}
			}
			if idx := strings.Index(trimmed, "/*"); idx != -1 {
				inBlockComment = true
				// Only process content before comment
				lineStr = lineStr[:idx]
			}
			// Search for symbol here
			idx := 0
			for {
				pos := strings.Index(lineStr[idx:], symbol)
				if pos == -1 {
					break
				}
				positions = append(positions, protocol.Position{
					Line:      uint32(startLine + i),
					Character: uint32(idx + pos),
				})
				idx += pos + len(symbol)
			}
		}
		if len(positions) > 0 {
			// Select the one closest to orig
			minDist := -1
			var nearest protocol.Position
			for _, pos := range positions {
				dist := absInt(int(pos.Line)-int(orig.Line)) + absInt(int(pos.Character)-int(orig.Character))
				if minDist == -1 || dist < minDist {
					minDist = dist
					nearest = pos
				}
			}
			return nearest
		}
	}
	return orig
}

// readLinesRange reads all lines in the [startLine, endLine) range of a file
func readLinesRange(filePath string, startLine, endLine int) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var lines []string
	scanner := bufio.NewScanner(file)
	curLine := 0
	for scanner.Scan() {
		if curLine >= startLine && curLine < endLine {
			lines = append(lines, scanner.Text())
		}
		if curLine >= endLine {
			break
		}
		curLine++
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// findProjectRoot automatically finds project root directory, supports multiple languages and node_modules special handling
// workspaceRoot defaults to "/", can be extended via configuration file
func findProjectRoot(fileUri, languageId, workspaceRoot string) string {
	filePath := strings.TrimPrefix(fileUri, "file://")
	// node_modules special handling
	if languageId == "typescript" || languageId == "javascript" {
		if idx := strings.Index(filePath, "node_modules"); idx > 0 {
			filePath = filePath[:idx]
		}
	}
	searchFiles := map[string][]string{
		"go":         {"go.mod"},
		"typescript": {"tsconfig.json"},
		"javascript": {"tsconfig.json"},
		"python":     {"pyproject.toml", "setup.py"},
		// ... other languages can be extended
	}
	dir := filepath.Dir(filePath)
	for {
		for _, marker := range searchFiles[languageId] {
			if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
				return "file://" + dir
			}
		}
		if dir == workspaceRoot || dir == "/" {
			break
		}
		dir = filepath.Dir(dir)
	}
	return "file://" + workspaceRoot
}
