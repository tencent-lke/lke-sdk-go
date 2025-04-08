package event

import (
	"bytes"
	"time"

	jsoniter "github.com/json-iterator/go"
)

// EventThought Agent 思考事件
const EventThought = "thought"

// AgentThoughtEvent Agent思考事件
type AgentThoughtEvent struct {
	SessionID string `json:"session_id"` // 会话 ID
	RequestID string `json:"request_id"` // 请求 ID
	TraceId   string `json:"trace_id"`
	RecordID  string `json:"record_id"` // 对应哪条会话, 会话 ID, 用于回答的消息存储使用, 可提前生成, 保存消息时使用

	Elapsed      uint32           `json:"elapsed"`       // 当前请求执行时间, 单位 ms
	IsWorkflow   bool             `json:"is_workflow"`   // 是否是工作流
	WorkflowName string           `json:"workflow_name"` // 工作流名称
	Procedures   []AgentProcedure `json:"procedures"`    // 过程列表, 从详细中去重获得

	StartTime      time.Time `json:"-"` // 开始时间, 用于记录总耗时
	EventSource    string    `json:"-"` // 本次 token 统计的 event 来源
	IsEvaluateTest bool      `json:"-"` // 是否应用评测
}

// AgentProcedure 执行过程
type AgentProcedure struct {
	Index        uint32                  `json:"index"`         // 过程索引
	Name         string                  `json:"name"`          // 英文名, 参考本文件常量定义
	Title        string                  `json:"title"`         // 中文名, 用于展示
	Status       string                  `json:"status"`        // 状态, 参考常量 ProcedureStatus* (使用中, 成功, 失败)
	Icon         string                  `json:"icon"`          // 图标, 用于展示
	Debugging    AgentProcedureDebugging `json:"debugging"`     // 调试信息
	Switch       string                  `json:"switch"`        // 是否切换Agent，取值workflow  or main
	WorkflowName string                  `json:"workflow_name"` // 工作流名称
	NodeName     string                  `json:"node_name"`     // 工作流节点名称
	ReplyIndex   uint32                  `json:"reply_index"`   // 回复索引，用于标记思考过程放在哪个回复气泡中
	PluginType   int32                   `json:"plugin_type"`   // 插件类型 0: 自定义插件；1: 官方插件；2: 工作流
	Elapsed      uint32                  `json:"elapsed"`       // 当前请求执行时间, 单位 ms
}

// AgentProcedureDebugging 调试信息
type AgentProcedureDebugging struct {
	Content        string      `json:"content,omitempty"`
	DisplayType    uint32      `json:"display_type,omitempty"`
	DisplayThought string      `json:"display_thought,omitempty"`
	DisplayContent string      `json:"display_content,omitempty"`
	References     []Reference `json:"references,omitempty"`
	QuoteInfos     []QuoteInfo `json:"quote_infos,omitempty"`
}

// QuoteInfo 引用信息
type QuoteInfo struct {
	Position int `json:"position"`
	Index    int `json:"index"`
}

// Name 事件名称
func (a AgentThoughtEvent) Name() string {
	return EventThought
}

// IsValid 判断事件是否合法
func (a AgentThoughtEvent) IsValid() bool {
	return true
}

// String 字符串化
func (a AgentThoughtEvent) String() string {
	buff := &bytes.Buffer{}
	enc := jsoniter.NewEncoder(buff)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(a)
	return buff.String()
}
