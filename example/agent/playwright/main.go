package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	lkesdk "github.com/tencent-lke/lke-sdk-go"
	"github.com/tencent-lke/lke-sdk-go/event"
	"github.com/tencent-lke/lke-sdk-go/model"
	"github.com/tencent-lke/lke-sdk-go/tool"
)

type myLogger struct {
	F *os.File
}

// Info ...
func (m myLogger) Info(msg string) {
	if m.F != nil {
		m.F.WriteString("INFO: " + msg + "\n")
	}
}

// Error ...
func (m myLogger) Error(msg string) {
	if m.F != nil {
		m.F.WriteString("ERROR: " + msg + "\n")
	}
}

const (
	visitorBizID = "custom-user-id"
	// 获取方法 https://cloud.tencent.com/document/product/1759/105561#8590003a-0a6d-4a8d-9a02-b706221a679d
	// botAppKey = "zIIRbxwI"
	botAppKey = "UcFBYLdzeFlZGGOXvSycRXDGHoUVTBGYSGgtkHkdINBZKNmQUZFgxhXQHidAyzoUGUeNYlFkzgYumUngLjawOurmuTwiDpnKoYVdLRXNQogdzuGaLsCuhWPoCLewNAmr"
)

func buildPlaywrightMcpClient() mcpclient.MCPClient {
	_, f, _, _ := runtime.Caller(0)
	serverPath := path.Join(path.Dir(f), "server.py")
	c, err := mcpclient.NewStdioMCPClient(
		"python3",
		[]string{}, // Empty ENV
		serverPath,
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	return c
}

// MyEventHandler 创建自定义事件处理器
type MyEventHandler struct {
	lastReply                  string
	replying                   bool
	lastThought                string
	lkesdk.DefaultEventHandler // 引用默认实现
}

// OnReply 自定义回复处理事件，使用增量输出 repley
func (e *MyEventHandler) OnReply(reply *event.ReplyEvent) {
	if reply.IsFromSelf {
		// 过滤输入重复回包
		fmt.Printf("\nUser: %s\n", reply.Content)
		return
	}
	if e.lastReply == "" {
		prefix := ""
		for range 20 {
			prefix = prefix + " "
		}
		fmt.Printf("\n%sAssistant(%s): ", prefix, reply.TraceId)
	}
	fmt.Printf("%s", strings.TrimPrefix(reply.Content, e.lastReply))
	e.lastReply = reply.Content
	e.replying = true
	e.lastThought = ""
	if reply.IsFinal {
		fmt.Println("\n")
		e.replying = false
	}
}

// OnReply 自定义思考处理事件,使用增量输出思考过程
func (e *MyEventHandler) OnThought(thought *event.AgentThoughtEvent) {
	if e.replying {
		return
	}
	if len(thought.Procedures) > 0 {
		m := map[string]interface{}{}
		json.Unmarshal([]byte(thought.Procedures[len(thought.Procedures)-1].Debugging.Content), &m)
		out := thought.Procedures[len(thought.Procedures)-1].Debugging.Content
		re := regexp.MustCompile(`"Answer":"(.*?)"`)
		matches := re.FindStringSubmatch(out)
		if len(matches) > 1 {
			out = strings.TrimSuffix(strings.TrimPrefix(matches[0], `"Answer":"`), `"`)
		}

		if e.lastThought == "" || !strings.HasPrefix(out, e.lastThought) {
			prefix := ""
			for range 20 {
				prefix = prefix + " "
			}
			fmt.Printf("\n\n%s思考(%s): %s", prefix, thought.TraceId, out)
		} else {
			fmt.Printf("%s", strings.TrimPrefix(out, e.lastThought))
		}
		e.lastReply = ""
		e.lastThought = out
	}
}

// AfterToolCallHook 工具调用后的钩子
// 如果是自定义的函数，output 类型是自定义函数的返回
// 如果是 mcp 工具，output 是 *mcp.CallToolResult 类型
func (e *MyEventHandler) AfterToolCallHook(t tool.Tool, input map[string]interface{},
	output interface{}, err error) {
	inputbs, _ := json.Marshal(input)
	outbs := []byte{}
	if _, ok := t.(*tool.McpTool); ok {
		mcpRsp, okout := output.(*mcp.CallToolResult)
		if okout {
			outbs, _ = json.Marshal(mcpRsp)
		}
	} else {
		outbs, _ = json.Marshal(output)
	}
	prefix := ""
	for range 20 {
		prefix = prefix + " "
	}
	fmt.Printf("\n\n%s called tools %s, input: %s, output: %s  \n\n", prefix, t.GetName(),
		string(inputbs), string(outbs))
}

func main() {
	sessionID := uuid.New().String()
	client := lkesdk.NewLkeClient(botAppKey, &MyEventHandler{})
	client.SetEndpoint("https://testwss.testsite.woa.com/v1/qbot/chat/experienceSse?qbot_env_set=2_11")
	client.SetEndpoint("https://testwss.testsite.woa.com/v1/qbot/chat/experienceSse")
	c := buildPlaywrightMcpClient() // 启动一个本地浏览器操作 mcp client
	defer c.Close()
	// 定义新闻搜索 agent
	downloadAgent := model.NewAgent(
		"下载助手agent",
		"根据用户输入需求，寻找到合适的下载链接。",
		"一个万能的下载助手",
		model.ModelFunctionCallPro,
	)
	browserAgent := model.NewAgent(
		"浏览器控制 agent",
		"涉及到实际操作浏览器",
		"涉及到实际浏览器控制和操作的需求都可以交给我。",
		model.ModelFunctionCallPro,
	)
	client.AddAgents([]model.Agent{downloadAgent, browserAgent})
	client.AddHandoffs("新闻搜索", []string{browserAgent.Name})
	client.AddHandoffs(downloadAgent.Name, []string{browserAgent.Name})

	addTools, err := client.AddMcpTools(browserAgent.Name, c, mcp.Implementation{
		Name:    "text",
		Version: "1.0.0",
	}, nil)
	if err != nil {
		log.Fatalf("Failed to AddMcpTools, error: %v", err)
	}
	for _, tools := range addTools {
		bs, _ := json.Marshal(tools.GetParametersSchema())
		fmt.Printf("toolname: %s\ndescribe: %s\nschema: %v\n\n",
			tools.GetName(), tools.GetDescription(), string(bs))
	}
	client.SetToolRunTimeout(20 * time.Second) // 设置工具超时时间
	// 设置入口 agent，如果不配置，默认从当前应用的云上的主 agent 开始执行
	client.SetStartAgent(browserAgent.Name)
	client.SetEnableSystemOpt(true)
	f, err := os.OpenFile("./logs.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		client.SetRunLogger(&myLogger{F: f}) // 设置 api 日志打印文件
		defer f.Close()
	}
	fmt.Printf("sessionID: %s\n", sessionID)
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
			CustomVariables:   map[string]string{}, // CustomVariables 调用工具不需要模型自动提取的参数，固定传入用户的参数
		}
		_, err = client.Run(query, sessionID, visitorBizID, options)
		if err != nil {
			log.Fatalf("run error: %v", err)
		}
	}
}
