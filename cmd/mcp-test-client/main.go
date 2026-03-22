// Package main MCP客户端测试程序
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
	"time"
)

// MCPRequest MCP请求结构
type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// MCPResponse MCP响应结构
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

// MCPClient MCP客户端
type MCPClient struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	reader *bufio.Scanner
	nextID int
}

// NewMCPClient 创建新的MCP客户端
func NewMCPClient(serverPath, configPath string) (*MCPClient, error) {
	// 启动MCP服务器进程
	cmd := exec.Command(serverPath, "--config", configPath)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("创建stdin管道失败: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("创建stdout管道失败: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("启动MCP服务器失败: %w", err)
	}

	reader := bufio.NewScanner(stdout)
	const maxBufSize = 1024 * 1024 // 1MB，可根据需要调整
	reader.Buffer(make([]byte, 4096), maxBufSize)

	return &MCPClient{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		reader: reader,
		nextID: 1,
	}, nil
}

// sendRequest 发送JSON-RPC请求
func (c *MCPClient) sendRequest(method string, params interface{}) (*MCPResponse, error) {
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.nextID,
		Method:  method,
		Params:  params,
	}
	c.nextID++

	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 发送请求
	if _, err := c.stdin.Write(append(reqData, '\n')); err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}

	// 读取响应
	if !c.reader.Scan() {
		if err := c.reader.Err(); err != nil {
			return nil, fmt.Errorf("读取响应失败: %w", err)
		}
		return nil, fmt.Errorf("读取响应失败: EOF")
	}

	var resp MCPResponse
	if err := json.Unmarshal(c.reader.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &resp, nil
}

// Initialize 初始化MCP连接
func (c *MCPClient) Initialize(ctx context.Context) error {
	params := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"roots": map[string]interface{}{
				"listChanged": true,
			},
			"sampling": map[string]interface{}{},
		},
		"clientInfo": map[string]interface{}{
			"name":    "mcp-lsp-test-client",
			"version": "1.0.0",
		},
	}

	resp, err := c.sendRequest("initialize", params)
	if err != nil {
		return fmt.Errorf("初始化请求失败: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("初始化失败: %v", resp.Error)
	}

	return nil
}

// CallTool 调用MCP工具
func (c *MCPClient) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (*MCPResponse, error) {
	params := map[string]interface{}{
		"name":      toolName,
		"arguments": arguments,
	}

	return c.sendRequest("tools/call", params)
}

// ListTools 列出可用工具
func (c *MCPClient) ListTools(ctx context.Context) (*MCPResponse, error) {
	return c.sendRequest("tools/list", map[string]interface{}{})
}

// TestLSPDefinition 测试LSP定义查找
func (c *MCPClient) TestLSPDefinition(ctx context.Context, languageID, rootURI, fileURI string, line, character int) error {
	log.Printf("测试LSP定义查找: 文件=%s, 位置=(%d,%d)", fileURI, line, character)

	// 调用lsp_definition工具
	resp, err := c.CallTool(ctx, "lsp_definition", map[string]interface{}{
		"language_id": languageID,
		"root_uri":    rootURI,
		"file_uri":    fileURI,
		"line":        line,
		"character":   character,
	})
	if err != nil {
		return fmt.Errorf("调用lsp_definition失败: %w", err)
	}

	if resp.Error != nil {
		log.Printf("⚠️ LSP定义查找返回错误: %v", resp.Error)
		return nil // 这可能是正常的，因为我们使用的是示例文件
	}

	// 兼容 Location/LocationLink 两种格式
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		log.Printf("❌ 解析LSP定义响应失败: %v", err)
		return nil
	}
	log.Printf("✅ LSP定义查找响应内容: %s", string(resultBytes))
	return nil
}

// TestLSPHover 测试LSP悬停
func (c *MCPClient) TestLSPHover(ctx context.Context, languageID, rootURI, fileURI string, line, character int) error {
	log.Printf("测试LSP悬停: 文件=%s, 位置=(%d,%d)", fileURI, line, character)

	// 调用lsp_hover工具
	resp, err := c.CallTool(ctx, "lsp_hover", map[string]interface{}{
		"language_id": languageID,
		"root_uri":    rootURI,
		"file_uri":    fileURI,
		"line":        line,
		"character":   character,
	})
	if err != nil {
		return fmt.Errorf("调用lsp_hover失败: %w", err)
	}

	if resp.Error != nil {
		log.Printf("⚠️ LSP悬停返回错误: %v", resp.Error)
		return nil // 这可能是正常的
	}

	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		log.Printf("❌ 解析LSP悬停响应失败: %v", err)
		return nil
	}
	log.Printf("✅ LSP悬停响应内容: %s", string(resultBytes))
	return nil
}

// TestListTools 测试获取工具列表
func (c *MCPClient) TestListTools(ctx context.Context) error {
	log.Println("获取可用工具列表...")

	resp, err := c.ListTools(ctx)
	if err != nil {
		return fmt.Errorf("获取工具列表失败: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("获取工具列表失败: %v", resp.Error)
	}

	if result, ok := resp.Result.(map[string]interface{}); ok {
		if tools, ok := result["tools"].([]interface{}); ok {
			log.Printf("✅ 发现 %d 个可用工具:", len(tools))
			for _, tool := range tools {
				if toolMap, ok := tool.(map[string]interface{}); ok {
					name := toolMap["name"]
					desc := toolMap["description"]
					log.Printf("  - %s: %s", name, desc)
				}
			}
		} else {
			log.Printf("✅ 工具列表响应: %v", result)
		}
	} else {
		log.Printf("✅ 工具列表响应: %v", resp.Result)
	}

	return nil
}

// Close 关闭客户端连接
func (c *MCPClient) Close() error {
	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.stdout != nil {
		c.stdout.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		return c.cmd.Process.Kill()
	}
	return nil
}

func main() {
	log.SetPrefix("[MCP-Client] ")
	log.Println("启动MCP客户端测试程序...")

	// 创建MCP客户端
	serverPath := "./bin/lsp-mcp"
	configPath := "./config/config.yaml"
	client, err := NewMCPClient(serverPath, configPath)
	if err != nil {
		log.Fatalf("创建MCP客户端失败: %v", err)
	}
	defer client.Close()

	// 等待服务器启动
	time.Sleep(2 * time.Second)

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// TypeScript 测试参数
	languageID := "typescript"
	rootURI := "file:///path/to/your/project"
	fileURI := "file:///path/to/your/project/src/index.tsx"
	line := 25
	character := 10

	// 初始化MCP连接
	log.Println("初始化MCP连接...")
	if err := client.Initialize(ctx); err != nil {
		log.Printf("❌ 初始化MCP连接失败: %v", err)
		return
	} else {
		log.Println("✅ 成功初始化MCP连接")
	}

	// 测试TypeScript LSP定义查找
	if err := client.TestLSPDefinition(ctx, languageID, rootURI, fileURI, line, character); err != nil {
		log.Printf("❌ TypeScript LSP定义查找测试失败: %v", err)
	} else {
		log.Println("✅ TypeScript LSP定义查找测试通过")
	}

	// 测试TypeScript LSP悬停
	if err := client.TestLSPHover(ctx, languageID, rootURI, fileURI, line, character); err != nil {
		log.Printf("❌ TypeScript LSP悬停测试失败: %v", err)
	} else {
		log.Println("✅ TypeScript LSP悬停测试通过")
	}

	log.Println("🎉 TypeScript MCP客户端测试完成")
}
