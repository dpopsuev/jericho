package orchestrate

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Log key constants for proxy.
const (
	logKeyTool     = "tool"
	logKeyToolsKey = "tools"
	logKeyEndpoint = "endpoint"
)

// ProxyServer creates a stdio MCP server that:
// 1. Connects to the remote endpoint as an MCP client
// 2. Discovers all remote tools and re-exposes them
// 3. Adds the local workers tool on top
//
// Claude Code sees one unified tool set: circuit, signal, workers, etc.
func ProxyServer(ctx context.Context, endpoint string, mgr *Manager) (*sdkmcp.Server, *sdkmcp.ClientSession, error) {
	// Connect to remote endpoint.
	transport := &sdkmcp.StreamableClientTransport{Endpoint: endpoint}
	client := sdkmcp.NewClient(
		&sdkmcp.Implementation{Name: "bugle-orchestrator-proxy", Version: "v0.1.0"},
		nil,
	)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("connect to endpoint %s: %w", endpoint, err)
	}

	// Discover remote tools.
	toolList, err := session.ListTools(ctx, nil)
	if err != nil {
		session.Close()
		return nil, nil, fmt.Errorf("list tools from %s: %w", endpoint, err)
	}

	// Create local server.
	server := sdkmcp.NewServer(
		&sdkmcp.Implementation{Name: "bugle-orchestrator", Version: "v0.1.0"},
		nil,
	)

	// Register proxied remote tools.
	toolNames := make([]string, 0, len(toolList.Tools)+1)
	for _, tool := range toolList.Tools {
		toolCopy := *tool
		toolNames = append(toolNames, toolCopy.Name)
		server.AddTool(
			&toolCopy,
			proxyHandler(session, toolCopy.Name),
		)
	}

	// Register local workers tool.
	RegisterTool(server, mgr)
	toolNames = append(toolNames, "workers")

	slog.InfoContext(ctx, "proxy server ready",
		slog.Any(logKeyToolsKey, toolNames),
		slog.String(logKeyEndpoint, endpoint))

	return server, session, nil
}

// ServeStdioProxy runs the unified proxy+workers server over stdio.
func ServeStdioProxy(ctx context.Context, endpoint string, mgr *Manager) error {
	server, remoteSession, err := ProxyServer(ctx, endpoint, mgr)
	if err != nil {
		return err
	}
	defer remoteSession.Close()

	stdioTransport := &sdkmcp.StdioTransport{}
	_, err = server.Connect(ctx, stdioTransport, nil)
	if err != nil {
		return fmt.Errorf("stdio connect: %w", err)
	}
	<-ctx.Done()
	return nil
}

// proxyHandler creates a tool handler that forwards calls to the remote session.
func proxyHandler(session *sdkmcp.ClientSession, toolName string) func(context.Context, *sdkmcp.CallToolRequest) (*sdkmcp.CallToolResult, error) {
	return func(ctx context.Context, req *sdkmcp.CallToolRequest) (*sdkmcp.CallToolResult, error) {
		var args map[string]any
		if req.Params.Arguments != nil {
			_ = json.Unmarshal(req.Params.Arguments, &args)
		}

		result, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
			Name:      toolName,
			Arguments: req.Params.Arguments,
		})
		if err != nil {
			return &sdkmcp.CallToolResult{
				IsError: true,
				Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: fmt.Sprintf("proxy %s: %v", toolName, err)}},
			}, nil
		}
		return result, nil
	}
}
