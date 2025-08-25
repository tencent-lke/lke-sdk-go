package runner

import (
	"context"

	"github.com/tencent-lke/lke-sdk-go/event"
	"github.com/tencent-lke/lke-sdk-go/model"
)

// Tool represents a capability that can be used by an agent
type Runner interface {
	// RunWithTimeout(ctx context.Context, f tool.Tool,
	// 	input map[string]interface{}) (output interface{}, err error)
	// RunTools(ctx context.Context, req *model.ChatRequest,
	// 	reply *event.ReplyEvent, output *[]string)
	RunWithContext(ctx context.Context,
		query, requestID, sessionID, visitorBizID string,
		options *model.Options) (finalReply *event.ReplyEvent, err error)
}
