// handlers.go
// MCP 工具请求的 LSP 适配与处理逻辑
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

// handleLSPInitialize 处理 LSP 初始化请求
// handleLSPInitialize handles LSP initialize requests
// 参数: ctx - 上下文，req - MCP 工具请求
// 返回值: MCP 工具调用结果和错误信息
// 用于初始化 LSP 会话，校验参数并调用 LSP 客户端
func (s *Server) handleLSPInitialize(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 从请求中提取参数
	languageId, err := req.RequireString("language_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	rootUri, err := req.RequireString("root_uri")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 验证语言ID是否支持
	if _, exists := s.config.GetLSPServerConfig(languageId); !exists {
		return mcp.NewToolResultError(fmt.Sprintf("不支持的语言: %s", languageId)), nil
	}

	// 直接调用LSP客户端的GetOrCreateSession方法来初始化会话
	// 这样避免了通过虚拟文件触发初始化的复杂逻辑
	_, err = s.sessionManager.GetLSPClient().GetOrCreateSession(languageId, rootUri)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("LSP会话初始化失败: %v", err)), nil
	}

	// 构建成功响应
	response := map[string]interface{}{
		"success":     true,
		"language_id": languageId,
		"root_uri":    rootUri,
		"message":     "LSP会话初始化成功",
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("序列化响应失败: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// handleLSPShutdown 处理LSP关闭请求
func (s *Server) handleLSPShutdown(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 从请求中提取参数
	languageId, err := req.RequireString("language_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	rootUri, err := req.RequireString("root_uri")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 构建响应
	response := map[string]interface{}{
		"success":     true,
		"language_id": languageId,
		"root_uri":    rootUri,
		"message":     "LSP会话关闭请求已接收",
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("序列化响应失败: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// handleInitialize 处理lsp.initialize工具调用
func (s *Server) handleInitialize(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return s.handleLSPInitialize(ctx, req)
}

// handleShutdown 处理lsp.shutdown工具调用
func (s *Server) handleShutdown(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return s.handleLSPShutdown(ctx, req)
}

// 封装：根据 symbol 智能提取修正 character，并输出日志
func adjustCharacterBySymbol(fileUri string, line, character int) (int, error) {
	filePath := strings.TrimPrefix(fileUri, "file://")
	lineContent, err := readFileLine(filePath, line)
	if err != nil {
		log.Printf("[symbol-extract] 读取文件失败: %s line=%d err=%v", fileUri, line, err)
		return character, err
	}
	symbol, start := extractSymbolSmart(lineContent, character)
	if start >= 0 && len(symbol) > 0 {
		if start <= character && character < start+len(symbol) {
			// character 已经在 symbol 内部，不修正
			log.Printf("[symbol-extract] %s line=%d 原始char=%d 已在symbol '%s' 内, 保持原位", fileUri, line, character, symbol)
			return character, nil
		} else {
			// character 不在 symbol 内部，修正到 symbol 起始
			log.Printf("[symbol-extract] %s line=%d 原始char=%d 修正char=%d symbol='%s'", fileUri, line, character, start, symbol)
			return start, nil
		}
	}
	log.Printf("[symbol-extract] %s line=%d 原始char=%d 未提取到symbol，保持原位", fileUri, line, character)
	return character, nil
}

// handleDefinition 处理lsp.definition工具调用
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
	// 自动查找 rootUri，找不到时用 rootUriInput 作为 workspaceRoot 兜底
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
		return mcp.NewToolResultError(fmt.Sprintf("查找定义失败: %v", err)), nil
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
		return mcp.NewToolResultError(fmt.Sprintf("序列化响应失败: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

// handleReferences 处理lsp.references工具调用
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
		return mcp.NewToolResultError(fmt.Sprintf("查找引用失败: %v", err)), nil
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
		return mcp.NewToolResultError(fmt.Sprintf("序列化响应失败: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

// handleHover 处理lsp.hover工具调用
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
		return mcp.NewToolResultError(fmt.Sprintf("获取悬停信息失败: %v", err)), nil
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
		return mcp.NewToolResultError(fmt.Sprintf("序列化响应失败: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

// handleCompletion 处理lsp.completion工具调用
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
		return mcp.NewToolResultError(fmt.Sprintf("获取代码补全失败: %v", err)), nil
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
		return mcp.NewToolResultError(fmt.Sprintf("序列化响应失败: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

// getCompletionCount 获取补全项数量
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

// formatLSPError 格式化LSP错误信息
func formatLSPError(method string, err error) string {
	if err == nil {
		return ""
	}

	errorMsg := err.Error()
	if strings.Contains(errorMsg, "timeout") {
		return fmt.Sprintf("LSP请求'%s'超时: %s", method, errorMsg)
	}

	if strings.Contains(errorMsg, "connection") {
		return fmt.Sprintf("LSP连接错误'%s': %s", method, errorMsg)
	}

	return fmt.Sprintf("LSP请求'%s'失败: %s", method, errorMsg)
}

// validateURI 验证URI格式
func validateURI(uri string) error {
	if uri == "" {
		return fmt.Errorf("URI不能为空")
	}

	if !strings.HasPrefix(uri, "file://") {
		return fmt.Errorf("URI必须以'file://'开头")
	}

	return nil
}

// validatePosition 验证位置参数
func validatePosition(line, character float64) error {
	if line < 0 {
		return fmt.Errorf("行号不能为负数")
	}

	if character < 0 {
		return fmt.Errorf("字符位置不能为负数")
	}

	return nil
}

// convertToJSONString 将对象转换为JSON字符串
func convertToJSONString(obj interface{}) string {
	if obj == nil {
		return "null"
	}

	data, err := json.Marshal(obj)
	if err != nil {
		return fmt.Sprintf("JSON序列化错误: %v", err)
	}

	return string(data)
}

// symbol 智能提取函数，参考方案文档
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

// 只读取指定行内容
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

// findSymbolPositionInFile 按方案在 fileUri 指定文件的 [line-10, line+10) 范围查找 symbol，未找到再扩大到 [line-50, line+50)，返回 protocol.Position，未找到则返回原始位置
func findSymbolPositionInFile(fileUri string, orig protocol.Position, symbol string) protocol.Position {
	if symbol == "" {
		return orig
	}
	filePath := strings.TrimPrefix(fileUri, "file://")
	// 尝试两次：先±10行，未找到再±50行
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
			// 跳过单行注释
			if strings.HasPrefix(trimmed, "//") {
				continue
			}
			// 处理多行注释
			if inBlockComment {
				if idx := strings.Index(trimmed, "*/"); idx != -1 {
					inBlockComment = false
					// 继续处理注释后面的内容
					trimmed = trimmed[idx+2:]
					lineStr = lineStr[idx+2:]
				} else {
					continue
				}
			}
			if idx := strings.Index(trimmed, "/*"); idx != -1 {
				inBlockComment = true
				// 只处理注释前的内容
				lineStr = lineStr[:idx]
			}
			// 这里再查找 symbol
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
			// 选取距离 orig 最近的
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

// readLinesRange 读取文件 [startLine, endLine) 范围的所有行
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

// findProjectRoot 自动查找项目根目录，支持多语言和 node_modules 特殊处理
// workspaceRoot 默认为 "/"，可通过配置文件扩展
func findProjectRoot(fileUri, languageId, workspaceRoot string) string {
	filePath := strings.TrimPrefix(fileUri, "file://")
	// node_modules 特殊处理
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
		// ... 其它语言可扩展
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
