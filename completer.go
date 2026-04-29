package troupe

import "context"

// Completer performs a single LLM completion. Prompt in, response out.
type Completer interface {
	Complete(ctx context.Context, prompt string) (string, error)
}

// CompleteFunc is a function that satisfies Completer.
type CompleteFunc func(ctx context.Context, prompt string) (string, error)

func (f CompleteFunc) Complete(ctx context.Context, prompt string) (string, error) {
	return f(ctx, prompt)
}
