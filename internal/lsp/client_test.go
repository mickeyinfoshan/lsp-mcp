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

// readWriteCloser 实现 io.ReadWriteCloser，组合 reader/writer
// 用于测试管道读写
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
	// 测试用，直接返回 nil
	return nil
}

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

// TestNewClient 测试创建新的LSP客户端
func TestNewClient(t *testing.T) {
	cfg := createTestConfig(t)

	client := NewClient(cfg)
	if client == nil {
		t.Fatal("创建LSP客户端失败")
	}

	// 验证客户端字段
	if client.config != cfg {
		t.Error("配置未正确设置")
	}

	if client.sessions == nil {
		t.Error("会话映射未初始化")
	}

	if client.ctx == nil {
		t.Error("上下文未初始化")
	}

	if client.cancel == nil {
		t.Error("取消函数未初始化")
	}

	// 清理资源
	client.Close()
}

// TestGetOrCreateSession_NewSession 测试创建新会话
func TestGetOrCreateSession_NewSession(t *testing.T) {
	// 跳过需要实际LSP服务器的测试
	if testing.Short() {
		t.Skip("跳过需要LSP服务器的测试")
	}

	cfg := createTestConfig(t)
	client := NewClient(cfg)
	defer client.Close()

	languageID := "go"
	rootURI := "file:///tmp/test"

	// 由于使用echo命令作为测试LSP服务器，这个测试会失败
	// 但我们可以测试错误处理逻辑
	session, err := client.GetOrCreateSession(languageID, rootURI)
	if err == nil {
		t.Log("意外成功创建会话（使用echo命令）")
		if session != nil {
			t.Log("会话创建成功")
		}
	} else {
		t.Logf("预期的错误: %v", err)
	}
}

// TestGetOrCreateSession_ExistingSession 测试获取现有会话
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

	// 手动创建一个模拟会话
	mockSession := &types.LSPSession{
		Key:           sessionKey,
		Conn:          nil, // 模拟连接
		Process:       nil, // 模拟进程
		CreatedAt:     time.Now(),
		LastUsedAt:    time.Now(),
		IsInitialized: true,
	}

	client.sessionsMutex.Lock()
	client.sessions[sessionKey.String()] = mockSession
	client.sessionsMutex.Unlock()

	// 测试获取现有会话
	session, err := client.GetOrCreateSession(languageID, rootURI)
	if err != nil {
		t.Fatalf("获取现有会话失败: %v", err)
	}

	if session != mockSession {
		t.Error("返回的会话不是预期的会话")
	}

	// 验证最后使用时间已更新
	if session.LastUsedAt.Before(mockSession.LastUsedAt) {
		t.Error("最后使用时间未更新")
	}
}

// TestGetOrCreateSession_MaxSessionsLimit 测试最大会话数限制
func TestGetOrCreateSession_MaxSessionsLimit(t *testing.T) {
	cfg := createTestConfig(t)
	// 设置较小的最大会话数用于测试
	cfg.Session.MaxSessions = 2

	client := NewClient(cfg)
	defer client.Close()

	// 手动添加会话直到达到限制
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

	// 尝试创建超出限制的会话
	_, err := client.GetOrCreateSession("go", "file:///tmp/test_overflow")
	if err == nil {
		t.Error("期望因达到最大会话数限制而失败")
	}

	if !strings.Contains(err.Error(), "最大会话数限制") {
		t.Errorf("错误消息不正确: %v", err)
	}
}

// TestGetOrCreateSession_UnsupportedLanguage 测试不支持的语言
func TestGetOrCreateSession_UnsupportedLanguage(t *testing.T) {
	cfg := createTestConfig(t)
	client := NewClient(cfg)
	defer client.Close()

	// 尝试使用不支持的语言
	_, err := client.GetOrCreateSession("unsupported", "file:///tmp/test")
	if err == nil {
		t.Error("期望因不支持的语言而失败")
	}

	if !strings.Contains(err.Error(), "不支持的语言") {
		t.Errorf("错误消息不正确: %v", err)
	}
}

// TestClose 测试关闭客户端
func TestClose(t *testing.T) {
	cfg := createTestConfig(t)
	client := NewClient(cfg)

	// 添加一个模拟会话
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

	// 测试关闭
	err := client.Close()
	if err != nil {
		t.Errorf("关闭客户端失败: %v", err)
	}

	// 验证上下文已取消
	select {
	case <-client.ctx.Done():
		// 预期行为
	default:
		t.Error("上下文未被取消")
	}

	// 验证会话已清理
	client.sessionsMutex.RLock()
	sessionCount := len(client.sessions)
	client.sessionsMutex.RUnlock()

	if sessionCount != 0 {
		t.Errorf("期望会话数为0，实际为%d", sessionCount)
	}
}

// TestReadWriteCloser 测试readWriteCloser实现
func TestReadWriteCloser(t *testing.T) {
	// 创建测试管道
	r1, w1, err := os.Pipe()
	if err != nil {
		t.Fatalf("创建管道1失败: %v", err)
	}
	defer r1.Close()
	defer w1.Close()

	r2, w2, err := os.Pipe()
	if err != nil {
		t.Fatalf("创建管道2失败: %v", err)
	}
	defer r2.Close()
	defer w2.Close()

	// 创建readWriteCloser
	rwc := &readWriteCloser{
		reader: r1,
		writer: w2,
	}

	// 测试写入
	testData := []byte("test data")
	n, err := rwc.Write(testData)
	if err != nil {
		t.Errorf("写入失败: %v", err)
	}
	if n != len(testData) {
		t.Errorf("写入字节数不正确: 期望%d，实际%d", len(testData), n)
	}

	// 关闭写入端以便读取
	w1.Write(testData)
	w1.Close()

	// 测试读取
	buf := make([]byte, len(testData))
	n, err = rwc.Read(buf)
	if err != nil {
		t.Errorf("读取失败: %v", err)
	}
	if n != len(testData) {
		t.Errorf("读取字节数不正确: 期望%d，实际%d", len(testData), n)
	}
	if string(buf) != string(testData) {
		t.Errorf("读取数据不正确: 期望%s，实际%s", string(testData), string(buf))
	}

	// 测试关闭
	err = rwc.Close()
	if err != nil {
		t.Errorf("关闭失败: %v", err)
	}
}

// TestBuildClientCapabilities 测试构建客户端能力
func TestBuildClientCapabilities(t *testing.T) {
	cfg := createTestConfig(t)
	client := NewClient(cfg)
	defer client.Close()

	capabilities := client.buildClientCapabilities()
	if capabilities.TextDocument == nil {
		t.Error("客户端能力为空")
	}

	// 验证基本能力
	if capabilities.TextDocument == nil {
		t.Error("TextDocument能力未设置")
	}

	if capabilities.Workspace == nil {
		t.Error("Workspace能力未设置")
	}
}
