package testkit

import "context"

// StubActorFunc returns an ActorFunc that returns canned responses.
// The responses cycle: first call returns responses[0], second returns
// responses[1], etc. After exhausting the list, wraps around.
func StubActorFunc(responses ...string) func(context.Context, string) (string, error) {
	var idx int
	return func(_ context.Context, _ string) (string, error) {
		resp := responses[idx%len(responses)]
		idx++
		return resp, nil
	}
}
