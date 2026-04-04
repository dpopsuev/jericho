package mcp

import (
	"context"
	"errors"
	"fmt"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCP tool errors.
var (
	ErrUnknownAction   = errors.New("unknown workers action")
	ErrSessionRequired = errors.New("session is required")
)

// WorkerManager is the minimal interface for MCP tool integration.
// Consumers provide their own implementation (e.g. workload.Controller).
type WorkerManager interface {
	Start(ctx context.Context, session string, count int) error
	Stop() error
	Health() map[string]any
}

// WorkersInput is the typed MCP tool input.
type WorkersInput struct {
	Action  string `json:"action"`
	Session string `json:"session,omitempty"`
	Count   int    `json:"count,omitempty"`
}

// RegisterTool adds the workers MCP tool to the given server.
func RegisterTool(server *sdkmcp.Server, mgr WorkerManager) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "workers",
		Description: "Agent worker management. Actions: start (spawn N workers), stop (kill all), health (status).",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input *WorkersInput) (*sdkmcp.CallToolResult, any, error) {
		switch input.Action {
		case "start":
			if input.Session == "" {
				return nil, nil, ErrSessionRequired
			}
			if err := mgr.Start(ctx, input.Session, input.Count); err != nil {
				return nil, nil, err
			}
			return nil, mgr.Health(), nil

		case "stop":
			if err := mgr.Stop(); err != nil {
				return nil, nil, err
			}
			return nil, map[string]any{"status": "stopped"}, nil

		case "health":
			return nil, mgr.Health(), nil

		default:
			return nil, nil, fmt.Errorf("%w: %q; valid actions: start, stop, health", ErrUnknownAction, input.Action)
		}
	})
}

// NewMCPServer creates an MCP server with the workers tool registered.
func NewMCPServer(mgr WorkerManager) *sdkmcp.Server {
	server := sdkmcp.NewServer(
		&sdkmcp.Implementation{Name: "troupe-orchestrator", Version: "v0.1.0"},
		nil,
	)
	RegisterTool(server, mgr)
	return server
}

// ServeStdio runs the MCP server over stdio (for Claude Code integration).
func ServeStdio(ctx context.Context, mgr WorkerManager) error {
	server := NewMCPServer(mgr)
	transport := &sdkmcp.StdioTransport{}
	_, err := server.Connect(ctx, transport, nil)
	if err != nil {
		return fmt.Errorf("stdio connect: %w", err)
	}
	<-ctx.Done()
	return nil
}
