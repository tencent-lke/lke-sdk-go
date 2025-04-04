# lke-sdk-go
腾讯云大模型知识引擎(lke) golang sdk。

相关概念和 API 请参考 [腾讯云大模型知识引擎对话接口文档](https://cloud.tencent.com/document/product/1759/105561)。

## Install
`go get github.com/tencent-lke/lke-sdk-go`

## Usage

1. 创建 client
```go
const (
  // 获取方法 https://cloud.tencent.com/document/product/1759/105561#8590003a-0a6d-4a8d-9a02-b706221a679d
  botAppKey = "custom-app-key"
)

// MyEventHandler 创建自定义事件处理器
type MyEventHandler struct {
  lkesdk.DefaultEventHandler // 引用默认实现
}

// Reply 自定义回复处理事件
func (MyEventHandler) Reply(reply *event.ReplyEvent) {
  if reply.IsFromSelf {
    // 过滤输入重复回包
    return
  }
  log.Printf("Reply: %v", reply.Content)
}

// Reply 自定义思考处理事件
func (MyEventHandler) Thought(thought *event.AgentThoughtEvent) {
  if len(thought.Procedures) > 0 {
    log.Printf("Thought: %s\n", thought.Procedures[len(thought.Procedures)-1].Debugging.Content)
  }
}

// 创建一个 client
client := lkesdk.NewLkeClient(botAppKey, sessionID)
```

2. 循环对话
```go
sessionID := uuid.New().String() // 生成唯一对话 id
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
  }
  finalReply, err := client.Run(input, sessionID, options)
  if err != nil {
    log.Fatalf("run error: %v", err)
  }
  log.Printf("finalReply: %v\n", finalReply.Content)
}
```

## Example
`go run example/general/main.go`

详细示例查看 [main.go](https://github.com/tencent-lke/lke-sdk-go/blob/main/example/general/main.go)

## 使用本地 tool

### fucntion tool

#### Usage
`client.AddFunctionTools("agentA", tools)`

其中 agentA 是需要增加 tools 的 agent 名字，tools 是函数插件列表


1. 方式1：自定义函数，除去 context，入参是一个 struct，并且 struct 中每个字段都有 tag，其中json tag 会转换参数名，doc tag 转换成字段描述。输出除了 error，只能有一个参数。


```go
// Location 地址
type Location struct {
  Address   string  `json:"Address" doc:"address of location"`
  Latitude  float32 `json:"Latitude" doc:"latitude of location"`
  Longitude float32 `json:"Longitude" doc:"longitude of location"`
}

// GetWeatherParams 获取天气的输入
type GetWeatherParams struct {
  Location Location `json:"Location" doc:"the location where you want to fetch the weather"`
  Date     string   `json:"Date" doc:"date of the weather"`
}

// GetWeather 获取天气
func GetWeather(ctx context.Context, params GetWeatherParams) (string, error) {
  str, _ := tool.InterfaceToString(params)
  fmt.Printf("call get weather: %s\n", str)
  return fmt.Sprintf("%s%s日天气很好", params.Location.Address, params.Date), nil
}

// 使用 fucntion 构建 tool
tools := []*tool.FunctionTool{}
t, err := tool.NewFunctionTool("GetWeather", "查询天气", GetWeather, nil)
if err == nil {
  tools = append(tools, t)
} else {
  log.Panicf("不支持的函数定义: %v", err)
}

// 给 Agent-A 增加 tools
client.AddFunctionTools("Agent-A", tools)
```

2. 方式2：自定义函数，除去 context，入参是一个严格的 map[string]interface{}，并且自定义 json schema。输出除了 error，只能有一个参数。

```go
// GetWeather2 获取天气2
func GetWeather2(ctx context.Context, params map[string]interface{}) (string, error) {
  str, _ := tool.InterfaceToString(params)
  fmt.Printf("call get weather2: %s\n", str)
  date, ok := params["Date"].(string)
  if !ok {
    return "", fmt.Errorf("miss Date param")
  }
  location, ok := params["Location"].(map[string]interface{})
  if !ok {
    return "", fmt.Errorf("miss Location param")
  }
  address, ok := location["Address"].(string)
  if !ok {
    return "", fmt.Errorf("miss Location.Address param")
  }
  return fmt.Sprintf("%s%s日天气很好", date, address), nil
}

// 自定义 json schema
schema := map[string]interface{}{
  "properties": map[string]interface{}{
    "Date": map[string]interface{}{
      "description": "date of the weather",
      "type":        "string",
    },
    "Location": map[string]interface{}{
      "description": "the location where you want to check the weather",
      "properties": map[string]interface{}{
        "Address": map[string]interface{}{
          "description": "address of location",
          "type":        "string",
        },
        "Latitude": map[string]interface{}{
          "description": "latitude of location",
          "type":        "number",
        },
        "Longitude": map[string]interface{}{
          "description": "longitude of location",
          "type":        "number",
        },
      },
      "required": []string{"Address", "Latitude", "Longitude"},
      "type":     "object",
    },
  },
  "required": []string{"Location", "Date"},
  "type":     "object",
}

// 使用 function + 自定义 json schema 构建 tool
t1, err := tool.NewFunctionTool("GetWeather2", "查询天气", GetWeather2, schema)
if err == nil {
  tools = append(tools, t1)
} else {
  log.Panicf("不支持的函数定义: %v", err)
}
// Agent-B 增加 tools
client.AddFunctionTools("Agent-B", tools)
```

#### Example
`go run example/fucntion_tool/main.go`

详细示例 [main.go](https://github.com/tencent-lke/lke-sdk-go/blob/main/example/fucntion_tool/main.go)

### mcp tool

#### Usage
`client.AddMcpTools("A", c, []string{"write_file", "move_file"})`

其中 agentA 是需要增加 tools 的 agent, c 是 mcp client 对象，可以选择加入哪些 tools，不选择默认增加全部 tools。



1. 增加自定义 stdio mcp
```go
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
  addTools, err := client.AddMcpTools("Agent-A", c, nil)
  if err != nil {
    log.Fatalf("Failed to AddMcpTools, error: %v", err)
  }
  for _, tools := range addTools {
    bs, _ := json.Marshal(tools.GetParametersSchema())
    fmt.Printf("toolname: %s\ndescribe: %s\nschema: %v\n\n",
      tools.GetName(), tools.GetDescription(), string(bs))
  }
}
```

2. 增加第三方 stdio mcp tool

```go
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
  // addTools, err := client.AddMcpTools("Agent-A", c, nil)
  addTools, err := client.AddMcpTools("Agent-A", c, []string{"write_file", "move_file"})
  if err != nil {
    log.Fatalf("Failed to AddMcpTools, error: %v", err)
  }
  for _, tools := range addTools {
    bs, _ := json.Marshal(tools.GetParametersSchema())
    fmt.Printf("toolname: %s\ndescribe: %s\nschema: %v\n\n",
      tools.GetName(), tools.GetDescription(), string(bs))
  }
}
```


3. 增加第三方 sse mcp tool

```go
sseUrl := "https://xxxx.com/sse"
c, err := mcpclient.NewSSEMCPClient(sseUrl)
if err != nil {
  log.Fatalf("Failed to connect sse server: %v", err)
}
if err := c.Start(context.Background()); err != nil {
  log.Fatalf("Failed to start client: %v", err)
}
addTools, err := client.AddMcpTools("Agent-A", c, nil)
if err != nil {
  log.Fatalf("Failed to AddMcpTools, error: %v", err)
}
for _, tools := range addTools {
  bs, _ := json.Marshal(tools.GetParametersSchema())
  fmt.Printf("toolname: %s\ndescribe: %s\nschema: %v\n\n",
    tools.GetName(), tools.GetDescription(), string(bs))
}
```

#### Example
`go run example/mcp_tool/main.go`

详细示例 [main.go](https://github.com/tencent-lke/lke-sdk-go/blob/main/example/mcp_tool/main.go)

