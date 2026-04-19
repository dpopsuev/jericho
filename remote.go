package troupe

import (
	"context"
	"errors"

	"github.com/dpopsuev/troupe/world"
)

// ErrRemoteNotImplemented is returned by Connect stubs until HTTP client is built.
var ErrRemoteNotImplemented = errors.New("troupe: remote operation not implemented")

type remoteBroker struct {
	serverURL string
}

func (r *remoteBroker) Pick(_ context.Context, _ Preferences) ([]ActorConfig, error) {
	return nil, ErrRemoteNotImplemented
}

func (r *remoteBroker) Spawn(_ context.Context, _ ActorConfig) (Actor, error) {
	return nil, ErrRemoteNotImplemented
}

func (r *remoteBroker) Discover(_ string) []AgentCard {
	return nil
}

type remoteAdmission struct {
	serverURL string
}

func (r *remoteAdmission) Admit(_ context.Context, _ ActorConfig) (world.EntityID, error) {
	return 0, ErrRemoteNotImplemented
}

func (r *remoteAdmission) Kick(_ context.Context, _ world.EntityID) error {
	return ErrRemoteNotImplemented
}

func (r *remoteAdmission) Ban(_ context.Context, _ world.EntityID, _ string) error {
	return ErrRemoteNotImplemented
}

func (r *remoteAdmission) Unban(_ context.Context, _ world.EntityID) error {
	return ErrRemoteNotImplemented
}
