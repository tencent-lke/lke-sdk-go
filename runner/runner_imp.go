package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tencent-lke/lke-sdk-go/event"
	"github.com/tencent-lke/lke-sdk-go/eventhandler"
	"github.com/tencent-lke/lke-sdk-go/model"
	"github.com/tencent-lke/lke-sdk-go/runlog"
	"github.com/tencent-lke/lke-sdk-go/tool"
	"github.com/tmaxmax/go-sse"
)

// runnerImp ...
type runnerImp struct {
	Name            string
	toolsMap        map[string][]tool.Tool // agentName -> tool lists  的映射
	agents          []model.Agent
	handoffs        []model.Handoff
	enableSystemOpt bool
	startAgent      string
	logger          runlog.RunLogger
	eventHandler    eventhandler.EventHandler
	maxToolTurns    uint // 单次对话本地工具调用最大次数
	endpoint        string
}

func (c *runnerImp) QueryOnce(ctx context.Context, req *model.ChatRequest) (
	finalReply *event.ReplyEvent, finalErr error) {
	return
}

func (c *runnerImp) RunWithTimeout(ctx context.Context, f tool.Tool,
	input map[string]interface{}) (output interface{}, err error) {
	var timeout time.Duration
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	signal := make(chan struct{}) // 无缓冲通道
	go func() {
		defer close(signal) // 关闭通道广播完成
		defer func() {
			if p := recover(); p != nil {
				err = fmt.Errorf("panic: %v", p)
			}
		}()
		begin := time.Now()
		output, err = f.Execute(runCtx, input)
		end := time.Now()
		if c.logger != nil {
			c.logger.Info(fmt.Sprintf("runWithTimeoutExecute: %s, cost: %v", f.GetName(), end.Sub(begin)))
		}
	}()
	t := time.NewTimer(timeout)
	defer t.Stop() // 确保定时器释放

	select {
	case <-t.C:
		return nil, fmt.Errorf("run tool %s timeout %ds", f.GetName(), int(timeout.Seconds()))
	case <-runCtx.Done():
		if err != nil {
			return output, err // 工具错误优先
		}
		return output, runCtx.Err()
	case <-signal:
		return output, err
	}
}

func (c *runnerImp) RunTools(ctx context.Context, req *model.ChatRequest,
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
				toolFuncs, ok := c.toolsMap[reply.InterruptInfo.CurrentAgent]
				if !ok {
					// agent map 未找到
					(*output)[index] = fmt.Sprintf("The current agent %s toolset does not exist, try another tool",
						reply.InterruptInfo.CurrentAgent)
					return
				}
				var f tool.Tool = nil
				hasTool := false
				for _, too := range toolFuncs {
					if too.GetName() == toolCall.Function.Name {
						f = too
						hasTool = true
						break
					}
				}
				if !hasTool {
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
				toolCallCtx := eventhandler.ToolCallContext{
					CallTool: f,
					CallId:   toolCall.ID,
					Input:    input,
				}
				c.eventHandler.BeforeToolCallHook(toolCallCtx)
				toolout, err := c.RunWithTimeout(ctx, f, input)
				toolCallCtx.Output = toolout
				toolCallCtx.Err = err
				c.eventHandler.AfterToolCallHook(toolCallCtx)
				if err != nil {
					(*output)[index] = fmt.Sprintf("Tool %s run failed, try another tool, error: %v",
						toolCall.Function.Name, err)
					return
				}
				(*output)[index] = f.ResultToString(toolout)
			} else {
				// functional call 返回错误
				(*output)[index] = fmt.Sprintf("The %dth tool of the thought process output is empty", index)
			}
		}(i)
	}
	wg.Wait()
}

func (c *runnerImp) buildReq(query, sessionID, visitorBizID string, botAppKey string,
	options *model.Options) *model.ChatRequest {
	req := &model.ChatRequest{
		Content:      query,
		VisitorBizID: visitorBizID,
		BotAppKey:    botAppKey,
		SessionID:    sessionID,
	}
	if options != nil {
		req.Options = *options
	}
	// 一次端云交互的过程使用一个 requestId
	req.Options.RequestID = uuid.New().String()
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

func (c *runnerImp) queryOnce(ctx context.Context, req *model.ChatRequest) (
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
		// if c.closed.Load() {
		// 	// client 关闭，不做任何处理
		// 	return nil, fmt.Errorf("client has been closed")
		// }
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

func (c *runnerImp) RunWithContext(ctx context.Context,
	query, sesionID, visitorBizID string,
	options *model.Options) (finalReply *event.ReplyEvent, err error) {
	var botAppKey string
	req := c.buildReq(query, sesionID, visitorBizID, botAppKey, options)
	for i := 0; i <= int(c.maxToolTurns); i++ {
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
		c.RunTools(ctx, req, reply, &outputs)
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

func (c *runnerImp) handlerEvent(data []byte) (finalReply *event.ReplyEvent, err error) {
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
