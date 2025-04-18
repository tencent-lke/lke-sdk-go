package main

import (
	"bufio"
	"context"
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
	visitorBizID = "custom-visior-id" // 访问者 id
	// 获取方法 https://cloud.tencent.com/document/product/1759/105561#8590003a-0a6d-4a8d-9a02-b706221a679d
	botAppKey = "custom-app-key"
)

// MyEventHandler 自定义事件处理器
type MyEventHandler struct {
	lkesdk.DefaultEventHandler // 引用默认实现
}

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

// GetWeather2 获取天气
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

// OnReply 自定义回复处理事件
func (MyEventHandler) OnReply(reply *event.ReplyEvent) {
	if reply.IsFromSelf {
		// 过滤输入重复回包
		return
	}
	log.Printf("Reply: %v", reply.Content)
}

func main() {
	sessionID := uuid.New().String()
	client := lkesdk.NewLkeClient(botAppKey, &MyEventHandler{})
	client.SetMock(true)
	// 方式1, 自定义函数，除去 context，入参是一个 struct，并且 struct 中每个字段都有 tag,
	// json tag 会转换参数名，doc tag 转换成字段描述
	tools := []*tool.FunctionTool{}
	t, err := tool.NewFunctionTool("GetWeather", "查询天气", GetWeather, nil)
	if err == nil {
		tools = append(tools, t)
	} else {
		log.Panicf("不支持的函数定义: %v", err)
	}
	client.AddFunctionTools("agentA", tools)

	// 方式2，自定义函数，除去 context，入参是一个严格的 map[string]interface{}，并且定义 json schema
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
	client.AddFunctionTools("agentA", tools)
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
