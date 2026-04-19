package troupe

import (
	"context"
	"os"
)

// ConfigSource provides runtime configuration with optional hot-reload.
type ConfigSource interface {
	Get(key string) (string, bool)
	Watch(ctx context.Context) <-chan ConfigChange
}

// ConfigChange is a configuration key-value update.
type ConfigChange struct {
	Key   string
	Value string
}

// EnvConfigSource reads from environment variables. Watch returns a
// closed channel (no hot-reload from env).
type EnvConfigSource struct{}

func (EnvConfigSource) Get(key string) (string, bool) {
	v := os.Getenv(key)
	return v, v != ""
}

func (EnvConfigSource) Watch(_ context.Context) <-chan ConfigChange {
	ch := make(chan ConfigChange)
	close(ch)
	return ch
}
