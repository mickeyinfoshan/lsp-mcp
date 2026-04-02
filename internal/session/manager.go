// Manages LSP session lifecycle, metrics, request dispatching, and parameter validation.
package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mickeyinfoshan/lsp-mcp/internal/config"
	"github.com/mickeyinfoshan/lsp-mcp/internal/lsp"
	"github.com/mickeyinfoshan/lsp-mcp/pkg/types"
)

// Manager manages LSP sessions and dispatches requests
type Manager struct {
	// LSP client instance
	lspClient *lsp.Client
	// Configuration information
	config *config.Config
	// Mutex for concurrent access
	mutex sync.RWMutex
	// Context for session management
	ctx context.Context
	// Cancel function for context
	cancel context.CancelFunc
	// Session metrics
	metrics *SessionMetrics
}

// SessionMetrics session metrics
type SessionMetrics struct {
	// TotalRequests total request count
	TotalRequests int64 `json:"total_requests"`
	// SuccessfulRequests successful request count
	SuccessfulRequests int64 `json:"successful_requests"`
	// FailedRequests failed request count
	FailedRequests int64 `json:"failed_requests"`
	// AverageResponseTime average response time (ms)
	AverageResponseTime float64 `json:"average_response_time_ms"`
	// SessionsCreated session count created
	SessionsCreated int64 `json:"sessions_created"`
	// SessionsClosed session count closed
	SessionsClosed int64 `json:"sessions_closed"`
	// LastRequestTime last request time
	LastRequestTime time.Time `json:"last_request_time"`
	// mutex protects metrics
	mutex sync.RWMutex
}

// NewManager creates a new session manager
func NewManager(cfg *config.Config) (*Manager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Create LSP client
	lspClient := lsp.NewClient(cfg)

	manager := &Manager{
		lspClient: lspClient,
		config:    cfg,
		ctx:       ctx,
		cancel:    cancel,
		metrics:   &SessionMetrics{},
	}

	return manager, nil
}

// GetLSPClient returns the LSP client instance
func (m *Manager) GetLSPClient() *lsp.Client {
	return m.lspClient
}

// FindDefinition finds the definition of a symbol
func (m *Manager) FindDefinition(ctx context.Context, req *types.FindDefinitionRequest) (*types.FindDefinitionResponse, error) {
	startTime := time.Now()
	m.metrics.incrementTotalRequests()
	m.metrics.updateLastRequestTime(startTime)

	// Validate request parameters
	if err := m.validateFindDefinitionRequest(req); err != nil {
		m.metrics.incrementFailedRequests()
		return &types.FindDefinitionResponse{
			Error: fmt.Sprintf("request parameter validation failed: %v", err),
		}, nil
	}

	// Call LSP client to find definition
	response, err := m.lspClient.FindDefinition(ctx, req)
	if err != nil {
		m.metrics.incrementFailedRequests()
		return &types.FindDefinitionResponse{
			Error: fmt.Sprintf("LSP find definition failed: %v", err),
		}, nil
	}

	// Check response for errors
	if response.Error != "" {
		m.metrics.incrementFailedRequests()
	} else {
		m.metrics.incrementSuccessfulRequests()
	}

	// Update response time metrics
	responseTime := time.Since(startTime)
	m.metrics.updateAverageResponseTime(responseTime)

	return response, nil
}

// validateFindDefinitionRequest validates a find definition request
func (m *Manager) validateFindDefinitionRequest(req *types.FindDefinitionRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if req.LanguageID == "" {
		return fmt.Errorf("language ID cannot be empty")
	}

	if req.RootURI == "" {
		return fmt.Errorf("root URI cannot be empty")
	}

	if req.FileURI == "" {
		return fmt.Errorf("file URI cannot be empty")
	}

	if req.Position.Line < 0 {
		return fmt.Errorf("line number cannot be negative")
	}

	if req.Position.Character < 0 {
		return fmt.Errorf("character position cannot be negative")
	}

	// Check if the language is supported
	if _, exists := m.config.GetLSPServerConfig(req.LanguageID); !exists {
		return fmt.Errorf("unsupported language: %s", req.LanguageID)
	}

	return nil
}

// validateFindReferencesRequest validates a find references request
func (m *Manager) validateFindReferencesRequest(req *types.FindReferencesRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if req.LanguageID == "" {
		return fmt.Errorf("language ID cannot be empty")
	}

	if req.FileURI == "" {
		return fmt.Errorf("file URI cannot be empty")
	}

	if req.Position.Line < 0 {
		return fmt.Errorf("line number cannot be negative")
	}

	if req.Position.Character < 0 {
		return fmt.Errorf("character position cannot be negative")
	}

	// Check if the language is supported
	if _, exists := m.config.GetLSPServerConfig(req.LanguageID); !exists {
		return fmt.Errorf("unsupported language: %s", req.LanguageID)
	}

	return nil
}

// validateHoverRequest validates a hover request
func (m *Manager) validateHoverRequest(req *types.HoverRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if req.LanguageID == "" {
		return fmt.Errorf("language ID cannot be empty")
	}

	if req.FileURI == "" {
		return fmt.Errorf("file URI cannot be empty")
	}

	if req.Position.Line < 0 {
		return fmt.Errorf("line number cannot be negative")
	}

	if req.Position.Character < 0 {
		return fmt.Errorf("character position cannot be negative")
	}

	// Check if the language is supported
	if _, exists := m.config.GetLSPServerConfig(req.LanguageID); !exists {
		return fmt.Errorf("unsupported language: %s", req.LanguageID)
	}

	return nil
}

// validateCompletionRequest validates a completion request
func (m *Manager) validateCompletionRequest(req *types.CompletionRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if req.LanguageID == "" {
		return fmt.Errorf("language ID cannot be empty")
	}

	if req.FileURI == "" {
		return fmt.Errorf("file URI cannot be empty")
	}

	if req.Position.Line < 0 {
		return fmt.Errorf("line number cannot be negative")
	}

	if req.Position.Character < 0 {
		return fmt.Errorf("character position cannot be negative")
	}

	// Check if the language is supported
	if _, exists := m.config.GetLSPServerConfig(req.LanguageID); !exists {
		return fmt.Errorf("unsupported language: %s", req.LanguageID)
	}

	return nil
}

// GetSessionInfo returns session info
func (m *Manager) GetSessionInfo() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	info := m.lspClient.GetSessionInfo()
	info["metrics"] = m.metrics.getMetrics()

	return info
}

// GetMetrics returns metrics info
func (m *Manager) GetMetrics() *SessionMetrics {
	return m.metrics
}

// Shutdown shuts down the session manager
func (m *Manager) Shutdown(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Cancel context
	if m.cancel != nil {
		m.cancel()
	}

	return nil
}

// Close closes the session manager
func (m *Manager) Close() error {
	m.cancel()

	if m.lspClient != nil {
		return m.lspClient.Close()
	}

	return nil
}

// FindReferences finds references
func (m *Manager) FindReferences(ctx context.Context, req *types.FindReferencesRequest) (*types.FindReferencesResponse, error) {
	startTime := time.Now()
	m.metrics.incrementTotalRequests()
	m.metrics.updateLastRequestTime(startTime)

	// Validate request parameters
	if err := m.validateFindReferencesRequest(req); err != nil {
		m.metrics.incrementFailedRequests()
		return &types.FindReferencesResponse{
			Error: fmt.Sprintf("request parameter validation failed: %v", err),
		}, nil
	}

	// Call LSP client to find references
	response, err := m.lspClient.FindReferences(ctx, req)
	if err != nil {
		m.metrics.incrementFailedRequests()
		return &types.FindReferencesResponse{
			Error: fmt.Sprintf("LSP find references failed: %v", err),
		}, nil
	}

	// Check response for errors
	if response.Error != "" {
		m.metrics.incrementFailedRequests()
	} else {
		m.metrics.incrementSuccessfulRequests()
	}

	// Update response time metrics
	responseTime := time.Since(startTime)
	m.metrics.updateAverageResponseTime(responseTime)

	return response, nil
}

// GetHover fetches hover info
func (m *Manager) GetHover(ctx context.Context, req *types.HoverRequest) (*types.HoverResponse, error) {
	startTime := time.Now()
	m.metrics.incrementTotalRequests()
	m.metrics.updateLastRequestTime(startTime)

	// Validate request parameters
	if err := m.validateHoverRequest(req); err != nil {
		m.metrics.incrementFailedRequests()
		return &types.HoverResponse{
			Error: fmt.Sprintf("request parameter validation failed: %v", err),
		}, nil
	}

	// Call LSP client to get hover info
	response, err := m.lspClient.GetHover(ctx, req)
	if err != nil {
		m.metrics.incrementFailedRequests()
		return &types.HoverResponse{
			Error: fmt.Sprintf("LSP hover request failed: %v", err),
		}, nil
	}

	// Check response for errors
	if response.Error != "" {
		m.metrics.incrementFailedRequests()
	} else {
		m.metrics.incrementSuccessfulRequests()
	}

	// Update response time metrics
	responseTime := time.Since(startTime)
	m.metrics.updateAverageResponseTime(responseTime)

	return response, nil
}

// GetCompletion fetches code completions
func (m *Manager) GetCompletion(ctx context.Context, req *types.CompletionRequest) (*types.CompletionResponse, error) {
	startTime := time.Now()
	m.metrics.incrementTotalRequests()
	m.metrics.updateLastRequestTime(startTime)

	// Validate request parameters
	if err := m.validateCompletionRequest(req); err != nil {
		m.metrics.incrementFailedRequests()
		return &types.CompletionResponse{
			Error: fmt.Sprintf("request parameter validation failed: %v", err),
		}, nil
	}

	// Call LSP client to get completions
	response, err := m.lspClient.GetCompletion(ctx, req)
	if err != nil {
		m.metrics.incrementFailedRequests()
		return &types.CompletionResponse{
			Error: fmt.Sprintf("LSP completion request failed: %v", err),
		}, nil
	}

	// Check response for errors
	if response.Error != "" {
		m.metrics.incrementFailedRequests()
	} else {
		m.metrics.incrementSuccessfulRequests()
	}

	// Update response time metrics
	responseTime := time.Since(startTime)
	m.metrics.updateAverageResponseTime(responseTime)

	return response, nil
}

// GetSupportedLanguages returns supported languages
func (m *Manager) GetSupportedLanguages() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	languages := make([]string, 0, len(m.config.LSPServers))
	for languageID := range m.config.LSPServers {
		languages = append(languages, languageID)
	}

	return languages
}

// GetLSPServerConfig returns the LSP server config
func (m *Manager) GetLSPServerConfig(languageID string) (*config.LSPServerConfig, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.config.GetLSPServerConfig(languageID)
}

// SessionMetrics methods

// incrementTotalRequests increments total requests
func (sm *SessionMetrics) incrementTotalRequests() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.TotalRequests++
}

// incrementSuccessfulRequests increments successful requests
func (sm *SessionMetrics) incrementSuccessfulRequests() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.SuccessfulRequests++
}

// incrementFailedRequests increments failed requests
func (sm *SessionMetrics) incrementFailedRequests() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.FailedRequests++
}

// updateAverageResponseTime updates average response time
func (sm *SessionMetrics) updateAverageResponseTime(responseTime time.Duration) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Simple moving average
	responseTimeMs := float64(responseTime.Nanoseconds()) / 1e6
	if sm.AverageResponseTime == 0 {
		sm.AverageResponseTime = responseTimeMs
	} else {
		// Exponential moving average with weight 0.1
		sm.AverageResponseTime = sm.AverageResponseTime*0.9 + responseTimeMs*0.1
	}
}

// updateLastRequestTime updates last request time
func (sm *SessionMetrics) updateLastRequestTime(t time.Time) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.LastRequestTime = t
}

// incrementSessionsCreated increments sessions created
func (sm *SessionMetrics) incrementSessionsCreated() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.SessionsCreated++
}

// incrementSessionsClosed increments sessions closed
func (sm *SessionMetrics) incrementSessionsClosed() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.SessionsClosed++
}

// getMetrics returns metrics (thread-safe)
func (sm *SessionMetrics) getMetrics() map[string]interface{} {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return map[string]interface{}{
		"total_requests":           sm.TotalRequests,
		"successful_requests":      sm.SuccessfulRequests,
		"failed_requests":          sm.FailedRequests,
		"average_response_time_ms": sm.AverageResponseTime,
		"sessions_created":         sm.SessionsCreated,
		"sessions_closed":          sm.SessionsClosed,
		"last_request_time":        sm.LastRequestTime,
		"success_rate":             sm.getSuccessRate(),
	}
}

// getSuccessRate returns success rate
func (sm *SessionMetrics) getSuccessRate() float64 {
	if sm.TotalRequests == 0 {
		return 0.0
	}
	return float64(sm.SuccessfulRequests) / float64(sm.TotalRequests) * 100.0
}

// Reset resets metrics
func (sm *SessionMetrics) Reset() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.TotalRequests = 0
	sm.SuccessfulRequests = 0
	sm.FailedRequests = 0
	sm.AverageResponseTime = 0
	sm.SessionsCreated = 0
	sm.SessionsClosed = 0
	sm.LastRequestTime = time.Time{}
}
