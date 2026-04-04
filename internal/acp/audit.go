// audit.go — ACP capability audit and event routing.
//
// AuditCapabilities documents which ACP features are currently handled.
// RouteEvent converts a classified ACP shape into a typed message suitable
// for consumer event loops.
package acp

import (
	"encoding/json"
	"fmt"
)

// Capability describes a single ACP feature and whether it's handled.
type Capability struct {
	Name        string
	Supported   bool
	Description string
}

// AuditCapabilities returns the known ACP capabilities and their support status.
func AuditCapabilities() []Capability {
	return []Capability{
		{Name: "text_streaming", Supported: true, Description: "Streamed text content from agent"},
		{Name: "thinking", Supported: true, Description: "Agent thinking/reasoning stream"},
		{Name: "tool_use", Supported: true, Description: "Agent tool call requests and results"},
		{Name: "plan_update", Supported: true, Description: "Structured plan steps via ShapeClassifier"},
		{Name: "diff_update", Supported: true, Description: "File diff/change events via ShapeClassifier"},
		{Name: "state_change", Supported: true, Description: "Key-value state transitions via ShapeClassifier"},
		{Name: "capability_list", Supported: true, Description: "Agent capability enumeration via ShapeClassifier"},
		{Name: "billing_info", Supported: false, Description: "Token usage and cost reporting (future)"},
		{Name: "project_index", Supported: false, Description: "Project file indexing events (future)"},
	}
}

// --- Routed message types ---

// PlanUpdateMsg carries plan steps extracted from an ACP event.
type PlanUpdateMsg struct{ Steps []string }

// DiffUpdateMsg carries file paths from a diff ACP event.
type DiffUpdateMsg struct{ Files []string }

// StateChangeMsg carries a key-value state change.
type StateChangeMsg struct{ Key, Value string }

// OutputMsg carries formatted output for unknown/fallback shapes.
type OutputMsg struct{ Line string }

// RouteEvent converts a classified ACP shape into a typed message.
//
// The returned value is one of:
//   - PlanUpdateMsg  (ShapePlanUpdate)
//   - DiffUpdateMsg  (ShapeDiffUpdate)
//   - StateChangeMsg (ShapeStateChange)
//   - OutputMsg      (ShapeUnknown or unhandled)
func RouteEvent(shape ShapeKind, data map[string]any) any {
	switch shape {
	case ShapePlanUpdate:
		return PlanUpdateMsg{Steps: extractStringSlice(data, "steps", "plan")}

	case ShapeDiffUpdate:
		return DiffUpdateMsg{Files: extractFileList(data)}

	case ShapeStateChange:
		key, _ := data["key"].(string)
		value := fmt.Sprintf("%v", data["value"])
		return StateChangeMsg{Key: key, Value: value}

	default:
		raw, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return OutputMsg{Line: fmt.Sprintf("[unknown ACP event: %v]", data)}
		}
		return OutputMsg{Line: string(raw)}
	}
}

func extractStringSlice(data map[string]any, keys ...string) []string {
	for _, key := range keys {
		v, ok := data[key]
		if !ok {
			continue
		}
		arr, isArr := v.([]any)
		if !isArr {
			continue
		}
		result := make([]string, 0, len(arr))
		for _, item := range arr {
			if s, ok := item.(string); ok {
				result = append(result, s)
			} else {
				result = append(result, fmt.Sprintf("%v", item))
			}
		}
		return result
	}
	return nil
}

func extractFileList(data map[string]any) []string {
	if files := extractStringSlice(data, "diff"); len(files) > 0 {
		return files
	}

	v, ok := data["changes"]
	if !ok {
		return nil
	}
	arr, isArr := v.([]any)
	if !isArr {
		return nil
	}
	var files []string
	for _, item := range arr {
		m, isMap := item.(map[string]any)
		if !isMap {
			continue
		}
		if f, ok := m["file"].(string); ok {
			files = append(files, f)
		}
	}
	return files
}
