package troupe

import "context"

// Gate is a predicate that decides whether an operation should proceed.
// Returns (allowed, reason, error). Reason explains rejection or provides
// audit context on pass. Error is reserved for infrastructure failures,
// not policy decisions — a rejected request returns (false, reason, nil).
type Gate func(ctx context.Context, subject any) (allowed bool, reason string, err error)

// AlwaysPass is a Gate that unconditionally allows.
var AlwaysPass Gate = func(_ context.Context, _ any) (bool, string, error) {
	return true, "", nil
}

// AlwaysDeny is a Gate that unconditionally rejects.
var AlwaysDeny Gate = func(_ context.Context, _ any) (bool, string, error) {
	return false, "denied", nil
}

// ComposeGates chains multiple gates with short-circuit AND semantics.
// The first gate to reject stops evaluation. If all pass, the composite passes.
// An empty chain passes unconditionally.
func ComposeGates(gates ...Gate) Gate {
	return func(ctx context.Context, subject any) (bool, string, error) {
		for _, g := range gates {
			allowed, reason, err := g(ctx, subject)
			if err != nil {
				return false, reason, err
			}
			if !allowed {
				return false, reason, nil
			}
		}
		return true, "", nil
	}
}
