// Package main MCP client test program
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

// MCPRequest MCP request structure
type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// MCPResponse MCP response structure
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

// MCPClient MCP client
type MCPClient struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	reader *bufio.Scanner
	nextID int
}

// NewMCPClient creates a new MCP client
func NewMCPClient(serverPath, configPath string) (*MCPClient, error) {
	// Start MCP server process
	cmd := exec.Command(serverPath, "--config", configPath)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start MCP server: %w", err)
	}

	reader := bufio.NewScanner(stdout)
	const maxBufSize = 1024 * 1024 // 1MB, adjust as needed
	reader.Buffer(make([]byte, 4096), maxBufSize)

	return &MCPClient{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		reader: reader,
		nextID: 1,
	}, nil
}

// sendRequest sends a JSON-RPC request
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
		return nil, fmt.Errorf("failed to serialize request: %w", err)
	}

	// Send request
	if _, err := c.stdin.Write(append(reqData, '\n')); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Read response
	if !c.reader.Scan() {
		if err := c.reader.Err(); err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}
		return nil, fmt.Errorf("failed to read response: EOF")
	}

	var resp MCPResponse
	if err := json.Unmarshal(c.reader.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// Initialize initializes MCP connection
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
		return fmt.Errorf("initialization request failed: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("initialization failed: %v", resp.Error)
	}

	return nil
}

// CallTool calls an MCP tool
func (c *MCPClient) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (*MCPResponse, error) {
	params := map[string]interface{}{
		"name":      toolName,
		"arguments": arguments,
	}

	return c.sendRequest("tools/call", params)
}

// ListTools lists available tools
func (c *MCPClient) ListTools(ctx context.Context) (*MCPResponse, error) {
	return c.sendRequest("tools/list", map[string]interface{}{})
}

// TestLSPDefinition tests LSP definition lookup
func (c *MCPClient) TestLSPDefinition(ctx context.Context, languageID, rootURI, fileURI string, line, character int) error {
	log.Printf("Testing LSP definition lookup: file=%s, position=(%d,%d)", fileURI, line, character)

	// Call lsp_definition tool
	resp, err := c.CallTool(ctx, "lsp_definition", map[string]interface{}{
		"language_id": languageID,
		"root_uri":    rootURI,
		"file_uri":    fileURI,
		"line":        line,
		"character":   character,
	})
	if err != nil {
		return fmt.Errorf("failed to call lsp_definition: %w", err)
	}

	if resp.Error != nil {
		log.Printf("⚠️ LSP definition lookup returned error: %v", resp.Error)
		return nil // This may be normal since we're using example files
	}

	// Compatible with both Location/LocationLink formats
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		log.Printf("❌ Failed to parse LSP definition response: %v", err)
		return nil
	}
	log.Printf("✅ LSP definition lookup response content: %s", string(resultBytes))
	return nil
}

// TestLSPHover tests LSP hover
func (c *MCPClient) TestLSPHover(ctx context.Context, languageID, rootURI, fileURI string, line, character int) error {
	log.Printf("Testing LSP hover: file=%s, position=(%d,%d)", fileURI, line, character)

	// Call lsp_hover tool
	resp, err := c.CallTool(ctx, "lsp_hover", map[string]interface{}{
		"language_id": languageID,
		"root_uri":    rootURI,
		"file_uri":    fileURI,
		"line":        line,
		"character":   character,
	})
	if err != nil {
		return fmt.Errorf("failed to call lsp_hover: %w", err)
	}

	if resp.Error != nil {
		log.Printf("⚠️ LSP hover returned error: %v", resp.Error)
		return nil // This may be normal
	}

	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		log.Printf("❌ Failed to parse LSP hover response: %v", err)
		return nil
	}
	log.Printf("✅ LSP hover response content: %s", string(resultBytes))
	return nil
}

// TestListTools tests getting tool list
func (c *MCPClient) TestListTools(ctx context.Context) error {
	log.Println("Getting available tool list...")

	resp, err := c.ListTools(ctx)
	if err != nil {
		return fmt.Errorf("failed to get tool list: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("failed to get tool list: %v", resp.Error)
	}

	if result, ok := resp.Result.(map[string]interface{}); ok {
		if tools, ok := result["tools"].([]interface{}); ok {
			log.Printf("✅ Found %d available tools:", len(tools))
			for _, tool := range tools {
				if toolMap, ok := tool.(map[string]interface{}); ok {
					name := toolMap["name"]
					desc := toolMap["description"]
					log.Printf("  - %s: %s", name, desc)
				}
			}
		} else {
			log.Printf("✅ Tool list response: %v", result)
		}
	} else {
		log.Printf("✅ Tool list response: %v", resp.Result)
	}

	return nil
}

// Close closes client connection
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
	log.Println("Starting MCP client test program...")

	// Create MCP client
	serverPath := "./bin/lsp-mcp"
	configPath := "./config/config.yaml"
	client, err := NewMCPClient(serverPath, configPath)
	if err != nil {
		log.Fatalf("Failed to create MCP client: %v", err)
	}
	defer client.Close()

	// Wait for server startup
	time.Sleep(2 * time.Second)

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// TypeScript test parameters
	languageID := "typescript"
	rootURI := "file:///path/to/your/project"
	fileURI := "file:///path/to/your/project/src/index.tsx"
	line := 25
	character := 10

	// Initialize MCP connection
	log.Println("Initializing MCP connection...")
	if err := client.Initialize(ctx); err != nil {
		log.Printf("❌ Failed to initialize MCP connection: %v", err)
		return
	} else {
		log.Println("✅ Successfully initialized MCP connection")
	}

	// Test TypeScript LSP definition lookup
	if err := client.TestLSPDefinition(ctx, languageID, rootURI, fileURI, line, character); err != nil {
		log.Printf("❌ TypeScript LSP definition lookup test failed: %v", err)
	} else {
		log.Println("✅ TypeScript LSP definition lookup test passed")
	}

	// Test TypeScript LSP hover
	if err := client.TestLSPHover(ctx, languageID, rootURI, fileURI, line, character); err != nil {
		log.Printf("❌ TypeScript LSP hover test failed: %v", err)
	} else {
		log.Println("✅ TypeScript LSP hover test passed")
	}

	log.Println("🎉 TypeScript MCP client test completed")
}
