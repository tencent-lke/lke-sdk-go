package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	mcpclient "git.woa.com/trpc-go/mcp-go/client"
	"git.woa.com/trpc-go/mcp-go/mcp"
	"git.woa.com/trpc-go/mcp-go/server"
	"github.com/google/uuid"
	openmcp "github.com/mark3labs/mcp-go/mcp"
	lkesdk "github.com/tencent-lke/lke-sdk-go"
	"github.com/tencent-lke/lke-sdk-go/model"
)

const (
	visitorBizID = "custom-visior-id" // 访问者 id
	// 获取方法 https://cloud.tencent.com/document/product/1759/105561#8590003a-0a6d-4a8d-9a02-b706221a679d
	botAppKey = "custom-app-key"
)

func buildSeeMcpClient() openmcp.MCPClient {
	// 启动一个 test sse server
	mcpServer := server.NewMCPServer(
		"test-server",
		"1.0.0",
	)

	// Add a test tool
	mcpServer.AddTool(mcp.NewTool(
		"test-tool",
		mcp.WithDescription("Test tool"),
		mcp.WithString("parameter-1", mcp.Description("A string tool parameter")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return nil, errors.New("arguments must be map[string]interface{}")
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "query parameter: " + args["parameter-1"].(string),
				},
			},
		}, nil
	})
	testServer := server.NewTestServer(mcpServer)
	if testServer == nil {
		log.Fatalf("test server is nil")
	}
	// ch := make(chan int, 1)
	// <-ch
	c, err := mcpclient.NewSSEMCPClient(testServer.URL + "/sse")
	if err != nil {
		log.Fatalf("Failed to connect sse server: %v", err)
	}
	if err := c.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start client: %v", err)
	}
	return c
}

func main() {
	sessionID := uuid.New().String()
	client := lkesdk.NewLkeClient(botAppKey, nil)
	// client.SetMock(true) // mock run

	// 增加 sse 插件
	c := buildSeeMcpClient()
	defer c.Close()
	addTools, err := client.AddMcpTools("Agent-A", c, mcp.Implementation{
		Name:    "test",
		Version: "1,0,0",
	}, nil)
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
		finalReply, err := client.Run(query, sessionID, visitorBizID, options)
		if err != nil {
			log.Fatalf("run error: %v", err)
		}
		log.Printf("finalReply: %v\n", finalReply.Content)
	}
}
