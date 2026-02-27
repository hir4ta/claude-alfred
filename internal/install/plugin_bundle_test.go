package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

	// Verify hooks.json has all 13 events (Stop removed by design).
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
			"UserPromptSubmit", "PreCompact", "SessionEnd",
			"SubagentStart", "SubagentStop", "Notification",
			"TeammateIdle", "TaskCompleted", "PermissionRequest",
		}
		for _, event := range expectedEvents {
			if _, ok := hooks[event]; !ok {
				t.Errorf("missing event: %s", event)
			}
		}

		// Stop hook should NOT be registered (removed by design).
		if _, ok := hooks["Stop"]; ok {
			t.Error("Stop hook should not be registered")
		}

		// All hook commands should use ${CLAUDE_PLUGIN_ROOT}.
		raw, _ := os.ReadFile(filepath.Join(outputDir, "hooks", "hooks.json"))
		content := string(raw)
		if !strings.Contains(content, "${CLAUDE_PLUGIN_ROOT}") {
			t.Error("hook commands should use ${CLAUDE_PLUGIN_ROOT}")
		}
		if strings.Contains(content, "$HOME/.claude/plugins") {
			t.Error("hook commands should not use hardcoded $HOME path")
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

	// Verify all skills exist, including init.
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

		// init skill must be user-invocable.
		initData, err := os.ReadFile(filepath.Join(outputDir, "skills", "init", "SKILL.md"))
		if err != nil {
			t.Fatalf("init skill missing: %v", err)
		}
		if !strings.Contains(string(initData), "user-invocable: true") {
			t.Error("init skill should be user-invocable")
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
