// Package event contains event definitions.
package event

import "encoding/json"

// Event 事件
type Event interface {
	Name() string
}

type EventExtend struct {
	Extend map[string]string `json:"extend,omitempty"`
}

// EventWrapper 事件 Wrapper
type EventWrapper struct {
	Type      string          `json:"type,omitempty"`
	Payload   json.RawMessage `json:"payload"`
	MessageID string          `json:"message_id,omitempty"`
}
