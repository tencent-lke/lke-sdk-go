// Package event contains event definitions.
package mcpserversse

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// McpServerSse
type McpServerSse struct {
	SseUrl               string
	Options              []transport.ClientOption
	InitRequest          mcp.InitializeRequest
	ClientSessionTimeout int64
	Cli                  *client.Client
}

func NewMcpServerSse(sseurl string, options []transport.ClientOption, initrequest mcp.InitializeRequest, clientsessiontimeout int64) *McpServerSse {
	mcpsse := &McpServerSse{
		SseUrl:               sseurl,
		Options:              options,
		InitRequest:          initrequest,
		ClientSessionTimeout: clientsessiontimeout,
	}
	mcpsse.init()
	return mcpsse
}

func isHTTPURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

func (sse *McpServerSse) initlocal() error {
	mcpClient, err := client.NewStdioMCPClient(
		"python3",
		[]string{}, // Empty ENV
		sse.SseUrl,
	)
	sse.Cli = mcpClient
	return err
}

func (sse *McpServerSse) init() error {
	if !isHTTPURL(sse.SseUrl) {
		return sse.initlocal()
	}
	options := sse.Options
	if sse.ClientSessionTimeout > 0 {
		httpClient := &http.Client{
			Timeout: time.Duration(sse.ClientSessionTimeout) * time.Second,
		}
		options = append(options, transport.WithHTTPClient(httpClient))
	}
	mcpClient, err := client.NewSSEMCPClient(sse.SseUrl, options...)
	if err != nil {
		return err
	}
	if err := mcpClient.Start(context.Background()); err != nil {
		return err
	}
	_, err = mcpClient.Initialize(context.Background(), sse.InitRequest)
	if err != nil {
		return fmt.Errorf("failed to initialize: %v, %v", err, sse.InitRequest)
	}
	sse.Cli = mcpClient
	return nil
}

func (sse *McpServerSse) Init() error {
	return sse.init()
}

func (sse *McpServerSse) ReConnect() error {
	return sse.init()
}

func (sse *McpServerSse) Ping(ctx context.Context) error {
	return sse.Cli.Ping(ctx)
}

func (sse *McpServerSse) ListTools(ctx context.Context, request mcp.ListToolsRequest) (*mcp.ListToolsResult, error) {
	return sse.Cli.ListTools(ctx, request)
}

func (sse *McpServerSse) CallTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return sse.Cli.CallTool(ctx, request)
}
