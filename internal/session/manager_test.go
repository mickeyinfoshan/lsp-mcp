package session

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/mickeyinfoshan/lsp-mcp/internal/config"
	"github.com/mickeyinfoshan/lsp-mcp/pkg/types"
	protocol "go.lsp.dev/protocol"
)

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
  python:
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

// TestNewManager tests creating a new session manager
func TestNewManager(t *testing.T) {
	cfg := createTestConfig(t)

	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("failed to create session manager: %v", err)
	}

	if manager == nil {
		t.Fatal("session manager is nil")
	}

	// Validate manager fields
	if manager.config != cfg {
		t.Error("config not set correctly")
	}

	if manager.lspClient == nil {
		t.Error("LSP client not initialized")
	}

	if manager.ctx == nil {
		t.Error("context not initialized")
	}

	if manager.cancel == nil {
		t.Error("cancel function not initialized")
	}

	if manager.metrics == nil {
		t.Error("metrics not initialized")
	}

	// Cleanup
	manager.Close()
}

// TestGetSupportedLanguages tests getting supported languages
func TestGetSupportedLanguages(t *testing.T) {
	cfg := createTestConfig(t)
	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("failed to create session manager: %v", err)
	}
	defer manager.Close()

	languages := manager.GetSupportedLanguages()
	if len(languages) == 0 {
		t.Error("expected at least one supported language")
	}

	// Verify configured languages are in the supported list
	expectedLanguages := []string{"go", "typescript", "python"}
	for _, expected := range expectedLanguages {
		found := false
		for _, lang := range languages {
			if lang == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected language %s not found in supported list", expected)
		}
	}
}

// TestFindDefinition_InvalidRequest tests invalid find definition request
func TestFindDefinition_InvalidRequest(t *testing.T) {
	cfg := createTestConfig(t)
	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("failed to create session manager: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Test nil request
	resp, err := manager.FindDefinition(ctx, nil)
	if err != nil {
		t.Errorf("expected error response, not error: %v", err)
	}
	if resp == nil {
		t.Fatal("response is nil")
	}
	if resp.Error == "" {
		t.Error("expected response to contain error")
	}

	// Test request missing required fields
	invalidReq := &types.FindDefinitionRequest{
		LanguageID: "", // empty language ID
		RootURI:    "file:///tmp/test",
		FileURI:    "file:///tmp/test/main.go",
		Position: protocol.Position{
			Line:      0, // was -1; uint32 does not support it, using 0 to simulate invalid input
			Character: 5,
		},
	}

	resp, err = manager.FindDefinition(ctx, invalidReq)
	if err != nil {
		t.Errorf("expected error response, not error: %v", err)
	}
	if resp == nil {
		t.Fatal("response is nil")
	}
	if resp.Error == "" {
		t.Error("expected response to contain error")
	}
}

// TestFindDefinition_ValidRequest tests valid find definition request
func TestFindDefinition_ValidRequest(t *testing.T) {
	// skip tests that require a real LSP server
	if testing.Short() {
		t.Skip("skip tests that require a real LSP server")
	}

	cfg := createTestConfig(t)
	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("failed to create session manager: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Create a valid request
	req := &types.FindDefinitionRequest{
		LanguageID: "go",
		RootURI:    "file:///tmp/test",
		FileURI:    "file:///tmp/test/main.go",
		Position: protocol.Position{
			Line:      0, // was -1; uint32 does not support it, using 0 to simulate invalid input
			Character: 5,
		},
	}

	// Since we use echo as the test LSP server, this test may fail
	// but we can validate request handling logic
	resp, err := manager.FindDefinition(ctx, req)
	if err != nil {
		t.Errorf("find definition failed: %v", err)
	}
	if resp == nil {
		t.Fatal("response is nil")
	}

	// Verify metrics updated
	metrics := manager.GetMetrics()
	if metrics.TotalRequests == 0 {
		t.Error("total requests not updated")
	}
}

// TestGetMetrics tests getting metrics
func TestGetMetrics(t *testing.T) {
	cfg := createTestConfig(t)
	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("failed to create session manager: %v", err)
	}
	defer manager.Close()

	metrics := manager.GetMetrics()
	if metrics == nil {
		t.Fatal("metrics is nil")
	}

	// Verify initial metrics
	if metrics.TotalRequests != 0 {
		t.Error("initial total requests should be 0")
	}
	if metrics.SuccessfulRequests != 0 {
		t.Error("initial successful requests should be 0")
	}
	if metrics.FailedRequests != 0 {
		t.Error("initial failed requests should be 0")
	}
	if metrics.SessionsCreated != 0 {
		t.Error("initial sessions created should be 0")
	}
	if metrics.SessionsClosed != 0 {
		t.Error("initial sessions closed should be 0")
	}
}

// TestShutdown tests closing the manager
func TestShutdown(t *testing.T) {
	cfg := createTestConfig(t)
	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("failed to create session manager: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test shutdown
	err = manager.Shutdown(ctx)
	if err != nil {
		t.Errorf("failed to close manager: %v", err)
	}

	// Verify context canceled
	select {
	case <-manager.ctx.Done():
		// Expected behavior
	default:
		t.Error("context was not canceled")
	}
}

// TestClose tests closing the manager
func TestClose(t *testing.T) {
	cfg := createTestConfig(t)
	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("failed to create session manager: %v", err)
	}

	// Test close
	err = manager.Close()
	if err != nil {
		t.Errorf("failed to close manager: %v", err)
	}

	// Verify context canceled
	select {
	case <-manager.ctx.Done():
		// Expected behavior
	default:
		t.Error("context was not canceled")
	}
}

// TestSessionMetrics_IncrementMethods tests metric increment methods
func TestSessionMetrics_IncrementMethods(t *testing.T) {
	metrics := &SessionMetrics{}

	// Test increment total requests
	metrics.incrementTotalRequests()
	if metrics.TotalRequests != 1 {
		t.Errorf("expected total requests to be 1, got %d", metrics.TotalRequests)
	}

	// Test increment successful requests
	metrics.incrementSuccessfulRequests()
	if metrics.SuccessfulRequests != 1 {
		t.Errorf("expected successful requests to be 1, got %d", metrics.SuccessfulRequests)
	}

	// Test increment failed requests
	metrics.incrementFailedRequests()
	if metrics.FailedRequests != 1 {
		t.Errorf("expected failed requests to be 1, got %d", metrics.FailedRequests)
	}

	// Test increment sessions created
	metrics.incrementSessionsCreated()
	if metrics.SessionsCreated != 1 {
		t.Errorf("expected sessions created to be 1, got %d", metrics.SessionsCreated)
	}

	// Test increment sessions closed
	metrics.incrementSessionsClosed()
	if metrics.SessionsClosed != 1 {
		t.Errorf("expected sessions closed to be 1, got %d", metrics.SessionsClosed)
	}
}

// TestSessionMetrics_UpdateMethods tests metric update methods
func TestSessionMetrics_UpdateMethods(t *testing.T) {
	metrics := &SessionMetrics{}

	// Test update last request time
	testTime := time.Now()
	metrics.updateLastRequestTime(testTime)
	if !metrics.LastRequestTime.Equal(testTime) {
		t.Error("last request time not updated correctly")
	}

	// Test update average response time
	responseTime := 100 * time.Millisecond
	metrics.updateAverageResponseTime(responseTime)
	expectedMs := float64(responseTime.Nanoseconds()) / 1e6
	if metrics.AverageResponseTime != expectedMs {
		t.Errorf("expected average response time %.2f, got %.2f", expectedMs, metrics.AverageResponseTime)
	}
}

// TestValidateFindDefinitionRequest tests validating find definition request
func TestValidateFindDefinitionRequest(t *testing.T) {
	cfg := createTestConfig(t)
	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("failed to create session manager: %v", err)
	}
	defer manager.Close()

	// Test valid request
	validReq := &types.FindDefinitionRequest{
		LanguageID: "go",
		RootURI:    "file:///tmp/test",
		FileURI:    "file:///tmp/test/main.go",
		Position: protocol.Position{
			Line:      0, // was -1; uint32 does not support it, using 0 to simulate invalid input
			Character: 5,
		},
	}

	err = manager.validateFindDefinitionRequest(validReq)
	if err != nil {
		t.Errorf("valid request validation failed: %v", err)
	}

	// Test empty language ID
	invalidReq1 := &types.FindDefinitionRequest{
		LanguageID: "",
		RootURI:    "file:///tmp/test",
		FileURI:    "file:///tmp/test/main.go",
		Position: protocol.Position{
			Line:      0, // was -1; uint32 does not support it, using 0 to simulate invalid input
			Character: 5,
		},
	}

	err = manager.validateFindDefinitionRequest(invalidReq1)
	if err == nil {
		t.Error("expected empty language ID validation to fail")
	}

	// Test empty root URI
	invalidReq2 := &types.FindDefinitionRequest{
		LanguageID: "go",
		RootURI:    "",
		FileURI:    "file:///tmp/test/main.go",
		Position: protocol.Position{
			Line:      0, // was -1; uint32 does not support it, using 0 to simulate invalid input
			Character: 5,
		},
	}

	err = manager.validateFindDefinitionRequest(invalidReq2)
	if err == nil {
		t.Error("expected empty root URI validation to fail")
	}

	// Test empty file URI
	invalidReq3 := &types.FindDefinitionRequest{
		LanguageID: "go",
		RootURI:    "file:///tmp/test",
		FileURI:    "",
		Position: protocol.Position{
			Line:      0, // was -1; uint32 does not support it, using 0 to simulate invalid input
			Character: 5,
		},
	}

	err = manager.validateFindDefinitionRequest(invalidReq3)
	if err == nil {
		t.Error("expected empty file URI validation to fail")
	}

	// Test unsupported language
	invalidReq6 := &types.FindDefinitionRequest{
		LanguageID: "unsupported",
		RootURI:    "file:///tmp/test",
		FileURI:    "file:///tmp/test/main.go",
		Position: protocol.Position{
			Line:      0, // was -1; uint32 does not support it, using 0 to simulate invalid input
			Character: 5,
		},
	}

	err = manager.validateFindDefinitionRequest(invalidReq6)
	if err == nil {
		t.Error("expected unsupported language validation to fail")
	}
}

// TestFindReferences_InvalidRequest tests invalid find references request
func TestFindReferences_InvalidRequest(t *testing.T) {
	cfg := createTestConfig(t)
	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("failed to create session manager: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Test nil request
	resp, err := manager.FindReferences(ctx, nil)
	if err != nil {
		t.Errorf("expected error response, not error: %v", err)
	}
	if resp == nil {
		t.Fatal("response is nil")
	}
	if resp.Error == "" {
		t.Error("expected response to contain error")
	}

	// Test request missing required fields
	invalidReq := &types.FindReferencesRequest{
		LanguageID: "", // empty language ID
		FileURI:    "file:///tmp/test/main.go",
		Position: protocol.Position{
			Line:      0, // was -1; uint32 does not support it, using 0 to simulate invalid input
			Character: 5,
		},
	}

	resp, err = manager.FindReferences(ctx, invalidReq)
	if err != nil {
		t.Errorf("expected error response, not error: %v", err)
	}
	if resp == nil {
		t.Fatal("response is nil")
	}
	if resp.Error == "" {
		t.Error("expected response to contain error")
	}
}

// TestGetHover_InvalidRequest tests invalid hover request
func TestGetHover_InvalidRequest(t *testing.T) {
	cfg := createTestConfig(t)
	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("failed to create session manager: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Test nil request
	resp, err := manager.GetHover(ctx, nil)
	if err != nil {
		t.Errorf("expected error response, not error: %v", err)
	}
	if resp == nil {
		t.Fatal("response is nil")
	}
	if resp.Error == "" {
		t.Error("expected response to contain error")
	}

	// Test request missing required fields
	invalidReq := &types.HoverRequest{
		LanguageID: "", // empty language ID
		FileURI:    "file:///tmp/test/main.go",
		Position: protocol.Position{
			Line:      0, // was -1; uint32 does not support it, using 0 to simulate invalid input
			Character: 5,
		},
	}

	resp, err = manager.GetHover(ctx, invalidReq)
	if err != nil {
		t.Errorf("expected error response, not error: %v", err)
	}
	if resp == nil {
		t.Fatal("response is nil")
	}
	if resp.Error == "" {
		t.Error("expected response to contain error")
	}
}

// TestGetCompletion_InvalidRequest tests invalid completion request
func TestGetCompletion_InvalidRequest(t *testing.T) {
	cfg := createTestConfig(t)
	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("failed to create session manager: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Test nil request
	resp, err := manager.GetCompletion(ctx, nil)
	if err != nil {
		t.Errorf("expected error response, not error: %v", err)
	}
	if resp == nil {
		t.Fatal("response is nil")
	}
	if resp.Error == "" {
		t.Error("expected response to contain error")
	}

	// Test request missing required fields
	invalidReq := &types.CompletionRequest{
		LanguageID: "", // empty language ID
		FileURI:    "file:///tmp/test/main.go",
		Position: protocol.Position{
			Line:      0, // was -1; uint32 does not support it, using 0 to simulate invalid input
			Character: 5,
		},
	}

	resp, err = manager.GetCompletion(ctx, invalidReq)
	if err != nil {
		t.Errorf("expected error response, not error: %v", err)
	}
	if resp == nil {
		t.Fatal("response is nil")
	}
	if resp.Error == "" {
		t.Error("expected response to contain error")
	}
}

// TestGetSessionInfo tests getting session info
func TestGetSessionInfo(t *testing.T) {
	cfg := createTestConfig(t)
	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("failed to create session manager: %v", err)
	}
	defer manager.Close()

	info := manager.GetSessionInfo()
	if info == nil {
		t.Fatal("session info is nil")
	}

	// Verify metrics included
	if _, exists := info["metrics"]; !exists {
		t.Error("session info missing metrics")
	}
}

// TestGetLSPServerConfig tests getting LSP server config
func TestGetLSPServerConfig(t *testing.T) {
	cfg := createTestConfig(t)
	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("failed to create session manager: %v", err)
	}
	defer manager.Close()

	// Test getting existing language config
	config, exists := manager.GetLSPServerConfig("go")
	if !exists {
		t.Error("expected to find Go language config")
	}
	if config == nil {
		t.Error("Go language config is nil")
	}

	// Test getting nonexistent language config
	_, exists = manager.GetLSPServerConfig("nonexistent")
	if exists {
		t.Error("expected nonexistent language config to return false")
	}
}

// TestSessionMetrics_Reset tests resetting metrics
func TestSessionMetrics_Reset(t *testing.T) {
	metrics := &SessionMetrics{
		TotalRequests:       10,
		SuccessfulRequests:  8,
		FailedRequests:      2,
		AverageResponseTime: 100.5,
		SessionsCreated:     5,
		SessionsClosed:      3,
		LastRequestTime:     time.Now(),
	}

	// Reset metrics
	metrics.Reset()

	// Verify all metrics reset to zero values
	if metrics.TotalRequests != 0 {
		t.Errorf("expected total requests 0, got %d", metrics.TotalRequests)
	}
	if metrics.SuccessfulRequests != 0 {
		t.Errorf("expected successful requests 0, got %d", metrics.SuccessfulRequests)
	}
	if metrics.FailedRequests != 0 {
		t.Errorf("expected failed requests 0, got %d", metrics.FailedRequests)
	}
	if metrics.AverageResponseTime != 0 {
		t.Errorf("expected average response time 0, got %.2f", metrics.AverageResponseTime)
	}
	if metrics.SessionsCreated != 0 {
		t.Errorf("expected sessions created 0, got %d", metrics.SessionsCreated)
	}
	if metrics.SessionsClosed != 0 {
		t.Errorf("expected sessions closed 0, got %d", metrics.SessionsClosed)
	}
	if !metrics.LastRequestTime.IsZero() {
		t.Error("expected last request time to be zero")
	}
}

// TestSessionMetrics_GetMetrics tests getting metrics info
func TestSessionMetrics_GetMetrics(t *testing.T) {
	metrics := &SessionMetrics{
		TotalRequests:       10,
		SuccessfulRequests:  8,
		FailedRequests:      2,
		AverageResponseTime: 100.5,
		SessionsCreated:     5,
		SessionsClosed:      3,
		LastRequestTime:     time.Now(),
	}

	metricsMap := metrics.getMetrics()
	if metricsMap == nil {
		t.Fatal("metrics map is nil")
	}

	// Verify all fields exist
	expectedFields := []string{
		"total_requests",
		"successful_requests",
		"failed_requests",
		"average_response_time_ms",
		"sessions_created",
		"sessions_closed",
		"last_request_time",
		"success_rate",
	}

	for _, field := range expectedFields {
		if _, exists := metricsMap[field]; !exists {
			t.Errorf("metrics map missing field: %s", field)
		}
	}

	// Verify success rate calculation
	successRate := metricsMap["success_rate"].(float64)
	expectedRate := float64(8) / float64(10) * 100.0
	if successRate != expectedRate {
		t.Errorf("expected success rate %.2f, got %.2f", expectedRate, successRate)
	}
}

// TestSessionMetrics_ConcurrentAccess tests concurrent metrics access
func TestSessionMetrics_ConcurrentAccess(t *testing.T) {
	metrics := &SessionMetrics{}
	const numGoroutines = 100
	const numOperations = 10

	// Use WaitGroup to wait for all goroutines
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Start multiple goroutines to access metrics concurrently
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				metrics.incrementTotalRequests()
				metrics.incrementSuccessfulRequests()
				metrics.updateLastRequestTime(time.Now())
				metrics.updateAverageResponseTime(time.Millisecond * 100)
			}
		}()
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify final result
	expectedTotal := int64(numGoroutines * numOperations)
	if metrics.TotalRequests != expectedTotal {
		t.Errorf("expected total requests %d, got %d", expectedTotal, metrics.TotalRequests)
	}
	if metrics.SuccessfulRequests != expectedTotal {
		t.Errorf("expected successful requests %d, got %d", expectedTotal, metrics.SuccessfulRequests)
	}
}
