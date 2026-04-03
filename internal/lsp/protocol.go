package lsp

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// MessageID represents a JSON-RPC ID which can be a string, number, or null
type MessageID struct {
	// The actual value of the ID, can be int32, string, or nil
	Value any
}

// MarshalJSON implements custom JSON marshaling for MessageID
// Returns: JSON bytes and error.
func (id *MessageID) MarshalJSON() ([]byte, error) {
	if id == nil || id.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(id.Value)
}

// UnmarshalJSON implements custom JSON unmarshaling for MessageID
// Parameters: data - JSON bytes.
// Returns: error.
func (id *MessageID) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		id.Value = nil
		return nil
	}

	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	// Convert float64 (default JSON number type) to int32 for backward compatibility
	if num, ok := value.(float64); ok {
		id.Value = int32(num)
	} else {
		id.Value = value
	}

	return nil
}

// String returns a string representation of the ID
// Returns: ID in string form.
func (id *MessageID) String() string {
	if id == nil || id.Value == nil {
		return "<null>"
	}

	switch v := id.Value.(type) {
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Message represents a JSON-RPC 2.0 message
type Message struct {
	// JSONRPC version, always "2.0"
	JSONRPC string `json:"jsonrpc"`
	// ID of the message, optional
	ID *MessageID `json:"id,omitempty"`
	// Name of the method
	Method string `json:"method,omitempty"`
	// Parameters of the method
	Params json.RawMessage `json:"params,omitempty"`
	// Result of the method call
	Result json.RawMessage `json:"result,omitempty"`
	// Error information
	Error *ResponseError `json:"error,omitempty"`
}

// ResponseError represents a JSON-RPC 2.0 error
type ResponseError struct {
	// Error code
	Code int `json:"code"`
	// Error message
	Message string `json:"message"`
}
