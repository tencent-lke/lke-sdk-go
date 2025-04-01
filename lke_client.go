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
)

const kDefaultEndpoint = "https://wss.lke.cloud.tencent.com/v1/qbot/chat/sse"

// LkeClient represents a client for interacting with the LKE service
type LkeClient struct {
	botAppKey    string // 机器人密钥 (从运营接口人处获取)
	sessionID    string // 会话 ID（外部系统提供，不能为空）
	visitorBizID string // 访客 ID（外部系统提供，需确认不同的访客使用不同的 ID）
	endpoint     string // 调用地址
	eventHandler EventHandler
}

// NewLkeClient creates a new LKE client with the provided parameters
func NewLkeClient(botAppKey, sessionID, visitorBizID string) *LkeClient {
	return &LkeClient{
		botAppKey:    botAppKey,
		sessionID:    sessionID,
		visitorBizID: visitorBizID,
		endpoint:     kDefaultEndpoint,
		eventHandler: nil,
	}
}

func (c LkeClient) GetBotAppKey() string {
	return c.botAppKey
}

// GetSessionID returns the session ID
func (c LkeClient) GetSessionID() string {
	return c.sessionID
}

// GetVisitorBizID returns the visitor business ID
func (c LkeClient) GetVisitorBizID() string {
	return c.visitorBizID
}

// SetBotAppKey sets the bot application key
func (c *LkeClient) SetBotAppKey(botAppKey string) {
	c.botAppKey = botAppKey
}

// SetSessionID sets the session ID
func (c *LkeClient) SetSessionID(sessionID string) {
	c.sessionID = sessionID
}

// SetVisitorBizID sets the visitor business ID
func (c *LkeClient) SetVisitorBizID(visitorBizID string) {
	c.visitorBizID = visitorBizID
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

// Chat 对话接口，query 用户的输入，options 可选参数，可以为空
func (c LkeClient) ChatWithContext(ctx context.Context, query string, options *model.Options) (
	finalReply event.ReplyEvent, err error) {
	req := model.ChatRequest{
		Content:      query,
		VisitorBizID: c.visitorBizID,
		BotAppKey:    c.botAppKey,
		SessionID:    c.sessionID,
	}
	if options != nil {
		req.Options = *options
	}
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
