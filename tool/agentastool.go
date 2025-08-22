package tool

import (
	"context"
	"time"
)

// AgentAsTool ...
type AgentAsTool struct {
	Name    string
	Timeout time.Duration
}

// GetName returns the name of the tool
func (m *AgentAsTool) GetName() string {
	return m.Name
}

// GetDescription returns the description of the tool
func (m *AgentAsTool) GetDescription() string {
	return ""
}

// GetParametersSchema returns the JSON schema for the tool parameters
func (m *AgentAsTool) GetParametersSchema() map[string]interface{} {
	return map[string]interface{}{}
}

// Execute executes the tool with the given parameter
func (m *AgentAsTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	return nil, nil
}

// ResultToString ...
func (m *AgentAsTool) ResultToString(output interface{}) string {
	return ""
}

// GetTimeout 获取超时时间
func (m *AgentAsTool) GetTimeout() time.Duration {
	return m.Timeout
}

// SetTimeout 工具输出结果转换成 string
func (m *AgentAsTool) SetTimeout(t time.Duration) {
	m.Timeout = t
}
