package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.yaml")

	configContent := `
lsp_servers:
  go:
    command: "gopls"
    args: []
    initialization_options: {}
  typescript:
    command: "typescript-language-server"
    args: ["--stdio"]
    initialization_options: {}

mcp_server:
  name: "test-lsp-bridge"
  version: "1.0.0"
  description: "Test LSP Bridge Service"

logging:
  level: "info"
  format: "text"
  file_output: false
  file_path: ""

session:
  max_sessions: 3
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	// 测试加载配置
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证配置内容
	if cfg.MCPServer.Name != "test-lsp-bridge" {
		t.Errorf("期望 MCP 服务器名称为 'test-lsp-bridge'，实际为 '%s'", cfg.MCPServer.Name)
	}

	if cfg.MCPServer.Version != "1.0.0" {
		t.Errorf("期望 MCP 服务器版本为 '1.0.0'，实际为 '%s'", cfg.MCPServer.Version)
	}

	if len(cfg.LSPServers) != 2 {
		t.Errorf("期望 2 个 LSP 服务器，实际为 %d", len(cfg.LSPServers))
	}

	// 验证 Go LSP 服务器配置
	goConfig, exists := cfg.LSPServers["go"]
	if !exists {
		t.Error("Go LSP 服务器配置不存在")
	}
	if goConfig.Command != "gopls" {
		t.Errorf("期望 Go LSP 命令为 'gopls'，实际为 '%s'", goConfig.Command)
	}

	// 验证 TypeScript LSP 服务器配置
	tsConfig, exists := cfg.LSPServers["typescript"]
	if !exists {
		t.Error("TypeScript LSP 服务器配置不存在")
	}
	if tsConfig.Command != "typescript-language-server" {
		t.Errorf("期望 TypeScript LSP 命令为 'typescript-language-server'，实际为 '%s'", tsConfig.Command)
	}
	if len(tsConfig.Args) != 1 || tsConfig.Args[0] != "--stdio" {
		t.Errorf("期望 TypeScript LSP 参数为 ['--stdio']，实际为 %v", tsConfig.Args)
	}

	// 验证日志配置
	if cfg.Logging.Level != "info" {
		t.Errorf("期望日志级别为 'info'，实际为 '%s'", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "text" {
		t.Errorf("期望日志格式为 'text'，实际为 '%s'", cfg.Logging.Format)
	}
	if cfg.Logging.FileOutput != false {
		t.Errorf("期望文件输出为 false，实际为 %v", cfg.Logging.FileOutput)
	}

	// 验证会话配置
	if cfg.Session.MaxSessions != 3 {
		t.Errorf("期望最大会话数为 3，实际为 %d", cfg.Session.MaxSessions)
	}
}

// TestGetLSPServerConfig 测试获取LSP服务器配置
func TestGetLSPServerConfig(t *testing.T) {
	config := &Config{
		LSPServers: map[string]*LSPServerConfig{
			"go": {
				Command: "gopls",
				Args:    []string{"serve"},
				InitializationOptions: map[string]interface{}{
					"usePlaceholders": true,
				},
			},
			"python": {
				Command: "pylsp",
				Args:    []string{},
			},
		},
	}

	// 测试存在的语言ID
	goConfig, exists := config.GetLSPServerConfig("go")
	if !exists {
		t.Error("期望找到Go LSP服务器配置")
	}
	if goConfig.Command != "gopls" {
		t.Errorf("期望Go LSP命令为 'gopls'，实际为 '%s'", goConfig.Command)
	}
	if len(goConfig.Args) != 1 || goConfig.Args[0] != "serve" {
		t.Errorf("期望Go LSP参数为 ['serve']，实际为 %v", goConfig.Args)
	}
	if goConfig.InitializationOptions["usePlaceholders"] != true {
		t.Error("期望Go LSP初始化选项包含 usePlaceholders: true")
	}

	// 测试不存在的语言ID
	_, exists = config.GetLSPServerConfig("nonexistent")
	if exists {
		t.Error("期望不存在的语言ID返回false")
	}
}

// TestLoadConfigFromDefault 测试从默认路径加载配置
func TestLoadConfigFromDefault(t *testing.T) {
	// 由于LoadConfigFromDefault依赖于当前工作目录和文件系统，
	// 这里主要测试函数调用不会panic
	_, err := LoadConfigFromDefault()
	// 期望返回错误，因为默认配置文件可能不存在
	if err == nil {
		t.Log("警告: 默认配置文件存在，这可能不是期望的测试环境")
	}
}

// TestValidateConfigEdgeCases 测试配置验证的边界情况
func TestValidateConfigEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "空语言ID",
			config: &Config{
				LSPServers: map[string]*LSPServerConfig{
					"": {
						Command: "gopls",
					},
				},
				MCPServer: &MCPServerConfig{
					Name:    "test",
					Version: "1.0.0",
				},
				Logging: &LoggingConfig{
					Level:  "info",
					Format: "text",
				},
				Session: &SessionConfig{
					MaxSessions: 5,
				},
			},
			expectError: true,
			errorMsg:    "语言ID不能为空",
		},
		{
			name: "无效日志级别",
			config: &Config{
				LSPServers: map[string]*LSPServerConfig{
					"go": {
						Command: "gopls",
					},
				},
				MCPServer: &MCPServerConfig{
					Name:    "test",
					Version: "1.0.0",
				},
				Logging: &LoggingConfig{
					Level:  "invalid",
					Format: "text",
				},
				Session: &SessionConfig{
					MaxSessions: 5,
				},
			},
			expectError: true,
			errorMsg:    "无效的日志级别",
		},
		{
			name: "无效日志格式",
			config: &Config{
				LSPServers: map[string]*LSPServerConfig{
					"go": {
						Command: "gopls",
					},
				},
				MCPServer: &MCPServerConfig{
					Name:    "test",
					Version: "1.0.0",
				},
				Logging: &LoggingConfig{
					Level:  "info",
					Format: "invalid",
				},
				Session: &SessionConfig{
					MaxSessions: 5,
				},
			},
			expectError: true,
			errorMsg:    "无效的日志格式",
		},
		{
			name: "最大会话数无效",
			config: &Config{
				LSPServers: map[string]*LSPServerConfig{
					"go": {
						Command: "gopls",
					},
				},
				MCPServer: &MCPServerConfig{
					Name:    "test",
					Version: "1.0.0",
				},
				Logging: &LoggingConfig{
					Level:  "info",
					Format: "text",
				},
				Session: &SessionConfig{
					MaxSessions: -1, // 无效值
				},
			},
			expectError: true,
			errorMsg:    "最大会话数必须大于0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				if err == nil {
					t.Error("期望返回错误，但没有错误")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("期望错误消息包含 '%s'，实际错误: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("不期望返回错误，但得到错误: %v", err)
				}
			}
		})
	}
}

// TestConfigWithInitializationOptions 测试带有初始化选项的配置
func TestConfigWithInitializationOptions(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "init_options_config.yaml")

	configContent := `
lsp_servers:
  go:
    command: "gopls"
    args: ["serve"]
    initialization_options:
      usePlaceholders: true
      completionDocumentation: true
      deepCompletion: true
  typescript:
    command: "typescript-language-server"
    args: ["--stdio"]
    initialization_options:
      preferences:
        disableSuggestions: false
        quotePreference: "double"

mcp_server:
  name: "init-options-test"
  version: "1.0.0"
  description: "Test with initialization options"

logging:
  level: "info"
  format: "text"
  file_output: false
  file_path: ""

session:
  max_sessions: 5
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("创建配置文件失败: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证Go LSP服务器的初始化选项
	goConfig, exists := cfg.GetLSPServerConfig("go")
	if !exists {
		t.Fatal("Go LSP服务器配置不存在")
	}

	if goConfig.InitializationOptions["usePlaceholders"] != true {
		t.Error("期望Go LSP初始化选项包含 usePlaceholders: true")
	}
	if goConfig.InitializationOptions["completionDocumentation"] != true {
		t.Error("期望Go LSP初始化选项包含 completionDocumentation: true")
	}
	if goConfig.InitializationOptions["deepCompletion"] != true {
		t.Error("期望Go LSP初始化选项包含 deepCompletion: true")
	}

	// 验证TypeScript LSP服务器的初始化选项
	tsConfig, exists := cfg.GetLSPServerConfig("typescript")
	if !exists {
		t.Fatal("TypeScript LSP服务器配置不存在")
	}

	preferences, ok := tsConfig.InitializationOptions["preferences"].(map[string]interface{})
	if !ok {
		t.Fatal("期望TypeScript LSP初始化选项包含preferences对象")
	}

	if preferences["disableSuggestions"] != false {
		t.Error("期望TypeScript preferences包含 disableSuggestions: false")
	}
	if preferences["quotePreference"] != "double" {
		t.Error("期望TypeScript preferences包含 quotePreference: 'double'")
	}
}

// contains 检查字符串是否包含子字符串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}()))
}

func TestLoadConfigFileNotFound(t *testing.T) {
	// 测试文件不存在的情况
	_, err := LoadConfig("/nonexistent/config.yaml")
	if err == nil {
		t.Error("期望加载不存在的配置文件时返回错误")
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	// 创建无效的 YAML 文件
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid_config.yaml")

	invalidContent := `
lsp_servers:
  go:
    command: "gopls"
    args: [
    # 无效的 YAML - 未闭合的数组
`

	err := os.WriteFile(configPath, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("创建无效配置文件失败: %v", err)
	}

	// 测试加载无效配置
	_, err = LoadConfig(configPath)
	if err == nil {
		t.Error("期望加载无效 YAML 配置文件时返回错误")
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name: "有效配置",
			config: &Config{
				LSPServers: map[string]*LSPServerConfig{
					"go": {
						Command: "gopls",
						Args:    []string{},
					},
				},
				MCPServer: &MCPServerConfig{
					Name:        "test-server",
					Version:     "1.0.0",
					Description: "Test server",
				},
				Logging: &LoggingConfig{
					Level:      "info",
					Format:     "text",
					FileOutput: false,
				},
				Session: &SessionConfig{
					MaxSessions: 5,
				},
			},
			expectError: false,
		},
		{
			name: "缺少 LSP 服务器",
			config: &Config{
				LSPServers: map[string]*LSPServerConfig{},
				MCPServer: &MCPServerConfig{
					Name:        "test-server",
					Version:     "1.0.0",
					Description: "Test server",
				},
			},
			expectError: true,
		},
		{
			name: "LSP 服务器命令为空",
			config: &Config{
				LSPServers: map[string]*LSPServerConfig{
					"go": {
						Command: "",
						Args:    []string{},
					},
				},
				MCPServer: &MCPServerConfig{
					Name:        "test-server",
					Version:     "1.0.0",
					Description: "Test server",
				},
			},
			expectError: true,
		},
		{
			name: "MCP 服务器名称为空",
			config: &Config{
				LSPServers: map[string]*LSPServerConfig{
					"go": {
						Command: "gopls",
						Args:    []string{},
					},
				},
				MCPServer: &MCPServerConfig{
					Name:        "",
					Version:     "1.0.0",
					Description: "Test server",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError && err == nil {
				t.Error("期望返回错误，但没有错误")
			}
			if !tt.expectError && err != nil {
				t.Errorf("不期望返回错误，但得到错误: %v", err)
			}
		})
	}
}

func TestConfigDefaults(t *testing.T) {
	// 测试完整配置加载
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "complete_config.yaml")

	// 完整配置
	completeContent := `
lsp_servers:
  go:
    command: "gopls"
    args: []
    initialization_options: {}

mcp_server:
  name: "complete-server"
  version: "1.0.0"
  description: "Complete test server"

logging:
  level: "debug"
  format: "json"
  file_output: true
  file_path: "/tmp/test.log"

session:
  max_sessions: 10
`

	err := os.WriteFile(configPath, []byte(completeContent), 0644)
	if err != nil {
		t.Fatalf("创建完整配置文件失败: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("加载完整配置失败: %v", err)
	}

	// 验证配置值
	if cfg.Logging.Level != "debug" {
		t.Errorf("期望日志级别为 'debug'，实际为 '%s'", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "json" {
		t.Errorf("期望日志格式为 'json'，实际为 '%s'", cfg.Logging.Format)
	}
	if cfg.Logging.FileOutput != true {
		t.Errorf("期望文件输出为 true，实际为 %v", cfg.Logging.FileOutput)
	}
	if cfg.Logging.FilePath != "/tmp/test.log" {
		t.Errorf("期望日志文件路径为 '/tmp/test.log'，实际为 '%s'", cfg.Logging.FilePath)
	}
	if cfg.Session.MaxSessions != 10 {
		t.Errorf("期望最大会话数为 10，实际为 %d", cfg.Session.MaxSessions)
	}
}
