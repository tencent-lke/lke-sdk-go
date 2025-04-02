package lkesdk

import "github.com/tencent-lke/lke-sdk-go/event"

// EventHandler 事件处理的接口，用户可以用默认的实现，也可以自定义
type EventHandler interface {

	// Error 错误处理
	Error(err *event.ErrorEvent)

	// Reply 回复处理
	Reply(reply *event.ReplyEvent)

	// Thought 思考过程处理
	Thought(thought *event.AgentThoughtEvent)

	// Reference 引用事件处理
	Reference(refer *event.ReferenceEvent)

	// TokenStat token 统计事件
	TokenStat(stat *event.TokenStatEvent)
}

// DefaultEventHandler 默认事件处理
type DefaultEventHandler struct {
}

// Error 错误处理
func (DefaultEventHandler) Error(err *event.ErrorEvent) {}

// Reply 回复处理
func (DefaultEventHandler) Reply(reply *event.ReplyEvent) {}

// Thought 思考过程处理
func (DefaultEventHandler) Thought(thought *event.AgentThoughtEvent) {}

// Reference 引用事件处理
func (DefaultEventHandler) Reference(refer *event.ReferenceEvent) {}

// TokenStat token 统计事件
func (DefaultEventHandler) TokenStat(stat *event.TokenStatEvent) {}
