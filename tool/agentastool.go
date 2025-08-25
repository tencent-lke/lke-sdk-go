package tool

import (
	"context"
	"time"

	"github.com/tencent-lke/lke-sdk-go/tool"
)

// AgentAsTool ...
type AgentAsTool struct {
	Name        string        // Tool名称
	Description string        // Tool描述
	AgentName   string        // Agent名称
	Timeout     time.Duration // 超时配置
	InputSchema string        // 输入参数schema
	Tools       []tool.Tool   // agent需要调用的tools
}

// GetName returns the name of the tool
func (m *AgentAsTool) GetName() string {
	return m.Name
}

// GetDescription returns the description of the tool
func (m *AgentAsTool) GetDescription() string {
	return m.Description
}

// GetParametersSchema returns the JSON schema for the tool parameters
func (m *AgentAsTool) GetParametersSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"request": map[string]interface{}{
				"type":        "string",
				"description": "The request to send to the agent",
			},
		},
		"required": []string{"request"},
	}
}

// Execute executes the tool with the given parameter
func (m *AgentAsTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	return nil, nil
}

// ResultToString ...
func (m *AgentAsTool) ResultToString(output interface{}) string {
	str, _ := InterfaceToString(output)
	return str
}

// GetTimeout 获取超时时间
func (m *AgentAsTool) GetTimeout() time.Duration {
	return m.Timeout
}

// SetTimeout 工具输出结果转换成 string
func (m *AgentAsTool) SetTimeout(t time.Duration) {
	m.Timeout = t
}
