package lkesdk

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/tencent-lke/lke-sdk-go/event"
	"github.com/tencent-lke/lke-sdk-go/model"
	"github.com/tencent-lke/lke-sdk-go/tool"
)

// LkeClient represents a client for interacting with the LKE service
type LkeClient interface {
	// AddFunctionTools 增加函数 tools
	AddFunctionTools(agentName string, tools []*tool.FunctionTool)

	// AddMcpTools 增加 mcptools
	AddMcpTools(agentName string, mcpClient client.MCPClient, selectedToolNames []string) (
		addTools []*tool.McpTool, err error)

	// AddAgents 添加一批 agents
	AddAgents(agents []model.Agent)

	// AddHandoffs 添加 handoffs
	// 其中 sourceAgentName, targetAgentNames 可以是应用对应的云上 agent，也可以是本地创建的 agent
	AddHandoffs(sourceAgentName string, targetAgentNames []string)

	// Run 执行 agent，query 用户的输入，sesionID 对话唯一标识，options 可选参数，可以为空
	// finalReply
	Run(query, sesionID string,
		options *model.Options) (finalReply *event.ReplyEvent, err error)

	// RunWithContext 执行 agent with context，query 用户的输入
	// sesionID 对话唯一标识，options 可选参数，可以为空
	RunWithContext(ctx context.Context, query, sesionID string,
		options *model.Options) (finalReply *event.ReplyEvent, err error)

	// GetBotAppKey 获取 BotAppKey
	GetBotAppKey() string

	// GetEndpoint returns the endpoint URL
	GetEndpoint() string

	// SetBotAppKey sets the bot application key
	SetBotAppKey(botAppKey string)

	// SetVisitorBizID 设置访问者 id
	SetVisitorBizID(visitorBizID string)

	// SetEndpoint sets the endpoint URL
	SetEndpoint(endpoint string)

	// SetEventHandler 设置时间处理函数
	SetEventHandler(eventHandler EventHandler)

	// SetMock 设置 Mock api 调用
	SetMock(mock bool)

	// SetEnableSystemOpt 是否开启系统优化开关，如果开启，子 agent 在完成任务后默认转回到父 agent
	SetEnableSystemOpt(disable bool)

	// SetStartAgent 设置开始执行的入口 agent
	SetStartAgent(agentName string)

	// SetHttpClient 设置自定义 http client
	SetHttpClient(cli *http.Client)

	// SetHttpClient 设置单轮对话，本地工具调用的最大轮数，不设置默认为 10
	SetMaxToolTurns(maxToolTurns uint)

	// SetHttpClient 设置本地工具调用的超时时间
	SetToolRunTimeout(toolRunTimeout time.Duration)

	// SetApiLogFile 设置 api 调用日志打印文件
	SetApiLogFile(f *os.File)
}

// NewLkeClient creates a new LKE client with the provided parameters,
// botAppKey 知识引擎应用 id,
// visitorBizID 访客的唯一标识,
// eventHandler 自定义事件处理
func NewLkeClient(botAppKey, visitorBizID string, eventHandler EventHandler) LkeClient {
	handler := eventHandler
	if handler == nil {
		handler = &DefaultEventHandler{}
	}
	return &lkeClient{
		botAppKey:    botAppKey,
		visitorBizID: visitorBizID,
		endpoint:     DefaultEndpoint,
		eventHandler: handler,
		toolsMap:     map[string]map[string]tool.Tool{},
		mock:         false,
		httpClient:   http.DefaultClient,
		maxToolTurns: 10,
	}
}
