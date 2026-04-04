package protocol

import "context"

// Server is the server-side protocol contract. Any process that serves the
// bugle MCP tool implements this interface. Transport-agnostic — MCP wiring
// is an adapter concern.
//
// Implementations: Origami circuit_server, Hegemony bugle-provider,
// testkit MockBugleServer.
type Server interface {
	Start(ctx context.Context, req StartRequest) (StartResponse, error)
	Pull(ctx context.Context, req PullRequest) (PullResponse, error)
	Push(ctx context.Context, req PushRequest) (PushResponse, error)
	Cancel(ctx context.Context, req CancelRequest) (CancelResponse, error)
	Status(ctx context.Context, req StatusRequest) (StatusResponse, error)
}
