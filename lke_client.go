package lkesdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/r3labs/sse/v2"
	"github.com/tencent-lke/lke-sdk-go/event"
	"github.com/tencent-lke/lke-sdk-go/model"
	"github.com/tencent-lke/lke-sdk-go/tool"
)

const kDefaultEndpoint = "https://wss.lke.cloud.tencent.com/v1/qbot/chat/sse"

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
		endpoint:     kDefaultEndpoint,
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
		toolFuncMap[tool.GetName()] = *tool
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
	fmt.Print(len(c.toolsMap))
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
	bs, _ := json.MarshalIndent(req, "  ", "  ")
	fmt.Println(string(bs))
	return req
}

// Chat 对话接口，query 用户的输入，options 可选参数，可以为空
func (c LkeClient) ChatWithContext(ctx context.Context, query string, options *model.Options) (
	finalReply event.ReplyEvent, err error) {
	req := c.buildReq(query, options)
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
					finalReply = reply
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

func (c LkeClient) Chat(query string, options *model.Options) (event.ReplyEvent, error) {
	return c.ChatWithContext(context.Background(), query, options)
}
