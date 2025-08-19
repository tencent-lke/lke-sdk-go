// Package event contains event definitions.
package mcpserversse

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// McpServerSse
type McpServerSse struct {
	SseUrl      string
	Option      transport.ClientOption
	InitRequest mcp.InitializeRequest
	Cli         *client.Client
}

func (sse *McpServerSse) init() error {
	mcpClient, err := client.NewSSEMCPClient(sse.SseUrl, sse.Option)
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
