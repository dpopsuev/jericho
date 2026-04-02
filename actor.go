package jericho

import "context"

// Actor is what the Broker gives back — an agent ready to perform.
// 3 methods. ISP: consumers see only what they need.
type Actor interface {
	// Perform sends a prompt to the actor and returns the response.
	Perform(ctx context.Context, prompt string) (string, error)

	// Ready reports whether the actor is available for work.
	Ready() bool

	// Kill stops the actor.
	Kill(ctx context.Context) error
}
