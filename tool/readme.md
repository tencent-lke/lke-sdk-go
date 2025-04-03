# 使用函数 tool

## fucntion tool

### Usage
`client.AddFunctionTools("agentA", tools)`

其中 agentA 是需要增加 tools 的 agent 名字，tools 是函数插件列表


1. 方式1：自定义函数，除去 context，入参是一个 struct，并且 struct 中每个字段都有 tag，其中json tag 会转换参数名，doc tag 转换成字段描述。输出除了 error，只能有一个参数。


```go
tools := []*tool.FunctionTool{}
t, err := tool.NewFunctionTool("GetWeather", "查询天气", GetWeather, nil)
if err == nil {
  tools = append(tools, t)
} else {
  log.Panicf("不支持的函数定义: %v", err)
}
// 给 agentA 增加 tools
client.AddFunctionTools("agentA", tools)
```

2. 方式2：自定义函数，除去 context，入参是一个严格的 map[string]interface{}，并且自定义 json schema。输出除了 error，只能有一个参数。
```go
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

t1, err := tool.NewFunctionTool("GetWeather2", "查询天气", GetWeather2, schema)
if err == nil {
  tools = append(tools, t1)
} else {
  log.Panicf("不支持的函数定义: %v", err)
}
// agentB 增加 tools
client.AddFunctionTools("agentB", tools)
```

### Example
`go run example/fucntion_tool/main.go`

详细示例 [main.go](https://github.com/tencent-lke/lke-sdk-go/blob/main/example/fucntion_tool/main.go)

## mcp tool

### Usage
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
addTools, err := client.AddMcpTools("A", c, nil)
if err != nil {
  log.Fatalf("Failed to AddMcpTools, error: %v", err)
}
for _, tools := range addTools {
  bs, _ := json.Marshal(tools.GetParametersSchema())
  fmt.Printf("toolname: %s\ndescribe: %s\nschema: %v\n\n",
    tools.GetName(), tools.GetDescription(), string(bs))
}
```

### Example
`go run example/mcp_tool/main.go`

详细示例 [main.go](https://github.com/tencent-lke/lke-sdk-go/blob/main/example/mcp_tool/main.go)
