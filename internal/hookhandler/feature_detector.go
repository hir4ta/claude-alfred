package hookhandler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/hir4ta/claude-alfred/internal/sessiondb"
)

// ProficiencyLevel represents the user's Claude Code proficiency.
type ProficiencyLevel string

const (
	LevelBeginner     ProficiencyLevel = "beginner"
	LevelIntermediate ProficiencyLevel = "intermediate"
	LevelAdvanced     ProficiencyLevel = "advanced"
)

// FeatureUsage captures which Claude Code features a project uses.
type FeatureUsage struct {
	HasCLAUDEMD    bool             `json:"has_claude_md"`
	HasHooks       bool             `json:"has_hooks"`
	HookCount      int              `json:"hook_count,omitempty"`
	HasSkills      bool             `json:"has_skills"`
	SkillCount     int              `json:"skill_count,omitempty"`
	HasAgents      bool             `json:"has_agents"`
	HasRules       bool             `json:"has_rules"`
	RuleCount      int              `json:"rule_count,omitempty"`
	HasMCP         bool             `json:"has_mcp"`
	MCPServerCount int              `json:"mcp_server_count,omitempty"`
	HasMemory      bool             `json:"has_memory"`
	HasPermissions bool             `json:"has_permissions"`
	Level          ProficiencyLevel `json:"level"`
}

// DetectFeatures scans the project directory for Claude Code feature usage.
// projectDir should be the git root or project root.
func DetectFeatures(projectDir string) *FeatureUsage {
	fu := &FeatureUsage{}

	// CLAUDE.md at project root.
	if fileExists(filepath.Join(projectDir, "CLAUDE.md")) {
		fu.HasCLAUDEMD = true
	}

	claudeDir := filepath.Join(projectDir, ".claude")

	// Hooks: .claude/hooks.json or hooks in settings.
	fu.HookCount = countHooks(claudeDir)
	fu.HasHooks = fu.HookCount > 0

	// Skills: .claude/skills/*/SKILL.md
	fu.SkillCount = countDirEntries(filepath.Join(claudeDir, "skills"))
	fu.HasSkills = fu.SkillCount > 0

	// Agents: .claude/agents/*.md
	fu.HasAgents = countFilesWithExt(filepath.Join(claudeDir, "agents"), ".md") > 0

	// Rules: .claude/rules/*.md
	fu.RuleCount = countFilesWithExt(filepath.Join(claudeDir, "rules"), ".md")
	fu.HasRules = fu.RuleCount > 0

	// MCP: .mcp.json or .claude/mcp.json
	fu.MCPServerCount = countMCPServers(projectDir, claudeDir)
	fu.HasMCP = fu.MCPServerCount > 0

	// Memory: .claude/memory/ or MEMORY.md
	fu.HasMemory = dirHasFiles(filepath.Join(claudeDir, "memory")) ||
		fileExists(filepath.Join(projectDir, "MEMORY.md"))

	// Permissions: .claude/permissions (settings.json permissions section)
	fu.HasPermissions = fileExists(filepath.Join(claudeDir, "permissions.json"))

	fu.Level = classifyLevel(fu)
	return fu
}

// classifyLevel determines proficiency based on feature adoption.
func classifyLevel(fu *FeatureUsage) ProficiencyLevel {
	featureCount := 0
	if fu.HasCLAUDEMD {
		featureCount++
	}
	if fu.HasHooks {
		featureCount++
	}
	if fu.HasSkills {
		featureCount++
	}
	if fu.HasAgents {
		featureCount++
	}
	if fu.HasRules {
		featureCount++
	}
	if fu.HasMCP {
		featureCount++
	}
	if fu.HasMemory {
		featureCount++
	}

	switch {
	case featureCount >= 5:
		return LevelAdvanced
	case featureCount >= 2:
		return LevelIntermediate
	default:
		return LevelBeginner
	}
}

// CacheFeatureUsage detects features and stores the result in sessiondb.
func CacheFeatureUsage(sdb *sessiondb.SessionDB, projectDir string) *FeatureUsage {
	if projectDir == "" {
		return nil
	}
	fu := DetectFeatures(projectDir)
	data, err := json.Marshal(fu)
	if err != nil {
		return fu
	}
	_ = sdb.SetContext("feature_usage", string(data))
	return fu
}

// GetCachedFeatureUsage retrieves the cached feature usage from sessiondb.
func GetCachedFeatureUsage(sdb *sessiondb.SessionDB) *FeatureUsage {
	raw, _ := sdb.GetContext("feature_usage")
	if raw == "" {
		return nil
	}
	var fu FeatureUsage
	if json.Unmarshal([]byte(raw), &fu) != nil {
		return nil
	}
	return &fu
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirHasFiles(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() {
			return true
		}
	}
	return false
}

func countDirEntries(path string) int {
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() {
			count++
		}
	}
	return count
}

func countFilesWithExt(dir, ext string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ext) {
			count++
		}
	}
	return count
}

func countHooks(claudeDir string) int {
	// Check .claude/hooks.json
	data, err := os.ReadFile(filepath.Join(claudeDir, "hooks.json"))
	if err != nil {
		return 0
	}
	var hooks map[string]any
	if json.Unmarshal(data, &hooks) != nil {
		return 0
	}
	count := 0
	for _, v := range hooks {
		if arr, ok := v.([]any); ok {
			count += len(arr)
		}
	}
	return count
}

func countMCPServers(projectDir, claudeDir string) int {
	count := 0
	for _, path := range []string{
		filepath.Join(projectDir, ".mcp.json"),
		filepath.Join(claudeDir, "mcp.json"),
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var cfg map[string]any
		if json.Unmarshal(data, &cfg) != nil {
			continue
		}
		if servers, ok := cfg["mcpServers"].(map[string]any); ok {
			count += len(servers)
		}
	}
	return count
}
