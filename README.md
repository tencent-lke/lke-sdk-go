# lke-sdk-go
腾讯云大模型知识引擎(lke) golang sdk。

相关概念和 API 请参考 [腾讯云大模型知识引擎对话接口文档](https://cloud.tencent.com/document/product/1759/105561)。

## Install
`go get github.com/tencent-lke/lke-sdk-go`

## Usage

1. 创建 client
```go
const botAppKey = "zIIRbxwI"
sessionId := uuid.New().String()
client := lkesdk.NewLkeClient(botAppKey, sessionId)
```

2. 自定义事件处理
```go
type MyEventHandler struct {}

// Reply 自定义回复处理事件
func (MyEventHandler) Reply(reply *event.ReplyEvent) {
	if reply.IsFromSelf {
		// 过滤输入重复回包
		return
	}
	log.Printf("Reply: %v", reply.Content)
}

// Reply 回复处理
func (MyEventHandler) Reply(reply *event.ReplyEvent) {}

// Thought 思考过程处理
func (MyEventHandler) Thought(thought *event.AgentThoughtEvent) {}

// Reference 引用事件处理
func (MyEventHandler) Reference(refer *event.ReferenceEvent) {}

// TokenStat token 统计事件
func (MyEventHandler) TokenStat(stat *event.TokenStatEvent) {}

```

3. 增加自定义事件处理到 client
```go
client.SetEventHandler(&MyEventHandler{}) // 配置自定义事件处理
```

4. 循环对话
```go
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
  finalReply, err := client.Chat(input, options) // 阻塞调用
  if err != nil {
    log.Fatalf("chat 出错: %v", err)
  }
  log.Printf("finalReply: %v\n", finalReply.Content)
}
```

## Example
`go run example/general/main.go`

详细示例查看 [main.go](https://github.com/tencent-lke/lke-sdk-go/blob/main/example/general/main.go)

## Tool
[使用 tool 教程](https://github.com/tencent-lke/lke-sdk-go/blob/main/tool/readme.md)