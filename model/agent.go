package model

// AgentConfig 对话的 agent 配置
type AgentConfig struct {
	StartAgentName   string      `json:"start_agent_name"`   // 入口 agent 的名字，如果不填默认从主 agent 开始执行
	Agents           []Agent     `json:"agents"`             // 每次对话的动态新增 agent
	DisableSystemOpt bool        `json:"disable_system_opt"` // 是否关闭系统内置优化
	Handoffs         []Handoff   `json:"handoffs"`           // 每次对话的动态新增工具
	AgentTools       []AgentTool `json:"agent_tools"`        // 每次对话的动态新增工具
}

// Agent agent 定义
type Agent struct {
	Name         string `json:"name"`
	Instructions string `json:"instructions"`
	Description  string `json:"description"`
	Model        model  `json:"model"`
}

// NewAgent 创建一个新的 Agent 实例
// name agent 的名字
// instructions 指令中定义Agent执行任务和响应方式，可以通过自然语言或者代码表达，也可称为提示词。
// description Agent任务目标的一句话简单介绍，转交时用于了解何时应当转交的描述信息。
// instructions与 description的区别是，description 是给到其他调用Agent的调用方了解何时需要调用Agent的简单描述，而instructions是给到思考模型理解并执行的Agent的详细工作逻辑。
// m 该 agent 需要用到的模型
func NewAgent(name, instruction, description string, m model) Agent {
	return Agent{
		Name:         name,
		Instructions: instruction,
		Description:  description,
		Model:        m,
	}
}
