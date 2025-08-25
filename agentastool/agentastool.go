package agentastool

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tencent-lke/lke-sdk-go/model"
	"github.com/tencent-lke/lke-sdk-go/runner"
	"github.com/tencent-lke/lke-sdk-go/tool"
)

// AgentAsTool ...
type AgentAsTool struct {
	Name         string        // Tool名称
	Description  string        // Tool描述
	AgentName    string        // Agent名称
	Timeout      time.Duration // 超时配置
	InputSchema  string        // 输入参数schema
	Tools        []tool.Tool   // agent需要调用的tools
	BotAppKey    string
	RequestID    string
	VisitorBizID string
	Conf         runner.RunnerConf
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
			"query": map[string]interface{}{
				"type":        "string",
				"description": "The request to send to the agent",
			},
		},
		"required": []string{"query"},
	}
}

// Execute executes the tool with the given parameter
func (m *AgentAsTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	query, ok := params["query"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid query parameter")
	}
	agents := []model.Agent{}
	toolsMap := map[string][]tool.Tool{}
	handoffs := []model.Handoff{}
	// toolsMap := map[string][]tool.Tool{}
	// for _, tool := range m.Tools {
	// 	toolFuncs = append(toolFuncs, tool)
	// 	toolsMap[m.AgentName] = toolFuncs
	// }
	toolsMap[m.AgentName] = m.Tools
	// handoffs := []model.Handoff{}
	runner := runner.NewRunnerImp(toolsMap, agents, handoffs, m.Conf)
	options := &model.Options{StreamingThrottle: 20,
		CustomVariables: map[string]string{
			"_user_guid":    m.VisitorBizID,
			"_user_task_id": m.RequestID,
		}}
	sessionID := uuid.New().String()
	result, err := runner.RunWithContext(ctx, query, m.RequestID, sessionID, m.VisitorBizID, options)
	if err != nil {
		return nil, err
	}
	return m.ResultToString(result), nil
}

// ResultToString ...
func (m *AgentAsTool) ResultToString(output interface{}) string {
	str, _ := tool.InterfaceToString(output)
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
