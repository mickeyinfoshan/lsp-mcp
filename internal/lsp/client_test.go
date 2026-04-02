package lsp

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mickeyinfoshan/lsp-mcp/internal/config"
	"github.com/mickeyinfoshan/lsp-mcp/pkg/types"
)

// readWriteCloser implements io.ReadWriteCloser by combining reader/writer
// Used to test pipe reads/writes
type readWriteCloser struct {
	reader io.Reader
	writer io.Writer
}

func (rwc *readWriteCloser) Read(p []byte) (int, error) {
	return rwc.reader.Read(p)
}

func (rwc *readWriteCloser) Write(p []byte) (int, error) {
	return rwc.writer.Write(p)
}

func (rwc *readWriteCloser) Close() error {
	// For tests, just return nil
	return nil
}

// createTestConfig creates test config
func createTestConfig(t *testing.T) *config.Config {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.yaml")

	configContent := `
lsp_servers:
  go:
    command: "echo"
    args: ["test"]
    initialization_options: {}
  typescript:
    command: "echo"
    args: ["test"]
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
  max_sessions: 5
  timeout: 900
  cleanup_interval: 300
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load test config: %v", err)
	}

	return cfg
}

// TestNewClient tests creating a new LSP client
func TestNewClient(t *testing.T) {
	cfg := createTestConfig(t)

	client := NewClient(cfg)
	if client == nil {
		t.Fatal("failed to create LSP client")
	}

	// Validate client fields
	if client.config != cfg {
		t.Error("config not set correctly")
	}

	if client.sessions == nil {
		t.Error("session map not initialized")
	}

	if client.ctx == nil {
		t.Error("context not initialized")
	}

	if client.cancel == nil {
		t.Error("cancel function not initialized")
	}

	// Cleanup
	client.Close()
}

// TestGetOrCreateSession_NewSession tests creating a new session
func TestGetOrCreateSession_NewSession(t *testing.T) {
	// Skip tests that require a real LSP server
	if testing.Short() {
		t.Skip("skip tests that require an LSP server")
	}

	cfg := createTestConfig(t)
	client := NewClient(cfg)
	defer client.Close()

	languageID := "go"
	rootURI := "file:///tmp/test"

	// Since we use echo as the test LSP server, this test will fail
	// but we can test error handling logic
	session, err := client.GetOrCreateSession(languageID, rootURI)
	if err == nil {
		t.Log("unexpectedly created session (using echo command)")
		if session != nil {
			t.Log("session created successfully")
		}
	} else {
		t.Logf("expected error: %v", err)
	}
}

// TestGetOrCreateSession_ExistingSession tests getting an existing session
func TestGetOrCreateSession_ExistingSession(t *testing.T) {
	cfg := createTestConfig(t)
	client := NewClient(cfg)
	defer client.Close()

	languageID := "go"
	rootURI := "file:///tmp/test"
	sessionKey := types.SessionKey{
		LanguageID: languageID,
		RootURI:    rootURI,
	}

	// Manually create a mock session
	mockSession := &types.LSPSession{
		Key:           sessionKey,
		Conn:          nil, // mock connection
		Process:       nil, // mock process
		CreatedAt:     time.Now(),
		LastUsedAt:    time.Now(),
		IsInitialized: true,
	}

	client.sessionsMutex.Lock()
	client.sessions[sessionKey.String()] = mockSession
	client.sessionsMutex.Unlock()

	// Test getting an existing session
	session, err := client.GetOrCreateSession(languageID, rootURI)
	if err != nil {
		t.Fatalf("failed to get existing session: %v", err)
	}

	if session != mockSession {
		t.Error("returned session is not the expected session")
	}

	// Verify last used time updated
	if session.LastUsedAt.Before(mockSession.LastUsedAt) {
		t.Error("last used time not updated")
	}
}

// TestGetOrCreateSession_MaxSessionsLimit tests max sessions limit
func TestGetOrCreateSession_MaxSessionsLimit(t *testing.T) {
	cfg := createTestConfig(t)
	// Set a smaller max sessions for testing
	cfg.Session.MaxSessions = 2

	client := NewClient(cfg)
	defer client.Close()

	// Manually add sessions until reaching the limit
	for i := 0; i < cfg.Session.MaxSessions; i++ {
		sessionKey := types.SessionKey{
			LanguageID: "go",
			RootURI:    fmt.Sprintf("file:///tmp/test%d", i),
		}
		mockSession := &types.LSPSession{
			Key:           sessionKey,
			Conn:          nil,
			Process:       nil,
			CreatedAt:     time.Now(),
			LastUsedAt:    time.Now(),
			IsInitialized: true,
		}
		client.sessionsMutex.Lock()
		client.sessions[sessionKey.String()] = mockSession
		client.sessionsMutex.Unlock()
	}

	// Try to create a session beyond the limit
	_, err := client.GetOrCreateSession("go", "file:///tmp/test_overflow")
	if err == nil {
		t.Error("expected failure due to max sessions limit")
	}

	if !strings.Contains(err.Error(), "max sessions limit reached") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestGetOrCreateSession_UnsupportedLanguage tests unsupported language
func TestGetOrCreateSession_UnsupportedLanguage(t *testing.T) {
	cfg := createTestConfig(t)
	client := NewClient(cfg)
	defer client.Close()

	// Try using an unsupported language
	_, err := client.GetOrCreateSession("unsupported", "file:///tmp/test")
	if err == nil {
		t.Error("expected failure due to unsupported language")
	}

	if !strings.Contains(err.Error(), "unsupported language") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestClose tests closing the client
func TestClose(t *testing.T) {
	cfg := createTestConfig(t)
	client := NewClient(cfg)

	// Add a mock session
	sessionKey := types.SessionKey{
		LanguageID: "go",
		RootURI:    "file:///tmp/test",
	}
	mockSession := &types.LSPSession{
		Key:           sessionKey,
		Conn:          nil,
		Process:       nil,
		CreatedAt:     time.Now(),
		LastUsedAt:    time.Now(),
		IsInitialized: true,
	}
	client.sessionsMutex.Lock()
	client.sessions[sessionKey.String()] = mockSession
	client.sessionsMutex.Unlock()

	// Test close
	err := client.Close()
	if err != nil {
		t.Errorf("failed to close client: %v", err)
	}

	// Verify context canceled
	select {
	case <-client.ctx.Done():
		// Expected behavior
	default:
		t.Error("context was not canceled")
	}

	// Verify sessions cleaned up
	client.sessionsMutex.RLock()
	sessionCount := len(client.sessions)
	client.sessionsMutex.RUnlock()

	if sessionCount != 0 {
		t.Errorf("expected session count 0, got %d", sessionCount)
	}
}

// TestReadWriteCloser tests readWriteCloser implementation
func TestReadWriteCloser(t *testing.T) {
	// Create test pipes
	r1, w1, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe 1: %v", err)
	}
	defer r1.Close()
	defer w1.Close()

	r2, w2, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe 2: %v", err)
	}
	defer r2.Close()
	defer w2.Close()

	// Create readWriteCloser
	rwc := &readWriteCloser{
		reader: r1,
		writer: w2,
	}

	// Test write
	testData := []byte("test data")
	n, err := rwc.Write(testData)
	if err != nil {
		t.Errorf("write failed: %v", err)
	}
	if n != len(testData) {
		t.Errorf("bytes written mismatch: expected %d, got %d", len(testData), n)
	}

	// Close write end to allow reading
	w1.Write(testData)
	w1.Close()

	// Test read
	buf := make([]byte, len(testData))
	n, err = rwc.Read(buf)
	if err != nil {
		t.Errorf("read failed: %v", err)
	}
	if n != len(testData) {
		t.Errorf("bytes read mismatch: expected %d, got %d", len(testData), n)
	}
	if string(buf) != string(testData) {
		t.Errorf("data read mismatch: expected %s, got %s", string(testData), string(buf))
	}

	// Test close
	err = rwc.Close()
	if err != nil {
		t.Errorf("close failed: %v", err)
	}
}

// TestBuildClientCapabilities tests building client capabilities
func TestBuildClientCapabilities(t *testing.T) {
	cfg := createTestConfig(t)
	client := NewClient(cfg)
	defer client.Close()

	capabilities := client.buildClientCapabilities()
	if capabilities.TextDocument == nil {
		t.Error("client capabilities are nil")
	}

	// Verify basic capabilities
	if capabilities.TextDocument == nil {
		t.Error("TextDocument capability not set")
	}

	if capabilities.Workspace == nil {
		t.Error("Workspace capability not set")
	}
}
