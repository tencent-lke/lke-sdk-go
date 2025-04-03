package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/google/uuid"
	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	lkesdk "github.com/tencent-lke/lke-sdk-go"
	"github.com/tencent-lke/lke-sdk-go/model"
)

const (
	botAppKey = "xxx"
)

// func compileTestServer() (string, error) {
// 	_, f, _, _ := runtime.Caller(0)
// 	serverPath := path.Join(path.Dir(f), "server", "server.go")
// 	outputPath := path.Join(path.Dir(f), "server", "server")
// 	cmd := exec.Command(
// 		"go",
// 		"build",
// 		"-o",
// 		outputPath,
// 		serverPath,
// 	)
// 	if output, err := cmd.CombinedOutput(); err != nil {
// 		return "", fmt.Errorf("compilation failed: %v\nOutput: %s", err, output)
// 	}
// 	return outputPath, nil
// }

func addCustomStdioMcp(client *lkesdk.LkeClient) {
	_, f, _, _ := runtime.Caller(0)
	serverPath := path.Join(path.Dir(f), "server", "server.go")
	c, err := mcpclient.NewStdioMCPClient(
		"go",
		[]string{}, // Empty ENV
		"run",
		serverPath,
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	addTools, err := client.AddMcpTools("A", c, nil)
	if err != nil {
		log.Fatalf("Failed to AddMcpTools, error: %v", err)
	}
	for _, tools := range addTools {
		bs, _ := json.Marshal(tools.GetParametersSchema())
		fmt.Printf("toolname: %s\ndescribe: %s\nschema: %v\n\n",
			tools.GetName(), tools.GetDescription(), string(bs))
	}
}

func addSSeMcp(client *lkesdk.LkeClient) {
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
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Input parameter: " + request.Params.Arguments["parameter-1"].(string),
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
	addTools, err := client.AddMcpTools("A", c, nil)
	if err != nil {
		log.Fatalf("Failed to AddMcpTools, error: %v", err)
	}
	for _, tools := range addTools {
		bs, _ := json.Marshal(tools.GetParametersSchema())
		fmt.Printf("toolname: %s\ndescribe: %s\nschema: %v\n\n",
			tools.GetName(), tools.GetDescription(), string(bs))
	}
}

func addFileSystemStdioMcp(client *lkesdk.LkeClient) {
	// 需要先安装 nodejs
	c, err := mcpclient.NewStdioMCPClient(
		"npx",
		[]string{}, // Empty ENV
		"-y",
		"@modelcontextprotocol/server-filesystem",
		"/tmp",
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	// addTools, err := client.AddMcpTools("A", c, nil)
	addTools, err := client.AddMcpTools("A", c, []string{"write_file", "move_file"})
	if err != nil {
		log.Fatalf("Failed to AddMcpTools, error: %v", err)
	}
	for _, tools := range addTools {
		bs, _ := json.Marshal(tools.GetParametersSchema())
		fmt.Printf("toolname: %s\ndescribe: %s\nschema: %v\n\n",
			tools.GetName(), tools.GetDescription(), string(bs))
	}
}

func main() {
	sessionId := uuid.New().String()
	client := lkesdk.NewLkeClient(botAppKey, sessionId)
	client.SetMock(true)
	addCustomStdioMcp(client)     // 增加自定义 mcp 插件
	addFileSystemStdioMcp(client) // 增加 npx mcp 插件
	addSSeMcp(client)             // 增加 sse 插件
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
