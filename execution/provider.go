package execution

import (
	"context"
	"fmt"
	"os"
	"strings"

	anyllmConfig "github.com/mozilla-ai/any-llm-go/config"
	anyllm "github.com/mozilla-ai/any-llm-go/providers"
	anyllmAnthropic "github.com/mozilla-ai/any-llm-go/providers/anthropic"
	anyllmGemini "github.com/mozilla-ai/any-llm-go/providers/gemini"
	anyllmOpenAI "github.com/mozilla-ai/any-llm-go/providers/openai"
)

// Env var names.
const (
	envProvider       = "TROUPE_PROVIDER"
	envUseVertex      = "CLAUDE_CODE_USE_VERTEX"
	envVertexRegion   = "CLOUD_ML_REGION"
	envVertexProject  = "ANTHROPIC_VERTEX_PROJECT_ID"
	envAnthropicKey   = "ANTHROPIC_API_KEY"
	envOpenAIKey      = "OPENAI_API_KEY"
	envGeminiKey      = "GEMINI_API_KEY"
	envOpenRouterKey  = "OPENROUTER_API_KEY"
	openRouterBaseURL = "https://openrouter.ai/api/v1"
)

// providerSpec defines a supported provider and its requirements.
type providerSpec struct {
	name     string   // canonical name
	aliases  []string // alternative names
	envVars  []string // required env vars (all must be set)
	envHint  string   // human-readable setup instruction
}

// providers is the single source of truth for supported providers.
var providers = []providerSpec{
	{name: "vertex-ai", envVars: []string{envVertexRegion, envVertexProject}, envHint: "set CLOUD_ML_REGION and ANTHROPIC_VERTEX_PROJECT_ID"},
	{name: "anthropic-api", aliases: []string{"anthropic", "claude"}, envVars: []string{envAnthropicKey}, envHint: "set ANTHROPIC_API_KEY"},
	{name: "openai-api", aliases: []string{"openai", "gpt"}, envVars: []string{envOpenAIKey}, envHint: "set OPENAI_API_KEY"},
	{name: "gemini-api", aliases: []string{"gemini"}, envVars: []string{envGeminiKey}, envHint: "set GEMINI_API_KEY"},
	{name: "openrouter", envVars: []string{envOpenRouterKey}, envHint: "set OPENROUTER_API_KEY"},
}

// ProviderNames returns the canonical names of all supported providers.
func ProviderNames() []string {
	names := make([]string, len(providers))
	for i, p := range providers {
		names[i] = p.name
	}
	return names
}

// providerOptionsHint returns a formatted string of available providers.
func providerOptionsHint() string {
	return strings.Join(ProviderNames(), ", ")
}

// findProvider returns the spec for the given name or alias.
func findProvider(name string) (providerSpec, bool) {
	for _, p := range providers {
		if p.name == name {
			return p, true
		}
		for _, alias := range p.aliases {
			if alias == name {
				return p, true
			}
		}
	}
	return providerSpec{}, false
}

// checkCredentials verifies all required env vars are set for a provider.
func checkCredentials(spec providerSpec) error {
	for _, env := range spec.envVars {
		if os.Getenv(env) == "" {
			return fmt.Errorf("%w: %s", ErrCredentialsMissing, spec.envHint)
		}
	}
	return nil
}

// NewProviderFromEnv creates the LLM provider specified by the given env var.
// Consumers pass their own env var name: "DJINN_PROVIDER", "ORIGAMI_PROVIDER", etc.
// If envName is empty, defaults to TROUPE_PROVIDER.
// Explicit only — no auto-detection, no fallback, no magic.
func NewProviderFromEnv(envName string) (anyllm.Provider, error) {
	if envName == "" {
		envName = envProvider
	}
	name := os.Getenv(envName)
	if name == "" {
		return nil, fmt.Errorf("%w: set %s (options: %s)", ErrProviderNotSet, envName, providerOptionsHint())
	}
	return NewProviderByName(name)
}

// NewProviderByName creates a provider by explicit name.
// Fails fast if required credentials are missing.
func NewProviderByName(name string) (anyllm.Provider, error) {
	spec, ok := findProvider(name)
	if !ok {
		return nil, fmt.Errorf("%w: %q (options: %s)", ErrProviderUnknown, name, providerOptionsHint())
	}
	if err := checkCredentials(spec); err != nil {
		return nil, err
	}
	return createProvider(spec)
}

func createProvider(spec providerSpec) (anyllm.Provider, error) {
	switch spec.name {
	case "vertex-ai":
		return NewVertexProvider(context.Background(), os.Getenv(envVertexRegion), os.Getenv(envVertexProject))
	case "anthropic-api":
		return anyllmAnthropic.New()
	case "openai-api":
		return anyllmOpenAI.New()
	case "gemini-api":
		return anyllmGemini.New()
	case "openrouter":
		return anyllmOpenAI.New(
			anyllmConfig.WithAPIKey(os.Getenv(envOpenRouterKey)),
			anyllmConfig.WithBaseURL(openRouterBaseURL),
		)
	default:
		return nil, fmt.Errorf("%w: %q", ErrProviderUnknown, spec.name)
	}
}
