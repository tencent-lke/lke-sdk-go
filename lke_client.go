package lkesdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"sync"

	"github.com/openai/openai-go"
	"github.com/r3labs/sse/v2"
	"github.com/tencent-lke/lke-sdk-go/event"
	"github.com/tencent-lke/lke-sdk-go/model"
	"github.com/tencent-lke/lke-sdk-go/tool"
)

const (
	DefaultEndpoint = "https://wss.lke.cloud.tencent.com/v1/qbot/chat/sse"
	maxToolTurns    = 10 // 工具调用最大轮数
)

// LkeClient represents a client for interacting with the LKE service
type LkeClient struct {
	botAppKey    string // 机器人密钥 (从运营接口人处获取)
	sessionID    string // 会话 ID（外部系统提供，不能为空）
	visitorBizID string // 访客 ID（外部系统提供，需确认不同的访客使用不同的 ID）
	endpoint     string // 调用地址
	eventHandler EventHandler
	toolsMap     map[string]map[string]tool.FunctionTool // agentName -> map[toolname -> FunctionTool] 的映射
}

// NewLkeClient creates a new LKE client with the provided parameters
func NewLkeClient(botAppKey, sessionID string) *LkeClient {
	return &LkeClient{
		botAppKey:    botAppKey,
		sessionID:    sessionID,
		visitorBizID: "123456789",
		endpoint:     DefaultEndpoint,
		eventHandler: nil,
		toolsMap:     map[string]map[string]tool.FunctionTool{},
	}
}

func (c LkeClient) GetBotAppKey() string {
	return c.botAppKey
}

// GetSessionID returns the session ID
func (c LkeClient) GetSessionID() string {
	return c.sessionID
}

// SetBotAppKey sets the bot application key
func (c *LkeClient) SetBotAppKey(botAppKey string) {
	c.botAppKey = botAppKey
}

// SetSessionID sets the session ID
func (c *LkeClient) SetSessionID(sessionID string) {
	c.sessionID = sessionID
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

func (c *LkeClient) AddFunctionTools(agentName string, tools []*tool.FunctionTool) {
	if len(tools) == 0 {
		return
	}

	toolFuncMap, ok := c.toolsMap[agentName]
	if !ok {
		toolFuncMap = map[string]tool.FunctionTool{}
		c.toolsMap[agentName] = toolFuncMap
	}
	for _, tool := range tools {
		if tool != nil {
			toolFuncMap[tool.GetName()] = *tool
		}
	}
}
func (c LkeClient) buildReq(query string, options *model.Options) *model.ChatRequest {
	req := &model.ChatRequest{
		Content:      query,
		VisitorBizID: c.visitorBizID,
		BotAppKey:    c.botAppKey,
		SessionID:    c.sessionID,
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
				dynamicTool.Tools = append(dynamicTool.Tools, tool.ToOpenAIToolPB(&t))
			}
			req.DynamicTools = append(req.DynamicTools, dynamicTool)
		}
	}
	return req
}

func (c LkeClient) queryOnce(ctx context.Context, req *model.ChatRequest) (
	finalReply *event.ReplyEvent, err error) {
	sseCli := sse.NewClient(c.endpoint, func(c *sse.Client) {
		body, _ := json.Marshal(req)
		c.Body = bytes.NewReader(body)
		c.Method = http.MethodPost
		c.Headers["Content-Type"] = "application/json"
	})
	handler := func(msg *sse.Event) {
		if c.eventHandler == nil {
			return
		}
		defer func() {
			if p := recover(); p != nil {
			}
		}()
		ev := event.EventWrapper{}
		_ = json.Unmarshal(msg.Data, &ev)
		switch ev.Type {
		case event.EventError:
			{
				errEvent := event.ErrorEvent{}
				json.Unmarshal(msg.Data, &errEvent)
				err = fmt.Errorf("get error event: %s", string(msg.Data))
				c.eventHandler.Error(&errEvent)
				break
			}
		case event.EventReference:
			{
				refer := event.ReferenceEvent{}
				json.Unmarshal(ev.Payload, &refer)
				c.eventHandler.Reference(&refer)
				break
			}
		case event.EventThought:
			{
				thought := event.AgentThoughtEvent{}
				json.Unmarshal(ev.Payload, &thought)
				c.eventHandler.Thought(&thought)
				break
			}
		case event.EventReply:
			{
				reply := event.ReplyEvent{}
				json.Unmarshal(ev.Payload, &reply)
				if reply.IsFinal {
					finalReply = &reply
				}
				c.eventHandler.Reply(&reply)
				break
			}
		case event.EventTokenStat:
			{
				tokenStat := event.TokenStatEvent{}
				json.Unmarshal(ev.Payload, &tokenStat)
				c.eventHandler.TokenStat(&tokenStat)
				break
			}
		}
	}
	e := sseCli.SubscribeRawWithContext(ctx, handler)
	if e != nil {
		err = e
	}
	return finalReply, err
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

// Chat 对话接口，query 用户的输入，options 可选参数，可以为空
func (c LkeClient) ChatWithContext(ctx context.Context, query string, options *model.Options) (
	finalReply *event.ReplyEvent, err error) {
	req := c.buildReq(query, options)
	for i := 0; i <= maxToolTurns; i++ {
		reply, err := c.queryOnce(ctx, req)
		if err != nil {
			return nil, err
		}
		if reply == nil {
			return nil, fmt.Errorf("no final reply from server")
		}
		// if i == 0 {
		// 	c.mockToolCall(reply) // mock tool call
		// }
		if reply.ReplyMethod != event.ReplyMethodInterrupt {
			return reply, err
		}
		outputs := []string{}
		if reply.InterruptInfo != nil {
			outputs = make([]string, len(reply.InterruptInfo.ToolCalls))
		}
		c.runTools(ctx, reply, &outputs)
		fmt.Printf("%v\n", outputs)
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

func (c LkeClient) Chat(query string, options *model.Options) (*event.ReplyEvent, error) {
	return c.ChatWithContext(context.Background(), query, options)
}

func (c LkeClient) mockToolCall(reply *event.ReplyEvent) {
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
	}
}
