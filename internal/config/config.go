// config.go
// 配置文件加载与校验，LSP/MCP/日志/会话等配置结构体
// Loads and validates configuration files for LSP, MCP, logging, and session management.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LSPServerConfig LSP 服务器配置
// LSPServerConfig represents the configuration for an LSP server
type LSPServerConfig struct {
	// Command LSP 服务器可执行文件路径或命令名
	// Path or command name of the LSP server executable
	Command string `yaml:"command"`
	// Args 启动参数
	// Startup arguments
	Args []string `yaml:"args"`
	// InitializationOptions 初始化选项
	// Initialization options
	InitializationOptions map[string]interface{} `yaml:"initialization_options"`
	// Env 环境变量
	// Environment variables
	Env map[string]string `yaml:"env"`
}

// MCPServerConfig MCP服务器配置
type MCPServerConfig struct {
	// Name 服务器名称
	Name string `yaml:"name"`
	// Version 服务器版本
	Version string `yaml:"version"`
	// Description 服务器描述
	Description string `yaml:"description"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	// Level 日志级别
	Level string `yaml:"level"`
	// Format 日志格式
	Format string `yaml:"format"`
	// FileOutput 是否输出到文件
	FileOutput bool `yaml:"file_output"`
	// FilePath 日志文件路径
	FilePath string `yaml:"file_path"`
}

// SessionConfig 会话管理配置
// SessionConfig represents session management configuration
type SessionConfig struct {
	// MaxSessions 最大并发会话数
	MaxSessions int `yaml:"max_sessions"`
}

// Config 应用程序配置
type Config struct {
	// LSPServers LSP服务器配置映射
	LSPServers map[string]*LSPServerConfig `yaml:"lsp_servers"`
	// MCPServer MCP服务器配置
	MCPServer *MCPServerConfig `yaml:"mcp_server"`
	// Logging 日志配置
	Logging *LoggingConfig `yaml:"logging"`
	// Session 会话管理配置
	Session *SessionConfig `yaml:"session"`
}

// LoadConfig 从指定路径加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("配置文件不存在: %s", configPath)
	}

	// 读取配置文件内容
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析YAML配置
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return &config, nil
}

// LoadConfigFromDefault 从默认路径加载配置文件
func LoadConfigFromDefault() (*Config, error) {
	// 获取当前可执行文件所在目录
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("获取可执行文件路径失败: %w", err)
	}
	dir := filepath.Dir(exePath)

	// 构建同目录下的 config.yaml 路径
	configPath := filepath.Join(dir, "config.yaml")
	return LoadConfig(configPath)
}

// Validate 验证配置的有效性
func (c *Config) Validate() error {
	// 验证LSP服务器配置
	if len(c.LSPServers) == 0 {
		return fmt.Errorf("至少需要配置一个LSP服务器")
	}

	for languageID, serverConfig := range c.LSPServers {
		if languageID == "" {
			return fmt.Errorf("语言ID不能为空")
		}
		if serverConfig.Command == "" {
			return fmt.Errorf("LSP服务器命令不能为空: %s", languageID)
		}
	}

	// 验证MCP服务器配置
	if c.MCPServer == nil {
		return fmt.Errorf("MCP服务器配置不能为空")
	}
	if c.MCPServer.Name == "" {
		return fmt.Errorf("MCP服务器名称不能为空")
	}
	if c.MCPServer.Version == "" {
		return fmt.Errorf("MCP服务器版本不能为空")
	}

	// 验证日志配置
	if c.Logging == nil {
		return fmt.Errorf("日志配置不能为空")
	}
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("无效的日志级别: %s", c.Logging.Level)
	}
	validFormats := map[string]bool{"json": true, "text": true}
	if !validFormats[c.Logging.Format] {
		return fmt.Errorf("无效的日志格式: %s", c.Logging.Format)
	}

	// 验证会话配置
	if c.Session == nil {
		return fmt.Errorf("会话配置不能为空")
	}

	if c.Session.MaxSessions <= 0 {
		return fmt.Errorf("最大会话数必须大于0")
	}

	return nil
}

// GetLSPServerConfig 根据语言ID获取LSP服务器配置
func (c *Config) GetLSPServerConfig(languageID string) (*LSPServerConfig, bool) {
	config, exists := c.LSPServers[languageID]
	return config, exists
}
