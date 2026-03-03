package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// suggestHandler returns a handler that analyzes recent code changes and
// suggests .claude/ configuration updates.
func suggestHandler(claudeHome string) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		projectPath := req.GetString("project_path", "")
		if projectPath == "" {
			return mcp.NewToolResultError("project_path is required"), nil
		}

		// Collect git diff information.
		diff := collectDiff(projectPath)
		if diff.err != "" {
			return mcp.NewToolResultError(diff.err), nil
		}
		if len(diff.files) == 0 {
			return marshalResult(map[string]any{
				"project_path": projectPath,
				"suggestions":  []string{},
				"summary":      "no recent changes detected",
			})
		}

		// Analyze current .claude/ config.
		config := analyzeConfig(projectPath, claudeHome)

		// Generate suggestions.
		suggestions := generateSuggestions(diff, config)

		result := map[string]any{
			"project_path":     projectPath,
			"changed_files":    len(diff.files),
			"diff_scope":       diff.scope,
			"suggestions":      suggestions,
			"suggestion_count": len(suggestions),
		}

		if len(suggestions) == 0 {
			result["summary"] = "no configuration changes suggested for the recent diff"
		}

		return marshalResult(result)
	}
}

// ---------------------------------------------------------------------------
// Git diff collection
// ---------------------------------------------------------------------------

type diffInfo struct {
	scope string   // "staged", "unstaged", or "recent_commits"
	files []string // changed file paths
	dirs  []string // unique top-level directories touched
	err   string
}

func collectDiff(projectPath string) diffInfo {
	// Check if we're in a git repo first.
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = projectPath
	if err := cmd.Run(); err != nil {
		return diffInfo{} // not a git repo — no changes
	}

	// Try staged changes first, then unstaged, then recent commits.
	if files := gitDiffFiles(projectPath, "--cached"); len(files) > 0 {
		return buildDiffInfo("staged", files)
	}
	if files := gitDiffFiles(projectPath); len(files) > 0 {
		return buildDiffInfo("unstaged", files)
	}
	if files := gitLogFiles(projectPath, 10); len(files) > 0 {
		return buildDiffInfo("recent_commits", files)
	}
	return diffInfo{}
}

func gitDiffFiles(projectPath string, args ...string) []string {
	cmdArgs := append([]string{"diff", "--name-only"}, args...)
	cmd := exec.Command("git", cmdArgs...)
	cmd.Dir = projectPath
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var files []string
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			files = append(files, line)
		}
	}
	return files
}

// gitLogFiles returns file paths changed in the last n commits.
func gitLogFiles(projectPath string, n int) []string {
	cmd := exec.Command("git", "log", "--name-only", "--format=", "-"+strconv.Itoa(n))
	cmd.Dir = projectPath
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var files []string
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		if line != "" && !seen[line] {
			seen[line] = true
			files = append(files, line)
		}
	}
	return files
}

func buildDiffInfo(scope string, files []string) diffInfo {
	dirSet := map[string]bool{}
	for _, f := range files {
		top := strings.SplitN(f, "/", 2)[0]
		dirSet[top] = true
	}
	dirs := make([]string, 0, len(dirSet))
	for d := range dirSet {
		dirs = append(dirs, d)
	}
	return diffInfo{scope: scope, files: files, dirs: dirs}
}

// ---------------------------------------------------------------------------
// Config analysis
// ---------------------------------------------------------------------------

type configState struct {
	hasClaudeMD bool
	claudeMDSections []string
	skillNames  []string
	ruleNames   []string
	hookEvents  []string
	hasMCP      bool
}

func analyzeConfig(projectPath, claudeHome string) configState {
	var cs configState

	// CLAUDE.md
	if data, err := os.ReadFile(filepath.Join(projectPath, "CLAUDE.md")); err == nil {
		cs.hasClaudeMD = true
		cs.claudeMDSections = extractH2Sections(string(data))
	}

	// Skills
	if entries, err := os.ReadDir(filepath.Join(projectPath, ".claude", "skills")); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				cs.skillNames = append(cs.skillNames, e.Name())
			}
		}
	}

	// Rules
	if entries, err := os.ReadDir(filepath.Join(projectPath, ".claude", "rules")); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				cs.ruleNames = append(cs.ruleNames, e.Name())
			}
		}
	}

	// Hooks
	cs.hookEvents = readHookEvents(claudeHome)

	// MCP
	if _, err := os.Stat(filepath.Join(projectPath, ".mcp.json")); err == nil {
		cs.hasMCP = true
	}

	return cs
}

func readHookEvents(claudeHome string) []string {
	data, err := os.ReadFile(filepath.Join(claudeHome, "settings.json"))
	if err != nil {
		return nil
	}
	// Quick extraction without full JSON parse.
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	hooks, ok := m["hooks"].(map[string]any)
	if !ok {
		return nil
	}
	events := make([]string, 0, len(hooks))
	for ev := range hooks {
		events = append(events, ev)
	}
	return events
}

// ---------------------------------------------------------------------------
// Suggestion generation
// ---------------------------------------------------------------------------

func generateSuggestions(diff diffInfo, config configState) []string {
	var s []string

	// Categorize changed files.
	var (
		configFiles  []string // .claude/ files
		testFiles    []string
		ciFiles      []string
		docFiles     []string
		sourceFiles  []string
	)

	for _, f := range diff.files {
		switch {
		case strings.HasPrefix(f, ".claude/"):
			configFiles = append(configFiles, f)
		case strings.Contains(f, "_test.go") || strings.HasPrefix(f, "test/") || strings.HasPrefix(f, "tests/"):
			testFiles = append(testFiles, f)
		case strings.HasPrefix(f, ".github/"):
			ciFiles = append(ciFiles, f)
		case strings.HasSuffix(f, ".md"):
			docFiles = append(docFiles, f)
		default:
			sourceFiles = append(sourceFiles, f)
		}
	}

	// 1. CLAUDE.md checks
	if !config.hasClaudeMD && len(sourceFiles) > 0 {
		s = append(s, "Create CLAUDE.md — source files changed but no CLAUDE.md exists to guide Claude Code")
	}

	if config.hasClaudeMD {
		hasCmdSection := false
		for _, sec := range config.claudeMDSections {
			if strings.Contains(strings.ToLower(sec), "command") {
				hasCmdSection = true
				break
			}
		}

		// New packages/directories might need structure update
		if hasNewDirs(diff, config) {
			s = append(s, "Update CLAUDE.md ## Structure — new directories detected in diff")
		}

		// CI changes might affect commands
		if len(ciFiles) > 0 && hasCmdSection {
			s = append(s, "Review CLAUDE.md ## Commands — CI workflow files changed")
		}
	}

	// 2. Test pattern changes
	if len(testFiles) > 0 && !hasRule(config, "test") {
		s = append(s, "Consider adding .claude/rules/ for testing conventions — test files changed")
	}

	// 3. .claude/ config files changed
	for _, f := range configFiles {
		switch {
		case strings.Contains(f, "skills/"):
			s = append(s, "Skill file changed: "+f+" — verify frontmatter (name, description) is complete")
		case strings.Contains(f, "rules/"):
			s = append(s, "Rule file changed: "+f+" — verify rule is clear and actionable")
		}
	}

	// 4. New file types that might need rules
	exts := uniqueExtensions(sourceFiles)
	for _, ext := range exts {
		if !hasRuleForExt(config, ext) && isSignificantExt(ext) {
			s = append(s, "New "+ext+" files detected — consider adding .claude/rules/ for "+extLanguage(ext)+" conventions")
		}
	}

	return s
}

func hasNewDirs(diff diffInfo, config configState) bool {
	if !config.hasClaudeMD {
		return false
	}
	lowerSections := strings.ToLower(strings.Join(config.claudeMDSections, " "))
	for _, d := range diff.dirs {
		if !strings.Contains(lowerSections, strings.ToLower(d)) {
			return true
		}
	}
	return false
}

func hasRule(config configState, keyword string) bool {
	for _, r := range config.ruleNames {
		if strings.Contains(strings.ToLower(r), keyword) {
			return true
		}
	}
	return false
}

func hasRuleForExt(config configState, ext string) bool {
	lang := extLanguage(ext)
	return hasRule(config, lang) || hasRule(config, strings.TrimPrefix(ext, "."))
}

func uniqueExtensions(files []string) []string {
	seen := map[string]bool{}
	var exts []string
	for _, f := range files {
		ext := filepath.Ext(f)
		if ext != "" && !seen[ext] {
			seen[ext] = true
			exts = append(exts, ext)
		}
	}
	return exts
}

func isSignificantExt(ext string) bool {
	switch ext {
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".py", ".rs", ".java", ".rb", ".swift", ".kt":
		return true
	}
	return false
}

func extLanguage(ext string) string {
	switch ext {
	case ".go":
		return "go"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx":
		return "javascript"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".rb":
		return "ruby"
	case ".swift":
		return "swift"
	case ".kt":
		return "kotlin"
	default:
		return strings.TrimPrefix(ext, ".")
	}
}

