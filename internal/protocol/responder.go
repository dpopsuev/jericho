package protocol

import "context"

// Responder handles a prompt and returns a response.
// The client-side interface — consumers inject their own implementation:
// Solo (Jericho), Pod exec (K8s), HTTP call, or mock (testkit).
type Responder interface {
	RespondTo(ctx context.Context, prompt string) (string, error)
}
