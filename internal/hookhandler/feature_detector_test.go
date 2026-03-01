package hookhandler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hir4ta/claude-alfred/internal/sessiondb"
)

func TestDetectFeatures_Empty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	fu := DetectFeatures(dir)
	if fu.Level != LevelBeginner {
		t.Errorf("empty project level = %q, want beginner", fu.Level)
	}
	if fu.HasCLAUDEMD {
		t.Error("expected HasCLAUDEMD = false")
	}
}

func TestDetectFeatures_CLAUDEMDOnly(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Project"), 0o644)

	fu := DetectFeatures(dir)
	if !fu.HasCLAUDEMD {
		t.Error("expected HasCLAUDEMD = true")
	}
	if fu.Level != LevelBeginner {
		t.Errorf("CLAUDE.md only level = %q, want beginner", fu.Level)
	}
}

func TestDetectFeatures_Intermediate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")

	// CLAUDE.md
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Project"), 0o644)

	// Hooks
	os.MkdirAll(claudeDir, 0o755)
	hooks := map[string]any{"PreToolUse": []any{map[string]any{"hooks": []any{}}}}
	data, _ := json.Marshal(hooks)
	os.WriteFile(filepath.Join(claudeDir, "hooks.json"), data, 0o644)

	// Skills (1 skill dir)
	os.MkdirAll(filepath.Join(claudeDir, "skills", "my-skill"), 0o755)

	fu := DetectFeatures(dir)
	if fu.Level != LevelIntermediate {
		t.Errorf("level = %q, want intermediate", fu.Level)
	}
	if !fu.HasHooks {
		t.Error("expected HasHooks = true")
	}
	if !fu.HasSkills {
		t.Error("expected HasSkills = true")
	}
	if fu.SkillCount != 1 {
		t.Errorf("SkillCount = %d, want 1", fu.SkillCount)
	}
}

func TestDetectFeatures_Advanced(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")

	// CLAUDE.md + hooks + skills + rules + MCP = 5 features → advanced
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Project"), 0o644)

	os.MkdirAll(claudeDir, 0o755)
	hooks := map[string]any{"PreToolUse": []any{map[string]any{}}}
	data, _ := json.Marshal(hooks)
	os.WriteFile(filepath.Join(claudeDir, "hooks.json"), data, 0o644)

	os.MkdirAll(filepath.Join(claudeDir, "skills", "s1"), 0o755)
	os.MkdirAll(filepath.Join(claudeDir, "skills", "s2"), 0o755)

	os.MkdirAll(filepath.Join(claudeDir, "rules"), 0o755)
	os.WriteFile(filepath.Join(claudeDir, "rules", "style.md"), []byte("# Style"), 0o644)

	mcpCfg := map[string]any{"mcpServers": map[string]any{"server1": map[string]any{}, "server2": map[string]any{}}}
	mcpData, _ := json.Marshal(mcpCfg)
	os.WriteFile(filepath.Join(dir, ".mcp.json"), mcpData, 0o644)

	fu := DetectFeatures(dir)
	if fu.Level != LevelAdvanced {
		t.Errorf("level = %q, want advanced", fu.Level)
	}
	if fu.MCPServerCount != 2 {
		t.Errorf("MCPServerCount = %d, want 2", fu.MCPServerCount)
	}
	if fu.SkillCount != 2 {
		t.Errorf("SkillCount = %d, want 2", fu.SkillCount)
	}
	if fu.RuleCount != 1 {
		t.Errorf("RuleCount = %d, want 1", fu.RuleCount)
	}
}

func TestDetectFeatures_Memory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// MEMORY.md at root.
	os.WriteFile(filepath.Join(dir, "MEMORY.md"), []byte("# Memory"), 0o644)

	fu := DetectFeatures(dir)
	if !fu.HasMemory {
		t.Error("expected HasMemory = true for MEMORY.md")
	}

	// .claude/memory/ directory.
	dir2 := t.TempDir()
	memDir := filepath.Join(dir2, ".claude", "memory")
	os.MkdirAll(memDir, 0o755)
	os.WriteFile(filepath.Join(memDir, "notes.md"), []byte("note"), 0o644)

	fu2 := DetectFeatures(dir2)
	if !fu2.HasMemory {
		t.Error("expected HasMemory = true for .claude/memory/")
	}
}

func TestCacheFeatureUsage(t *testing.T) {
	t.Parallel()
	sdb, err := sessiondb.Open("test-feature-cache-" + t.Name())
	if err != nil {
		t.Fatalf("sessiondb.Open: %v", err)
	}
	t.Cleanup(func() { _ = sdb.Destroy() })

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Test"), 0o644)

	fu := CacheFeatureUsage(sdb, dir)
	if fu == nil {
		t.Fatal("CacheFeatureUsage returned nil")
	}
	if !fu.HasCLAUDEMD {
		t.Error("expected HasCLAUDEMD = true")
	}

	// Retrieve cached.
	cached := GetCachedFeatureUsage(sdb)
	if cached == nil {
		t.Fatal("GetCachedFeatureUsage returned nil")
	}
	if cached.Level != fu.Level {
		t.Errorf("cached level = %q, want %q", cached.Level, fu.Level)
	}
}
