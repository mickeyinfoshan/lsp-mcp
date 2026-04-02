package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
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
		t.Fatalf("failed to create test config file: %v", err)
	}

	// Load config
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Validate config contents
	if cfg.MCPServer.Name != "test-lsp-bridge" {
		t.Errorf("expected MCP server name to be 'test-lsp-bridge', got '%s'", cfg.MCPServer.Name)
	}

	if cfg.MCPServer.Version != "1.0.0" {
		t.Errorf("expected MCP server version to be '1.0.0', got '%s'", cfg.MCPServer.Version)
	}

	if len(cfg.LSPServers) != 2 {
		t.Errorf("expected 2 LSP servers, got %d", len(cfg.LSPServers))
	}

	// Validate Go LSP server config
	goConfig, exists := cfg.LSPServers["go"]
	if !exists {
		t.Error("Go LSP server config missing")
	}
	if goConfig.Command != "gopls" {
		t.Errorf("expected Go LSP command to be 'gopls', got '%s'", goConfig.Command)
	}

	// Validate TypeScript LSP server config
	tsConfig, exists := cfg.LSPServers["typescript"]
	if !exists {
		t.Error("TypeScript LSP server config missing")
	}
	if tsConfig.Command != "typescript-language-server" {
		t.Errorf("expected TypeScript LSP command to be 'typescript-language-server', got '%s'", tsConfig.Command)
	}
	if len(tsConfig.Args) != 1 || tsConfig.Args[0] != "--stdio" {
		t.Errorf("expected TypeScript LSP args to be ['--stdio'], got %v", tsConfig.Args)
	}

	// Validate logging config
	if cfg.Logging.Level != "info" {
		t.Errorf("expected log level 'info', got '%s'", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "text" {
		t.Errorf("expected log format 'text', got '%s'", cfg.Logging.Format)
	}
	if cfg.Logging.FileOutput != false {
		t.Errorf("expected file output false, got %v", cfg.Logging.FileOutput)
	}

	// Validate session config
	if cfg.Session.MaxSessions != 3 {
		t.Errorf("expected max sessions 3, got %d", cfg.Session.MaxSessions)
	}
}

// TestGetLSPServerConfig tests fetching LSP server config
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

	// Test an existing language ID
	goConfig, exists := config.GetLSPServerConfig("go")
	if !exists {
		t.Error("expected to find Go LSP server config")
	}
	if goConfig.Command != "gopls" {
		t.Errorf("expected Go LSP command to be 'gopls', got '%s'", goConfig.Command)
	}
	if len(goConfig.Args) != 1 || goConfig.Args[0] != "serve" {
		t.Errorf("expected Go LSP args to be ['serve'], got %v", goConfig.Args)
	}
	if goConfig.InitializationOptions["usePlaceholders"] != true {
		t.Error("expected Go LSP initialization options to include usePlaceholders: true")
	}

	// Test a nonexistent language ID
	_, exists = config.GetLSPServerConfig("nonexistent")
	if exists {
		t.Error("expected nonexistent language ID to return false")
	}
}

// TestLoadConfigFromDefault tests loading config from the default path
func TestLoadConfigFromDefault(t *testing.T) {
	// LoadConfigFromDefault depends on current working directory and filesystem,
	// so we mainly ensure the function call does not panic.
	_, err := LoadConfigFromDefault()
	// Expect an error because the default config may not exist
	if err == nil {
		t.Log("warning: default config file exists; this may not be the expected test environment")
	}
}

// TestValidateConfigEdgeCases tests edge cases for config validation
func TestValidateConfigEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "empty language ID",
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
			errorMsg:    "language ID cannot be empty",
		},
		{
			name: "invalid log level",
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
			errorMsg:    "invalid log level",
		},
		{
			name: "invalid log format",
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
			errorMsg:    "invalid log format",
		},
		{
			name: "invalid max sessions",
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
					MaxSessions: -1, // Invalid value
				},
			},
			expectError: true,
			errorMsg:    "max sessions must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain '%s', got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("did not expect error, got: %v", err)
				}
			}
		})
	}
}

// TestConfigWithInitializationOptions tests config with initialization options
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
		t.Fatalf("failed to create config file: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Validate Go LSP initialization options
	goConfig, exists := cfg.GetLSPServerConfig("go")
	if !exists {
		t.Fatal("Go LSP server config missing")
	}

	if goConfig.InitializationOptions["usePlaceholders"] != true {
		t.Error("expected Go LSP initialization options to include usePlaceholders: true")
	}
	if goConfig.InitializationOptions["completionDocumentation"] != true {
		t.Error("expected Go LSP initialization options to include completionDocumentation: true")
	}
	if goConfig.InitializationOptions["deepCompletion"] != true {
		t.Error("expected Go LSP initialization options to include deepCompletion: true")
	}

	// Validate TypeScript LSP initialization options
	tsConfig, exists := cfg.GetLSPServerConfig("typescript")
	if !exists {
		t.Fatal("TypeScript LSP server config missing")
	}

	preferences, ok := tsConfig.InitializationOptions["preferences"].(map[string]interface{})
	if !ok {
		t.Fatal("expected TypeScript initialization options to include preferences object")
	}

	if preferences["disableSuggestions"] != false {
		t.Error("expected TypeScript preferences to include disableSuggestions: false")
	}
	if preferences["quotePreference"] != "double" {
		t.Error("expected TypeScript preferences to include quotePreference: 'double'")
	}
}

// contains checks whether a string contains a substring
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
	// Test file not found
	_, err := LoadConfig("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected an error when loading a nonexistent config file")
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	// Create an invalid YAML file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid_config.yaml")

	invalidContent := `
lsp_servers:
  go:
    command: "gopls"
    args: [
    # Invalid YAML - unclosed array
`

	err := os.WriteFile(configPath, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("failed to create invalid config file: %v", err)
	}

	// Test loading invalid config
	_, err = LoadConfig(configPath)
	if err == nil {
		t.Error("expected an error when loading invalid YAML config")
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name: "valid config",
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
			name: "missing LSP servers",
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
			name: "LSP server command is empty",
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
			name: "MCP server name is empty",
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
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("did not expect error, got: %v", err)
			}
		})
	}
}

func TestConfigDefaults(t *testing.T) {
	// Test loading a complete config
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "complete_config.yaml")

	// Complete config
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
		t.Fatalf("failed to create complete config file: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load complete config: %v", err)
	}

	// Validate config values
	if cfg.Logging.Level != "debug" {
		t.Errorf("expected log level 'debug', got '%s'", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "json" {
		t.Errorf("expected log format 'json', got '%s'", cfg.Logging.Format)
	}
	if cfg.Logging.FileOutput != true {
		t.Errorf("expected file output true, got %v", cfg.Logging.FileOutput)
	}
	if cfg.Logging.FilePath != "/tmp/test.log" {
		t.Errorf("expected log file path '/tmp/test.log', got '%s'", cfg.Logging.FilePath)
	}
	if cfg.Session.MaxSessions != 10 {
		t.Errorf("expected max sessions 10, got %d", cfg.Session.MaxSessions)
	}
}
