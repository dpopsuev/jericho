package providers

import (
	"errors"
	"os"
	"testing"

	anyllm "github.com/mozilla-ai/any-llm-go/providers"
)

func TestSentinel_ProviderNotSet(t *testing.T) {
	os.Unsetenv("TEST_PROVIDER_SENTINEL")
	_, err := NewProviderFromEnv("TEST_PROVIDER_SENTINEL")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrProviderNotSet) {
		t.Errorf("expected ErrProviderNotSet, got: %v", err)
	}
}

func TestSentinel_CredentialsMissing_Anthropic(t *testing.T) {
	orig := os.Getenv("ANTHROPIC_API_KEY")
	os.Unsetenv("ANTHROPIC_API_KEY")
	defer func() {
		if orig != "" {
			os.Setenv("ANTHROPIC_API_KEY", orig)
		}
	}()

	_, err := NewProviderByName("anthropic-api")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrCredentialsMissing) {
		t.Errorf("expected ErrCredentialsMissing, got: %v", err)
	}
}

func TestSentinel_ProviderUnknown(t *testing.T) {
	_, err := NewProviderByName("bogus-provider")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrProviderUnknown) {
		t.Errorf("expected ErrProviderUnknown, got: %v", err)
	}
}

func TestSentinel_ModelRequired(t *testing.T) {
	region := os.Getenv("GOOGLE_CLOUD_LOCATION")
	project := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if region == "" || project == "" {
		t.Skip("Vertex credentials not configured")
	}

	p, err := NewVertexProvider(t.Context(), region, project)
	if err != nil {
		t.Fatalf("NewVertexProvider: %v", err)
	}

	_, err = p.Completion(t.Context(), anyllm.CompletionParams{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrModelRequired) {
		t.Errorf("expected ErrModelRequired, got: %v", err)
	}
}

func TestProviderNames_MatchesRegistry(t *testing.T) {
	names := ProviderNames()
	if len(names) != len(providers) {
		t.Fatalf("ProviderNames() returned %d, want %d", len(names), len(providers))
	}
	for _, name := range names {
		if _, ok := findProvider(name); !ok {
			t.Errorf("ProviderNames() contains %q but findProvider can't find it", name)
		}
	}
}

func TestFindProvider_ByAlias(t *testing.T) {
	spec, ok := findProvider("claude")
	if !ok {
		t.Fatal("findProvider(claude) not found")
	}
	if spec.name != "anthropic-api" {
		t.Errorf("claude resolved to %q, want anthropic-api", spec.name)
	}
}
