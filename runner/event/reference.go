package event

// 参考来源类型
const (
	ReferTypeQA      = 1
	ReferTypeSegment = 2
	ReferTypeDoc     = 3
)

// OnReference 参考来源
type Reference struct {
	ID       uint64 `json:"id,string"`
	Type     uint32 `json:"type"`
	URL      string `json:"url"`
	Name     string `json:"name"`
	DocID    uint64 `json:"doc_id,string"`
	DocBizID uint64 `json:"doc_biz_id,string"` // 前端需要biz id用于反馈
	DocName  string `json:"doc_name"`
	QABizID  uint64 `json:"qa_biz_id,string"`
}

// EventReference 参考来源事件
const EventReference = "reference"

// ReferenceEvent 参考来源事件消息体
type ReferenceEvent struct {
	RecordID   string      `json:"record_id"`
	References []Reference `json:"references"`
	Extend     EventExtend `json:"extend,omitempty"`
}

// Name 事件名称
func (e ReferenceEvent) Name() string {
	return EventReference
}
