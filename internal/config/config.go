// config.go
// Loads and validates configuration files for LSP, MCP, logging, and session management.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LSPServerConfig represents the configuration for an LSP server
type LSPServerConfig struct {
	// Command path or name of the LSP server executable
	Command string `yaml:"command"`
	// Args startup arguments
	Args []string `yaml:"args"`
	// InitializationOptions initialization options
	InitializationOptions map[string]interface{} `yaml:"initialization_options"`
	// Env environment variables
	Env map[string]string `yaml:"env"`
}

// MCPServerConfig MCP server configuration
type MCPServerConfig struct {
	// Name server name
	Name string `yaml:"name"`
	// Version server version
	Version string `yaml:"version"`
	// Description server description
	Description string `yaml:"description"`
}

// LoggingConfig logging configuration
type LoggingConfig struct {
	// Level log level
	Level string `yaml:"level"`
	// Format log format
	Format string `yaml:"format"`
	// FileOutput whether to output to a file
	FileOutput bool `yaml:"file_output"`
	// FilePath log file path
	FilePath string `yaml:"file_path"`
}

// SessionConfig represents session management configuration
type SessionConfig struct {
	// MaxSessions maximum concurrent sessions
	MaxSessions int `yaml:"max_sessions"`
}

// Config application configuration
type Config struct {
	// LSPServers LSP server configuration map
	LSPServers map[string]*LSPServerConfig `yaml:"lsp_servers"`
	// MCPServer MCP server configuration
	MCPServer *MCPServerConfig `yaml:"mcp_server"`
	// Logging logging configuration
	Logging *LoggingConfig `yaml:"logging"`
	// Session session management configuration
	Session *SessionConfig `yaml:"session"`
}

// LoadConfig loads a configuration file from the given path
func LoadConfig(configPath string) (*Config, error) {
	// Check if the config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file does not exist: %s", configPath)
	}

	// Read config file contents
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML config
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// LoadConfigFromDefault loads a config file from the default path
func LoadConfigFromDefault() (*Config, error) {
	// Get the directory of the current executable
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}
	dir := filepath.Dir(exePath)

	// Build config.yaml path in the same directory
	configPath := filepath.Join(dir, "config.yaml")
	return LoadConfig(configPath)
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate LSP server configuration
	if len(c.LSPServers) == 0 {
		return fmt.Errorf("at least one LSP server must be configured")
	}

	for languageID, serverConfig := range c.LSPServers {
		if languageID == "" {
			return fmt.Errorf("language ID cannot be empty")
		}
		if serverConfig.Command == "" {
			return fmt.Errorf("LSP server command cannot be empty: %s", languageID)
		}
	}

	// Validate MCP server configuration
	if c.MCPServer == nil {
		return fmt.Errorf("MCP server configuration cannot be nil")
	}
	if c.MCPServer.Name == "" {
		return fmt.Errorf("MCP server name cannot be empty")
	}
	if c.MCPServer.Version == "" {
		return fmt.Errorf("MCP server version cannot be empty")
	}

	// Validate logging configuration
	if c.Logging == nil {
		return fmt.Errorf("logging configuration cannot be nil")
	}
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level: %s", c.Logging.Level)
	}
	validFormats := map[string]bool{"json": true, "text": true}
	if !validFormats[c.Logging.Format] {
		return fmt.Errorf("invalid log format: %s", c.Logging.Format)
	}

	// Validate session configuration
	if c.Session == nil {
		return fmt.Errorf("session configuration cannot be nil")
	}

	if c.Session.MaxSessions <= 0 {
		return fmt.Errorf("max sessions must be greater than 0")
	}

	return nil
}

// GetLSPServerConfig returns the LSP server config by language ID
func (c *Config) GetLSPServerConfig(languageID string) (*LSPServerConfig, bool) {
	config, exists := c.LSPServers[languageID]
	return config, exists
}
