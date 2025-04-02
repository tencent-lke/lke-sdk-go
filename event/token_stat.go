package event

import (
	"time"
)

// EventTokenStat token 统计事件
const EventTokenStat = "token_stat"

// ProcedureStatus 过程状态定义
type ProcedureStatus string

const (
	// ProcedureStatusProcessing 使用中
	ProcedureStatusProcessing ProcedureStatus = "processing" // 使用中
	// ProcedureStatusSuccess 成功
	ProcedureStatusSuccess ProcedureStatus = "success" // 成功
	// ProcedureStatusFailed 失败
	ProcedureStatusFailed ProcedureStatus = "failed" // 失败
	// ProcedureStatusStop 停止
	ProcedureStatusStop ProcedureStatus = "stop" // 停止
)

// 过程定义, 过程名字, 中文(title)存放到配置文件
const (
	ProcedureKnowledge     = "knowledge"            // 调用知识库
	ProcedureTaskFlow      = "task_flow"            // 调用任务流程
	ProcedureSE            = "search_engine"        // 调用搜索引擎
	ProcedureImage         = "image"                // 调用图片理解
	ProcedureLLM           = "large_language_model" // 大模型回复
	ProcedurePOTMath       = "pot_math"             // 调用计算器
	ProcedureFile          = "file"                 // 阅读文件
	ProcedureWorkflow      = "workflow"             // 工作流
	ProcedureAgent         = "agent"                // 智能体
	ProcedureLLMGen        = "model_generate"       // 大模型生成中
	ProcedureThinkingModel = "thinking_model"       // 调用思考模型
	ProcedureAgentTool     = "tool_call"            // 调用插件工具
)

const (
	// ResourceStatusAvailable 计费资源可用
	ResourceStatusAvailable = uint32(1)
	// ResourceStatusUnAvailable 计费资源不可用
	ResourceStatusUnAvailable = uint32(2)
)

// TokenStatEvent token 统计事件
type TokenStatEvent struct {
	SessionID string `json:"session_id"` // 会话 ID
	RequestID string `json:"request_id"` // 请求 ID
	RecordID  string `json:"record_id"`  // 对应哪条会话, 会话 ID, 用于回答的消息存储使用, 可提前生成, 保存消息时使用

	MainModelName string `json:"-"` // 主模型名
	// BalanceType string `json:"-"`           // 余额状态, 体验: experience; 云计费: cloud
	UsedCount  uint32 `json:"used_count"`  // token 已使用数
	FreeCount  uint32 `json:"free_count"`  // 免费 token 数
	OrderCount uint32 `json:"order_count"` // 订单总 token 数

	StatusSummary      ProcedureStatus `json:"status_summary"`       // 当前执行状态汇总, 参考常量 ProcedureStatus* (使用中, 成功, 失败)
	StatusSummaryTitle string          `json:"status_summary_title"` // 当前执行状态汇总后中文展示
	Elapsed            uint32          `json:"elapsed"`              // 当前请求执行时间, 单位 ms
	TokenCount         uint32          `json:"token_count"`          // 当前请求消耗 token 数

	// ProceduresDetail []Procedure `json:"procedures_detail"` // 过程列表详细, 支持多个相同过程
	Procedures []Procedure `json:"procedures"` // 过程列表, 从详细中去重获得

	StartTime         time.Time `json:"-"` // 开始时间, 用于记录总耗时
	EventSource       string    `json:"-"` // 本次 token 统计的 event 来源
	FinanceSubBizType string    `json:"-"` // 计费子类型
}

// Procedure 执行过程
type Procedure struct {
	Name   string          `json:"name"`   // 英文名, 参考本文件常量定义
	Title  string          `json:"title"`  // 中文名, 用于展示
	Status ProcedureStatus `json:"status"` // 状态, 参考常量 ProcedureStatus* (使用中, 成功, 失败)

	InputCount        uint32        `json:"input_count"`                   // 输入消耗 token 数
	OutputCount       uint32        `json:"output_count"`                  // 输出消耗 token 数
	Count             uint32        `json:"count"`                         // 消耗 token 数
	TokenUsageDetails []*TokenUsage `json:"token_usage_details,omitempty"` // token 用量

	Debugging ProcedureDebugging `json:"debugging"` // 调试信息

	ResourceStatus uint32 `json:"resource_status"` // 计费资源状态，1：可用，2：不可用
}

// TokenUsage Token用量
type TokenUsage struct {
	ModelName    string `json:"model_name"`    // 模型名
	InputTokens  uint32 `json:"input_tokens"`  // 输入 token 数
	OutputTokens uint32 `json:"output_tokens"` // 输出 token 数
	TotalTokens  uint32 `json:"total_tokens"`  // 总 token 数
}

// ProcedureDebugging 调试信息
type ProcedureDebugging struct {
	Content      string             `json:"content,omitempty"`
	System       string             `json:"system,omitempty"`
	Histories    []HistorySummary   `json:"histories,omitempty"`     // 多轮历史信息
	Knowledge    []KnowledgeSummary `json:"knowledge,omitempty"`     // 检索知识
	TaskFlow     TaskFlowSummary    `json:"task_flow,omitempty"`     // 任务流程
	Workflow     WorkflowSummary    `json:"work_flow,omitempty"`     // 工作流
	Agent        AgentDebugInfo     `json:"agent,omitempty"`         // 智能体
	RewriteQuery string             `json:"rewrite_query,omitempty"` // 改写后query
}

// HistorySummary 多轮历史信息
type HistorySummary struct {
	User      string `json:"user,omitempty"`
	Assistant string `json:"assistant,omitempty"`
}

// KnowledgeSummary 知识片段信息
type KnowledgeSummary struct {
	Type    uint32 `json:"type"` // 1是QA 2是segment
	Content string `json:"content,omitempty"`
}

// 值的类型
type ValueType int32

type ValueInfo struct {
	ID            string    `protobuf:"bytes,1,opt,name=ID,proto3" json:"ID,omitempty"`
	Name          string    `protobuf:"bytes,2,opt,name=Name,proto3" json:"Name,omitempty"`
	ValueType     ValueType `protobuf:"varint,3,opt,name=ValueType,proto3,enum=trpc.KEP.bot_dm_server.ValueType" json:"ValueType,omitempty"`
	ValueStr      string    `protobuf:"bytes,4,opt,name=ValueStr,proto3" json:"ValueStr,omitempty"`
	ValueInt      int64     `protobuf:"varint,5,opt,name=ValueInt,proto3" json:"ValueInt,omitempty"`
	ValueFloat    float32   `protobuf:"fixed32,6,opt,name=ValueFloat,proto3" json:"ValueFloat,omitempty"`
	ValueBool     bool      `protobuf:"varint,7,opt,name=ValueBool,proto3" json:"ValueBool,omitempty"`
	ValueStrArray []string  `protobuf:"bytes,8,rep,name=ValueStrArray,proto3" json:"ValueStrArray,omitempty"`
}

// 节点类型
type FlowNodeType int32

type StrValue struct {
	Name  string `protobuf:"bytes,1,opt,name=Name,proto3" json:"Name,omitempty"`
	Value string `protobuf:"bytes,2,opt,name=Value,proto3" json:"Value,omitempty"`
}

type InvokeAPI struct {
	Method          string       `protobuf:"bytes,1,opt,name=Method,proto3" json:"Method,omitempty"`                   // 请求方法，如GET/POST等
	URL             string       `protobuf:"bytes,2,opt,name=URL,proto3" json:"URL,omitempty"`                         // 请求地址。
	HeaderValues    []*StrValue  `protobuf:"bytes,3,rep,name=HeaderValues,proto3" json:"HeaderValues,omitempty"`       // header参数
	QueryValues     []*StrValue  `protobuf:"bytes,4,rep,name=QueryValues,proto3" json:"QueryValues,omitempty"`         // 入参Query
	RequestPostBody string       `protobuf:"bytes,5,opt,name=RequestPostBody,proto3" json:"RequestPostBody,omitempty"` // Post请求的原始数据
	ResponseBody    string       `protobuf:"bytes,6,opt,name=ResponseBody,proto3" json:"ResponseBody,omitempty"`       // 返回的原始数据
	ResponseValues  []*ValueInfo `protobuf:"bytes,7,rep,name=ResponseValues,proto3" json:"ResponseValues,omitempty"`   // 出参
	FailMessage     string       `protobuf:"bytes,8,opt,name=FailMessage,proto3" json:"FailMessage,omitempty"`         // 异常信息
}

type RunNodeInfo struct {
	NodeType   FlowNodeType `protobuf:"varint,1,opt,name=NodeType,proto3,enum=trpc.KEP.bot_dm_server.FlowNodeType" json:"NodeType,omitempty"` // 节点类型
	NodeID     string       `protobuf:"bytes,2,opt,name=NodeID,proto3" json:"NodeID,omitempty"`                                               // 节点ID
	NodeName   string       `protobuf:"bytes,3,opt,name=NodeName,proto3" json:"NodeName,omitempty"`                                           // 节点名称
	InvokeAPI  *InvokeAPI   `protobuf:"bytes,4,opt,name=InvokeAPI,proto3" json:"InvokeAPI,omitempty"`                                         // 请求的API
	SlotValues []*ValueInfo `protobuf:"bytes,5,rep,name=SlotValues,proto3" json:"SlotValues,omitempty"`                                       // 当前节点的所有槽位的值，key：SlotID。没有值的时候也要返回空。
}

// TaskFlowSummary 任务流程信息
type TaskFlowSummary struct {
	IntentName        string         `json:"intent_name,omitempty"`         // 任务流程名
	UpdatedSlotValues []*ValueInfo   `json:"updated_slot_values,omitempty"` // 实体列表
	Purposes          []string       `json:"purposes,omitempty"`            // 意图判断                   //
	RunNodes          []*RunNodeInfo `json:"run_nodes,omitempty"`           // 节点列表
}

// WorkflowSummary 工作流信息
type WorkflowSummary struct {
	WorkflowID      string          `json:"workflow_id,omitempty"`       // 工作流ID
	WorkflowName    string          `json:"workflow_name,omitempty"`     // 工作流名称
	RunNodes        []*RunNodeInfo  `json:"run_nodes,omitempty"`         // 节点列表
	WorkflowRunID   string          `json:"workflow_run_id,omitempty"`   // 工作流运行ID
	OptionCardIndex OptionCardIndex `json:"option_card_index,omitempty"` // 选项卡索引
}

// OptionCardIndex 选项卡索引
type OptionCardIndex struct {
	RecordID string `json:"record_id,omitempty"`
	Index    int32  `json:"index,omitempty"` // 选项卡索引, 从1开始
}

// AgentDebugInfo 智能体调试信息
type AgentDebugInfo struct {
	Input  string `json:"input,omitempty"`  // 输入
	Output string `json:"output,omitempty"` // 输出
}

// ProcedureOption Procedure参数
type ProcedureOption func(p *Procedure)

// Name 事件名称
func (e *TokenStatEvent) Name() string {
	return EventTokenStat
}

// IsValid 判断事件是否合法
func (e *TokenStatEvent) IsValid() bool {
	return true
}
