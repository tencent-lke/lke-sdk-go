package model

// Handoff 转交关系
type Handoff struct {
	SourceAgentName string `json:"source_agent_name"`
	TargetAgentName string `json:"target_agent_name"`
}
