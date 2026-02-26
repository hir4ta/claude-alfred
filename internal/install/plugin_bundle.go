package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Bundle generates the plugin directory structure from Go source definitions.
// The outputDir will contain .claude-plugin/, hooks/, skills/, agents/, and .mcp.json.
func Bundle(outputDir, version string) error {
	// 1. Create directory structure.
	dirs := []string{
		filepath.Join(outputDir, ".claude-plugin"),
		filepath.Join(outputDir, "hooks"),
		filepath.Join(outputDir, "agents"),
	}
	for _, skill := range buddySkills {
		dirs = append(dirs, filepath.Join(outputDir, "skills", skill.Dir))
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", d, err)
		}
	}

	// 2. Write plugin.json.
	pluginJSON := map[string]any{
		"name":        "claude-buddy",
		"version":     version,
		"description": "Proactive session advisor for Claude Code",
		"author":      map[string]string{"name": "hir4ta"},
		"homepage":    "https://github.com/hir4ta/claude-buddy",
		"repository":  "https://github.com/hir4ta/claude-buddy",
		"license":     "MIT",
		"keywords":    []string{"session-advisor", "anti-pattern", "workflow", "productivity"},
	}
	if err := writeJSON(filepath.Join(outputDir, ".claude-plugin", "plugin.json"), pluginJSON); err != nil {
		return fmt.Errorf("write plugin.json: %w", err)
	}

	// 3. Write hooks.json from buddyHookEntries.
	hooksJSON := map[string]any{
		"hooks": buddyHookEntries("claude-buddy"),
	}
	if err := writeJSON(filepath.Join(outputDir, "hooks", "hooks.json"), hooksJSON); err != nil {
		return fmt.Errorf("write hooks.json: %w", err)
	}

	// 4. Write .mcp.json.
	mcpJSON := map[string]any{
		"mcpServers": map[string]any{
			"claude-buddy": map[string]any{
				"command": "claude-buddy",
				"args":    []string{"serve"},
			},
		},
	}
	if err := writeJSON(filepath.Join(outputDir, ".mcp.json"), mcpJSON); err != nil {
		return fmt.Errorf("write .mcp.json: %w", err)
	}

	// 5. Write skills.
	for _, skill := range buddySkills {
		p := filepath.Join(outputDir, "skills", skill.Dir, "SKILL.md")
		if err := os.WriteFile(p, []byte(skill.Content), 0o644); err != nil {
			return fmt.Errorf("write skill %s: %w", skill.Dir, err)
		}
	}

	// 6. Write agent.
	agentPath := filepath.Join(outputDir, "agents", "buddy.md")
	if err := os.WriteFile(agentPath, []byte(buddyAgentContent), 0o644); err != nil {
		return fmt.Errorf("write buddy agent: %w", err)
	}

	hookCount := len(buddyHookEntries("claude-buddy"))

	fmt.Printf("✓ Plugin bundle generated at %s\n", outputDir)
	fmt.Printf("  - plugin.json (v%s)\n", version)
	fmt.Printf("  - hooks.json (%d events)\n", hookCount)
	fmt.Printf("  - .mcp.json\n")
	fmt.Printf("  - %d skills\n", len(buddySkills))
	fmt.Printf("  - 1 agent (buddy)\n")
	return nil
}

func writeJSON(path string, data any) error {
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0o644)
}
