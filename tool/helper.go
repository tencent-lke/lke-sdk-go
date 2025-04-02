package tool

import (
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
	"github.com/openai/openai-go/shared"
)

// ToOpenAITool converts a Tool to the OpenAI tool format
func ToOpenAIToolPB(tool Tool) *openai.FunctionToolParam {
	return &openai.FunctionToolParam{
		Type: "function",
		Function: shared.FunctionDefinitionParam{
			Name:        tool.GetName(),
			Description: param.Opt[string]{Value: tool.GetDescription()},
			Parameters:  tool.GetParametersSchema(),
		},
	}
}

// ToOpenAITool converts a Tool to the OpenAI tool format
func ToOpenAITool(tool Tool) map[string]interface{} {
	return map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name":        tool.GetName(),
			"description": tool.GetDescription(),
			"parameters":  tool.GetParametersSchema(),
		},
	}
}

// ToOpenAITools converts a slice of Tools to the OpenAI tool format
func ToOpenAITools(tools []Tool) []map[string]interface{} {
	result := make([]map[string]interface{}, len(tools))
	for i, tool := range tools {
		result[i] = ToOpenAITool(tool)
	}
	return result
}

// CreateToolFromDefinition creates a Tool from an OpenAI tool definition
func CreateToolFromDefinition(definition map[string]interface{}, executeFn func(map[string]interface{}) (interface{}, error)) Tool {
	// Extract function details
	functionDef := definition["function"].(map[string]interface{})
	name := functionDef["name"].(string)
	description := functionDef["description"].(string)
	parameters := functionDef["parameters"].(map[string]interface{})

	// Create a new function tool
	tool, _ := NewFunctionTool(name, description, func(ctx interface{}, params map[string]interface{}) (interface{}, error) {
		return executeFn(params)
	}, parameters)

	// Set the schema
	tool.WithSchema(parameters)

	return tool
}
