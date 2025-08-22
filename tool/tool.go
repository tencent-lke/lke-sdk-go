package tool

import (
	"context"
	"time"
)

// Tool represents a capability that can be used by an agent
type Tool interface {
	// GetName returns the name of the tool
	GetName() string

	// GetDescription returns the description of the tool
	GetDescription() string

	// GetParametersSchema returns the JSON schema for the tool parameters
	GetParametersSchema() map[string]interface{}

	// Execute executes the tool with the given parameters
	Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)

	// ResultToString 工具输出结果转换成 string
	ResultToString(result interface{}) string

	// GetTimeout 获取超时时间
	GetTimeout() time.Duration

	// SetTimeout 配置工具超时时间
	SetTimeout(t time.Duration)
}
