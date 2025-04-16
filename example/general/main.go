package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
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
}

// OnReply 自定义回复处理事件
func (MyEventHandler) OnReply(reply *event.ReplyEvent) {
	if reply.IsFromSelf {
		// 过滤输入重复回包
		return
	}
	log.Printf("Reply: %v", reply.Content)
}

// OnReply 自定义思考处理事件
func (MyEventHandler) OnThought(thought *event.AgentThoughtEvent) {
	if len(thought.Procedures) > 0 {
		log.Printf("Thought: %s\n", thought.Procedures[len(thought.Procedures)-1].Debugging.Content)
	}
}

func main() {
	sessionID := uuid.New().String()
	client := lkesdk.NewLkeClient(botAppKey, &MyEventHandler{})
	// client.SetMock(true)

	for {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("请输入你想问的问题：")

		// 读取用户输入，直到遇到换行符
		query, err := reader.ReadString('\n')
		if err != nil {
			log.Println("读取输入时出错:", err)
			return
		}
		query = strings.TrimSuffix(query, "\n")
		options := &model.Options{
			StreamingThrottle: 5,
		}
		finalReply, err := client.Run(query, sessionID, visitorBizID, options)
		if err != nil {
			log.Fatalf("run error: %v", err)
		}
		log.Printf("finalReply: %v\n", finalReply.Content)
	}

}
