package provider

import (
	"context"
	"fmt"
	"strings"
	"testing"

	bd "github.com/dpopsuev/jericho/dispatch"
)

var _ bd.Dispatcher = (*Router)(nil)

type mockDispatcher struct {
	name    string
	called  bool
	lastCtx bd.Context
	err     error
}

func (m *mockDispatcher) Dispatch(_ context.Context, ctx bd.Context) ([]byte, error) { //nolint:gocritic
	m.called = true
	m.lastCtx = ctx
	if m.err != nil {
		return nil, m.err
	}
	return []byte(m.name + "-output"), nil
}

func TestRouter_DefaultRoute(t *testing.T) {
	def := &mockDispatcher{name: "default"}
	router := NewRouter(def, nil)

	result, err := router.Dispatch(context.Background(), bd.Context{
		CaseID: "C1", Step: "F0",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !def.called {
		t.Error("default dispatcher not called")
	}
	if string(result) != "default-output" {
		t.Errorf("result = %q", result)
	}
}

func TestRouter_NamedRoute(t *testing.T) {
	def := &mockDispatcher{name: "default"}
	codex := &mockDispatcher{name: "codex"}
	claude := &mockDispatcher{name: "claude"}

	router := NewRouter(def, map[string]bd.Dispatcher{
		"codex":  codex,
		"claude": claude,
	})

	result, err := router.Dispatch(context.Background(), bd.Context{
		CaseID: "C1", Step: "F1", Provider: "codex",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !codex.called {
		t.Error("codex dispatcher not called")
	}
	if def.called {
		t.Error("default dispatcher should not be called")
	}
	if string(result) != "codex-output" {
		t.Errorf("result = %q", result)
	}
}

func TestRouter_UnknownProvider(t *testing.T) {
	def := &mockDispatcher{name: "default"}
	router := NewRouter(def, nil)

	_, err := router.Dispatch(context.Background(), bd.Context{
		CaseID: "C1", Step: "F0", Provider: "nonexistent",
	})
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	if !strings.Contains(err.Error(), "unknown provider") {
		t.Errorf("error = %v", err)
	}
}

func TestRouter_Register(t *testing.T) {
	def := &mockDispatcher{name: "default"}
	router := NewRouter(def, nil)

	openai := &mockDispatcher{name: "openai"}
	router.Register("openai", openai)

	result, err := router.Dispatch(context.Background(), bd.Context{
		CaseID: "C1", Step: "F3", Provider: "openai",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !openai.called {
		t.Error("openai dispatcher not called")
	}
	if string(result) != "openai-output" {
		t.Errorf("result = %q", result)
	}
}

func TestRouter_EmptyProviderUsesDefault(t *testing.T) {
	def := &mockDispatcher{name: "default"}
	codex := &mockDispatcher{name: "codex"}
	router := NewRouter(def, map[string]bd.Dispatcher{"codex": codex})

	_, err := router.Dispatch(context.Background(), bd.Context{
		CaseID: "C1", Step: "F0", Provider: "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !def.called {
		t.Error("default should be called for empty provider")
	}
	if codex.called {
		t.Error("codex should not be called for empty provider")
	}
}

func TestRouter_AutoRouteFromPersonaSheet(t *testing.T) {
	def := &mockDispatcher{name: "default"}
	anthropic := &mockDispatcher{name: "anthropic"}
	openai := &mockDispatcher{name: "openai"}

	router := NewRouter(def, map[string]bd.Dispatcher{
		"anthropic": anthropic,
		"openai":    openai,
	})
	router.StepProviderHints = map[string]string{
		"investigate": "anthropic",
		"triage":      "openai",
	}

	result, err := router.Dispatch(context.Background(), bd.Context{
		CaseID: "C1", Step: "investigate", Provider: "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !anthropic.called {
		t.Error("anthropic should be called via auto-route")
	}
	if def.called {
		t.Error("default should not be called when auto-route matches")
	}
	if string(result) != "anthropic-output" {
		t.Errorf("result = %q, want anthropic-output", result)
	}

	result, err = router.Dispatch(context.Background(), bd.Context{
		CaseID: "C1", Step: "triage", Provider: "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !openai.called {
		t.Error("openai should be called via auto-route for triage")
	}
	if string(result) != "openai-output" {
		t.Errorf("result = %q, want openai-output", result)
	}
}

func TestRouter_AutoRoute_NoHint_FallsToDefault(t *testing.T) {
	def := &mockDispatcher{name: "default"}
	router := NewRouter(def, nil)
	router.StepProviderHints = map[string]string{
		"investigate": "anthropic",
	}

	_, err := router.Dispatch(context.Background(), bd.Context{
		CaseID: "C1", Step: "recall", Provider: "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !def.called {
		t.Error("default should be called when no auto-route hint exists for step")
	}
}

func TestRouter_ExplicitProvider_OverridesAutoRoute(t *testing.T) {
	def := &mockDispatcher{name: "default"}
	anthropic := &mockDispatcher{name: "anthropic"}
	openai := &mockDispatcher{name: "openai"}

	router := NewRouter(def, map[string]bd.Dispatcher{
		"anthropic": anthropic,
		"openai":    openai,
	})
	router.StepProviderHints = map[string]string{
		"investigate": "anthropic",
	}

	_, err := router.Dispatch(context.Background(), bd.Context{
		CaseID: "C1", Step: "investigate", Provider: "openai",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !openai.called {
		t.Error("explicit provider should override auto-route")
	}
	if anthropic.called {
		t.Error("auto-route should not override explicit provider")
	}
}

func TestRouter_Fallback_PrimaryFails(t *testing.T) {
	primary := &mockDispatcher{name: "primary", err: fmt.Errorf("rate limited")}
	backup := &mockDispatcher{name: "backup"}

	router := NewRouter(primary, map[string]bd.Dispatcher{
		"primary": primary,
		"backup":  backup,
	}, WithFallbacks(map[string][]string{
		"primary": {"backup"},
	}))

	result, err := router.Dispatch(context.Background(), bd.Context{
		CaseID: "C1", Step: "F0", Provider: "primary",
	})
	if err != nil {
		t.Fatalf("expected fallback to succeed, got: %v", err)
	}
	if !backup.called {
		t.Error("backup should be called on primary failure")
	}
	if string(result) != "backup-output" {
		t.Errorf("result = %q, want backup-output", result)
	}
}

func TestRouter_Fallback_AllFail(t *testing.T) {
	primary := &mockDispatcher{name: "primary", err: fmt.Errorf("error 1")}
	fb1 := &mockDispatcher{name: "fb1", err: fmt.Errorf("error 2")}
	fb2 := &mockDispatcher{name: "fb2", err: fmt.Errorf("error 3")}

	router := NewRouter(primary, map[string]bd.Dispatcher{
		"primary": primary,
		"fb1":     fb1,
		"fb2":     fb2,
	}, WithFallbacks(map[string][]string{
		"primary": {"fb1", "fb2"},
	}))

	_, err := router.Dispatch(context.Background(), bd.Context{
		CaseID: "C1", Step: "F0", Provider: "primary",
	})
	if err == nil {
		t.Fatal("expected error when all providers fail")
	}
	if !strings.Contains(err.Error(), "all providers failed") {
		t.Errorf("error = %v", err)
	}
}

func TestRouter_Fallback_NoFallbacks(t *testing.T) {
	primary := &mockDispatcher{name: "primary", err: fmt.Errorf("fail")}

	router := NewRouter(primary, map[string]bd.Dispatcher{
		"primary": primary,
	})

	_, err := router.Dispatch(context.Background(), bd.Context{
		CaseID: "C1", Step: "F0", Provider: "primary",
	})
	if err == nil {
		t.Fatal("expected error with no fallbacks")
	}
	if !strings.Contains(err.Error(), "fail") {
		t.Errorf("error = %v", err)
	}
}

func TestRouter_Fallback_DefaultProvider(t *testing.T) {
	def := &mockDispatcher{name: "default", err: fmt.Errorf("default fail")}
	backup := &mockDispatcher{name: "backup"}

	router := NewRouter(def, map[string]bd.Dispatcher{
		"backup": backup,
	}, WithFallbacks(map[string][]string{
		"default": {"backup"},
	}))

	result, err := router.Dispatch(context.Background(), bd.Context{
		CaseID: "C1", Step: "F0",
	})
	if err != nil {
		t.Fatalf("expected fallback to work for default: %v", err)
	}
	if string(result) != "backup-output" {
		t.Errorf("result = %q, want backup-output", result)
	}
}

func TestRouter_FallbackCallback(t *testing.T) {
	primary := &mockDispatcher{name: "primary", err: fmt.Errorf("fail")}
	backup := &mockDispatcher{name: "backup"}

	var gotPrimary, gotFallback string
	router := NewRouter(primary, map[string]bd.Dispatcher{
		"primary": primary,
		"backup":  backup,
	},
		WithFallbacks(map[string][]string{"primary": {"backup"}}),
		WithFallbackCallback(func(p, fb string, _ error) {
			gotPrimary = p
			gotFallback = fb
		}),
	)

	_, err := router.Dispatch(context.Background(), bd.Context{
		CaseID: "C1", Step: "F0", Provider: "primary",
	})
	if err != nil {
		t.Fatalf("expected success: %v", err)
	}
	if gotPrimary != "primary" || gotFallback != "backup" {
		t.Errorf("callback: primary=%q fallback=%q", gotPrimary, gotFallback)
	}
}
