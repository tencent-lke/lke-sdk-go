package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

type mcpClientCache struct {
	Cli           client.MCPClient
	Data          map[string]mcp.Tool
	OrderedName   []string
	LastFetchTime time.Time
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

func (cache *mcpClientCache) GetParametersSchema(name string) map[string]interface{} {
	cache.fetch()
	schema := map[string]interface{}{}
	if info, ok := cache.Data[name]; ok {
		bs, _ := json.Marshal(info.InputSchema)
		_ = json.Unmarshal(bs, &schema)
		replaceDefaultWithJson(schema)
		return schema
	}
	return schema
}

// 构建一个新的 mcp client cache
func NewMcpClientCache(cli client.MCPClient) (*mcpClientCache, error) {
	if cli == nil {
		return nil, fmt.Errorf("mcp client is nil")
	}
	rsp, err := ListMcpTools(cli)
	if err != nil {
		return nil, fmt.Errorf("mcp client is list tools error: %v", err)
	}
	cache := &mcpClientCache{
		Cli:           cli,
		LastFetchTime: time.Now(),
		Data:          map[string]mcp.Tool{},
		OrderedName:   []string{},
	}
	for _, tool := range rsp.Tools {
		cache.Data[tool.Name] = tool
		cache.OrderedName = append(cache.OrderedName, tool.GetName())
	}
	return cache, nil
}

func (cache *mcpClientCache) fetch() {
	defer func() {
		if p := recover(); p != nil {
			return
		}
	}()
	// 2s 刷新一次
	if time.Now().After(cache.LastFetchTime.Add(2 * time.Second)) {
		rsp, err := ListMcpTools(cache.Cli)
		if err != nil {
			return
		}
		cache.LastFetchTime = time.Now()
		for _, tool := range rsp.Tools {
			cache.Data[tool.Name] = tool
		}
	}
}

// McpTool ...
type McpTool struct {
	Name    string
	Cache   *mcpClientCache
	Timeout time.Duration
}

// GetName returns the name of the tool
func (m *McpTool) GetName() string {
	return m.Name
}

func (cache *mcpClientCache) GetDescription(name string) string {
	cache.fetch()
	if info, ok := cache.Data[name]; ok {
		return info.Description
	}
	return ""
}

// GetDescription returns the description of the tool
func (m *McpTool) GetDescription() string {
	if m.Cache != nil {
		return m.Cache.GetDescription(m.GetName())
	}
	return ""
}

// GetParametersSchema returns the JSON schema for the tool parameters
func (m *McpTool) GetParametersSchema() map[string]interface{} {
	if m.Cache != nil {
		return m.Cache.GetParametersSchema(m.GetName())
	}
	return map[string]interface{}{}
}

// Execute executes the tool with the given parameter
func (m *McpTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if m.Cache == nil || m.Cache.Cli == nil {
		return nil, fmt.Errorf("mcp client is nil")
	}
	req := mcp.CallToolRequest{}
	req.Params.Name = m.Name
	req.Params.Arguments = params
	toolCtx, toolCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer toolCancel()
	errp := m.Cache.Cli.Ping(toolCtx)
	if errp != nil {
		cli := m.Cache.Cli.(*client.Client)
		cli.GetTransport().Start(ctx)
		m.Cache.Cli = cli
	}
	result, err := m.Cache.Cli.CallTool(ctx, req)
	if err != nil {
		return nil, err
	}
	return result, nil
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
