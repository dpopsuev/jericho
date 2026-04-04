package acp

import "testing"

func TestAuditCapabilities_Count(t *testing.T) {
	caps := AuditCapabilities()
	if len(caps) != 9 {
		t.Fatalf("AuditCapabilities() returned %d items, want 9", len(caps))
	}
}

func TestAuditCapabilities_SupportedList(t *testing.T) {
	caps := AuditCapabilities()
	supported := make(map[string]bool)
	for _, c := range caps {
		supported[c.Name] = c.Supported
	}

	wantSupported := []string{
		"text_streaming", "thinking", "tool_use",
		"plan_update", "diff_update", "state_change", "capability_list",
	}
	for _, name := range wantSupported {
		if !supported[name] {
			t.Errorf("capability %q should be supported", name)
		}
	}

	wantUnsupported := []string{"billing_info", "project_index"}
	for _, name := range wantUnsupported {
		if supported[name] {
			t.Errorf("capability %q should NOT be supported (future)", name)
		}
	}
}

func TestRouteEvent_PlanUpdate(t *testing.T) {
	data := map[string]any{
		"steps": []any{"step 1", "step 2", "step 3"},
	}
	msg := RouteEvent(ShapePlanUpdate, data)

	plan, ok := msg.(PlanUpdateMsg)
	if !ok {
		t.Fatalf("expected PlanUpdateMsg, got %T", msg)
	}
	if len(plan.Steps) != 3 {
		t.Fatalf("Steps len = %d, want 3", len(plan.Steps))
	}
}

func TestRouteEvent_DiffUpdate(t *testing.T) {
	data := map[string]any{
		"changes": []any{
			map[string]any{"file": "main.go", "action": "modify"},
			map[string]any{"file": "test.go", "action": "create"},
		},
	}
	msg := RouteEvent(ShapeDiffUpdate, data)

	diff, ok := msg.(DiffUpdateMsg)
	if !ok {
		t.Fatalf("expected DiffUpdateMsg, got %T", msg)
	}
	if len(diff.Files) != 2 {
		t.Fatalf("Files len = %d, want 2", len(diff.Files))
	}
}

func TestRouteEvent_StateChange(t *testing.T) {
	data := map[string]any{
		"key":   "mode",
		"value": "agent",
	}
	msg := RouteEvent(ShapeStateChange, data)

	sc, ok := msg.(StateChangeMsg)
	if !ok {
		t.Fatalf("expected StateChangeMsg, got %T", msg)
	}
	if sc.Key != "mode" || sc.Value != "agent" {
		t.Fatalf("got %+v", sc)
	}
}

func TestRouteEvent_Unknown(t *testing.T) {
	data := map[string]any{"random": "payload"}
	msg := RouteEvent(ShapeUnknown, data)

	out, ok := msg.(OutputMsg)
	if !ok {
		t.Fatalf("expected OutputMsg, got %T", msg)
	}
	if out.Line == "" {
		t.Error("OutputMsg.Line should not be empty")
	}
}
