package model

import "github.com/openai/openai-go"

// AgentTool Agent 的工具
type AgentTool struct {
	AgentName string                      `json:"agent_name"` // agent 的名字
	Tools     []*openai.FunctionToolParam `json:"tools"`      // 工具列表
}
