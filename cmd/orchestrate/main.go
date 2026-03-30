// Command orchestrate is a stdio MCP server that proxies remote tools
// and adds local worker management. Provides a unified facade:
// circuit + signal + workers through one MCP connection.
//
// Register in Claude Code:
//
//	claude mcp add origami -- origami orchestrate --endpoint http://localhost:9000/mcp
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/dpopsuev/bugle/orchestrate"
)

const (
	logKeyError    = "error"
	logKeyEndpoint = "endpoint"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "orchestrator: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	endpoint := flag.String("endpoint", envOr("BUGLE_ENDPOINT", "http://localhost:9000/mcp"), "MCP endpoint to proxy and connect workers to")
	session := flag.String("session", "", "auto-start workers for this session (optional)")
	agent := flag.String("agent", "claude", "agent CLI name")
	count := flag.Int("count", 4, "number of workers (for auto-start)")
	tool := flag.String("tool", "circuit", "MCP tool name for step/submit")
	pullAction := flag.String("pull-action", "step", "action name for pulling work")
	pushAction := flag.String("push-action", "submit", "action name for submitting results")
	sessionKey := flag.String("session-key", "session_id", "session key name in arguments")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cfg := orchestrate.WorkerConfig{
		ToolName:   *tool,
		PullAction: *pullAction,
		PushAction: *pushAction,
		SessionKey: *sessionKey,
	}
	mgr := orchestrate.NewManager(*endpoint, cfg)

	if *session != "" {
		if err := mgr.Start(ctx, *session, *agent, *count); err != nil {
			slog.ErrorContext(ctx, "auto-start failed", slog.Any(logKeyError, err))
			return err
		}
	}

	slog.InfoContext(ctx, "orchestrator starting",
		slog.String(logKeyEndpoint, *endpoint))

	return orchestrate.ServeStdioProxy(ctx, *endpoint, mgr)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
