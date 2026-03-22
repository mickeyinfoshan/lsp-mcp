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

// createTestConfig 创建测试配置
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
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("加载测试配置失败: %v", err)
	}

	return cfg
}

// TestNewManager 测试创建新的会话管理器
func TestNewManager(t *testing.T) {
	cfg := createTestConfig(t)

	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("创建会话管理器失败: %v", err)
	}

	if manager == nil {
		t.Fatal("会话管理器为空")
	}

	// 验证管理器字段
	if manager.config != cfg {
		t.Error("配置未正确设置")
	}

	if manager.lspClient == nil {
		t.Error("LSP客户端未初始化")
	}

	if manager.ctx == nil {
		t.Error("上下文未初始化")
	}

	if manager.cancel == nil {
		t.Error("取消函数未初始化")
	}

	if manager.metrics == nil {
		t.Error("指标未初始化")
	}

	// 清理资源
	manager.Close()
}

// TestGetSupportedLanguages 测试获取支持的语言
func TestGetSupportedLanguages(t *testing.T) {
	cfg := createTestConfig(t)
	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("创建会话管理器失败: %v", err)
	}
	defer manager.Close()

	languages := manager.GetSupportedLanguages()
	if len(languages) == 0 {
		t.Error("期望至少支持一种语言")
	}

	// 验证配置的语言是否在支持列表中
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
			t.Errorf("期望的语言 %s 不在支持列表中", expected)
		}
	}
}

// TestFindDefinition_InvalidRequest 测试无效的查找定义请求
func TestFindDefinition_InvalidRequest(t *testing.T) {
	cfg := createTestConfig(t)
	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("创建会话管理器失败: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// 测试空请求
	resp, err := manager.FindDefinition(ctx, nil)
	if err != nil {
		t.Errorf("期望返回错误响应而不是错误: %v", err)
	}
	if resp == nil {
		t.Fatal("响应为空")
	}
	if resp.Error == "" {
		t.Error("期望响应包含错误信息")
	}

	// 测试缺少必需字段的请求
	invalidReq := &types.FindDefinitionRequest{
		LanguageID: "", // 空语言ID
		RootURI:    "file:///tmp/test",
		FileURI:    "file:///tmp/test/main.go",
		Position: protocol.Position{
			Line:      0, // 原为-1，uint32不支持，0用于测试非法输入
			Character: 5,
		},
	}

	resp, err = manager.FindDefinition(ctx, invalidReq)
	if err != nil {
		t.Errorf("期望返回错误响应而不是错误: %v", err)
	}
	if resp == nil {
		t.Fatal("响应为空")
	}
	if resp.Error == "" {
		t.Error("期望响应包含错误信息")
	}
}

// TestFindDefinition_ValidRequest 测试有效的查找定义请求
func TestFindDefinition_ValidRequest(t *testing.T) {
	// 跳过需要实际LSP服务器的测试
	if testing.Short() {
		t.Skip("跳过需要LSP服务器的测试")
	}

	cfg := createTestConfig(t)
	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("创建会话管理器失败: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// 创建有效请求
	req := &types.FindDefinitionRequest{
		LanguageID: "go",
		RootURI:    "file:///tmp/test",
		FileURI:    "file:///tmp/test/main.go",
		Position: protocol.Position{
			Line:      0, // 原为-1，uint32不支持，0用于测试非法输入
			Character: 5,
		},
	}

	// 由于使用echo命令作为测试LSP服务器，这个测试可能会失败
	// 但我们可以验证请求处理逻辑
	resp, err := manager.FindDefinition(ctx, req)
	if err != nil {
		t.Errorf("查找定义失败: %v", err)
	}
	if resp == nil {
		t.Fatal("响应为空")
	}

	// 验证指标已更新
	metrics := manager.GetMetrics()
	if metrics.TotalRequests == 0 {
		t.Error("总请求数未更新")
	}
}

// TestGetMetrics 测试获取指标
func TestGetMetrics(t *testing.T) {
	cfg := createTestConfig(t)
	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("创建会话管理器失败: %v", err)
	}
	defer manager.Close()

	metrics := manager.GetMetrics()
	if metrics == nil {
		t.Fatal("指标为空")
	}

	// 验证初始指标值
	if metrics.TotalRequests != 0 {
		t.Error("初始总请求数应为0")
	}
	if metrics.SuccessfulRequests != 0 {
		t.Error("初始成功请求数应为0")
	}
	if metrics.FailedRequests != 0 {
		t.Error("初始失败请求数应为0")
	}
	if metrics.SessionsCreated != 0 {
		t.Error("初始创建会话数应为0")
	}
	if metrics.SessionsClosed != 0 {
		t.Error("初始关闭会话数应为0")
	}
}

// TestShutdown 测试关闭管理器
func TestShutdown(t *testing.T) {
	cfg := createTestConfig(t)
	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("创建会话管理器失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 测试关闭
	err = manager.Shutdown(ctx)
	if err != nil {
		t.Errorf("关闭管理器失败: %v", err)
	}

	// 验证上下文已取消
	select {
	case <-manager.ctx.Done():
		// 预期行为
	default:
		t.Error("上下文未被取消")
	}
}

// TestClose 测试关闭管理器
func TestClose(t *testing.T) {
	cfg := createTestConfig(t)
	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("创建会话管理器失败: %v", err)
	}

	// 测试关闭
	err = manager.Close()
	if err != nil {
		t.Errorf("关闭管理器失败: %v", err)
	}

	// 验证上下文已取消
	select {
	case <-manager.ctx.Done():
		// 预期行为
	default:
		t.Error("上下文未被取消")
	}
}

// TestSessionMetrics_IncrementMethods 测试指标增量方法
func TestSessionMetrics_IncrementMethods(t *testing.T) {
	metrics := &SessionMetrics{}

	// 测试增加总请求数
	metrics.incrementTotalRequests()
	if metrics.TotalRequests != 1 {
		t.Errorf("期望总请求数为1，实际为%d", metrics.TotalRequests)
	}

	// 测试增加成功请求数
	metrics.incrementSuccessfulRequests()
	if metrics.SuccessfulRequests != 1 {
		t.Errorf("期望成功请求数为1，实际为%d", metrics.SuccessfulRequests)
	}

	// 测试增加失败请求数
	metrics.incrementFailedRequests()
	if metrics.FailedRequests != 1 {
		t.Errorf("期望失败请求数为1，实际为%d", metrics.FailedRequests)
	}

	// 测试增加创建会话数
	metrics.incrementSessionsCreated()
	if metrics.SessionsCreated != 1 {
		t.Errorf("期望创建会话数为1，实际为%d", metrics.SessionsCreated)
	}

	// 测试增加关闭会话数
	metrics.incrementSessionsClosed()
	if metrics.SessionsClosed != 1 {
		t.Errorf("期望关闭会话数为1，实际为%d", metrics.SessionsClosed)
	}
}

// TestSessionMetrics_UpdateMethods 测试指标更新方法
func TestSessionMetrics_UpdateMethods(t *testing.T) {
	metrics := &SessionMetrics{}

	// 测试更新最后请求时间
	testTime := time.Now()
	metrics.updateLastRequestTime(testTime)
	if !metrics.LastRequestTime.Equal(testTime) {
		t.Error("最后请求时间未正确更新")
	}

	// 测试更新平均响应时间
	responseTime := 100 * time.Millisecond
	metrics.updateAverageResponseTime(responseTime)
	expectedMs := float64(responseTime.Nanoseconds()) / 1e6
	if metrics.AverageResponseTime != expectedMs {
		t.Errorf("期望平均响应时间为%.2f，实际为%.2f", expectedMs, metrics.AverageResponseTime)
	}
}

// TestValidateFindDefinitionRequest 测试验证查找定义请求
func TestValidateFindDefinitionRequest(t *testing.T) {
	cfg := createTestConfig(t)
	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("创建会话管理器失败: %v", err)
	}
	defer manager.Close()

	// 测试有效请求
	validReq := &types.FindDefinitionRequest{
		LanguageID: "go",
		RootURI:    "file:///tmp/test",
		FileURI:    "file:///tmp/test/main.go",
		Position: protocol.Position{
			Line:      0, // 原为-1，uint32不支持，0用于测试非法输入
			Character: 5,
		},
	}

	err = manager.validateFindDefinitionRequest(validReq)
	if err != nil {
		t.Errorf("有效请求验证失败: %v", err)
	}

	// 测试空语言ID
	invalidReq1 := &types.FindDefinitionRequest{
		LanguageID: "",
		RootURI:    "file:///tmp/test",
		FileURI:    "file:///tmp/test/main.go",
		Position: protocol.Position{
			Line:      0, // 原为-1，uint32不支持，0用于测试非法输入
			Character: 5,
		},
	}

	err = manager.validateFindDefinitionRequest(invalidReq1)
	if err == nil {
		t.Error("期望空语言ID验证失败")
	}

	// 测试空根URI
	invalidReq2 := &types.FindDefinitionRequest{
		LanguageID: "go",
		RootURI:    "",
		FileURI:    "file:///tmp/test/main.go",
		Position: protocol.Position{
			Line:      0, // 原为-1，uint32不支持，0用于测试非法输入
			Character: 5,
		},
	}

	err = manager.validateFindDefinitionRequest(invalidReq2)
	if err == nil {
		t.Error("期望空根URI验证失败")
	}

	// 测试空文件URI
	invalidReq3 := &types.FindDefinitionRequest{
		LanguageID: "go",
		RootURI:    "file:///tmp/test",
		FileURI:    "",
		Position: protocol.Position{
			Line:      0, // 原为-1，uint32不支持，0用于测试非法输入
			Character: 5,
		},
	}

	err = manager.validateFindDefinitionRequest(invalidReq3)
	if err == nil {
		t.Error("期望空文件URI验证失败")
	}

	// 测试不支持的语言
	invalidReq6 := &types.FindDefinitionRequest{
		LanguageID: "unsupported",
		RootURI:    "file:///tmp/test",
		FileURI:    "file:///tmp/test/main.go",
		Position: protocol.Position{
			Line:      0, // 原为-1，uint32不支持，0用于测试非法输入
			Character: 5,
		},
	}

	err = manager.validateFindDefinitionRequest(invalidReq6)
	if err == nil {
		t.Error("期望不支持的语言验证失败")
	}
}

// TestFindReferences_InvalidRequest 测试无效的查找引用请求
func TestFindReferences_InvalidRequest(t *testing.T) {
	cfg := createTestConfig(t)
	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("创建会话管理器失败: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// 测试空请求
	resp, err := manager.FindReferences(ctx, nil)
	if err != nil {
		t.Errorf("期望返回错误响应而不是错误: %v", err)
	}
	if resp == nil {
		t.Fatal("响应为空")
	}
	if resp.Error == "" {
		t.Error("期望响应包含错误信息")
	}

	// 测试缺少必需字段的请求
	invalidReq := &types.FindReferencesRequest{
		LanguageID: "", // 空语言ID
		FileURI:    "file:///tmp/test/main.go",
		Position: protocol.Position{
			Line:      0, // 原为-1，uint32不支持，0用于测试非法输入
			Character: 5,
		},
	}

	resp, err = manager.FindReferences(ctx, invalidReq)
	if err != nil {
		t.Errorf("期望返回错误响应而不是错误: %v", err)
	}
	if resp == nil {
		t.Fatal("响应为空")
	}
	if resp.Error == "" {
		t.Error("期望响应包含错误信息")
	}
}

// TestGetHover_InvalidRequest 测试无效的悬停请求
func TestGetHover_InvalidRequest(t *testing.T) {
	cfg := createTestConfig(t)
	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("创建会话管理器失败: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// 测试空请求
	resp, err := manager.GetHover(ctx, nil)
	if err != nil {
		t.Errorf("期望返回错误响应而不是错误: %v", err)
	}
	if resp == nil {
		t.Fatal("响应为空")
	}
	if resp.Error == "" {
		t.Error("期望响应包含错误信息")
	}

	// 测试缺少必需字段的请求
	invalidReq := &types.HoverRequest{
		LanguageID: "", // 空语言ID
		FileURI:    "file:///tmp/test/main.go",
		Position: protocol.Position{
			Line:      0, // 原为-1，uint32不支持，0用于测试非法输入
			Character: 5,
		},
	}

	resp, err = manager.GetHover(ctx, invalidReq)
	if err != nil {
		t.Errorf("期望返回错误响应而不是错误: %v", err)
	}
	if resp == nil {
		t.Fatal("响应为空")
	}
	if resp.Error == "" {
		t.Error("期望响应包含错误信息")
	}
}

// TestGetCompletion_InvalidRequest 测试无效的代码补全请求
func TestGetCompletion_InvalidRequest(t *testing.T) {
	cfg := createTestConfig(t)
	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("创建会话管理器失败: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// 测试空请求
	resp, err := manager.GetCompletion(ctx, nil)
	if err != nil {
		t.Errorf("期望返回错误响应而不是错误: %v", err)
	}
	if resp == nil {
		t.Fatal("响应为空")
	}
	if resp.Error == "" {
		t.Error("期望响应包含错误信息")
	}

	// 测试缺少必需字段的请求
	invalidReq := &types.CompletionRequest{
		LanguageID: "", // 空语言ID
		FileURI:    "file:///tmp/test/main.go",
		Position: protocol.Position{
			Line:      0, // 原为-1，uint32不支持，0用于测试非法输入
			Character: 5,
		},
	}

	resp, err = manager.GetCompletion(ctx, invalidReq)
	if err != nil {
		t.Errorf("期望返回错误响应而不是错误: %v", err)
	}
	if resp == nil {
		t.Fatal("响应为空")
	}
	if resp.Error == "" {
		t.Error("期望响应包含错误信息")
	}
}

// TestGetSessionInfo 测试获取会话信息
func TestGetSessionInfo(t *testing.T) {
	cfg := createTestConfig(t)
	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("创建会话管理器失败: %v", err)
	}
	defer manager.Close()

	info := manager.GetSessionInfo()
	if info == nil {
		t.Fatal("会话信息为空")
	}

	// 验证包含指标信息
	if _, exists := info["metrics"]; !exists {
		t.Error("会话信息中缺少指标信息")
	}
}

// TestGetLSPServerConfig 测试获取LSP服务器配置
func TestGetLSPServerConfig(t *testing.T) {
	cfg := createTestConfig(t)
	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("创建会话管理器失败: %v", err)
	}
	defer manager.Close()

	// 测试获取存在的语言配置
	config, exists := manager.GetLSPServerConfig("go")
	if !exists {
		t.Error("期望找到go语言配置")
	}
	if config == nil {
		t.Error("go语言配置为空")
	}

	// 测试获取不存在的语言配置
	_, exists = manager.GetLSPServerConfig("nonexistent")
	if exists {
		t.Error("期望不存在的语言配置返回false")
	}
}

// TestSessionMetrics_Reset 测试重置指标
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

	// 重置指标
	metrics.Reset()

	// 验证所有指标都被重置为零值
	if metrics.TotalRequests != 0 {
		t.Errorf("期望总请求数为0，实际为%d", metrics.TotalRequests)
	}
	if metrics.SuccessfulRequests != 0 {
		t.Errorf("期望成功请求数为0，实际为%d", metrics.SuccessfulRequests)
	}
	if metrics.FailedRequests != 0 {
		t.Errorf("期望失败请求数为0，实际为%d", metrics.FailedRequests)
	}
	if metrics.AverageResponseTime != 0 {
		t.Errorf("期望平均响应时间为0，实际为%.2f", metrics.AverageResponseTime)
	}
	if metrics.SessionsCreated != 0 {
		t.Errorf("期望创建会话数为0，实际为%d", metrics.SessionsCreated)
	}
	if metrics.SessionsClosed != 0 {
		t.Errorf("期望关闭会话数为0，实际为%d", metrics.SessionsClosed)
	}
	if !metrics.LastRequestTime.IsZero() {
		t.Error("期望最后请求时间为零值")
	}
}

// TestSessionMetrics_GetMetrics 测试获取指标信息
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
		t.Fatal("指标映射为空")
	}

	// 验证所有字段都存在
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
			t.Errorf("指标映射中缺少字段: %s", field)
		}
	}

	// 验证成功率计算
	successRate := metricsMap["success_rate"].(float64)
	expectedRate := float64(8) / float64(10) * 100.0
	if successRate != expectedRate {
		t.Errorf("期望成功率为%.2f，实际为%.2f", expectedRate, successRate)
	}
}

// TestSessionMetrics_ConcurrentAccess 测试并发访问指标
func TestSessionMetrics_ConcurrentAccess(t *testing.T) {
	metrics := &SessionMetrics{}
	const numGoroutines = 100
	const numOperations = 10

	// 使用WaitGroup等待所有goroutine完成
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// 启动多个goroutine并发访问指标
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

	// 等待所有goroutine完成
	wg.Wait()

	// 验证最终结果
	expectedTotal := int64(numGoroutines * numOperations)
	if metrics.TotalRequests != expectedTotal {
		t.Errorf("期望总请求数为%d，实际为%d", expectedTotal, metrics.TotalRequests)
	}
	if metrics.SuccessfulRequests != expectedTotal {
		t.Errorf("期望成功请求数为%d，实际为%d", expectedTotal, metrics.SuccessfulRequests)
	}
}
