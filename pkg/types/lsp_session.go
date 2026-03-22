// LSP 会话相关类型定义
// Types for LSP session management
// Package types 定义项目中使用的核心数据类型
package types

import (
	"fmt"
	"strings"
	"time"
)

// SessionKey 会话键，用于唯一标识一个LSP会话
type SessionKey struct {
	// LanguageID 编程语言标识符
	LanguageID string `json:"language_id"`
	// RootURI 工作区根目录URI
	RootURI string `json:"root_uri"`
}

// String 返回会话键的字符串表示
func (sk SessionKey) String() string {
	return sk.LanguageID + ":" + sk.RootURI
}

// ParseSessionKey 从字符串解析会话键
func ParseSessionKey(s string) (SessionKey, error) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return SessionKey{}, fmt.Errorf("invalid session key format: %s", s)
	}
	return SessionKey{
		LanguageID: parts[0],
		RootURI:    parts[1],
	}, nil
}

// LSPSession LSP会话信息
type LSPSession struct {
	// Key 会话键
	Key SessionKey `json:"key"`
	// Conn LSP连接
	Conn interface{} `json:"-"`
	// Process LSP服务器进程（如果有）
	Process interface{} `json:"-"`
	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at"`
	// LastUsedAt 最后使用时间
	LastUsedAt time.Time `json:"last_used_at"`
	// IsInitialized 是否已初始化
	IsInitialized bool `json:"is_initialized"`
	// InitializeParams 初始化参数
	InitializeParams *LSPInitializeParams `json:"initialize_params,omitempty"`
	// OpenedDocuments 已打开的文档集合，key为文档URI
	OpenedDocuments map[string]bool `json:"opened_documents,omitempty"`
}

// UpdateLastUsed 更新最后使用时间
func (s *LSPSession) UpdateLastUsed() {
	s.LastUsedAt = time.Now()
}

// IsExpired 检查会话是否已过期
func (s *LSPSession) IsExpired(timeout time.Duration) bool {
	return time.Since(s.LastUsedAt) > timeout
}
