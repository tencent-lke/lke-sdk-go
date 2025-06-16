package main

import (
	"bytes"
	"encoding/json"
	"log"
	"strings"

	"github.com/google/uuid"
	lkesdk "github.com/tencent-lke/lke-sdk-go"
	"github.com/tencent-lke/lke-sdk-go/event"
	"github.com/tencent-lke/lke-sdk-go/model"
)

const (
	visitorBizID = "custom-visior-id" // 访问者 id
	// 获取方法 https://cloud.tencent.com/document/product/1759/105561#8590003a-0a6d-4a8d-9a02-b706221a679d
	botAppKey = "custom-app-key"
)

// MyEventHandler 创建自定义事件处理器
type MyEventHandler struct {
	lkesdk.DefaultEventHandler // 引用默认实现
	traceID                    string
	runNodes                   []string
	runNodeMap                 map[string]*event.RunNodeInfo
}

// OnReply 自定义回复处理事件
func (h *MyEventHandler) OnReply(reply *event.ReplyEvent) {
	if reply.IsFromSelf {
		h.traceID = reply.TraceId
		log.Printf("traceID: %s", h.traceID)
		// 过滤输入重复回包
		return
	}
	log.Printf("Reply: %v", reply.Content)
}

// OnThought 自定义思考处理事件
func (h *MyEventHandler) OnThought(thought *event.AgentThoughtEvent) {
	if len(thought.Procedures) > 0 {
		log.Printf("Thought: %s\n", thought.Procedures[len(thought.Procedures)-1].Debugging.Content)
	}
}

// OnTokenStat 自定义 token 统计处理事件
func (h *MyEventHandler) OnTokenStat(tokenStat *event.TokenStatEvent) {
	for _, procedure := range tokenStat.Procedures {
		workflowSummary := procedure.Debugging.Workflow
		if workflowSummary.WorkflowID == "" {
			// 只关注工作流执行情况
			continue
		}
		for _, runNode := range workflowSummary.RunNodes {
			if _, ok := h.runNodeMap[runNode.NodeID]; !ok {
				h.runNodeMap[runNode.NodeID] = runNode
				h.runNodes = append(h.runNodes, runNode.NodeID)
				log.Printf("runNode: %s", ToJsonString(runNode))
			} else {
				h.runNodeMap[runNode.NodeID] = runNode
			}
		}
	}
}

// ToJsonString 转成显示的内容
func ToJsonString(valueI interface{}) string {
	valueStr, ok := valueI.(string)
	if ok {
		return valueStr
	}
	bf := bytes.NewBuffer([]byte{})
	jsonEncoder := json.NewEncoder(bf)
	jsonEncoder.SetEscapeHTML(false)
	err := jsonEncoder.Encode(valueI)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(bf.String())
}

func main() {
	sessionID := uuid.New().String()
	client := lkesdk.NewLkeClient(botAppKey, &MyEventHandler{
		runNodeMap: make(map[string]*event.RunNodeInfo),
	})
	// client.SetMock(true)
	query := "请帮我分析一下这个Excel文件，并告诉我每个产品的销售情况"
	options := &model.Options{
		CustomVariables: map[string]string{
			"ExcelFile": "custom-excel-file-url",
		},
	}
	finalReply, err := client.Run(query, sessionID, visitorBizID, options)
	if err != nil {
		log.Fatalf("run error: %v", err)
	}
	log.Printf("finalReply: %v\n", finalReply.Content)
}
