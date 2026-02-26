package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBundle(t *testing.T) {
	t.Parallel()
	outputDir := t.TempDir()

	if err := Bundle(outputDir, "0.15.0-test"); err != nil {
		t.Fatalf("Bundle() error: %v", err)
	}

	// Verify plugin.json exists and has correct structure.
	t.Run("plugin.json", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(outputDir, ".claude-plugin", "plugin.json"))
		if err != nil {
			t.Fatalf("read plugin.json: %v", err)
		}
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			t.Fatalf("parse plugin.json: %v", err)
		}
		if got := m["name"]; got != "claude-buddy" {
			t.Errorf("name = %v, want claude-buddy", got)
		}
		if got := m["version"]; got != "0.15.0-test" {
			t.Errorf("version = %v, want 0.15.0-test", got)
		}
	})

	// Verify hooks.json has all 14 events + Stop's prompt hook.
	t.Run("hooks.json", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(outputDir, "hooks", "hooks.json"))
		if err != nil {
			t.Fatalf("read hooks.json: %v", err)
		}
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			t.Fatalf("parse hooks.json: %v", err)
		}
		hooks, ok := m["hooks"].(map[string]any)
		if !ok {
			t.Fatal("hooks key missing or wrong type")
		}

		expectedEvents := []string{
			"SessionStart", "PreToolUse", "PostToolUse", "PostToolUseFailure",
			"UserPromptSubmit", "PreCompact", "SessionEnd", "Stop",
			"SubagentStart", "SubagentStop", "Notification",
			"TeammateIdle", "TaskCompleted", "PermissionRequest",
		}
		for _, event := range expectedEvents {
			if _, ok := hooks[event]; !ok {
				t.Errorf("missing event: %s", event)
			}
		}

		// Stop should have 2 entries (command + prompt).
		stopEntries, ok := hooks["Stop"].([]any)
		if !ok {
			t.Fatal("Stop is not an array")
		}
		if len(stopEntries) != 2 {
			t.Errorf("Stop entries = %d, want 2", len(stopEntries))
		}
	})

	// Verify .mcp.json.
	t.Run("mcp.json", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(outputDir, ".mcp.json"))
		if err != nil {
			t.Fatalf("read .mcp.json: %v", err)
		}
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			t.Fatalf("parse .mcp.json: %v", err)
		}
		servers, ok := m["mcpServers"].(map[string]any)
		if !ok {
			t.Fatal("mcpServers key missing")
		}
		if _, ok := servers["claude-buddy"]; !ok {
			t.Error("claude-buddy server missing from .mcp.json")
		}
	})

	// Verify all skills exist.
	t.Run("skills", func(t *testing.T) {
		for _, skill := range buddySkills {
			p := filepath.Join(outputDir, "skills", skill.Dir, "SKILL.md")
			data, err := os.ReadFile(p)
			if err != nil {
				t.Errorf("skill %s: %v", skill.Dir, err)
				continue
			}
			if len(data) == 0 {
				t.Errorf("skill %s: empty file", skill.Dir)
			}
		}
	})

	// Verify agent exists.
	t.Run("agent", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(outputDir, "agents", "buddy.md"))
		if err != nil {
			t.Fatalf("read buddy.md: %v", err)
		}
		if len(data) == 0 {
			t.Error("buddy.md is empty")
		}
	})
}

func TestBundleIdempotent(t *testing.T) {
	t.Parallel()
	outputDir := t.TempDir()

	// Run twice — should succeed both times.
	if err := Bundle(outputDir, "1.0.0"); err != nil {
		t.Fatalf("first Bundle() error: %v", err)
	}
	if err := Bundle(outputDir, "1.0.1"); err != nil {
		t.Fatalf("second Bundle() error: %v", err)
	}

	// Verify version was updated.
	data, err := os.ReadFile(filepath.Join(outputDir, ".claude-plugin", "plugin.json"))
	if err != nil {
		t.Fatalf("read plugin.json: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("parse plugin.json: %v", err)
	}
	if got := m["version"]; got != "1.0.1" {
		t.Errorf("version = %v, want 1.0.1", got)
	}
}
