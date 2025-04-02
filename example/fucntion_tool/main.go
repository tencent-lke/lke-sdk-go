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
	"github.com/tencent-lke/lke-sdk-go/tool"
)

const (
	kBotAppKey = "zIIRbxwI"
)

type MyEventHandler struct {
	lkesdk.DefaultEventHandler // 引用默认实现
}

func add(a int, b int) int {
	return a + b
}

type AddParam struct {
	A int `json:"a"`
	B int `json:"b"`
}

func add2(param1 AddParam) int {
	return param1.A + param1.B
}

// Reply 自定义回复处理事件
func (MyEventHandler) Reply(reply *event.ReplyEvent) {
	if reply.IsFromSelf {
		// 过滤输入重复回包
		return
	}
	log.Printf("Reply: %v", reply.Content)
}

func main() {
	sessionId := uuid.New().String()
	client := lkesdk.NewLkeClient(kBotAppKey, sessionId)
	client.SetEndpoint("https://testwss.testsite.woa.com/v1/qbot/chat/experienceSse?qbot_env_set=2_10")
	client.SetEventHandler(&MyEventHandler{})
	// 方式1, 自定义函数如参是一个 struct，并且 struct 中每个字段都有 tag
	tools := []*tool.FunctionTool{}
	t, err := tool.NewFunctionTool("add", "计算两个数的和", add2, nil)
	if err != nil {
		tools = append(tools, t)
	} else {
		log.Panicf("不支持的函数定义: %v", err)
	}

	client.AddFunctionTools("agentA", tools)
	for {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("请输入你想问的问题：")

		// 读取用户输入，直到遇到换行符
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Println("读取输入时出错:", err)
			return
		}
		input = strings.TrimSuffix(input, "\n")
		options := &model.Options{
			StreamingThrottle: 5,
			RequestID:         "test",
		}
		finalReply, err := client.Chat(input, options)
		if err != nil {
			log.Fatalf("chat 出错: %v", err)
		}
		log.Printf("finalReply: %v\n", finalReply.Content)
	}

}
