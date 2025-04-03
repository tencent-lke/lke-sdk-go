package tool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// McpTool ...
type McpTool struct {
	Name        string
	Description string
	Schame      map[string]interface{}
	Cli         client.MCPClient
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

// GetParametersSchema returns the JSON schema for the tool parameters
func (m *McpTool) GetParametersSchema() map[string]interface{} {
	m.fetch()
	return m.Schame
}

// Execute executes the tool with the given parameter
func (m *McpTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	m.fetch()
	req := mcp.CallToolRequest{}
	req.Params.Name = m.Name
	req.Params.Arguments = params
	result, err := m.Cli.CallTool(ctx, req)
	if err != nil {
		return nil, err
	}
	totalResult := ""
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			totalResult = totalResult + textContent.Text
		} else {
			jsonBytes, _ := json.Marshal(content)
			totalResult = totalResult + string(jsonBytes)
		}
	}
	return totalResult, nil
}

func (m *McpTool) fetch() {
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
			if err := json.Unmarshal(bs, &tmpSchema); err != nil {
				m.Schame = tmpSchema
			}
		}
	}
}

func ListMcpTools(cli client.MCPClient) (*mcp.ListToolsResult, error) {
	ctx := context.Background()
	// Initialize the client
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "mcp-client",
		Version: "1.0.0",
	}
	_, err := cli.Initialize(ctx, initRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize: %v", err)
	}
	toolsRequest := mcp.ListToolsRequest{}
	return cli.ListTools(ctx, toolsRequest)
}
