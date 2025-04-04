package lkesdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"sync"

	"github.com/mark3labs/mcp-go/client"
	"github.com/openai/openai-go"
	"github.com/tencent-lke/lke-sdk-go/event"
	"github.com/tencent-lke/lke-sdk-go/model"
	"github.com/tencent-lke/lke-sdk-go/tool"
	sse "github.com/tmaxmax/go-sse"
)

const (
	DefaultEndpoint = "https://wss.lke.cloud.tencent.com/v1/qbot/chat/sse"
	maxToolTurns    = 10 // 工具调用最大轮数
)

// LkeClient represents a client for interacting with the LKE service
type LkeClient struct {
	botAppKey    string // 机器人密钥 (从运营接口人处获取)
	visitorBizID string // 访客 ID（外部系统提供，需确认不同的访客使用不同的 ID）
	endpoint     string // 调用地址
	eventHandler EventHandler
	toolsMap     map[string]map[string]tool.Tool // agentName -> map[toolname -> FunctionTool] 的映射
	mock         bool
	httpClient   *http.Client
}

// NewLkeClient creates a new LKE client with the provided parameters
// eventHandler 自定义事件处理
func NewLkeClient(botAppKey string, eventHandler EventHandler) *LkeClient {
	handler := eventHandler
	if handler == nil {
		handler = &DefaultEventHandler{}
	}
	return &LkeClient{
		botAppKey:    botAppKey,
		visitorBizID: "123456789",
		endpoint:     DefaultEndpoint,
		eventHandler: handler,
		toolsMap:     map[string]map[string]tool.Tool{},
		mock:         false,
		httpClient:   http.DefaultClient,
	}
}

func (c LkeClient) GetBotAppKey() string {
	return c.botAppKey
}

// SetBotAppKey sets the bot application key
func (c *LkeClient) SetBotAppKey(botAppKey string) {
	c.botAppKey = botAppKey
}

// GetEndpoint returns the endpoint URL
func (c LkeClient) GetEndpoint() string {
	return c.endpoint
}

// SetEndpoint sets the endpoint URL
func (c *LkeClient) SetEndpoint(endpoint string) {
	c.endpoint = endpoint
}

// SetEventHandler 设置时间处理函数
func (c *LkeClient) SetEventHandler(eventHandler EventHandler) {
	c.eventHandler = eventHandler
}

// SetMock 设置 Mock
func (c *LkeClient) SetMock(mock bool) {
	c.mock = mock
}

// SetHttpClient 设置自定义 http client
func (c *LkeClient) SetHttpClient(cli *http.Client) {
	if cli != nil {
		c.httpClient = cli
	}
}

// AddFunctionTools 增加函数 tools
func (c *LkeClient) AddFunctionTools(agentName string, tools []*tool.FunctionTool) {
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
func (c *LkeClient) AddMcpTools(agentName string, mcpClient client.MCPClient, selectedToolNames []string) (
	addTools []*tool.McpTool, err error) {
	tools, err := tool.ListMcpTools(mcpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %v", err)
	}
	selectMap := map[string]struct{}{}
	for _, t := range selectTools {
		selectMap[t] = struct{}{}
	}
	toolMcpMap, ok := c.toolsMap[agentName]
	if !ok {
		toolMcpMap = map[string]tool.Tool{}
		c.toolsMap[agentName] = toolMcpMap
	}
	for _, t := range tools.Tools {
		add := true
		if len(selectTools) > 0 {
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
			json.Unmarshal(bs, &newtool.Schame)
			toolMcpMap[t.Name] = newtool
			addTools = append(addTools, newtool)
		}
	}
	return addTools, err
}

func (c LkeClient) buildReq(query, sessionID string, options *model.Options) *model.ChatRequest {
	req := &model.ChatRequest{
		Content:      query,
		VisitorBizID: c.visitorBizID,
		BotAppKey:    c.botAppKey,
		SessionID:    sessionID,
	}
	if options != nil {
		req.Options = *options
	}
	for agentName, toolFuncMap := range c.toolsMap {
		if len(toolFuncMap) > 0 {
			dynamicTool := model.DynamicTool{
				AgentName: agentName,
			}
			for _, t := range toolFuncMap {
				dynamicTool.Tools = append(dynamicTool.Tools, tool.ToOpenAIToolPB(t))
			}
			req.DynamicTools = append(req.DynamicTools, dynamicTool)
		}
	}
	return req
}

func (c LkeClient) handlerEvent(data []byte) (finalReply *event.ReplyEvent, err error) {
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
			c.eventHandler.Error(&errEvent)
			return nil, err
		}
	case event.EventReference:
		{
			refer := event.ReferenceEvent{}
			json.Unmarshal(ev.Payload, &refer)
			c.eventHandler.Reference(&refer)
			return nil, nil
		}
	case event.EventThought:
		{
			thought := event.AgentThoughtEvent{}
			json.Unmarshal(ev.Payload, &thought)
			c.eventHandler.Thought(&thought)
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
				c.eventHandler.Reply(&reply)
			}
			return finalReply, nil
		}
	case event.EventTokenStat:
		{
			tokenStat := event.TokenStatEvent{}
			json.Unmarshal(ev.Payload, &tokenStat)
			c.eventHandler.TokenStat(&tokenStat)
			return finalReply, nil
		}
	}
	return nil, nil
}

func (c LkeClient) queryOnce(ctx context.Context, req *model.ChatRequest) (
	finalReply *event.ReplyEvent, finalErr error) {
	bs, _ := json.Marshal(req)
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

	for ev, err := range sse.Read(res.Body, nil) {
		if err != nil {
			return nil, fmt.Errorf("sse.Read error: %v", err)
		}
		finalReply, finalErr = c.handlerEvent([]byte(ev.Data))
	}
	return finalReply, finalErr
}

func (c LkeClient) runTools(ctx context.Context, reply *event.ReplyEvent, output *[]string) {
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
			toolCall := reply.InterruptInfo.ToolCalls[index]
			if toolCall != nil {
				defer func() {
					if p := recover(); p != nil {
						(*output)[index] = fmt.Sprintf("Tool %s run failed, try another tool, error: %v",
							toolCall.Function.Name, string(debug.Stack()))
					}
					wg.Done()
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
				toolout, err := f.Execute(ctx, input)
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
// sesionID 对话唯一标识，options 可选参数，可以为空
func (c LkeClient) RunWithContext(ctx context.Context, query, sesionID string,
	options *model.Options) (finalReply *event.ReplyEvent, err error) {
	if c.mock {
		return c.mockRun()
	}
	req := c.buildReq(query, sesionID, options)
	for i := 0; i <= maxToolTurns; i++ {
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
		c.runTools(ctx, reply, &outputs)
		req.LocalToolOuputs = nil
		for i, out := range outputs {
			req.LocalToolOuputs = append(req.LocalToolOuputs, model.LocalToolOuput{
				ToolName: reply.InterruptInfo.ToolCalls[i].Function.Name,
				Output:   out,
			})
		}
	}
	return nil, fmt.Errorf("reached maximum tool call turns")
}

// Run 执行 agent，query 用户的输入，sesionID 对话唯一标识，options 可选参数，可以为空
func (c LkeClient) Run(query, sesionID string,
	options *model.Options) (*event.ReplyEvent, error) {
	return c.RunWithContext(context.Background(), query, sesionID, options)
}

func (c LkeClient) mockRun() (finalReply *event.ReplyEvent, err error) {
	reply := &event.ReplyEvent{
		IsFinal: true,
		Content: "mock text",
	}
	c.mockToolCall(reply)
	outputs := []string{}
	if reply.InterruptInfo != nil {
		outputs = make([]string, len(reply.InterruptInfo.ToolCalls))
	}
	c.runTools(context.Background(), reply, &outputs)
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

func (c LkeClient) mockToolCall(reply *event.ReplyEvent) {
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
