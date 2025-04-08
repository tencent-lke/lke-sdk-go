package lkesdk

import (
	"github.com/tencent-lke/lke-sdk-go/event"
	"github.com/tencent-lke/lke-sdk-go/tool"
)

// EventHandler 事件处理的接口，用户可以用默认的实现，也可以自定义
type EventHandler interface {

	// OnError 错误处理
	OnError(err *event.ErrorEvent)

	// OnReply 回复处理
	OnReply(reply *event.ReplyEvent)

	// OnThought 思考过程处理
	OnThought(thought *event.AgentThoughtEvent)

	// OnReference 引用事件处理
	OnReference(refer *event.ReferenceEvent)

	// OnTokenStat token 统计事件
	OnTokenStat(stat *event.TokenStatEvent)

	// ToolCallHook 工具调用的后的钩子
	ToolCallHook(tool tool.Tool, output interface{}, err error)
}

// DefaultEventHandler 默认事件处理
type DefaultEventHandler struct {
}

// OnError 错误处理
func (DefaultEventHandler) OnError(err *event.ErrorEvent) {}

// OnReply 回复处理
func (DefaultEventHandler) OnReply(reply *event.ReplyEvent) {}

// OnThought 思考过程处理
func (DefaultEventHandler) OnThought(thought *event.AgentThoughtEvent) {}

// OnReference 引用事件处理
func (DefaultEventHandler) OnReference(refer *event.ReferenceEvent) {}

// OnTokenStattoken 统计事件
func (DefaultEventHandler) OnTokenStat(stat *event.TokenStatEvent) {}

// ToolCallHook 工具调用后钩子
func (DefaultEventHandler) ToolCallHook(tool tool.Tool, output interface{}, err error) {}
