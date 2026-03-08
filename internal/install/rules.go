package install

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed content/rules/*.md
var rulesFS embed.FS

type ruleDef struct {
	File    string // filename under rules/
	Content string // markdown content
}

// loadRules reads all rule definitions from the embedded filesystem.
func loadRules() []ruleDef {
	var rules []ruleDef
	entries, err := fs.ReadDir(rulesFS, "content/rules")
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := fs.ReadFile(rulesFS, "content/rules/"+e.Name())
		if err != nil {
			continue
		}
		rules = append(rules, ruleDef{File: e.Name(), Content: string(data)})
	}
	return rules
}

// InstallUserRules copies alfred rule files to ~/.claude/rules/ so they are
// loaded by Claude Code globally. Files are prefixed with "alfred-" to avoid
// collisions with user rules. Skips files whose content is already up-to-date.
// Also removes deprecated rule files from previous versions.
// Returns the number of files actually written (not skipped).
func InstallUserRules() (int, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return 0, fmt.Errorf("user home dir: %w", err)
	}
	rulesDir := filepath.Join(home, ".claude", "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		return 0, fmt.Errorf("mkdir %s: %w", rulesDir, err)
	}

	// Clean up deprecated rule files from previous versions.
	for _, name := range deprecatedRuleFiles {
		_ = os.Remove(filepath.Join(rulesDir, name))
		_ = os.Remove(filepath.Join(rulesDir, "alfred-"+name))
	}

	var written int
	rules := loadRules()
	for _, r := range rules {
		name := r.File
		// Ensure alfred- prefix to avoid collisions.
		if !strings.HasPrefix(name, "alfred") {
			name = "alfred-" + name
		}
		p := filepath.Join(rulesDir, name)
		content := []byte(r.Content)

		// Skip write if content is already up-to-date.
		if existing, err := os.ReadFile(p); err == nil && bytes.Equal(existing, content) {
			continue
		}

		if err := os.WriteFile(p, content, 0o644); err != nil {
			return written, fmt.Errorf("write rule %s: %w", name, err)
		}
		written++
	}
	return written, nil
}

// deprecatedRuleFiles lists rule files from previous versions that
// should be cleaned up during install/uninstall.
var deprecatedRuleFiles = []string{
	"butler-protocol.md",
	// v0.43 era: moved to skill supporting files
	"agents.md",
	"claude-md.md",
	"hooks.md",
	"memory.md",
	"mcp-config.md",
	"rules.md",
	"skills.md",
}
