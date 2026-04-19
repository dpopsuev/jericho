package troupe

import "context"

// StateStore persists and restores World state. Signal already has
// EventStore for bus replay; StateStore is the World equivalent.
type StateStore interface {
	SaveWorld(ctx context.Context, snapshot []byte) error
	LoadWorld(ctx context.Context) ([]byte, error)
}
