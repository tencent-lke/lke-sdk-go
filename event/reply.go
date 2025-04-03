package event

import (
	openai "github.com/openai/openai-go"
)

// ReplyMethod 回复方式
type ReplyMethod uint8

// 回复方式
const (
	ReplyMethodModel          ReplyMethod = 1  // 大模型直接回复
	ReplyMethodBare           ReplyMethod = 2  // 保守回复, 未知问题回复
	ReplyMethodRejected       ReplyMethod = 3  // 拒答问题回复
	ReplyMethodEvil           ReplyMethod = 4  // 敏感回复
	ReplyMethodPriorityQA     ReplyMethod = 5  // 问答对直接回复, 已采纳问答对优先回复
	ReplyMethodGreeting       ReplyMethod = 6  // 欢迎语回复
	ReplyMethodBusy           ReplyMethod = 7  // 并发超限回复
	ReplyGlobalKnowledge      ReplyMethod = 8  // 全局干预知识
	ReplyMethodTaskFlow       ReplyMethod = 9  // 任务流程过程回复, 当历史记录中 task_flow.type = 0 时, 为大模型回复
	ReplyMethodTaskAnswer     ReplyMethod = 10 // 任务流程答案回复
	ReplyMethodSearch         ReplyMethod = 11 // 搜索引擎回复
	ReplyMethodDecorator      ReplyMethod = 12 // 知识润色后回复
	ReplyMethodImage          ReplyMethod = 13 // 图片理解回复
	ReplyMethodFile           ReplyMethod = 14 // 实时文档回复
	ReplyMethodClarifyConfirm ReplyMethod = 15 // 澄清确认回复
	ReplyMethodWorkflow       ReplyMethod = 16 // 工作流回复
	ReplyMethodWorkflowAnswer ReplyMethod = 17 // 工作流运行结束
	ReplyMethodAgent          ReplyMethod = 18 // 智能体回复
	ReplyMethodMultiIntent    ReplyMethod = 19 // 多意图回复
	ReplyMethodInterrupt      ReplyMethod = 20 // 中断回复
)

// EventReply 回复/确认事件
const EventReply = "reply"

// ReplyEvent 回复/确认事件消息体
type ReplyEvent struct {
	RequestID       string           `json:"request_id"`
	SessionID       string           `json:"session_id"`
	Content         string           `json:"content"`
	FromName        string           `json:"from_name"`
	FromAvatar      string           `json:"from_avatar"`
	RecordID        string           `json:"record_id"`
	RelatedRecordID string           `json:"related_record_id"`
	Timestamp       int64            `json:"timestamp"`
	IsFinal         bool             `json:"is_final"`
	IsFromSelf      bool             `json:"is_from_self"`
	CanRating       bool             `json:"can_rating"`
	CanFeedback     bool             `json:"can_feedback"`
	IsEvil          bool             `json:"is_evil"`
	IsLLMGenerated  bool             `json:"is_llm_generated"`
	Knowledge       []ReplyKnowledge `json:"knowledge"`
	ReplyMethod     ReplyMethod      `json:"reply_method"`
	IntentCategory  string           `json:"intent_category"`
	OptionCards     []string         `json:"option_cards"`            // 选项卡, 用于多轮对话,如果没有选项卡 为[]
	Tags            []*ReplyTag      `json:"tags,omitempty"`          // 命中标签列表
	CustomParams    []string         `json:"custom_params,omitempty"` // 自定义参数, 用户透传用户自定义参数，如果没有自定义参数 为[]
	InterruptInfo   *InterruptInfo   `json:"interrupt_info"`          // 中断信息，意味着必须要端上配合 chat 才能继续执行
}

type InterruptInfo struct {
	CurrentAgent string                       `json:"current_agent"` // 当前 agent 的 name
	ToolCalls    []*openai.ToolCallDeltaUnion `json:"tool_calls"`    // 需要本地调用的工具
}

// Name 事件名称
func (e ReplyEvent) Name() string {
	return EventReply
}

// ReplyKnowledge 回复事件中的知识
type ReplyKnowledge struct {
	ID   string `json:"id"`
	Type uint32 `json:"type"`
}

// ReplyTag 回复事件中的标签
type ReplyTag struct {
	Name       string   `json:"name"`        // 标签名称
	ValueRange []string `json:"value_range"` // 命中标签的范围
}
