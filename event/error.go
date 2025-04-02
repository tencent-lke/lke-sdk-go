// Package event 事件
package event

import (
	"context"
	"encoding/json"
	"time"

	jsoniter "github.com/json-iterator/go"
	"go.opentelemetry.io/otel/trace"

	"git.code.oa.com/trpc-go/trpc-go/log"
	"git.woa.com/dialogue-platform/lke_proto/pb-protocol/KEP_DM"
	"git.woa.com/dialogue-platform/lke_proto/pb-protocol/KEP_WF_DM"
	llmm "git.woa.com/dialogue-platform/proto/pb-stub/llm-manager-server"

	"git.woa.com/ivy/qbot/qbot/chat/internal/config"
	"git.woa.com/ivy/qbot/qbot/chat/internal/model"
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

	LLMMStatisticInfo *llmm.StatisticInfo `json:"-"` // 大模型返回的 token 信息

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

// TaskFlowSummary 任务流程信息
type TaskFlowSummary struct {
	IntentName        string                `json:"intent_name,omitempty"`         // 任务流程名
	UpdatedSlotValues []*KEP_DM.ValueInfo   `json:"updated_slot_values,omitempty"` // 实体列表
	Purposes          []string              `json:"purposes,omitempty"`            // 意图判断                   //
	RunNodes          []*KEP_DM.RunNodeInfo `json:"run_nodes,omitempty"`           // 节点列表
}

// WorkflowSummary 工作流信息
type WorkflowSummary struct {
	WorkflowID      string                   `json:"workflow_id,omitempty"`       // 工作流ID
	WorkflowName    string                   `json:"workflow_name,omitempty"`     // 工作流名称
	RunNodes        []*KEP_WF_DM.RunNodeInfo `json:"run_nodes,omitempty"`         // 节点列表
	WorkflowRunID   string                   `json:"workflow_run_id,omitempty"`   // 工作流运行ID
	OptionCardIndex OptionCardIndex          `json:"option_card_index,omitempty"` // 选项卡索引
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

// UpdateSuccessProcedure 过程更新并上报到统计
func (e *TokenStatEvent) UpdateSuccessProcedure(p Procedure) {
	e.UpdateProcedure(p)
}

// UpdateProcedure 过程更新
func (e *TokenStatEvent) UpdateProcedure(p Procedure) {
	if e == nil {
		return
	}
	if len(p.Name) == 0 {
		return
	}

	var hasProcedure bool // 使用名字标识是否为同一个过程
	for i := range e.Procedures {
		if e.Procedures[i].Name == p.Name { // 存在, 则更新状态, 补充 count
			e.Procedures[i].Status = p.Status
			e.Procedures[i].ResourceStatus = p.ResourceStatus
			if p.LLMMStatisticInfo != nil {
				s0 := e.Procedures[i].LLMMStatisticInfo
				if s0 == nil {
					s0 = p.LLMMStatisticInfo
				} else {
					s0.InputTokens += p.LLMMStatisticInfo.GetInputTokens()
					s0.OutputTokens += p.LLMMStatisticInfo.GetOutputTokens()
					s0.TotalTokens += p.LLMMStatisticInfo.GetTotalTokens()
				}
				e.Procedures[i].InputCount += p.LLMMStatisticInfo.GetInputTokens()
				e.Procedures[i].OutputCount += p.LLMMStatisticInfo.GetOutputTokens()
				e.Procedures[i].Count += p.LLMMStatisticInfo.GetTotalTokens()
				e.Procedures[i].LLMMStatisticInfo = s0
			}
			e.Procedures[i].Debugging = p.Debugging
			e.Procedures[i].TokenUsageDetails = p.TokenUsageDetails
			hasProcedure = true
			break
		}
	}
	if !hasProcedure {
		e.Procedures = append(e.Procedures, p)
	}

	if len(e.Procedures) == 0 {
		return
	}
	// 只要有一个失败, 就认为整体失败
	p0 := e.Procedures[len(e.Procedures)-1] // 取最后一个元素
	// 检查过程中, 是否有失败, 如果有, 则用失败的
	for i := 0; i < len(e.Procedures); i++ {
		if e.Procedures[i].Status == ProcedureStatusFailed {
			p0 = e.Procedures[i]
			break
		}
	}
	// 资源不可用产生的失败，只记录失败过程，不计入整个运行周期的失败
	if p0.Status == ProcedureStatusFailed && p0.ResourceStatus == ResourceStatusUnAvailable {
		p0.Status = ProcedureStatusSuccess
	}
	e.StatusSummary = p0.Status
	if p0.Status == ProcedureStatusFailed {
		e.StatusSummaryTitle = p0.Title + "失败"
	} else {
		e.StatusSummaryTitle = p0.Title
	}
	if p.LLMMStatisticInfo != nil {
		e.TokenCount += p.LLMMStatisticInfo.GetTotalTokens() // 总数更新
	}
	e.Elapsed = uint32(time.Since(e.StartTime).Milliseconds())
}

// func (e *TokenStatEvent) hasProcedure(name string) (Procedure, bool) {
//	for i := range e.Procedures {
//		if e.Procedures[i].Name == name {
//			return e.Procedures[i], true
//		}
//	}
//	return Procedure{}, false
// }
//
// func (e *TokenStatEvent) replaceProcedure(p Procedure) {
//	for i := range e.Procedures {
//		if e.Procedures[i].Name == p.Name { // 存在, 则更新状态, 补充 count
//			e.Procedures[i] = p
//			break
//		}
//	}
// }

// NewProcessingTSProcedure 创建进行中过程
func NewProcessingTSProcedure(name string) Procedure {
	return Procedure{
		Name:   name,
		Title:  config.App().Procedure[name], // 名字使用配置文件的配置
		Status: ProcedureStatusProcessing,
	}
}

// NewSuccessTSProcedure 创建成功过程
func NewSuccessTSProcedure(name string, llmmStat *llmm.StatisticInfo,
	debugging ProcedureDebugging, usage []*TokenUsage) Procedure {
	return Procedure{
		Name:              name,
		Title:             config.App().Procedure[name], // 名字使用配置文件的配置
		Status:            ProcedureStatusSuccess,
		LLMMStatisticInfo: llmmStat,
		InputCount:        llmmStat.GetInputTokens(),
		OutputCount:       llmmStat.GetOutputTokens(),
		Count:             llmmStat.GetTotalTokens(),
		TokenUsageDetails: usage,
		Debugging:         debugging,
	}
}

// NewProcessingTSProcedureWithDebug 创建进行中过程
func NewProcessingTSProcedureWithDebug(name string, llmmStat *llmm.StatisticInfo,
	debugging ProcedureDebugging, usage []*TokenUsage) Procedure {
	return Procedure{
		Name:              name,
		Title:             config.App().Procedure[name], // 名字使用配置文件的配置
		Status:            ProcedureStatusProcessing,
		LLMMStatisticInfo: llmmStat,
		InputCount:        llmmStat.GetInputTokens(),
		OutputCount:       llmmStat.GetOutputTokens(),
		Count:             llmmStat.GetTotalTokens(),
		TokenUsageDetails: usage,
		Debugging:         debugging,
	}
}

// NewFailedTSProcedure 创建失败过程
func NewFailedTSProcedure(name string, options ...ProcedureOption) Procedure {
	p := Procedure{
		Name:   name,
		Title:  config.App().Procedure[name], // 名字使用配置文件的配置
		Status: ProcedureStatusFailed,
	}
	for _, option := range options {
		option(&p)
	}
	return p
}

// GetLLMMStatsJSON 获取大模型回来的 token 统计数据
func (e *TokenStatEvent) GetLLMMStatsJSON() string {
	var s0 []*llmm.StatisticInfo
	for i := 0; i < len(e.Procedures); i++ {
		if e.Procedures[i].LLMMStatisticInfo == nil {
			continue
		}
		s0 = append(s0, e.Procedures[i].LLMMStatisticInfo)
	}
	if len(s0) == 0 {
		return ""
	}
	d, err := json.Marshal(s0)
	if err != nil {
		return ""
	}
	return string(d)
}

// WithResourceStatusUnAvailable 设置过程的资源状态不可用
func WithResourceStatusUnAvailable() ProcedureOption {
	return func(p *Procedure) {
		p.ResourceStatus = ResourceStatusUnAvailable
	}
}

// WithProcedureDebugging 设置过程的debug信息
func WithProcedureDebugging(debugging ProcedureDebugging) ProcedureOption {
	return func(p *Procedure) {
		p.Debugging = debugging
	}
}

// ConvertToMsgRecordTokenStat 转换为消息记录统计信息
func (e *TokenStatEvent) ConvertToMsgRecordTokenStat(traceID string) *model.MsgRecordTokenStat {
	var procedures string
	if len(e.Procedures) > 0 {
		procedures, _ = jsoniter.MarshalToString(e.Procedures)
	}
	return &model.MsgRecordTokenStat{
		RecordID:           e.RecordID,
		TraceID:            traceID,
		UsedCount:          e.UsedCount,
		FreeCount:          e.FreeCount,
		OrderCount:         e.OrderCount,
		StatusSummary:      string(e.StatusSummary),
		StatusSummaryTitle: e.StatusSummaryTitle,
		Elapsed:            e.Elapsed,
		TokenCount:         e.TokenCount,
		Procedures:         procedures,
		IsDeleted:          false,
		CreateTime:         time.Now(),
		UpdateTime:         time.Now(),
	}
}

// GetMsgRecordAndTokenStat 转换并获取消息记录统计信息
func GetMsgRecordAndTokenStat(ctx context.Context, record model.MsgRecord) (model.MsgRecord,
	*model.MsgRecordTokenStat) {
	if len(record.TokenStat) <= 0 {
		return record, nil
	}

	traceID := trace.SpanContextFromContext(ctx).TraceID().String()

	// 避免调试信息数据超出text类型
	// 剥离调试信息存储到调试信息表

	var tokenStat TokenStatEvent
	err := jsoniter.UnmarshalFromString(record.TokenStat, &tokenStat)
	if err != nil {
		log.Warnf("ConvertMsgRecordTokenStat|procedures.json.Unmarshal %s", err.Error())
		return record, nil
	}
	stat := tokenStat.ConvertToMsgRecordTokenStat(traceID)

	recordWithoutStatProcedures := record
	tokenStat.Procedures = []Procedure{}
	st, _ := jsoniter.MarshalToString(tokenStat)
	recordWithoutStatProcedures.TokenStat = st

	return recordWithoutStatProcedures, stat
}
