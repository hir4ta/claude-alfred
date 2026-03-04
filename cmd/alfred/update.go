package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"
)

const (
	githubRepo  = "hir4ta/claude-alfred"
	installPath = "github.com/hir4ta/claude-alfred/cmd/alfred"
)

// showVersion prints a styled version display.
func showVersion() {
	ver := resolvedVersion()
	c := resolvedCommit()
	d := resolvedDate()

	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7571F9"))
	verStyle := lipgloss.NewStyle().Bold(true)
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))

	line := nameStyle.Render("alfred") + " " + verStyle.Render(ver)
	if c != "" {
		meta := c
		if d != "" {
			// Show date only (strip time).
			if t, err := time.Parse(time.RFC3339, d); err == nil {
				meta += " " + t.Format("2006-01-02")
			} else {
				meta += " " + d
			}
		}
		line += " " + metaStyle.Render("("+meta+")")
	}
	fmt.Println(line)
}

// --- update TUI ---

type updatePhase int

const (
	updateChecking updatePhase = iota
	updateUpToDate
	updateInstalling
	updateDone
	updateError
)

type (
	latestVersionMsg struct {
		version string
		err     error
	}
	installDoneMsg struct{ err error }
)

type updateModel struct {
	phase     updatePhase
	current   string
	latest    string
	err       error
	spinner   spinner.Model
	startTime time.Time
}

func newUpdateModel() updateModel {
	s := spinner.New(spinner.WithSpinner(spinner.Dot))
	s.Style = dimStyle
	return updateModel{
		phase:     updateChecking,
		current:   resolvedVersion(),
		spinner:   s,
		startTime: time.Now(),
	}
}

func (m updateModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, checkLatestVersion)
}

func (m updateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}

	case latestVersionMsg:
		if msg.err != nil {
			m.phase = updateError
			m.err = msg.err
			return m, tea.Quit
		}
		m.latest = msg.version
		if m.latest == m.current {
			m.phase = updateUpToDate
			return m, tea.Quit
		}
		m.phase = updateInstalling
		return m, doInstall(m.latest)

	case installDoneMsg:
		if msg.err != nil {
			m.phase = updateError
			m.err = msg.err
			return m, tea.Quit
		}
		m.phase = updateDone
		return m, tea.Quit

	case spinner.TickMsg:
		if m.phase == updateChecking || m.phase == updateInstalling {
			sm, cmd := m.spinner.Update(msg)
			m.spinner = sm
			return m, cmd
		}
		return m, nil
	}

	return m, nil
}

func (m updateModel) View() tea.View {
	var b strings.Builder

	b.WriteString("\n")

	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7571F9"))
	verStyle := lipgloss.NewStyle().Bold(true)
	arrowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7571F9"))

	switch m.phase {
	case updateChecking:
		b.WriteString(fmt.Sprintf("  %s Checking latest version %s\n",
			nameStyle.Render("alfred"), m.spinner.View()))

	case updateUpToDate:
		b.WriteString(fmt.Sprintf("  %s %s %s\n",
			nameStyle.Render("alfred"),
			verStyle.Render(m.current),
			doneStyle.Render("already up to date")))

	case updateInstalling:
		b.WriteString(fmt.Sprintf("  %s %s %s %s\n",
			nameStyle.Render("alfred"),
			dimStyle.Render(m.current),
			arrowStyle.Render("→"),
			verStyle.Render(m.latest)))
		b.WriteString(fmt.Sprintf("  Installing %s\n", m.spinner.View()))

	case updateDone:
		elapsed := time.Since(m.startTime).Round(time.Second)
		b.WriteString(fmt.Sprintf("  %s %s %s %s\n",
			nameStyle.Render("alfred"),
			dimStyle.Render(m.current),
			arrowStyle.Render("→"),
			verStyle.Render(m.latest)))
		b.WriteString(fmt.Sprintf("  %s (%s)\n",
			doneStyle.Render("✓ Updated"),
			elapsed))

	case updateError:
		b.WriteString(fmt.Sprintf("  %s %v\n",
			errStyle.Render("✗ Error:"), m.err))
	}

	b.WriteString("\n")
	return tea.NewView(b.String())
}

// checkLatestVersion fetches the latest release tag from GitHub API.
// This is more reliable than go list -m which depends on the Go module proxy.
func checkLatestVersion() tea.Msg {
	url := "https://api.github.com/repos/" + githubRepo + "/releases/latest"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return latestVersionMsg{err: fmt.Errorf("failed to check latest version: %w", err)}
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return latestVersionMsg{err: fmt.Errorf("failed to check latest version: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return latestVersionMsg{err: fmt.Errorf("failed to check latest version: HTTP %d", resp.StatusCode)}
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return latestVersionMsg{err: fmt.Errorf("failed to parse version info: %w", err)}
	}

	ver := strings.TrimPrefix(release.TagName, "v")
	return latestVersionMsg{version: ver}
}

// doInstall installs the specified version via go install, then regenerates
// the plugin bundle at the installed location so skills, rules, hooks, and
// run.sh are updated to match the new binary.
func doInstall(version string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("go", "install", installPath+"@v"+version)
		if out, err := cmd.CombinedOutput(); err != nil {
			return installDoneMsg{err: fmt.Errorf("%w: %s", err, out)}
		}

		// Regenerate plugin bundle at installed location (best-effort).
		if root := findInstalledPluginRoot(); root != "" {
			cmd = exec.Command("alfred", "plugin-bundle", root)
			cmd.Run()
		}

		return installDoneMsg{}
	}
}

// findInstalledPluginRoot reads ~/.claude/plugins/installed_plugins.json
// and returns the installPath for the alfred plugin.
func findInstalledPluginRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(home, ".claude", "plugins", "installed_plugins.json"))
	if err != nil {
		return ""
	}
	var manifest struct {
		Plugins map[string][]struct {
			InstallPath string `json:"installPath"`
		} `json:"plugins"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return ""
	}
	for key, entries := range manifest.Plugins {
		if strings.Contains(key, "alfred") && len(entries) > 0 {
			return entries[0].InstallPath
		}
	}
	return ""
}

func runUpdate() error {
	m := newUpdateModel()
	_, err := tea.NewProgram(m).Run()
	return err
}
