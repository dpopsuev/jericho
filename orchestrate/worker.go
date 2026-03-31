// Package orchestrate manages agent workers that connect to MCP endpoints
// and loop pull/push via the Bugle Protocol.
package orchestrate

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dpopsuev/jericho/bugle"
)

// Log key constants for sloglint compliance.
const (
	logKeyWorker = "worker"
	logKeyItems  = "items"
	logKeyItem   = "item"
	logKeyError  = "error"
)

// WorkerConfig configures the Bugle Protocol pull/push loop.
type WorkerConfig struct {
	// MCP tool name (default: bugle.DefaultToolName).
	ToolName string
	// Action for pulling work (default: bugle.ActionPull).
	PullAction string
	// Action for pushing results (default: bugle.ActionPush).
	PushAction string
	// Session key name in arguments (default: bugle.DefaultSessionKey).
	SessionKey string
	// AndonFunc is called before push to report worker health. Nil = omit.
	AndonFunc func() *bugle.Andon
	// BudgetFunc is called before push to report resource consumption. Nil = omit.
	BudgetFunc func() *bugle.BudgetActual
	// OnPull is called after each pull response with protocol metadata. Nil = no-op.
	OnPull func(bugle.PullMeta)
}

func (c *WorkerConfig) defaults() {
	if c.ToolName == "" {
		c.ToolName = bugle.DefaultToolName
	}
	if c.PullAction == "" {
		c.PullAction = string(bugle.ActionPull)
	}
	if c.PushAction == "" {
		c.PushAction = string(bugle.ActionPush)
	}
	if c.SessionKey == "" {
		c.SessionKey = bugle.DefaultSessionKey
	}
}

// RunWorker is the Bugle Protocol client loop: pull work, send to responder,
// push results. Blocks until done or ctx canceled.
//
// The caller owns agent lifecycle and MCP connection — RunWorker is pure protocol.
//
//nolint:funlen // protocol loop with pull/push/abort/blocked paths
func RunWorker(ctx context.Context, session *sdkmcp.ClientSession, responder bugle.Responder, sessionID, workerID string, cfg WorkerConfig) error {
	cfg.defaults()

	items := 0
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		pullArgs := map[string]any{
			"action":       cfg.PullAction,
			cfg.SessionKey: sessionID,
			"worker_id":    workerID,
			"timeout_ms":   30000,
		}
		result, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
			Name:      cfg.ToolName,
			Arguments: marshalArgs(pullArgs),
		})
		if err != nil {
			return fmt.Errorf("%s/%s: %w", cfg.ToolName, cfg.PullAction, err)
		}

		if result.IsError {
			return fmt.Errorf("%w: %s", ErrStepFailed, textContent(result))
		}

		text := textContent(result)
		var pullResp bugle.PullResponse
		if err := json.Unmarshal([]byte(text), &pullResp); err != nil {
			return fmt.Errorf("parse pull response: %w", err)
		}

		// Notify callback with protocol metadata.
		if cfg.OnPull != nil {
			cfg.OnPull(bugle.PullMeta{
				Andon:           pullResp.Andon,
				BudgetRemaining: pullResp.BudgetRemaining,
			})
		}

		// Andon dead = abort signal.
		if pullResp.Andon == bugle.AndonDead {
			slog.WarnContext(ctx, "abort signal received",
				slog.String(logKeyWorker, workerID))
			return nil
		}

		if pullResp.Done {
			slog.InfoContext(ctx, "work complete",
				slog.String(logKeyWorker, workerID),
				slog.Int(logKeyItems, items))
			return nil
		}
		if !pullResp.Available {
			continue
		}

		response, err := responder.Respond(ctx, pullResp.PromptContent)
		if err != nil {
			slog.ErrorContext(ctx, "responder failed",
				slog.String(logKeyWorker, workerID),
				slog.String(logKeyItem, pullResp.Item),
				slog.Any(logKeyError, err))

			// Push as blocked instead of silently continuing.
			pushBlocked(ctx, session, cfg, sessionID, workerID, pullResp, err)
			continue
		}

		pushArgs := map[string]any{
			"action":       cfg.PushAction,
			cfg.SessionKey: sessionID,
			"worker_id":    workerID,
			"dispatch_id":  pullResp.DispatchID,
			"item":         pullResp.Item,
			"fields":       json.RawMessage(response),
		}
		if cfg.AndonFunc != nil {
			if a := cfg.AndonFunc(); a != nil {
				pushArgs["andon"] = a
			}
		}
		if cfg.BudgetFunc != nil {
			if b := cfg.BudgetFunc(); b != nil {
				pushArgs["budget_actual"] = b
			}
		}

		_, err = session.CallTool(ctx, &sdkmcp.CallToolParams{
			Name:      cfg.ToolName,
			Arguments: marshalArgs(pushArgs),
		})
		if err != nil {
			slog.WarnContext(ctx, "push failed",
				slog.String(logKeyWorker, workerID),
				slog.String(logKeyItem, pullResp.Item),
				slog.Any(logKeyError, err))
		}
		items++
	}
}

// pushBlocked sends a blocked status when the responder fails.
func pushBlocked(ctx context.Context, session *sdkmcp.ClientSession, cfg WorkerConfig, sessionID, workerID string, pullResp bugle.PullResponse, respondErr error) {
	blockedArgs := map[string]any{
		"action":       cfg.PushAction,
		cfg.SessionKey: sessionID,
		"worker_id":    workerID,
		"dispatch_id":  pullResp.DispatchID,
		"item":         pullResp.Item,
		"status":       bugle.StatusBlocked,
		"fields":       map[string]any{"reason": respondErr.Error()},
	}
	_, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      cfg.ToolName,
		Arguments: marshalArgs(blockedArgs),
	})
	if err != nil {
		slog.WarnContext(ctx, "push blocked failed",
			slog.String(logKeyItem, pullResp.Item),
			slog.Any(logKeyError, err))
	}
}

// ConnectEndpoint creates an MCP client session to the given endpoint.
// Convenience helper — callers may create sessions any way they prefer.
func ConnectEndpoint(ctx context.Context, endpoint, workerName string) (*sdkmcp.ClientSession, error) {
	transport := &sdkmcp.StreamableClientTransport{Endpoint: endpoint}
	client := sdkmcp.NewClient(
		&sdkmcp.Implementation{Name: "jericho-" + workerName, Version: "v0.1.0"},
		nil,
	)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("connect to endpoint: %w", err)
	}
	return session, nil
}

func marshalArgs(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func textContent(result *sdkmcp.CallToolResult) string {
	for _, c := range result.Content {
		if tc, ok := c.(*sdkmcp.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}
