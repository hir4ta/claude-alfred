package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/hir4ta/claude-alfred/internal/embedder"
	"github.com/hir4ta/claude-alfred/internal/store"
)

type checkResult struct {
	name    string
	status  string // "ok", "warn", "fail"
	message string
}

func runDoctor() error {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7571F9"))
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
	okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB627"))
	failStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4672"))

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString("  " + headerStyle.Render("alfred doctor") + "\n")
	b.WriteString("  " + mutedStyle.Render(strings.Repeat("─", 42)) + "\n\n")

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
		checks = append(checks, checkResult{"Plugin", "fail", "not found"})
	}

	// 6. Hooks registered.
	if pluginRoot != "" {
		hooksPath := filepath.Join(pluginRoot, "hooks", "hooks.json")
		if _, err := os.Stat(hooksPath); err == nil {
			checks = append(checks, checkResult{"Hooks", "ok", "hooks.json present"})
		} else {
			checks = append(checks, checkResult{"Hooks", "warn", "hooks.json missing"})
		}
	}

	// 7. run.sh executable.
	if pluginRoot != "" {
		runSh := filepath.Join(pluginRoot, "bin", "run.sh")
		if fi, err := os.Stat(runSh); err == nil {
			if fi.Mode()&0111 != 0 {
				checks = append(checks, checkResult{"Bootstrap", "ok", "run.sh executable"})
			} else {
				checks = append(checks, checkResult{"Bootstrap", "fail", "run.sh not executable"})
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
			checks = append(checks, checkResult{"Embeddings", "warn", "none"})
		}
	}

	// 10. Crawl freshness.
	if st != nil {
		if t, err := st.LastCrawledAt(); err == nil {
			checks = append(checks, checkResult{"Last crawl", "ok", t.Format("2006-01-02")})
		} else {
			checks = append(checks, checkResult{"Last crawl", "warn", "never crawled"})
		}
	}

	// 11. Config dir.
	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".claude-alfred")
	if _, err := os.Stat(configDir); err == nil {
		checks = append(checks, checkResult{"Config dir", "ok", "~/.claude-alfred/"})
	} else {
		checks = append(checks, checkResult{"Config dir", "warn", "~/.claude-alfred/ not found"})
	}

	// Render results.
	for _, c := range checks {
		var icon string
		switch c.status {
		case "ok":
			icon = okStyle.Render("[ok]  ")
		case "warn":
			icon = warnStyle.Render("[warn]")
		case "fail":
			icon = failStyle.Render("[fail]")
		}
		b.WriteString(fmt.Sprintf("  %s %-16s %s\n", icon, c.name, mutedStyle.Render(c.message)))
	}

	// Summary.
	fails := 0
	warns := 0
	for _, c := range checks {
		switch c.status {
		case "fail":
			fails++
		case "warn":
			warns++
		}
	}
	b.WriteString("\n")
	if fails > 0 {
		b.WriteString("  " + failStyle.Render(fmt.Sprintf("%d issue(s) need attention", fails)))
		if warns > 0 {
			b.WriteString(", " + warnStyle.Render(fmt.Sprintf("%d warning(s)", warns)))
		}
		b.WriteString("\n")
	} else if warns > 0 {
		b.WriteString("  " + warnStyle.Render(fmt.Sprintf("%d warning(s), no critical issues", warns)) + "\n")
	} else {
		b.WriteString("  " + okStyle.Render("All checks passed") + "\n")
	}

	b.WriteString("\n")
	fmt.Print(b.String())
	return nil
}
