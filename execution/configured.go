package execution

import (
	"context"

	anyllm "github.com/mozilla-ai/any-llm-go/providers"
)

// ProviderConfig holds common defaults propagated to all providers.
// Consumers set this once — all providers respect it.
type ProviderConfig struct {
	// MaxTokens is the default max output tokens. When the caller's
	// CompletionParams.MaxTokens is nil, this value is used.
	// Zero means no override (provider's own default applies).
	MaxTokens int
}

// ConfiguredProvider wraps an anyllm.Provider and injects defaults
// from ProviderConfig into every CompletionParams before forwarding.
type ConfiguredProvider struct {
	inner  anyllm.Provider
	config ProviderConfig
}

// NewConfiguredProvider wraps a provider with common defaults.
func NewConfiguredProvider(inner anyllm.Provider, cfg ProviderConfig) *ConfiguredProvider {
	return &ConfiguredProvider{inner: inner, config: cfg}
}

// Name returns the inner provider's name.
func (p *ConfiguredProvider) Name() string { return p.inner.Name() }

// Completion injects defaults and forwards to the inner provider.
func (p *ConfiguredProvider) Completion(ctx context.Context, params anyllm.CompletionParams) (*anyllm.ChatCompletion, error) {
	if params.MaxTokens == nil && p.config.MaxTokens > 0 {
		mt := p.config.MaxTokens
		params.MaxTokens = &mt
	}
	return p.inner.Completion(ctx, params)
}

// CompletionStream forwards to the inner provider.
func (p *ConfiguredProvider) CompletionStream(ctx context.Context, params anyllm.CompletionParams) (<-chan anyllm.ChatCompletionChunk, <-chan error) {
	if params.MaxTokens == nil && p.config.MaxTokens > 0 {
		mt := p.config.MaxTokens
		params.MaxTokens = &mt
	}
	return p.inner.CompletionStream(ctx, params)
}

var _ anyllm.Provider = (*ConfiguredProvider)(nil)
