package tui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	"github.com/sergi/go-diff/diffmatchpatch"

	"github.com/hir4ta/claude-alfred/internal/spec"
)

// clipboardMsg is sent after clipboard operation completes.
type clipboardMsg struct{ err error }

// clipboardCommand returns the OS-appropriate clipboard command and args.
// Returns empty name if no clipboard tool is available.
func clipboardCommand() (name string, args []string) {
	switch runtime.GOOS {
	case "darwin":
		return "pbcopy", nil
	case "linux":
		if _, err := exec.LookPath("xclip"); err == nil {
			return "xclip", []string{"-selection", "clipboard"}
		}
		if _, err := exec.LookPath("xsel"); err == nil {
			return "xsel", []string{"--clipboard", "--input"}
		}
	}
	return "", nil
}

// copyToClipboard copies text to system clipboard using OS-appropriate tool.
func copyToClipboard(text string) tea.Cmd {
	return func() tea.Msg {
		name, args := clipboardCommand()
		if name == "" {
			return clipboardMsg{fmt.Errorf("no clipboard tool available")}
		}
		cmd := exec.Command(name, args...)
		cmd.Stdin = strings.NewReader(text)
		err := cmd.Run()
		return clipboardMsg{err}
	}
}

func (m *Model) openOverlay(title, content string, crumbs ...string) {
	m.overlayActive = true
	m.overlayTitle = title
	m.overlayRawMD = ""
	m.overlayCopied = false
	m.breadcrumbs = crumbs

	// Size the overlay viewport — use 85% of terminal width.
	w := min(m.width-4, m.width*85/100)
	if w < 60 {
		w = min(m.width-4, 60)
	}
	h := m.height - 8
	if h < 5 {
		h = 5
	}

	m.overlayVP = viewport.New(
		viewport.WithWidth(w-4), // padding inside border
		viewport.WithHeight(h-3),
	)
	m.overlayVP.SoftWrap = true
	m.overlayVP.SetContent(content)
}

func (m *Model) updateOverlay(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Review comment editing mode — text input takes all keys.
	if m.reviewEditing {
		return m.updateReviewInput(msg)
	}

	// Review mode navigation.
	if m.reviewMode {
		return m.updateReviewMode(msg)
	}

	// Reset "Copied!" flash on any key press.
	m.overlayCopied = false

	switch {
	case key.Matches(msg, keys.Back), key.Matches(msg, keys.Quit):
		m.overlayActive = false
		return m, nil
	case msg.String() == "r" && m.activeTab == tabTasks && m.taskLevel > 0:
		// Enter review mode — only when task has pending review.
		if m.specGroupCursor < len(m.specGroups) {
			slug := m.specGroups[m.specGroupCursor].Slug
			status := spec.ReviewStatusFor(m.ds.ProjectPath(), slug)
			if status != spec.ReviewPending {
				return m, nil // not pending — ignore
			}
		}
		m.enterReviewMode()
		return m, nil
	case msg.String() == "c" && m.overlayRawMD != "":
		// Copy raw markdown to clipboard.
		cmd := copyToClipboard(m.overlayRawMD)
		m.overlayCopied = true
		return m, cmd
	case msg.String() == "d" && m.activeTab == tabTasks && m.taskLevel > 0 && !m.reviewMode:
		// Show diff against previous version.
		m.showSpecDiff()
		return m, nil
	default:
		var cmd tea.Cmd
		m.overlayVP, cmd = m.overlayVP.Update(msg)
		return m, cmd
	}
}

// showSpecDiff shows a diff between the current spec file and its last saved version.
func (m *Model) showSpecDiff() {
	if m.specGroupCursor >= len(m.specGroups) {
		return
	}
	g := m.specGroups[m.specGroupCursor]
	if m.specFileCursor >= len(g.Files) {
		return
	}
	f := g.Files[m.specFileCursor]
	sd := &spec.SpecDir{ProjectPath: m.ds.ProjectPath(), TaskSlug: f.TaskSlug}

	// Get history entries.
	history, err := sd.History(spec.SpecFile(f.File))
	if err != nil || len(history) == 0 {
		m.overlayVP.SetContent(dimStyle.Render("  no previous versions"))
		m.overlayTitle = "Diff: " + f.File
		return
	}

	// Read current and previous version.
	current := m.ds.SpecContent(f.TaskSlug, f.File)
	prevData, err := os.ReadFile(history[0].Path)
	if err != nil {
		m.overlayVP.SetContent(dimStyle.Render("  cannot read previous version"))
		m.overlayTitle = "Diff: " + f.File
		return
	}
	previous := string(prevData)

	// Unified diff using go-diff.
	diff := renderUnifiedDiff(previous, current)
	ts, _ := time.Parse("20060102-150405", history[0].Timestamp)
	age := time.Since(ts)
	m.overlayTitle = fmt.Sprintf("Diff: %s (vs %s ago)", f.File, formatDuration(age))
	m.overlayVP.SetContent(diff)
}

// renderUnifiedDiff produces a colored unified diff between two texts.
// Uses go-diff for semantic line diff with context lines.
func renderUnifiedDiff(old, new string) string {
	diffs := dmp.DiffMain(old, new, true)
	diffs = dmp.DiffCleanupSemantic(diffs)

	addStyle := lipgloss.NewStyle().Foreground(green)
	delStyle := lipgloss.NewStyle().Foreground(red)
	headerStyle := lipgloss.NewStyle().Foreground(aqua)

	var b strings.Builder
	addedCount, removedCount := 0, 0

	for _, d := range diffs {
		lines := strings.Split(d.Text, "\n")
		// Last empty element from trailing newline — keep it attached.
		for i, line := range lines {
			// Skip the empty string that results from a trailing newline.
			if i == len(lines)-1 && line == "" {
				continue
			}
			switch d.Type {
			case diffmatchpatch.DiffDelete:
				b.WriteString(delStyle.Render("  - "+line) + "\n")
				removedCount++
			case diffmatchpatch.DiffInsert:
				b.WriteString(addStyle.Render("  + "+line) + "\n")
				addedCount++
			case diffmatchpatch.DiffEqual:
				b.WriteString("    " + line + "\n")
			}
		}
	}

	if removedCount == 0 && addedCount == 0 {
		b.WriteString(dimStyle.Render("  no changes"))
	} else {
		b.WriteString(fmt.Sprintf("\n  %s",
			headerStyle.Render(fmt.Sprintf("%d removed, %d added", removedCount, addedCount))))
	}

	return b.String()
}

func (m Model) renderOverlayView(bg string) string {
	w := min(m.width-2, m.width*87/100)
	if w < 64 {
		w = min(m.width-2, 64)
	}
	h := m.height - 4

	// Title bar only — no breadcrumb.
	titleBar := "  " + overlayTitleStyle.Render(m.overlayTitle) + "\n"

	// Viewport content.
	content := titleBar + m.overlayVP.View()

	// Footer: hints + scroll position.
	hint := "esc: close  j/k: scroll"
	if m.overlayRawMD != "" {
		hint += "  c: copy"
	}
	if m.activeTab == tabTasks && m.taskLevel > 0 && !m.reviewMode {
		isPending := false
		if m.specGroupCursor < len(m.specGroups) {
			slug := m.specGroups[m.specGroupCursor].Slug
			isPending = spec.ReviewStatusFor(m.ds.ProjectPath(), slug) == spec.ReviewPending
		}
		if isPending {
			hint = "esc: close  j/k: scroll  r: review  d: diff"
		} else {
			hint = "esc: close  j/k: scroll  d: diff"
		}
	} else if m.reviewMode && !m.reviewEditing {
		isLatest := m.reviewRoundIdx >= len(m.reviewRounds)
		if isLatest {
			hint = "esc: back  j/k: move  </>: rounds  c: comment  d: del  a: approve  x: changes"
		} else {
			hint = "esc: back  j/k: scroll  </>: rounds  (read-only)"
		}
	} else if m.reviewEditing {
		hint = "ctrl+s: save  esc: cancel"
	}
	pct := m.overlayVP.ScrollPercent()
	copiedFlash := ""
	if m.overlayCopied {
		copiedFlash = "  " + scoreStyle.Render("Copied!")
	}
	footer := dimStyle.Render(fmt.Sprintf("  %s  %d%%", hint, int(pct*100))) + copiedFlash
	content += "\n" + footer

	// Panel with border.
	panel := overlayStyle.
		Width(w).
		Height(h).
		Render(content)

	// Center the panel on screen.
	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		panel,
		lipgloss.WithWhitespaceChars("·"),
		lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#1a1a1a"))),
	)
}
