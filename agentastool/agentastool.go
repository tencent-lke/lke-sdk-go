package agentastool

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/tencent-lke/lke-sdk-go/model"
	"github.com/tencent-lke/lke-sdk-go/runner"
	"github.com/tencent-lke/lke-sdk-go/tool"
	"github.com/tencent-lke/lke-sdk-go/util"
)

// AgentAsTool ...
type AgentAsTool struct {
	Name        string // Tool名称
	Description string // Tool描述
	// Instructions string        // Tool推广信息
	// ModelName    string        // Tool模型信息
	// AgentName    string        // Agent名称
	Timeout time.Duration // 超时配置
	Agent   model.Agent
	// OutputSchema map[string]interface{} `json:"outputSchema"`
	// InputSchema  map[string]interface{} `json:"inputSchema"`
	Tools        []tool.Tool // agent需要调用的tools
	RequestID    string
	VisitorBizID string
	SessionID    string
	index        int64
	Conf         runner.RunnerConf
	RunnerImpl   *runner.RunnerImp
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
	if m.Agent.InputSchema != nil {
		return m.Agent.InputSchema
	} else {
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
}

// Execute executes the tool with the given parameter
func (m *AgentAsTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	input := ""
	if m.Agent.InputSchema != nil {
		_, err := govalidator.ValidateMap(params, m.Agent.InputSchema)
		if err != nil {
			return nil, fmt.Errorf("failed to validate parameters: %v", err)
		}
		paramsBytes, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal parameters: %v", err)
		}
		input = string(paramsBytes)
	} else {
		query, ok := params["query"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid query parameter")
		}
		input = query
	}
	agents := []model.Agent{m.Agent}
	toolsMap := map[string][]tool.Tool{}
	toolsMap[m.Agent.Name] = m.Tools
	handoffs := []model.Handoff{}
	m.RunnerImpl = runner.NewRunnerImp(toolsMap, agents, handoffs, m.Conf)
	sessionID := fmt.Sprintf("%s_%d", m.SessionID, m.index)
	options := &model.Options{StreamingThrottle: 20,
		CustomVariables: map[string]string{
			"_user_guid":    m.VisitorBizID,
			"_user_task_id": m.SessionID,
		},
	}
	if envSet := util.GetEnvSetFromContext(ctx); envSet != "" {
		options.EnvSet = envSet
	}
	m.index = m.index + 1
	instruction := input + "\n\n" + m.generateJSONInstructions()
	result, err := m.RunnerImpl.RunWithContext(ctx, instruction, m.RequestID, sessionID, m.VisitorBizID, options)
	if err != nil {
		return nil, err
	}
	return m.ResultToString(result), nil
}

// generateJSONInstructions generates JSON output instructions based on the output schema.
func (m *AgentAsTool) generateJSONInstructions() string {
	if m.Agent.OutputSchema == nil {
		return ""
	}
	// Convert schema to a readable format for the instruction
	schemaStr := m.formatSchemaForInstruction(m.Agent.OutputSchema)
	return fmt.Sprintf("IMPORTANT: You must respond with valid JSON in the following format:\n%s\n\n"+
		"Your response must be valid JSON that matches this schema exactly. "+
		"Do not include ```json or ``` in the beginning or end of the response.", schemaStr)
}

// formatSchemaForInstruction formats the schema for inclusion in instructions.
func (p *AgentAsTool) formatSchemaForInstruction(schema map[string]interface{}) string {
	// For now, we'll create a simple JSON representation.
	// In a more sophisticated implementation, we could parse the schema more intelligently.
	jsonBytes, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		// Fallback to a simple string representation.
		return fmt.Sprintf("%v", schema)
	}
	return string(jsonBytes)
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
