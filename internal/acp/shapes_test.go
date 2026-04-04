package acp

import "testing"

func TestClassifyShape_TextStream(t *testing.T) {
	tests := []struct {
		name string
		data map[string]any
	}{
		{"content field", map[string]any{"content": "hello"}},
		{"text field", map[string]any{"text": "world"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyShape(tt.data)
			if got != ShapeTextStream {
				t.Fatalf("ClassifyShape(%v) = %v, want ShapeTextStream", tt.data, got)
			}
		})
	}
}

func TestClassifyShape_PlanUpdate(t *testing.T) {
	tests := []struct {
		name string
		data map[string]any
	}{
		{"steps array", map[string]any{"steps": []any{"step1", "step2"}}},
		{"plan array", map[string]any{"plan": []any{"a", "b"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyShape(tt.data)
			if got != ShapePlanUpdate {
				t.Fatalf("ClassifyShape(%v) = %v, want ShapePlanUpdate", tt.data, got)
			}
		})
	}
}

func TestClassifyShape_DiffUpdate(t *testing.T) {
	tests := []struct {
		name string
		data map[string]any
	}{
		{
			"changes with file entries",
			map[string]any{
				"changes": []any{
					map[string]any{"file": "main.go", "hunks": []any{}},
				},
			},
		},
		{
			"diff array",
			map[string]any{
				"diff": []any{"--- a/file.go", "+++ b/file.go"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyShape(tt.data)
			if got != ShapeDiffUpdate {
				t.Fatalf("ClassifyShape(%v) = %v, want ShapeDiffUpdate", tt.data, got)
			}
		})
	}
}

func TestClassifyShape_StructuredList(t *testing.T) {
	data := map[string]any{"items": []any{"one", "two", "three"}}
	got := ClassifyShape(data)
	if got != ShapeStructuredList {
		t.Fatalf("ClassifyShape(%v) = %v, want ShapeStructuredList", data, got)
	}
}

func TestClassifyShape_ActionLifecycle(t *testing.T) {
	data := map[string]any{"status": "running"}
	got := ClassifyShape(data)
	if got != ShapeActionLifecycle {
		t.Fatalf("ClassifyShape(%v) = %v, want ShapeActionLifecycle", data, got)
	}
}

func TestClassifyShape_StateChange(t *testing.T) {
	data := map[string]any{"key": "mode", "value": "agent"}
	got := ClassifyShape(data)
	if got != ShapeStateChange {
		t.Fatalf("ClassifyShape(%v) = %v, want ShapeStateChange", data, got)
	}
}

func TestClassifyShape_CapabilityList(t *testing.T) {
	tests := []struct {
		name string
		data map[string]any
	}{
		{"tools array", map[string]any{"tools": []any{"read", "write"}}},
		{"capabilities array", map[string]any{"capabilities": []any{"sandbox"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyShape(tt.data)
			if got != ShapeCapabilityList {
				t.Fatalf("ClassifyShape(%v) = %v, want ShapeCapabilityList", tt.data, got)
			}
		})
	}
}

func TestClassifyShape_Unknown(t *testing.T) {
	data := map[string]any{"random": 42}
	got := ClassifyShape(data)
	if got != ShapeUnknown {
		t.Fatalf("ClassifyShape(%v) = %v, want ShapeUnknown", data, got)
	}
}

func TestClassifyShape_NilMap(t *testing.T) {
	got := ClassifyShape(nil)
	if got != ShapeUnknown {
		t.Fatalf("ClassifyShape(nil) = %v, want ShapeUnknown", got)
	}
}

func TestShapeKind_String(t *testing.T) {
	kinds := []struct {
		kind ShapeKind
		want string
	}{
		{ShapeTextStream, "text_stream"},
		{ShapeStructuredList, "structured_list"},
		{ShapeActionLifecycle, "action_lifecycle"},
		{ShapeStateChange, "state_change"},
		{ShapeCapabilityList, "capability_list"},
		{ShapePlanUpdate, "plan_update"},
		{ShapeDiffUpdate, "diff_update"},
		{ShapeUnknown, "unknown"},
	}
	for _, tt := range kinds {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.kind.String()
			if got != tt.want {
				t.Fatalf("ShapeKind(%d).String() = %q, want %q", tt.kind, got, tt.want)
			}
		})
	}
}
