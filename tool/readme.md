# 使用函数 tool

1. 方式1, 自定义函数，除去 context，入参是一个 struct，并且 struct 中每个字段都有 tag，其中json tag 会转换参数名，doc tag 转换成字段描述
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

2. 方式2，自定义函数，除去 context，入参是一个严格的 map[string]interface{}，并且自定义 json schema。
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

详细示例 [main.go](https://github.com/tencent-lke/lke-sdk-go/blob/main/example/fucntion_tool/main.go)