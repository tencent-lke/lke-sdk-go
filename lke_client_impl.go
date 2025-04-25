package lkesdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/openai/openai-go"
	"github.com/tencent-lke/lke-sdk-go/event"
	"github.com/tencent-lke/lke-sdk-go/model"
	"github.com/tencent-lke/lke-sdk-go/tool"
	sse "github.com/tmaxmax/go-sse"
)

const (
	DefaultEndpoint = "https://wss.lke.cloud.tencent.com/v1/qbot/chat/sse"
)

// lkeClient represents a client for interacting with the LKE service
type lkeClient struct {
	botAppKey    string // 机器人密钥 (从运营接口人处获取)
	endpoint     string // 调用地址
	eventHandler EventHandler
	mock         bool
	httpClient   *http.Client

	toolsMap        map[string]map[string]tool.Tool // agentName -> map[toolname -> FunctionTool] 的映射
	agents          []model.Agent
	handoffs        []model.Handoff
	enableSystemOpt bool
	startAgent      string
	logger          RunLogger
	toolRunTimeout  time.Duration
	maxToolTurns    uint // 单次对话本地工具调用最大次数
	closed          atomic.Bool
}

// GetBotAppKey 获取 BotAppKey
func (c *lkeClient) GetBotAppKey() string {
	return c.botAppKey
}

// SetBotAppKey sets the bot application key
func (c *lkeClient) SetBotAppKey(botAppKey string) {
	c.botAppKey = botAppKey
}

// GetEndpoint returns the endpoint URL
func (c *lkeClient) GetEndpoint() string {
	return c.endpoint
}

// SetEndpoint sets the endpoint URL
func (c *lkeClient) SetEndpoint(endpoint string) {
	c.endpoint = endpoint
}

// SetEventHandler 设置时间处理函数
func (c *lkeClient) SetEventHandler(eventHandler EventHandler) {
	c.eventHandler = eventHandler
}

// SetMock 设置 Mock
func (c *lkeClient) SetMock(mock bool) {
	c.mock = mock
}

// DisableSystemOpt 配置 agent 运行时的系统优化开关
func (c *lkeClient) SetEnableSystemOpt(enable bool) {
	c.enableSystemOpt = enable
}

// SetStartAgent 设置开始执行的入口 agent
func (c *lkeClient) SetStartAgent(agentName string) {
	c.startAgent = agentName
}

// SetHttpClient 设置自定义 http client
func (c *lkeClient) SetHttpClient(cli *http.Client) {
	if cli != nil {
		c.httpClient = cli
	}
}

// SetHttpClient 设置单轮对话，本地工具调用的最大轮数，不设置默认为 10
func (c *lkeClient) SetMaxToolTurns(maxToolTurns uint) {
	c.maxToolTurns = maxToolTurns
}

// SetHttpClient 设置本地工具调用的超时时间
func (c *lkeClient) SetToolRunTimeout(toolRunTimeout time.Duration) {
	c.toolRunTimeout = toolRunTimeout
}

// SetRunLogger 设置 sdk 执行日志 logger
func (c *lkeClient) SetRunLogger(logger RunLogger) {
	c.logger = logger
}

// AddFunctionTools 增加函数 tools
func (c *lkeClient) AddFunctionTools(agentName string, tools []*tool.FunctionTool) {
	if len(tools) == 0 {
		return
	}

	toolFuncMap, ok := c.toolsMap[agentName]
	if !ok {
		toolFuncMap = map[string]tool.Tool{}
		c.toolsMap[agentName] = toolFuncMap
	}
	for _, tool := range tools {
		if tool != nil {
			toolFuncMap[tool.GetName()] = tool
		}
	}
}

// AddMcpTools 增加 mcptools
func (c *lkeClient) AddMcpTools(agentName string, mcpClient client.MCPClient,
	impl mcp.Implementation, selectedToolNames []string) (
	addTools []*tool.McpTool, err error) {
	// Initialize the client
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = impl
	_, err = mcpClient.Initialize(context.Background(), initRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize: %v", err)
	}
	tools, err := tool.ListMcpTools(mcpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %v", err)
	}
	selectMap := map[string]struct{}{}
	for _, t := range selectedToolNames {
		selectMap[t] = struct{}{}
	}
	toolMcpMap, ok := c.toolsMap[agentName]
	if !ok {
		toolMcpMap = map[string]tool.Tool{}
		c.toolsMap[agentName] = toolMcpMap
	}
	for _, t := range tools.Tools {
		add := true
		if len(selectedToolNames) > 0 {
			if _, ok := selectMap[t.Name]; !ok {
				add = false
			}
		}
		if add {
			newtool := &tool.McpTool{
				Name:        t.Name,
				Description: t.Description,
				Cli:         mcpClient,
			}
			bs, _ := json.Marshal(t.InputSchema)
			_ = json.Unmarshal(bs, &newtool.Schame)
			toolMcpMap[t.Name] = newtool
			addTools = append(addTools, newtool)
		}
	}
	return addTools, err
}

// AddAgents 添加一批 agent
func (c *lkeClient) AddAgents(agents []model.Agent) {
	c.agents = append(c.agents, agents...)
}

// AddHandoffs 添加 handoffs
// 其中 sourceAgentName, targetAgentNames 可以是应用对应的云上 agent，也可以是本地创建的 agent
func (c *lkeClient) AddHandoffs(sourceAgentName string, targetAgentNames []string) {
	for _, target := range targetAgentNames {
		c.handoffs = append(c.handoffs, model.Handoff{
			SourceAgentName: sourceAgentName,
			TargetAgentName: target,
		})
	}
}

func (c *lkeClient) buildReq(query, sessionID, visitorBizID string,
	options *model.Options) *model.ChatRequest {
	req := &model.ChatRequest{
		Content:      query,
		VisitorBizID: visitorBizID,
		BotAppKey:    c.botAppKey,
		SessionID:    sessionID,
	}
	if options != nil {
		req.Options = *options
	}
	// 构建 agent 参数
	req.AgentConfig.Agents = c.agents
	// 构建 handoff 参数
	req.AgentConfig.Handoffs = c.handoffs
	req.AgentConfig.DisableSystemOpt = !c.enableSystemOpt
	req.AgentConfig.StartAgentName = c.startAgent
	// 构建工具参数
	for agentName, toolFuncMap := range c.toolsMap {
		if len(toolFuncMap) > 0 {
			agentTool := model.AgentTool{
				AgentName: agentName,
			}
			for _, t := range toolFuncMap {
				agentTool.Tools = append(agentTool.Tools, tool.ToOpenAIToolPB(t))
			}
			req.AgentConfig.AgentTools = append(req.AgentConfig.AgentTools, agentTool)
		}
	}
	return req
}

func (c *lkeClient) handlerEvent(data []byte) (finalReply *event.ReplyEvent, err error) {
	defer func() {
		if p := recover(); p != nil {
		}
	}()
	ev := event.EventWrapper{}
	_ = json.Unmarshal(data, &ev)
	switch ev.Type {
	case event.EventError:
		{
			errEvent := event.ErrorEvent{}
			json.Unmarshal(data, &errEvent)
			err = fmt.Errorf("get error event: %s", string(data))
			c.eventHandler.OnError(&errEvent)
			return nil, err
		}
	case event.EventReference:
		{
			refer := event.ReferenceEvent{}
			json.Unmarshal(ev.Payload, &refer)
			c.eventHandler.OnReference(&refer)
			return nil, nil
		}
	case event.EventThought:
		{
			thought := event.AgentThoughtEvent{}
			json.Unmarshal(ev.Payload, &thought)
			c.eventHandler.OnThought(&thought)
			return nil, nil
		}
	case event.EventReply:
		{
			reply := event.ReplyEvent{}
			json.Unmarshal(ev.Payload, &reply)
			if reply.IsFinal {
				finalReply = &reply
			}
			if reply.ReplyMethod != event.ReplyMethodInterrupt {
				c.eventHandler.OnReply(&reply)
			}
			return finalReply, nil
		}
	case event.EventTokenStat:
		{
			tokenStat := event.TokenStatEvent{}
			json.Unmarshal(ev.Payload, &tokenStat)
			c.eventHandler.OnTokenStat(&tokenStat)
			return finalReply, nil
		}
	}
	return nil, nil
}

func (c *lkeClient) queryOnce(ctx context.Context, req *model.ChatRequest) (
	finalReply *event.ReplyEvent, finalErr error) {
	bs, _ := json.Marshal(req)
	if c.logger != nil {
		c.logger.Info(fmt.Sprintf("[lkesdk]api call, request: %s", string(bs)))
	}
	payload := bytes.NewReader(bs)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, payload)
	if err != nil {
		return nil, fmt.Errorf("NewRequestWithContext error: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("httpClient do request error: %v", err)
	}
	defer res.Body.Close() // don't forget!!
	for ev, err := range sse.Read(res.Body, &sse.ReadConfig{
		MaxEventSize: 10 * 1024 * 1024, // 10M buffer
	}) {
		if c.closed.Load() {
			// client 关闭，不做任何处理
			return nil, fmt.Errorf("client has been closed")
		}
		if err != nil {
			return nil, fmt.Errorf("sse.Read error: %v", err)
		}
		finalReply, finalErr = c.handlerEvent([]byte(ev.Data))
	}
	if c.logger != nil {
		if finalErr != nil {
			c.logger.Error(fmt.Sprintf("[lkesdk]api final error: %v", finalErr))
		} else {
			bs, _ := json.Marshal(finalReply)
			c.logger.Info(fmt.Sprintf("[lkesdk]api final reply: %s", string(bs)))
		}
	}
	return finalReply, finalErr
}

func (c *lkeClient) runWithTimeout(ctx context.Context, f tool.Tool,
	input map[string]interface{}) (output interface{}, err error) {
	if c.toolRunTimeout.Seconds() == 0 {
		// 不设置超时时间
		return f.Execute(ctx, input)
	}
	runCtx, cancel := context.WithCancel(ctx)
	t := time.NewTimer(c.toolRunTimeout)
	defer cancel()
	signal := make(chan struct{})
	go func() {
		defer func() {
			if p := recover(); p != nil {
				err = fmt.Errorf("tool %s run failed, try another tool, error: %v",
					f.GetName(), string(debug.Stack()))
			}
		}()
		output, err = f.Execute(runCtx, input)
		select {
		case <-runCtx.Done():
			return
		case signal <- struct{}{}:
		}
	}()
	for {
		select {
		case <-t.C:
			cancel()
			err = fmt.Errorf("run tool %s timeout %ds", f.GetName(), int(c.toolRunTimeout.Seconds()))
			return nil, err
		case <-runCtx.Done():
			if runCtx.Err() != nil {
				return output, runCtx.Err()
			}
			return output, err
		case <-signal:
			return output, err
		}
	}
}

func (c *lkeClient) runTools(ctx context.Context, req *model.ChatRequest,
	reply *event.ReplyEvent, output *[]string) {
	if reply == nil {
		return
	}
	if reply.InterruptInfo == nil {
		return
	}
	if output == nil {
		return
	}
	if len(*output) != len(reply.InterruptInfo.ToolCalls) {
		return
	}
	// 处理工具调用，并行调用工具
	wg := sync.WaitGroup{}
	for i := range reply.InterruptInfo.ToolCalls {
		wg.Add(1)
		go func(index int) {
			defer func() {
				wg.Done()
			}()
			toolCall := reply.InterruptInfo.ToolCalls[index]
			if toolCall != nil {
				defer func() {
					if p := recover(); p != nil {
						(*output)[index] = fmt.Sprintf("Tool %s run failed, try another tool, error: %v",
							toolCall.Function.Name, string(debug.Stack()))
					}
				}()
				toolFuncMap, ok := c.toolsMap[reply.InterruptInfo.CurrentAgent]
				if !ok {
					// agent map 未找到
					(*output)[index] = fmt.Sprintf("The current agent %s toolset does not exist, try another tool",
						reply.InterruptInfo.CurrentAgent)
					return
				}
				f, ok := toolFuncMap[toolCall.Function.Name]
				if !ok {
					// tool name 未找到
					(*output)[index] = fmt.Sprintf("Tool %s not found in currant agent %s's toolset, try another tool",
						toolCall.Function.Name, reply.InterruptInfo.CurrentAgent)
					return
				}
				input := map[string]interface{}{}
				err := json.Unmarshal([]byte(toolCall.Function.Arguments), &input)
				if err != nil {
					// functional call 输出的函数参数有误
					(*output)[index] = fmt.Sprintf("The parameters of the thinking process output are wrong, error: %v", err)
					return
				}
				// 用户自定义参数放到 tool input 中
				if req != nil {
					for k, v := range req.CustomVariables {
						input[k] = v
					}
				}
				c.eventHandler.BeforeToolCallHook(f, input)
				toolout, err := c.runWithTimeout(ctx, f, input)
				c.eventHandler.AfterToolCallHook(f, input, toolout, err)
				if err != nil {
					(*output)[index] = fmt.Sprintf("Tool %s run failed, try another tool, error: %v",
						toolCall.Function.Name, err)
					return
				}

				str, _ := tool.InterfaceToString(toolout)
				(*output)[index] = str
			} else {
				// functional call 返回错误
				(*output)[index] = fmt.Sprintf("The %dth tool of the thought process output is empty", index)
			}
		}(i)
	}
	wg.Wait()
}

// RunWithContext 执行 agent with context，query 用户的输入
// sesionID 对话唯一标识，options 可选参数，可以为空，visitorBizID 用户的唯一标识
func (c *lkeClient) RunWithContext(ctx context.Context,
	query, sesionID, visitorBizID string,
	options *model.Options) (finalReply *event.ReplyEvent, err error) {
	if c.mock {
		return c.mockRun()
	}
	req := c.buildReq(query, sesionID, visitorBizID, options)
	for i := 0; i <= int(c.maxToolTurns); i++ {
		if c.closed.Load() {
			// client 关闭，不做任何处理
			return nil, fmt.Errorf("client has been closed")
		}
		reply, err := c.queryOnce(ctx, req)
		if err != nil {
			return nil, err
		}
		if reply == nil {
			return nil, fmt.Errorf("no final reply from server")
		}
		if reply.ReplyMethod != event.ReplyMethodInterrupt {
			return reply, err
		}
		outputs := []string{}
		if reply.InterruptInfo != nil {
			outputs = make([]string, len(reply.InterruptInfo.ToolCalls))
		}
		c.runTools(ctx, req, reply, &outputs)
		req.ToolOuputs = nil
		for i, out := range outputs {
			req.ToolOuputs = append(req.ToolOuputs, model.ToolOuput{
				ToolName: reply.InterruptInfo.ToolCalls[i].Function.Name,
				Output:   out,
			})
		}
	}
	return nil, fmt.Errorf("reached maximum tool call turns")
}

// Run 执行 agent，query 用户的输入，sesionID 对话唯一标识，options 可选参数，可以为空
// visitorBizID 用户的唯一标识
func (c *lkeClient) Run(query, sesionID, visitorBizID string,
	options *model.Options) (*event.ReplyEvent, error) {
	return c.RunWithContext(context.Background(), query, sesionID, visitorBizID, options)
}

func (c *lkeClient) mockRun() (finalReply *event.ReplyEvent, err error) {
	reply := &event.ReplyEvent{
		IsFinal: true,
		Content: "mock text",
	}
	c.mockToolCall(reply)
	outputs := []string{}
	if reply.InterruptInfo != nil {
		outputs = make([]string, len(reply.InterruptInfo.ToolCalls))
	}
	c.runTools(context.Background(), nil, reply, &outputs)
	if c.mock {
		for i, out := range outputs {
			fmt.Printf("run tool %s, input: %s, output: %s\n", reply.InterruptInfo.ToolCalls[i].Function.Name,
				reply.InterruptInfo.ToolCalls[i].Function.Arguments, out)
		}
	}
	finalReply = &event.ReplyEvent{
		IsFinal: true,
		Content: "mock text",
	}
	return finalReply, err
}

func (c *lkeClient) mockToolCall(reply *event.ReplyEvent) {
	// mock tool call
	for agentName, toolMap := range c.toolsMap {
		reply.InterruptInfo = &event.InterruptInfo{
			CurrentAgent: agentName,
		}
		for toolName, f := range toolMap {
			reply.ReplyMethod = event.ReplyMethodInterrupt
			jsonData := tool.GenerateRandomSchema(f.GetParametersSchema())
			str, _ := tool.InterfaceToString(jsonData)
			reply.InterruptInfo.ToolCalls = append(reply.InterruptInfo.ToolCalls,
				&openai.ToolCallDeltaUnion{
					Index: 1,
					Type:  "function",
					ID:    "mock-id",
					Function: openai.FunctionToolCallDeltaFunction{
						Name:      toolName,
						Arguments: str,
					},
				},
			)
		}
		return
	}
}

// Close 关闭所有 client 上的任务
func (c *lkeClient) Close() {
	c.closed.Store(true)
}

// Open Open 已经 Close 的 client
func (c *lkeClient) Open() {
	c.closed.Store(false)
}
