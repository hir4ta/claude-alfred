package mcpserver

import (
	"encoding/json"
	"testing"
)

func TestFormatConcise(t *testing.T) {
	t.Parallel()

	t.Run("scalars kept", func(t *testing.T) {
		t.Parallel()
		obj := map[string]any{
			"health":       0.85,
			"phase":        "implement",
			"total_tools":  float64(47),
			"total_errors": float64(2),
			"active":       true,
		}
		result := formatConcise(obj)

		kd, ok := result["key_data"].(map[string]any)
		if !ok {
			t.Fatal("key_data missing or wrong type")
		}
		if kd["health"] != 0.85 {
			t.Errorf("health = %v, want 0.85", kd["health"])
		}
		if kd["phase"] != "implement" {
			t.Errorf("phase = %v, want implement", kd["phase"])
		}
		if kd["active"] != true {
			t.Errorf("active = %v, want true", kd["active"])
		}
	})

	t.Run("arrays become count", func(t *testing.T) {
		t.Parallel()
		obj := map[string]any{
			"patterns": []any{"a", "b", "c"},
		}
		result := formatConcise(obj)
		kd := result["key_data"].(map[string]any)
		if kd["patterns_count"] != 3 {
			t.Errorf("patterns_count = %v, want 3", kd["patterns_count"])
		}
		if _, exists := kd["patterns"]; exists {
			t.Error("raw patterns array should not be in key_data")
		}
	})

	t.Run("long strings truncated", func(t *testing.T) {
		t.Parallel()
		long := ""
		for i := 0; i < 120; i++ {
			long += "x"
		}
		obj := map[string]any{"desc": long}
		result := formatConcise(obj)
		kd := result["key_data"].(map[string]any)
		s := kd["desc"].(string)
		if len(s) != 100 {
			t.Errorf("truncated string len = %d, want 100", len(s))
		}
	})

	t.Run("summary from health fields", func(t *testing.T) {
		t.Parallel()
		obj := map[string]any{
			"health":       0.92,
			"phase":        "test",
			"total_tools":  float64(30),
			"total_errors": float64(0),
		}
		result := formatConcise(obj)
		summary, ok := result["summary"].(string)
		if !ok || summary == "" {
			t.Error("expected non-empty summary")
		}
	})

	t.Run("small nested objects flattened", func(t *testing.T) {
		t.Parallel()
		obj := map[string]any{
			"burst": map[string]any{
				"state": "active",
				"count": float64(5),
			},
		}
		result := formatConcise(obj)
		kd := result["key_data"].(map[string]any)
		if _, ok := kd["burst"].(map[string]any); !ok {
			t.Error("small nested object should be kept as-is")
		}
	})

	t.Run("large nested objects become key count", func(t *testing.T) {
		t.Parallel()
		obj := map[string]any{
			"details": map[string]any{
				"a": "1", "b": "2", "c": "3", "d": "4", "e": "5",
			},
		}
		result := formatConcise(obj)
		kd := result["key_data"].(map[string]any)
		if _, exists := kd["details"]; exists {
			t.Error("large nested object should not be in key_data directly")
		}
		if kd["details_keys"] != 5 {
			t.Errorf("details_keys = %v, want 5", kd["details_keys"])
		}
	})

	t.Run("meta excluded", func(t *testing.T) {
		t.Parallel()
		obj := map[string]any{
			"health": 0.5,
			"_meta":  map[string]any{"source": "session"},
		}
		result := formatConcise(obj)
		kd := result["key_data"].(map[string]any)
		if _, exists := kd["_meta"]; exists {
			t.Error("_meta should be excluded from key_data")
		}
	})

	t.Run("concise output is smaller than detailed", func(t *testing.T) {
		t.Parallel()
		obj := map[string]any{
			"health":       0.85,
			"phase":        "implement",
			"total_tools":  float64(47),
			"total_errors": float64(2),
			"patterns":     []any{"a", "b", "c", "d", "e"},
			"details": map[string]any{
				"a": "very long value here that takes up space",
				"b": "another long value",
				"c": "yet another",
				"d": "and more",
				"e": "even more data",
			},
		}
		detailed, _ := json.Marshal(obj)
		concise := formatConcise(obj)
		conciseJSON, _ := json.Marshal(concise)
		if len(conciseJSON) >= len(detailed) {
			t.Errorf("concise (%d bytes) should be smaller than detailed (%d bytes)", len(conciseJSON), len(detailed))
		}
	})
}
