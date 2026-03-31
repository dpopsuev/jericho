package provider

import (
	"os"
	"path/filepath"
	"testing"

	bd "github.com/dpopsuev/jericho/dispatch"
)

func TestParseConfig_Valid(t *testing.T) {
	data := []byte(`
providers:
  - name: openai
    type: http
    config:
      base_url: "https://api.openai.com"
      model: "gpt-4o"
  - name: fallback
    type: stdin
fallbacks:
  openai: [fallback]
`)
	cfg, err := ParseConfig(data)
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if len(cfg.Providers) != 2 {
		t.Fatalf("providers = %d, want 2", len(cfg.Providers))
	}
	if cfg.Providers[0].Name != "openai" {
		t.Errorf("name = %q, want openai", cfg.Providers[0].Name)
	}
	if cfg.Providers[0].Type != "http" {
		t.Errorf("type = %q, want http", cfg.Providers[0].Type)
	}
	if cfg.Providers[0].Config["base_url"] != "https://api.openai.com" {
		t.Errorf("base_url = %v", cfg.Providers[0].Config["base_url"])
	}
	if len(cfg.Fallbacks["openai"]) != 1 || cfg.Fallbacks["openai"][0] != "fallback" {
		t.Errorf("fallbacks = %v", cfg.Fallbacks)
	}
}

func TestParseConfig_NoProviders(t *testing.T) {
	_, err := ParseConfig([]byte(`fallbacks: {}`))
	if err == nil {
		t.Fatal("expected error for empty providers")
	}
}

func TestParseConfig_MissingName(t *testing.T) {
	_, err := ParseConfig([]byte(`
providers:
  - type: http
`))
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestParseConfig_MissingType(t *testing.T) {
	_, err := ParseConfig([]byte(`
providers:
  - name: foo
`))
	if err == nil {
		t.Fatal("expected error for missing type")
	}
}

func TestLoadConfig_FromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dispatch.yaml")
	content := []byte(`
providers:
  - name: test
    type: stdin
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Providers[0].Name != "test" {
		t.Errorf("name = %q", cfg.Providers[0].Name)
	}
}

func TestBuildRouter_StdinProvider(t *testing.T) {
	cfg := &Config{
		Providers: []Def{
			{Name: "interactive", Type: "stdin"},
		},
	}

	router, err := BuildRouter(cfg, nil)
	if err != nil {
		t.Fatalf("BuildRouter: %v", err)
	}

	if router.Default == nil {
		t.Fatal("default dispatcher is nil")
	}
	if _, ok := router.Routes["interactive"]; !ok {
		t.Fatal("interactive route not registered")
	}
}

func TestBuildRouter_HTTPProvider(t *testing.T) {
	cfg := &Config{
		Providers: []Def{
			{
				Name: "openai",
				Type: "http",
				Config: map[string]any{
					"base_url":    "https://api.openai.com",
					"model":       "gpt-4o",
					"api_key_env": "OPENAI_KEY",
				},
			},
		},
	}

	router, err := BuildRouter(cfg, nil)
	if err != nil {
		t.Fatalf("BuildRouter: %v", err)
	}

	d, ok := router.Routes["openai"]
	if !ok {
		t.Fatal("openai route not registered")
	}
	httpD, ok := d.(*bd.HTTPDispatcher)
	if !ok {
		t.Fatalf("expected *bd.HTTPDispatcher, got %T", d)
	}
	if httpD.Model != "gpt-4o" {
		t.Errorf("model = %q, want gpt-4o", httpD.Model)
	}
}

func TestBuildRouter_FallbackChains(t *testing.T) {
	cfg := &Config{
		Providers: []Def{
			{Name: "primary", Type: "stdin"},
			{Name: "secondary", Type: "stdin"},
		},
		Fallbacks: map[string][]string{
			"primary": {"secondary"},
		},
	}

	router, err := BuildRouter(cfg, nil)
	if err != nil {
		t.Fatalf("BuildRouter: %v", err)
	}

	if len(router.Fallbacks["primary"]) != 1 || router.Fallbacks["primary"][0] != "secondary" {
		t.Errorf("fallbacks = %v", router.Fallbacks)
	}
}

func TestBuildRouter_ExtraFactory(t *testing.T) {
	called := false
	mockFactory := func(_ map[string]any) (bd.Dispatcher, error) {
		called = true
		return bd.NewStdinDispatcher(), nil
	}

	cfg := &Config{
		Providers: []Def{
			{Name: "custom", Type: "custom-type"},
		},
	}

	router, err := BuildRouter(cfg, map[string]DispatcherFactory{
		"custom-type": mockFactory,
	})
	if err != nil {
		t.Fatalf("BuildRouter: %v", err)
	}
	if !called {
		t.Fatal("custom factory was not called")
	}
	if _, ok := router.Routes["custom"]; !ok {
		t.Fatal("custom route not registered")
	}
}

func TestBuildRouter_UnknownType(t *testing.T) {
	cfg := &Config{
		Providers: []Def{
			{Name: "bad", Type: "unknown"},
		},
	}

	_, err := BuildRouter(cfg, nil)
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
}

func TestBuildRouter_FirstProviderIsDefault(t *testing.T) {
	cfg := &Config{
		Providers: []Def{
			{Name: "first", Type: "stdin"},
			{Name: "second", Type: "stdin"},
		},
	}

	router, err := BuildRouter(cfg, nil)
	if err != nil {
		t.Fatalf("BuildRouter: %v", err)
	}

	if router.Default != router.Routes["first"] {
		t.Error("default dispatcher should be the first provider")
	}
}
