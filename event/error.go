// Package event 事件
package event

// EventError 错误事件
const EventError = "error"

// ErrorEvent 错误事件消息体
type ErrorEvent struct {
	Error     Error       `json:"error"`
	RequestID string      `json:"request_id"`
	TraceId   string      `json:"trace_id"`
	Extend    EventExtend `json:"extend,omitempty"`
}

// Name 事件名称
func (e ErrorEvent) Name() string {
	return EventError
}

// Error 错误
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
