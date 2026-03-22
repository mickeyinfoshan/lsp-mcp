// LSP 会话生命周期与指标管理，负责请求分发与参数校验
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

// Manager 会话管理器
// Manager manages LSP sessions and dispatches requests
type Manager struct {
	// lspClient LSP 客户端实例
	// LSP client instance
	lspClient *lsp.Client
	// config 配置信息
	// Configuration information
	config *config.Config
	// mutex 互斥锁
	// Mutex for concurrent access
	mutex sync.RWMutex
	// ctx 上下文
	// Context for session management
	ctx context.Context
	// cancel 取消函数
	// Cancel function for context
	cancel context.CancelFunc
	// metrics 会话指标
	// Session metrics
	metrics *SessionMetrics
}

// SessionMetrics 会话指标
type SessionMetrics struct {
	// TotalRequests 总请求数
	TotalRequests int64 `json:"total_requests"`
	// SuccessfulRequests 成功请求数
	SuccessfulRequests int64 `json:"successful_requests"`
	// FailedRequests 失败请求数
	FailedRequests int64 `json:"failed_requests"`
	// AverageResponseTime 平均响应时间（毫秒）
	AverageResponseTime float64 `json:"average_response_time_ms"`
	// SessionsCreated 创建的会话数
	SessionsCreated int64 `json:"sessions_created"`
	// SessionsClosed 关闭的会话数
	SessionsClosed int64 `json:"sessions_closed"`
	// LastRequestTime 最后请求时间
	LastRequestTime time.Time `json:"last_request_time"`
	// mutex 指标互斥锁
	mutex sync.RWMutex
}

// NewManager 创建新的会话管理器
func NewManager(cfg *config.Config) (*Manager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// 创建LSP客户端
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

// GetLSPClient 获取LSP客户端实例
func (m *Manager) GetLSPClient() *lsp.Client {
	return m.lspClient
}

// FindDefinition 查找变量定义
func (m *Manager) FindDefinition(ctx context.Context, req *types.FindDefinitionRequest) (*types.FindDefinitionResponse, error) {
	startTime := time.Now()
	m.metrics.incrementTotalRequests()
	m.metrics.updateLastRequestTime(startTime)

	// 验证请求参数
	if err := m.validateFindDefinitionRequest(req); err != nil {
		m.metrics.incrementFailedRequests()
		return &types.FindDefinitionResponse{
			Error: fmt.Sprintf("请求参数验证失败: %v", err),
		}, nil
	}

	// 调用LSP客户端查找定义
	response, err := m.lspClient.FindDefinition(ctx, req)
	if err != nil {
		m.metrics.incrementFailedRequests()
		return &types.FindDefinitionResponse{
			Error: fmt.Sprintf("LSP查找定义失败: %v", err),
		}, nil
	}

	// 检查响应是否包含错误
	if response.Error != "" {
		m.metrics.incrementFailedRequests()
	} else {
		m.metrics.incrementSuccessfulRequests()
	}

	// 更新响应时间指标
	responseTime := time.Since(startTime)
	m.metrics.updateAverageResponseTime(responseTime)

	return response, nil
}

// validateFindDefinitionRequest 验证查找定义请求
func (m *Manager) validateFindDefinitionRequest(req *types.FindDefinitionRequest) error {
	if req == nil {
		return fmt.Errorf("请求不能为空")
	}

	if req.LanguageID == "" {
		return fmt.Errorf("语言ID不能为空")
	}

	if req.RootURI == "" {
		return fmt.Errorf("根URI不能为空")
	}

	if req.FileURI == "" {
		return fmt.Errorf("文件URI不能为空")
	}

	if req.Position.Line < 0 {
		return fmt.Errorf("行号不能为负数")
	}

	if req.Position.Character < 0 {
		return fmt.Errorf("字符位置不能为负数")
	}

	// 检查是否支持该语言
	if _, exists := m.config.GetLSPServerConfig(req.LanguageID); !exists {
		return fmt.Errorf("不支持的语言: %s", req.LanguageID)
	}

	return nil
}

// validateFindReferencesRequest 验证查找引用请求的参数
func (m *Manager) validateFindReferencesRequest(req *types.FindReferencesRequest) error {
	if req == nil {
		return fmt.Errorf("请求不能为空")
	}

	if req.LanguageID == "" {
		return fmt.Errorf("语言ID不能为空")
	}

	if req.FileURI == "" {
		return fmt.Errorf("文件URI不能为空")
	}

	if req.Position.Line < 0 {
		return fmt.Errorf("行号不能为负数")
	}

	if req.Position.Character < 0 {
		return fmt.Errorf("字符位置不能为负数")
	}

	// 检查是否支持该语言
	if _, exists := m.config.GetLSPServerConfig(req.LanguageID); !exists {
		return fmt.Errorf("不支持的语言: %s", req.LanguageID)
	}

	return nil
}

// validateHoverRequest 验证悬停请求的参数
func (m *Manager) validateHoverRequest(req *types.HoverRequest) error {
	if req == nil {
		return fmt.Errorf("请求不能为空")
	}

	if req.LanguageID == "" {
		return fmt.Errorf("语言ID不能为空")
	}

	if req.FileURI == "" {
		return fmt.Errorf("文件URI不能为空")
	}

	if req.Position.Line < 0 {
		return fmt.Errorf("行号不能为负数")
	}

	if req.Position.Character < 0 {
		return fmt.Errorf("字符位置不能为负数")
	}

	// 检查是否支持该语言
	if _, exists := m.config.GetLSPServerConfig(req.LanguageID); !exists {
		return fmt.Errorf("不支持的语言: %s", req.LanguageID)
	}

	return nil
}

// validateCompletionRequest 验证代码补全请求的参数
func (m *Manager) validateCompletionRequest(req *types.CompletionRequest) error {
	if req == nil {
		return fmt.Errorf("请求不能为空")
	}

	if req.LanguageID == "" {
		return fmt.Errorf("语言ID不能为空")
	}

	if req.FileURI == "" {
		return fmt.Errorf("文件URI不能为空")
	}

	if req.Position.Line < 0 {
		return fmt.Errorf("行号不能为负数")
	}

	if req.Position.Character < 0 {
		return fmt.Errorf("字符位置不能为负数")
	}

	// 检查是否支持该语言
	if _, exists := m.config.GetLSPServerConfig(req.LanguageID); !exists {
		return fmt.Errorf("不支持的语言: %s", req.LanguageID)
	}

	return nil
}

// GetSessionInfo 获取会话信息
func (m *Manager) GetSessionInfo() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	info := m.lspClient.GetSessionInfo()
	info["metrics"] = m.metrics.getMetrics()

	return info
}

// GetMetrics 获取指标信息
func (m *Manager) GetMetrics() *SessionMetrics {
	return m.metrics
}

// Shutdown 关闭会话管理器
func (m *Manager) Shutdown(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 取消上下文
	if m.cancel != nil {
		m.cancel()
	}

	return nil
}

// Close 关闭会话管理器
func (m *Manager) Close() error {
	m.cancel()

	if m.lspClient != nil {
		return m.lspClient.Close()
	}

	return nil
}

// FindReferences 查找引用
func (m *Manager) FindReferences(ctx context.Context, req *types.FindReferencesRequest) (*types.FindReferencesResponse, error) {
	startTime := time.Now()
	m.metrics.incrementTotalRequests()
	m.metrics.updateLastRequestTime(startTime)

	// 验证请求参数
	if err := m.validateFindReferencesRequest(req); err != nil {
		m.metrics.incrementFailedRequests()
		return &types.FindReferencesResponse{
			Error: fmt.Sprintf("请求参数验证失败: %v", err),
		}, nil
	}

	// 调用LSP客户端查找引用
	response, err := m.lspClient.FindReferences(ctx, req)
	if err != nil {
		m.metrics.incrementFailedRequests()
		return &types.FindReferencesResponse{
			Error: fmt.Sprintf("LSP查找引用失败: %v", err),
		}, nil
	}

	// 检查响应是否包含错误
	if response.Error != "" {
		m.metrics.incrementFailedRequests()
	} else {
		m.metrics.incrementSuccessfulRequests()
	}

	// 更新响应时间指标
	responseTime := time.Since(startTime)
	m.metrics.updateAverageResponseTime(responseTime)

	return response, nil
}

// GetHover 获取悬停信息
func (m *Manager) GetHover(ctx context.Context, req *types.HoverRequest) (*types.HoverResponse, error) {
	startTime := time.Now()
	m.metrics.incrementTotalRequests()
	m.metrics.updateLastRequestTime(startTime)

	// 验证请求参数
	if err := m.validateHoverRequest(req); err != nil {
		m.metrics.incrementFailedRequests()
		return &types.HoverResponse{
			Error: fmt.Sprintf("请求参数验证失败: %v", err),
		}, nil
	}

	// 调用LSP客户端获取悬停信息
	response, err := m.lspClient.GetHover(ctx, req)
	if err != nil {
		m.metrics.incrementFailedRequests()
		return &types.HoverResponse{
			Error: fmt.Sprintf("LSP获取悬停信息失败: %v", err),
		}, nil
	}

	// 检查响应是否包含错误
	if response.Error != "" {
		m.metrics.incrementFailedRequests()
	} else {
		m.metrics.incrementSuccessfulRequests()
	}

	// 更新响应时间指标
	responseTime := time.Since(startTime)
	m.metrics.updateAverageResponseTime(responseTime)

	return response, nil
}

// GetCompletion 获取代码补全
func (m *Manager) GetCompletion(ctx context.Context, req *types.CompletionRequest) (*types.CompletionResponse, error) {
	startTime := time.Now()
	m.metrics.incrementTotalRequests()
	m.metrics.updateLastRequestTime(startTime)

	// 验证请求参数
	if err := m.validateCompletionRequest(req); err != nil {
		m.metrics.incrementFailedRequests()
		return &types.CompletionResponse{
			Error: fmt.Sprintf("请求参数验证失败: %v", err),
		}, nil
	}

	// 调用LSP客户端获取代码补全
	response, err := m.lspClient.GetCompletion(ctx, req)
	if err != nil {
		m.metrics.incrementFailedRequests()
		return &types.CompletionResponse{
			Error: fmt.Sprintf("LSP获取代码补全失败: %v", err),
		}, nil
	}

	// 检查响应是否包含错误
	if response.Error != "" {
		m.metrics.incrementFailedRequests()
	} else {
		m.metrics.incrementSuccessfulRequests()
	}

	// 更新响应时间指标
	responseTime := time.Since(startTime)
	m.metrics.updateAverageResponseTime(responseTime)

	return response, nil
}

// GetSupportedLanguages 获取支持的语言列表
func (m *Manager) GetSupportedLanguages() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	languages := make([]string, 0, len(m.config.LSPServers))
	for languageID := range m.config.LSPServers {
		languages = append(languages, languageID)
	}

	return languages
}

// GetLSPServerConfig 获取LSP服务器配置
func (m *Manager) GetLSPServerConfig(languageID string) (*config.LSPServerConfig, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.config.GetLSPServerConfig(languageID)
}

// SessionMetrics 方法

// incrementTotalRequests 增加总请求数
func (sm *SessionMetrics) incrementTotalRequests() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.TotalRequests++
}

// incrementSuccessfulRequests 增加成功请求数
func (sm *SessionMetrics) incrementSuccessfulRequests() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.SuccessfulRequests++
}

// incrementFailedRequests 增加失败请求数
func (sm *SessionMetrics) incrementFailedRequests() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.FailedRequests++
}

// updateAverageResponseTime 更新平均响应时间
func (sm *SessionMetrics) updateAverageResponseTime(responseTime time.Duration) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// 使用简单的移动平均算法
	responseTimeMs := float64(responseTime.Nanoseconds()) / 1e6
	if sm.AverageResponseTime == 0 {
		sm.AverageResponseTime = responseTimeMs
	} else {
		// 使用指数移动平均，权重为0.1
		sm.AverageResponseTime = sm.AverageResponseTime*0.9 + responseTimeMs*0.1
	}
}

// updateLastRequestTime 更新最后请求时间
func (sm *SessionMetrics) updateLastRequestTime(t time.Time) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.LastRequestTime = t
}

// incrementSessionsCreated 增加创建的会话数
func (sm *SessionMetrics) incrementSessionsCreated() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.SessionsCreated++
}

// incrementSessionsClosed 增加关闭的会话数
func (sm *SessionMetrics) incrementSessionsClosed() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.SessionsClosed++
}

// getMetrics 获取指标信息（线程安全）
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

// getSuccessRate 获取成功率
func (sm *SessionMetrics) getSuccessRate() float64 {
	if sm.TotalRequests == 0 {
		return 0.0
	}
	return float64(sm.SuccessfulRequests) / float64(sm.TotalRequests) * 100.0
}

// Reset 重置指标
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
