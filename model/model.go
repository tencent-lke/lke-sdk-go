package model

import "fmt"

// ModelName 模型名类型
type ModelName string

// model 模型配置
type model struct {
	ModelName   ModelName `json:"model_name"`
	Temperature float32   `json:"temperature"`
	TopP        float32   `json:"top_p"`
}

// 模型枚举，目前支持的 function call 模型
const (
	FunctionCallPro ModelName = "function-call-pro"
	DeepSeekR1      ModelName = "lke-deepseek-r1"
	DeepSeekV30324  ModelName = "lke-deepseek-v3-function-call"
)

// ModelFunctionCallPro pro 模型
var ModelFunctionCallPro = model{
	ModelName:   FunctionCallPro,
	Temperature: 0.5,
	TopP:        0.5,
}

// ModelDeepSeekR1 R1模型
var ModelDeepSeekR1 = model{
	ModelName:   DeepSeekR1,
	Temperature: 0.5,
	TopP:        0.5,
}

// ModelDeepSeekV3 V3 0324 模型
var ModelDeepSeekV30324 = model{
	ModelName:   DeepSeekV30324,
	Temperature: 0.5,
	TopP:        0.5,
}

// DefaultModel 默认模型
var DefaultModel = ModelFunctionCallPro

// NewModel ...
func NewModel(modelName ModelName) (model, error) {
	switch modelName {
	case FunctionCallPro, DeepSeekR1:
		return model{
			ModelName:   modelName,
			Temperature: 0.5,
			TopP:        0.5,
		}, nil
	}
	return model{}, fmt.Errorf("unsupport mode name %s", modelName)
}

// NewModel ...
func NewModelWithParam(modelName ModelName, temperature, topP float32) (model, error) {
	switch modelName {
	case FunctionCallPro, DeepSeekR1:
		return model{
			ModelName:   modelName,
			Temperature: temperature,
			TopP:        topP,
		}, nil
	}
	return model{}, fmt.Errorf("unsupport mode name %s", modelName)
}
