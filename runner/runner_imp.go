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

	"github.com/tencent-lke/lke-sdk-go/event"
	"github.com/tencent-lke/lke-sdk-go/eventhandler"
	"github.com/tencent-lke/lke-sdk-go/model"
	"github.com/tencent-lke/lke-sdk-go/runlog"
	"github.com/tencent-lke/lke-sdk-go/tool"
	"github.com/tmaxmax/go-sse"
)

// RunnerConf TODO
type RunnerConf struct {
	EnableSystemOpt     bool
	StartAgent          string
	Logger              runlog.RunLogger
	EventHandler        eventhandler.EventHandler
	MaxToolTurns        uint // 单次对话本地工具调用最大次数
	Endpoint            string
	BotAppKey           string
	HttpClient          *http.Client
	LocalToolRunTimeout time.Duration
}

// RunnerImp TODO
type RunnerImp struct {
	toolsMap map[string][]tool.Tool // agentName -> tool lists  的映射
	agents   []model.Agent
	handoffs []model.Handoff
	runconf  RunnerConf
}

// NewRunnerImp TODO
func NewRunnerImp(toolsMap map[string][]tool.Tool, agents []model.Agent,
	handoffs []model.Handoff, conf RunnerConf) *RunnerImp {
	runner := &RunnerImp{
		toolsMap: toolsMap,
		agents:   agents,
		handoffs: handoffs,
		runconf:  conf,
	}
	return runner
}

// RunWithTimeout TODO
func (c *RunnerImp) RunWithTimeout(ctx context.Context, f tool.Tool,
	input map[string]interface{}) (output interface{}, err error) {
	if c.runconf.LocalToolRunTimeout.Seconds() == 0 && f.GetTimeout() == 0 {
		return f.Execute(ctx, input)
	}
	var timeout time.Duration
	if f.GetTimeout() != 0 {
		timeout = f.GetTimeout()
	} else {
		timeout = c.runconf.LocalToolRunTimeout
	}
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
		if c.runconf.Logger != nil {
			c.runconf.Logger.Info(fmt.Sprintf("runWithTimeoutExecute: %s, cost: %v", f.GetName(), end.Sub(begin)))
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

// RunTools TODO
func (c *RunnerImp) RunTools(ctx context.Context, req *model.ChatRequest,
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
					CallToolName: f.GetName(),
					CallId:       toolCall.ID,
					Input:        input,
				}
				c.runconf.EventHandler.BeforeToolCallHook(toolCallCtx)
				toolout, err := c.RunWithTimeout(ctx, f, input)
				toolCallCtx.Output = toolout
				toolCallCtx.Err = err
				c.runconf.EventHandler.AfterToolCallHook(toolCallCtx)
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

func (c *RunnerImp) buildReq(query, requestID, sessionID, visitorBizID string, botAppKey string,
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
	// req.Options.RequestID = uuid.New().String()
	req.Options.RequestID = requestID
	// 构建 agent 参数
	req.AgentConfig.Agents = c.agents
	// 构建 handoff 参数
	req.AgentConfig.Handoffs = c.handoffs
	req.AgentConfig.DisableSystemOpt = !c.runconf.EnableSystemOpt
	req.AgentConfig.StartAgentName = c.runconf.StartAgent
	// 构建工具参数
	for agentName, toolFuncMap := range c.toolsMap {
		fmt.Printf("agentName: %s, toolFuncMap: %v\n", agentName, toolFuncMap)
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
	// bs, _ := json.Marshal(req)
	// fmt.Printf("req: %v\n", string(bs))
	return req
}

func (c *RunnerImp) queryOnce(ctx context.Context, req *model.ChatRequest) (
	finalReply *event.ReplyEvent, finalErr error) {
	bs, _ := json.Marshal(req)
	if c.runconf.Logger != nil {
		c.runconf.Logger.Info(fmt.Sprintf("[lkesdk]api call, request: %s", string(bs)))
	}
	payload := bytes.NewReader(bs)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.runconf.Endpoint, payload)
	if err != nil {
		return nil, fmt.Errorf("NewRequestWithContext error: %v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	res, err := c.runconf.HttpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("httpClient do request error: %v", err)
	}
	defer res.Body.Close() // don't forget!!
	for ev, err := range sse.Read(res.Body, &sse.ReadConfig{
		MaxEventSize: 10 * 1024 * 1024, // 10M buffer
	}) {
		if err != nil {
			return nil, fmt.Errorf("sse.Read error: %v", err)
		}
		finalReply, finalErr = c.handlerEvent([]byte(ev.Data))
	}
	if c.runconf.Logger != nil {
		if finalErr != nil {
			c.runconf.Logger.Error(fmt.Sprintf("[lkesdk]api final error: %v", finalErr))
		} else {
			bs, _ := json.Marshal(finalReply)
			c.runconf.Logger.Info(fmt.Sprintf("[lkesdk]api final reply: %s", string(bs)))
		}
	}
	return finalReply, finalErr
}

// RunWithContext TODO
func (c *RunnerImp) RunWithContext(ctx context.Context,
	query, requestID, sessionID, visitorBizID string,
	options *model.Options) (finalReply *event.ReplyEvent, err error) {
	req := c.buildReq(query, requestID, sessionID, visitorBizID, c.runconf.BotAppKey, options)
	// c.runconf.Logger.Info(fmt.Sprintf("buildReq: %v", req))
	for i := 0; i <= int(c.runconf.MaxToolTurns); i++ {
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

func (c *RunnerImp) handlerEvent(data []byte) (finalReply *event.ReplyEvent, err error) {
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
			errEvent.EContent.Content = make(map[string]string)
			errEvent.EContent.Content["agentname"] = c.runconf.StartAgent
			c.runconf.EventHandler.OnError(&errEvent)
			return nil, err
		}
	case event.EventReference:
		{
			refer := event.ReferenceEvent{}
			json.Unmarshal(ev.Payload, &refer)
			refer.EContent.Content = make(map[string]string)
			refer.EContent.Content["agentname"] = c.runconf.StartAgent
			c.runconf.EventHandler.OnReference(&refer)
			return nil, nil
		}
	case event.EventThought:
		{
			thought := event.AgentThoughtEvent{}
			json.Unmarshal(ev.Payload, &thought)
			thought.EContent.Content = make(map[string]string)
			thought.EContent.Content["agentname"] = c.runconf.StartAgent
			c.runconf.EventHandler.OnThought(&thought)
			return nil, nil
		}
	case event.EventReply:
		{
			reply := event.ReplyEvent{}
			json.Unmarshal(ev.Payload, &reply)
			reply.EContent.Content = make(map[string]string)
			reply.EContent.Content["agentname"] = c.runconf.StartAgent
			if reply.IsFinal {
				finalReply = &reply
			}
			if reply.ReplyMethod != event.ReplyMethodInterrupt {
				c.runconf.EventHandler.OnReply(&reply)
			}
			return finalReply, nil
		}
	case event.EventTokenStat:
		{
			tokenStat := event.TokenStatEvent{}
			json.Unmarshal(ev.Payload, &tokenStat)
			tokenStat.EContent.Content = make(map[string]string)
			tokenStat.EContent.Content["agentname"] = c.runconf.StartAgent
			c.runconf.EventHandler.OnTokenStat(&tokenStat)
			return finalReply, nil
		}
	}
	return nil, nil
}
