package lkesdk

import (
	"github.com/tencent-lke/lke-sdk-go/event"
	"github.com/tencent-lke/lke-sdk-go/tool"
)

// ToolCallContext 工具调用的上下文
type ToolCallContext struct {
	CallTool tool.Tool
	CallId   string
	Input    map[string]interface{}
	// 如果是自定义的函数，output 类型是自定义函数的返回
	// 如果是 mcp 工具，output 是 *mcp.CallToolResult 类型
	Output interface{}
	Err    error
}

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

	// BeforeToolCallHook 工具调用前的钩子
	BeforeToolCallHook(tollCallCtx ToolCallContext)

	// AfterToolCallHook 工具调用后的钩子
	AfterToolCallHook(tollCallCtx ToolCallContext)
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

// BeforeToolCallHook 工具调用前钩子
func (DefaultEventHandler) BeforeToolCallHook(tollCallCtx ToolCallContext) {}

// AfterToolCallHook 工具调用后的钩子
func (DefaultEventHandler) AfterToolCallHook(tollCallCtx ToolCallContext) {}
