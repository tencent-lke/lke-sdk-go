package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/google/uuid"
	mcpclient "github.com/mark3labs/mcp-go/client"
	lkesdk "github.com/tencent-lke/lke-sdk-go"
	"github.com/tencent-lke/lke-sdk-go/model"
)

const (
	// 获取方法 https://cloud.tencent.com/document/product/1759/105561#8590003a-0a6d-4a8d-9a02-b706221a679d
	botAppKey = "custom-app-key"
)

func buildCustomStdioMcpClient() mcpclient.MCPClient {
	_, f, _, _ := runtime.Caller(0)
	serverPath := path.Join(path.Dir(f), "custom_server", "server.go")
	c, err := mcpclient.NewStdioMCPClient(
		"go",
		[]string{}, // Empty ENV
		"run",
		serverPath,
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	return c
}

func main() {
	sessionID := uuid.New().String()
	client := lkesdk.NewLkeClient(botAppKey, nil)
	// client.SetMock(true) // mock run

	// 增加自定义 mcp 插件
	c := buildCustomStdioMcpClient()
	defer c.Close()
	addTools, err := client.AddMcpTools("Agent-A", c, nil) // add all tools in mcp client
	if err != nil {
		log.Fatalf("Failed to AddMcpTools, error: %v", err)
	}

	for _, tools := range addTools {
		bs, _ := json.Marshal(tools.GetParametersSchema())
		fmt.Printf("toolname: %s\ndescribe: %s\nschema: %v\n\n",
			tools.GetName(), tools.GetDescription(), string(bs))
	}

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
		finalReply, err := client.Run(query, sessionID, options)
		if err != nil {
			log.Fatalf("run error: %v", err)
		}
		log.Printf("finalReply: %v\n", finalReply.Content)
	}
}
