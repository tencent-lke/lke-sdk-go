package lkesdk

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/tencent-lke/lke-sdk-go/event"
	"github.com/tencent-lke/lke-sdk-go/eventhandler"
	"github.com/tencent-lke/lke-sdk-go/mcpserversse"
	"github.com/tencent-lke/lke-sdk-go/model"
	"github.com/tencent-lke/lke-sdk-go/runlog"
	"github.com/tencent-lke/lke-sdk-go/tool"
)

// LkeClient represents a client for interacting with the LKE service
type LkeClient interface {
	// AddFunctionTools 增加函数 tools
	AddFunctionTools(agentName string, tools []*tool.FunctionTool)

	// AddMcpTools 增加 mcptools
	AddMcpTools(agentName string, mcpServerSse *mcpserversse.McpServerSse,
		selectedToolNames []string) (addTools []*tool.McpTool, err error)

	AddAgentAsTool(agentName string, agentastoolName string, toolName string, toolDescription string) error

	// AddAgents 添加一批 agents
	AddAgents(agents []model.Agent)
	// AddHandoffs 添加 handoffs
	// 其中 sourceAgentName, targetAgentNames 可以是应用对应的云上 agent，也可以是本地创建的 agent
	AddHandoffs(sourceAgentName string, targetAgentNames []string)

	// Run 执行 agent，query 用户的输入，sesionID 对话唯一标识，visitorBizID 用户的唯一标识
	// options 可选参数，可以为空。finalReply 最终的回复。
	Run(query, sesionID string,
		options *model.Options) (finalReply *event.ReplyEvent, err error)

	// RunWithContext 执行 agent with context，query 用户的输入
	// sesionID 对话唯一标识，options 可选参数，可以为空
	RunWithContext(ctx context.Context, query, sesionID string,
		options *model.Options) (finalReply *event.ReplyEvent, err error)

	// Close 关闭所有 client 上的任务
	Close()

	// Open 已经 Close 的 client
	Open()

	// GetBotAppKey 获取 BotAppKey
	GetBotAppKey() string

	// GetEndpoint returns the endpoint URL
	GetEndpoint() string

	// SetBotAppKey sets the bot application key
	SetBotAppKey(botAppKey string)

	// SetEndpoint sets the endpoint URL
	SetEndpoint(endpoint string)

	// SetEventHandler 设置时间处理函数
	SetEventHandler(eventHandler eventhandler.EventHandler)

	// SetMock 设置 Mock api 调用
	SetMock(mock bool)

	// SetEnableSystemOpt 是否开启系统优化开关，如果开启，子 agent 在完成任务后默认转回到父 agent
	SetEnableSystemOpt(disable bool)

	// SetStartAgent 设置开始执行的入口 agent
	SetStartAgent(agentName string)

	// SetHttpClient 设置自定义 http client
	SetHttpClient(cli *http.Client)

	// SetMaxToolTurns TODO
	// SetHttpClient 设置单轮对话，本地工具调用的最大轮数，不设置默认为 10
	SetMaxToolTurns(maxToolTurns uint)

	// SetToolRunTimeout TODO
	// SetHttpClient 设置本地工具调用的超时时间
	SetToolRunTimeout(toolRunTimeout time.Duration)

	// SetRunLogger 设置 sdk 执行日志 logger
	SetRunLogger(logger runlog.RunLogger)
}

// NewLkeClient creates a new LKE client with the provided parameters,
// botAppKey 知识引擎应用 id,
// eventHandler 自定义事件处理
// visitorBizID 访客唯一标识
func NewLkeClient(botAppKey string, userID string, eventHandler eventhandler.EventHandler) LkeClient {
	handler := eventHandler
	if handler == nil {
		handler = &eventhandler.DefaultEventHandler{}
	}
	return &lkeClient{
		botAppKey:    botAppKey,
		endpoint:     DefaultEndpoint,
		eventHandler: handler,
		toolsMap:     map[string][]tool.Tool{},
		mock:         false,
		httpClient:   http.DefaultClient,
		maxToolTurns: 10,
		requestID:    uuid.New().String(),
		visitorBizID: userID,
	}
}
