// Types for LSP session management
// Package types defines core data types used by the project
package types

import (
	"fmt"
	"strings"
	"time"
)

// SessionKey uniquely identifies an LSP session
type SessionKey struct {
	// LanguageID programming language identifier
	LanguageID string `json:"language_id"`
	// RootURI workspace root URI
	RootURI string `json:"root_uri"`
}

// String returns the string representation of a session key
func (sk SessionKey) String() string {
	return sk.LanguageID + ":" + sk.RootURI
}

// ParseSessionKey parses a session key from string
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

// LSPSession LSP session info
type LSPSession struct {
	// Key session key
	Key SessionKey `json:"key"`
	// Conn LSP connection
	Conn interface{} `json:"-"`
	// Process LSP server process (if any)
	Process interface{} `json:"-"`
	// CreatedAt creation time
	CreatedAt time.Time `json:"created_at"`
	// LastUsedAt last used time
	LastUsedAt time.Time `json:"last_used_at"`
	// IsInitialized whether initialized
	IsInitialized bool `json:"is_initialized"`
	// InitializeParams initialization params
	InitializeParams *LSPInitializeParams `json:"initialize_params,omitempty"`
	// OpenedDocuments opened document set, keyed by document URI
	OpenedDocuments map[string]bool `json:"opened_documents,omitempty"`
}

// UpdateLastUsed updates the last used time
func (s *LSPSession) UpdateLastUsed() {
	s.LastUsedAt = time.Now()
}

// IsExpired checks whether the session has expired
func (s *LSPSession) IsExpired(timeout time.Duration) bool {
	return time.Since(s.LastUsedAt) > timeout
}
