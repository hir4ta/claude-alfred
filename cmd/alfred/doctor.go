package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"

	"github.com/hir4ta/claude-alfred/internal/embedder"
	"github.com/hir4ta/claude-alfred/internal/store"
)

type checkResult struct {
	name    string
	status  string // "ok", "warn", "fail"
	message string
}

type doctorModel struct {
	table    table.Model
	fails    int
	warns    int
	showHelp bool
}

func newDoctorModel() doctorModel {
	checks := runChecks()

	var rows []table.Row
	var fails, warns int
	for _, c := range checks {
		var icon string
		switch c.status {
		case "ok":
			icon = "✓ ok"
		case "warn":
			icon = "⚠ warn"
			warns++
		case "fail":
			icon = "✗ fail"
			fails++
		}
		rows = append(rows, table.Row{icon, c.name, c.message})
	}

	columns := []table.Column{
		{Title: "Status", Width: 8},
		{Title: "Check", Width: 16},
		{Title: "Details", Width: 56},
	}

	totalWidth := 0
	for _, c := range columns {
		totalWidth += c.Width + 2 // column width + padding
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithWidth(totalWidth),
		table.WithHeight(len(rows)),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		Bold(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(lipgloss.Color("#626262"))
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)
	t.UpdateViewport()

	return doctorModel{table: t, fails: fails, warns: warns}
}

func (m doctorModel) Init() tea.Cmd { return nil }

func (m doctorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			if m.showHelp {
				m.showHelp = false
				return m, nil
			}
			return m, tea.Quit
		case "?":
			m.showHelp = !m.showHelp
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m doctorModel) View() tea.View {
	if m.showHelp {
		return tea.NewView(m.renderHelpOverlay())
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7571F9"))
	okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB627"))
	failStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4672"))

	var b strings.Builder
	b.WriteString("\n  " + headerStyle.Render("alfred doctor") + "\n\n")
	b.WriteString("  " + m.table.View() + "\n\n")

	// Summary.
	if m.fails > 0 {
		b.WriteString("  " + failStyle.Render(fmt.Sprintf("%d issue(s) need attention", m.fails)))
		if m.warns > 0 {
			b.WriteString(", " + warnStyle.Render(fmt.Sprintf("%d warning(s)", m.warns)))
		}
		b.WriteString("\n")
	} else if m.warns > 0 {
		b.WriteString("  " + warnStyle.Render(fmt.Sprintf("%d warning(s), no critical issues", m.warns)) + "\n")
	} else {
		b.WriteString("  " + okStyle.Render("All checks passed") + "\n")
	}

	b.WriteString("\n")
	h := newHelp()
	b.WriteString("  " + h.View(simpleKeyMap{keyUp, keyDown, keyHelp, keyQuit}) + "\n")

	return tea.NewView(b.String())
}

func (m doctorModel) renderHelpOverlay() string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7571F9"))
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFB627"))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA"))

	var b strings.Builder
	b.WriteString("\n  " + headerStyle.Render("Doctor Checks") + "\n\n")

	sections := []struct{ title, desc string }{
		{"Database", "SQLite database existence, accessibility, and file size."},
		{"Schema", "Whether the DB schema version matches the expected version.\n" +
			"    A mismatch means a migration may be needed (run 'alfred init')."},
		{"Seed docs", "Number of knowledge documents in the database.\n" +
			"    If zero, run 'alfred init' to populate the knowledge base."},
		{"FTS index", "Full-text search index integrity check.\n" +
			"    A failure means the FTS index may be corrupted (rebuild with 'alfred init')."},
		{"Plugin", "Whether the alfred MCP plugin is installed in ~/.claude/plugins/."},
		{"Hooks", "Whether hooks.json exists in the plugin directory.\n" +
			"    Hooks enable automatic knowledge injection on each prompt."},
		{"Bootstrap", "Whether run.sh is present and executable in the plugin.\n" +
			"    Required for Claude Code to start the MCP server."},
		{"Voyage API", "Voyage AI API key availability.\n" +
			"    Optional — without it, alfred uses FTS-only mode (no vector search)."},
		{"Embeddings", "Number of vector embeddings stored in the database.\n" +
			"    Zero is normal in FTS-only mode (no Voyage API key)."},
		{"Last crawl", "When the knowledge base was last refreshed from source docs.\n" +
			"    Auto-crawl runs every 7 days (configurable via ALFRED_CRAWL_INTERVAL_DAYS)."},
		{"Config dir", "Whether ~/.claude-alfred/ directory exists.\n" +
			"    Used for global settings, custom dictionary, and source overrides."},
	}

	for _, s := range sections {
		b.WriteString("  " + titleStyle.Render(s.title) + "\n")
		b.WriteString("  " + descStyle.Render(s.desc) + "\n\n")
	}

	b.WriteString("  " + descStyle.Render("Press ? or Esc to close") + "\n")
	return b.String()
}

// runChecks gathers all diagnostic check results.
func runChecks() []checkResult {
	var checks []checkResult

	// 1. DB exists & opens.
	dbPath := store.DefaultDBPath()
	st, dbErr := store.OpenDefault()
	if dbErr != nil {
		checks = append(checks, checkResult{"Database", "fail", fmt.Sprintf("cannot open: %v", dbErr)})
	} else {
		defer st.Close()
		if fi, err := os.Stat(dbPath); err == nil {
			checks = append(checks, checkResult{"Database", "ok", fmt.Sprintf("%.1f MB", float64(fi.Size())/(1024*1024))})
		} else {
			checks = append(checks, checkResult{"Database", "ok", "opened"})
		}
	}

	// 2. Schema version.
	if st != nil {
		current := st.SchemaVersionCurrent()
		expected := store.SchemaVersion()
		if current == expected {
			checks = append(checks, checkResult{"Schema", "ok", fmt.Sprintf("version %d (current)", current)})
		} else {
			checks = append(checks, checkResult{"Schema", "warn", fmt.Sprintf("version %d (expected %d)", current, expected)})
		}
	}

	// 3. Seed docs.
	if st != nil {
		count, _ := st.SeedDocsCount()
		if count > 0 {
			checks = append(checks, checkResult{"Seed docs", "ok", fmt.Sprintf("%d docs", count)})
		} else {
			checks = append(checks, checkResult{"Seed docs", "warn", "no docs — run 'alfred init'"})
		}
	}

	// 4. FTS integrity.
	if st != nil {
		if err := st.FTSIntegrityCheck(); err != nil {
			checks = append(checks, checkResult{"FTS index", "warn", fmt.Sprintf("integrity check failed: %v", err)})
		} else {
			checks = append(checks, checkResult{"FTS index", "ok", "integrity check passed"})
		}
	}

	// 5. Plugin installed.
	pluginRoot := findInstalledPluginRoot()
	if pluginRoot != "" {
		home, _ := os.UserHomeDir()
		display := pluginRoot
		if home != "" {
			display = strings.Replace(display, home, "~", 1)
		}
		checks = append(checks, checkResult{"Plugin", "ok", display})
	} else {
		checks = append(checks, checkResult{"Plugin", "fail", "not found — run 'claude mcp add' or reinstall"})
	}

	// 6. Hooks registered.
	if pluginRoot != "" {
		hooksPath := filepath.Join(pluginRoot, "hooks", "hooks.json")
		if _, err := os.Stat(hooksPath); err == nil {
			checks = append(checks, checkResult{"Hooks", "ok", "hooks.json present"})
		} else {
			checks = append(checks, checkResult{"Hooks", "warn", "hooks.json missing — reinstall plugin"})
		}
	}

	// 7. run.sh executable.
	if pluginRoot != "" {
		runSh := filepath.Join(pluginRoot, "bin", "run.sh")
		if fi, err := os.Stat(runSh); err == nil {
			if fi.Mode()&0111 != 0 {
				checks = append(checks, checkResult{"Bootstrap", "ok", "run.sh executable"})
			} else {
				checks = append(checks, checkResult{"Bootstrap", "fail", fmt.Sprintf("run.sh not executable — chmod +x %s", runSh)})
			}
		} else {
			checks = append(checks, checkResult{"Bootstrap", "warn", "run.sh not found"})
		}
	}

	// 8. Voyage API key.
	if _, err := embedder.NewEmbedder(); err == nil {
		checks = append(checks, checkResult{"Voyage API", "ok", "key valid"})
	} else {
		checks = append(checks, checkResult{"Voyage API", "warn", "not set (FTS-only mode)"})
	}

	// 9. Embeddings.
	if st != nil {
		count, _ := st.CountEmbeddings()
		if count > 0 {
			checks = append(checks, checkResult{"Embeddings", "ok", fmt.Sprintf("%d vectors", count)})
		} else {
			checks = append(checks, checkResult{"Embeddings", "warn", "none (normal without API key; 'alfred init' with key to enable)"})
		}
	}

	// 10. Crawl freshness.
	if st != nil {
		if t, err := st.LastCrawledAt(); err == nil {
			checks = append(checks, checkResult{"Last crawl", "ok", t.Format("2006-01-02")})
		} else {
			checks = append(checks, checkResult{"Last crawl", "warn", "never — auto-crawls on next session, or 'alfred harvest'"})
		}
	}

	// 11. Config dir.
	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".claude-alfred")
	if _, err := os.Stat(configDir); err == nil {
		checks = append(checks, checkResult{"Config dir", "ok", "~/.claude-alfred/"})
	} else {
		checks = append(checks, checkResult{"Config dir", "warn", "~/.claude-alfred/ not found (created on first use)"})
	}

	// 12. MCP server reachability.
	checks = append(checks, checkMCPReachability()...)

	return checks
}

// knownPackageRunners are commands that download and run packages at runtime.
// exec.LookPath confirms the runner exists but cannot verify package availability.
var knownPackageRunners = map[string]bool{
	"npx": true, "uvx": true, "bunx": true, "pipx": true,
}

// checkMCPReachability verifies that MCP server commands in .mcp.json are accessible.
func checkMCPReachability() []checkResult {
	// Walk upward from CWD to find .mcp.json.
	mcpPath := findMCPConfig()
	if mcpPath == "" {
		return []checkResult{{"MCP servers", "ok", "no .mcp.json found"}}
	}

	data, err := os.ReadFile(mcpPath)
	if err != nil {
		return []checkResult{{"MCP servers", "warn", fmt.Sprintf("cannot read %s: %v", mcpPath, err)}}
	}

	var config struct {
		MCPServers map[string]struct {
			Command string `json:"command"`
			URL     string `json:"url"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return []checkResult{{"MCP servers", "warn", fmt.Sprintf("invalid JSON in %s", mcpPath)}}
	}

	if len(config.MCPServers) == 0 {
		return []checkResult{{"MCP servers", "ok", "no servers configured"}}
	}

	var results []checkResult
	for name, srv := range config.MCPServers {
		label := "MCP:" + name
		if srv.Command != "" {
			// stdio transport: check command accessibility.
			bin := strings.Fields(srv.Command)[0]
			resolved, err := exec.LookPath(bin)
			if err != nil {
				results = append(results, checkResult{label, "warn", fmt.Sprintf("command not found: %s", bin)})
			} else {
				msg := resolved
				if knownPackageRunners[bin] {
					msg += " (package availability not verified)"
				}
				results = append(results, checkResult{label, "ok", msg})
			}
		} else if srv.URL != "" {
			// http/sse transport: no binary to check.
			results = append(results, checkResult{label, "info", fmt.Sprintf("http transport: %s", srv.URL)})
		} else {
			results = append(results, checkResult{label, "warn", "no command or url configured"})
		}
	}
	return results
}

// findMCPConfig walks upward from CWD to find .mcp.json.
func findMCPConfig() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(dir, ".mcp.json")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	// Also check ~/.claude/.mcp.json as fallback.
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	candidate := filepath.Join(home, ".claude", ".mcp.json")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return ""
}

func runDoctor() error {
	m := newDoctorModel()
	_, err := tea.NewProgram(m).Run()
	return err
}
