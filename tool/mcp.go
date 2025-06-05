package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// McpTool ...
type McpTool struct {
	Name        string
	Description string
	Schame      map[string]interface{}
	Cli         client.MCPClient
	Timeout     time.Duration
}

// GetName returns the name of the tool
func (m *McpTool) GetName() string {
	m.fetch()
	return m.Name
}

// GetDescription returns the description of the tool
func (m *McpTool) GetDescription() string {
	m.fetch()
	return m.Description
}

func replaceDefaultWithJson(m map[string]interface{}) error {
	for key, value := range m {
		if key == "default" {
			jsonValue, err := json.Marshal(value)
			if err != nil {
				return err
			}
			m[key] = string(jsonValue)
		} else if nestedMap, ok := value.(map[string]interface{}); ok {
			// 如果值是一个嵌套的 map，则递归处理
			err := replaceDefaultWithJson(nestedMap)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// GetParametersSchema returns the JSON schema for the tool parameters
func (m *McpTool) GetParametersSchema() map[string]interface{} {
	m.fetch()
	return m.Schame
}

// Execute executes the tool with the given parameter
func (m *McpTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	req := mcp.CallToolRequest{}
	req.Params.Name = m.Name
	req.Params.Arguments = params
	result, err := m.Cli.CallTool(ctx, req)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (m *McpTool) fetch() {
	defer func() {
		if p := recover(); p != nil {
			return
		}
	}()

	rsp, err := ListMcpTools(m.Cli)
	if err != nil {
		// 如果失败，继续用缓存数据
		return
	}
	for _, tool := range rsp.Tools {
		if tool.Name == m.Name {
			m.Description = tool.Description
			bs, _ := json.Marshal(tool.InputSchema)
			tmpSchema := map[string]interface{}{}
			if err := json.Unmarshal(bs, &tmpSchema); err == nil {
				m.Schame = tmpSchema
				replaceDefaultWithJson(m.Schame)
			}
		}
	}
}

// ListMcpTools 获取 mcp 工具列表
func ListMcpTools(cli client.MCPClient) (res *mcp.ListToolsResult, err error) {
	ctx := context.Background()
	runCtx, cancel := context.WithCancel(ctx)
	t := time.NewTimer(5 * time.Second)
	defer cancel()
	signal := make(chan struct{})
	go func() {
		defer func() {
			select {
			case <-runCtx.Done():
				return
			case signal <- struct{}{}:
			}
		}()
		if err = cli.Ping(ctx); err != nil {
			return
		}
		toolsRequest := mcp.ListToolsRequest{}
		res, err = cli.ListTools(runCtx, toolsRequest)
	}()
	for {
		select {
		case <-t.C:
			err = fmt.Errorf("ListMcpTools timeout")
			return nil, err
		case <-signal:
			return res, err
		}
	}
}

// ResultToString ...
func (m *McpTool) ResultToString(output interface{}) string {
	result, ok := output.(*mcp.CallToolResult)
	if !ok {
		return ""
	}
	totalResult := []string{}
	for _, content := range result.Content {

		if textContent, ok := content.(mcp.TextContent); ok {
			totalResult = append(totalResult, textContent.Text)
		} else {
			jsonBytes, _ := json.Marshal(content)
			totalResult = append(totalResult, string(jsonBytes))
		}
	}
	if len(totalResult) == 1 {
		return totalResult[0]
	}
	str, _ := InterfaceToString(totalResult)
	return str
}

// GetTimeout 获取超时时间
func (m *McpTool) GetTimeout() time.Duration {
	return m.Timeout
}

// SetTimeout 工具输出结果转换成 string
func (m *McpTool) SetTimeout(t time.Duration) {
	m.Timeout = t
}
