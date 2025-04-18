package model

// ChatRequest chat 的请求
type ChatRequest struct {
	Options
	Content      string `json:"content"`
	SessionID    string `json:"session_id"`
	BotAppKey    string `json:"bot_app_key"`
	VisitorBizID string `json:"visitor_biz_id"`
}

// Options 定义了发送消息的选项
type Options struct {
	RequestID         string            `json:"request_id,omitempty"`         // 请求ID，用于标识一个请求，建议每个请求使用不同的request_id，便于问题排查
	FileInfos         []FileInfo        `json:"file_infos,omitempty"`         // 文件信息，如果填写该字段，content字段可以为空
	VisitorLabels     []VisitorLabel    `json:"visitor_labels,omitempty"`     // 知识标签，用于知识库中知识的检索过滤（即将下线）
	StreamingThrottle int32             `json:"streaming_throttle,omitempty"` // 流式回复频率控制，控制应用回包频率，默认值5
	CustomVariables   map[string]string `json:"custom_variables,omitempty"`   // 自定义参数，可用于传递参数给工作流或设置知识库检索范围
	SystemRole        string            `json:"system_role,omitempty"`        // 角色指令（提示词），为空时使用应用配置默认设定
	Incremental       bool              `json:"incremental,omitempty"`        // 控制回复事件和思考事件中的content是否是增量输出的内容，默认false
	SearchNetwork     string            `json:"search_network"`
	// 用于端上sdk的参数
	ToolOuputs  []ToolOuput `json:"tool_ouputs"`  // 端上调用工具的输出提交到云上
	AgentConfig AgentConfig `json:"agent_config"` // agent配置
}

// VisitorLabel 定义了知识标签的结构
type VisitorLabel struct {
	Name   string   `json:"name"`   // 知识标签名
	Values []string `json:"values"` // 知识标签值
}

// FileInfo 定义了文件信息的结构
type FileInfo struct {
	FileName string `json:"file_name"` // 文件名称
	FileSize string `json:"file_size"` // 实时文档解析接口返回的文件大小
	FileURL  string `json:"file_url"`  // 实时文档解析接口返回的文件URL
	FileType string `json:"file_type"` // 文件类型
	DocID    string `json:"doc_id"`    // 实时文档解析接口返回的doc_id
}

// ToolOuput 本地工具的输出结果
type ToolOuput struct {
	ToolName string `json:"tool_name"`
	Output   string `json:"output"`
}
