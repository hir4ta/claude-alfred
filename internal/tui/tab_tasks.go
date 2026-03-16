package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"

	"github.com/hir4ta/claude-alfred/internal/spec"
)

func (m *Model) rebuildTasksViewport() {
	if m.activeTab != tabTasks || m.width == 0 || m.taskLevel != 0 {
		return
	}

	var b strings.Builder
	maxW := m.width - 6

	// Active task details at top.
	task := m.findActiveTask()
	if task != nil {
		b.WriteString(m.renderTaskOverview(task))
		b.WriteString("\n  " + strings.Repeat("\u2500", min(maxW, 60)) + "\n\n")
	}

	// Active Tasks list with cursor.
	if len(m.allTasks) > 0 {
		b.WriteString("  " + sectionHeader.Render("Active Tasks") + "\n")
		for i, t := range m.allTasks {
			isCompleted := t.Status == "completed" || t.Status == "done" || t.Status == "implementation-complete"

			marker := "  "
			if i == m.taskCursor {
				marker = "> "
			}
			slug := fmt.Sprintf("%-24s", truncStr(t.Slug, 24))
			progBar := ""
			if t.Total > 0 {
				pct := float64(t.Completed) / float64(t.Total)
				barW := 10
				filled := int(pct * float64(barW))
				progBar = strings.Repeat("#", filled) + strings.Repeat("-", barW-filled)
				progBar += fmt.Sprintf(" %d%%", int(pct*100))
			}
			// Show status only for non-active (completed tasks need the label; active is implied).
			status := ""
			if isCompleted {
				status = styledStatus(t.Status)
			}
			blocker := " "
			if t.HasBlocker {
				blocker = blockerStyle.Render("!")
			}

			line := marker + slug + " " + progBar + " " + status + " " + blocker
			if i == m.taskCursor {
				b.WriteString(titleStyle.Render(marker+slug) + " " + progBar + " " + status + " " + blocker + "\n")
			} else if isCompleted {
				b.WriteString(dimStyle.Render(line) + "\n")
			} else {
				b.WriteString(line + "\n")
			}
		}
	}

	m.viewport.SetContent(b.String())
}

// rebuildTaskOverlay updates the Tasks tab overlay content for shimmer animation.
func (m *Model) rebuildTaskOverlay() {
	if !m.overlayActive || m.activeTab != tabTasks || m.taskLevel != 0 {
		return
	}
	if m.taskCursor >= len(m.allTasks) {
		return
	}
	task := &m.allTasks[m.taskCursor]
	// Preserve scroll position.
	yOff := m.overlayVP.YOffset()
	m.overlayVP.SetContent(m.renderTaskOverview(task))
	m.overlayVP.SetYOffset(yOff)
}

func (m *Model) findActiveTask() *TaskDetail {
	for i := range m.allTasks {
		if m.allTasks[i].Slug == m.activeSlug {
			return &m.allTasks[i]
		}
	}
	if len(m.allTasks) > 0 {
		return &m.allTasks[0]
	}
	return nil
}

// tryDirectReview attempts to enter review mode for the active task from Overview.
func (m *Model) tryDirectReview() (tea.Model, tea.Cmd) {
	task := m.findActiveTask()
	if task == nil {
		return m, nil
	}
	status := spec.ReviewStatusFor(m.ds.ProjectPath(), task.Slug)
	if status != spec.ReviewPending {
		return m, nil
	}
	// Find the task in specGroups and open review on the first file.
	for gi, g := range m.specGroups {
		if g.Slug == task.Slug && len(g.Files) > 0 {
			m.specGroupCursor = gi
			m.specFileCursor = 0
			f := g.Files[0]
			content := m.ds.SpecContent(f.TaskSlug, f.File)
			m.openOverlay(f.File, m.renderMarkdown(content), "Specs", g.Slug, f.File)
			m.enterReviewMode()
			return m, nil
		}
	}
	return m, nil
}

func (m *Model) updateTasksList(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Down):
		if m.taskCursor < len(m.allTasks)-1 {
			m.taskCursor++
			m.rebuildTasksViewport()
		}
	case key.Matches(msg, keys.Up):
		if m.taskCursor > 0 {
			m.taskCursor--
			m.rebuildTasksViewport()
		}
	case key.Matches(msg, keys.Enter):
		if m.taskCursor < len(m.allTasks) {
			task := m.allTasks[m.taskCursor]
			// Try to enter spec file view for this task.
			for gi, g := range m.specGroups {
				if g.Slug == task.Slug {
					m.specGroupCursor = gi
					m.specFileCursor = 0
					m.specLevel = 1
					m.taskLevel = 1
					return m, nil
				}
			}
			// No specs — show task detail in overlay.
			m.openOverlay(task.Slug, m.renderTaskOverview(&task), "Tasks", task.Slug)
		}
	}
	return m, nil
}

func (m *Model) updateSpecs(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.specLevel {
	case 0: // task group list
		switch {
		case key.Matches(msg, keys.Down):
			if m.specGroupCursor < len(m.specGroups)-1 {
				m.specGroupCursor++
			}
		case key.Matches(msg, keys.Up):
			if m.specGroupCursor > 0 {
				m.specGroupCursor--
			}
		case key.Matches(msg, keys.Enter):
			if m.specGroupCursor < len(m.specGroups) {
				m.specFileCursor = 0
				m.specLevel = 1
			}
		case key.Matches(msg, keys.Back):
			m.taskLevel = 0
			m.rebuildTasksViewport()
		}
	case 1: // file list
		switch {
		case key.Matches(msg, keys.Down):
			if m.specGroupCursor < len(m.specGroups) {
				g := m.specGroups[m.specGroupCursor]
				if m.specFileCursor < len(g.Files)-1 {
					m.specFileCursor++
				}
			}
		case key.Matches(msg, keys.Up):
			if m.specFileCursor > 0 {
				m.specFileCursor--
			}
		case key.Matches(msg, keys.Enter):
			if m.specGroupCursor < len(m.specGroups) {
				g := m.specGroups[m.specGroupCursor]
				if m.specFileCursor < len(g.Files) {
					f := g.Files[m.specFileCursor]
					content := m.ds.SpecContent(f.TaskSlug, f.File)
					m.openOverlay(
						f.File,
						m.renderMarkdown(content),
						"Specs", g.Slug, f.File,
					)
				}
			}
		case key.Matches(msg, keys.Back):
			m.specLevel = 0
			m.taskLevel = 0
			m.rebuildTasksViewport()
		}
	}
	return m, nil
}

func (m Model) renderTaskOverview(td *TaskDetail) string {
	var b strings.Builder
	maxW := m.width - 6

	// Header: slug + status + progress.
	b.WriteString("  " + titleStyle.Render(td.Slug))
	b.WriteString("  " + styledStatus(td.Status))
	if td.Total > 0 {
		pct := float64(td.Completed) / float64(td.Total)
		b.WriteString("  " + m.progress.ViewAs(pct))
		b.WriteString(fmt.Sprintf(" %d/%d", td.Completed, td.Total))
	}
	if td.EpicSlug != "" {
		b.WriteString("  " + dimStyle.Render("epic:"+td.EpicSlug))
	}
	b.WriteString("\n")

	// Focus — warm white for readability.
	if td.Focus != "" {
		focusStyle := lipgloss.NewStyle().Foreground(fgWarm)
		b.WriteString("  " + focusStyle.Render(td.Focus) + "\n")
	}
	b.WriteString("\n")

	// Blockers (prominent if present).
	if td.HasBlocker {
		b.WriteString("  " + blockerStyle.Render("! BLOCKER") + "  " + td.BlockerText + "\n\n")
	}

	// Next Steps. First unchecked item = currently active → shimmer.
	if len(td.NextSteps) > 0 {
		b.WriteString("  " + sectionHeader.Render("Next Steps") + "\n")
		foundActive := false
		for _, s := range td.NextSteps {
			check := checkUndone
			if s.Done {
				check = checkDone
			}
			text := truncStr(s.Text, maxW-6)
			if s.Done {
				b.WriteString("  " + check + " " + dimStyle.Render(text) + "\n")
			} else if !foundActive {
				foundActive = true
				b.WriteString("  " + check + " " + renderShimmerBold(text, m.shimmerFrame) + "\n")
			} else {
				uncheckedStyle := lipgloss.NewStyle().Foreground(fgWarm)
				b.WriteString("  " + check + " " + uncheckedStyle.Render(text) + "\n")
			}
		}
		b.WriteString("\n")
	}

	// Recent Decisions.
	if len(td.Decisions) > 0 {
		b.WriteString("  " + sectionHeader.Render("Recent Decisions") + "\n")
		for i, d := range td.Decisions {
			b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, truncStr(d, maxW-5)))
		}
		b.WriteString("\n")
	}

	// Modified Files.
	if len(td.ModFiles) > 0 {
		b.WriteString("  " + sectionHeader.Render("Modified Files") + fmt.Sprintf("  %s\n", dimStyle.Render(fmt.Sprintf("(%d)", len(td.ModFiles)))))
		for _, f := range td.ModFiles {
			b.WriteString("  " + dimStyle.Render(truncStr(f, maxW-4)) + "\n")
		}
	}

	return b.String()
}

func (m Model) tasksView() string {
	if len(m.allTasks) == 0 {
		return "\n" + dimStyle.Render("  no tasks — use dossier init to start")
	}
	var hint string
	task := m.findActiveTask()
	if task != nil {
		status := spec.ReviewStatusFor(m.ds.ProjectPath(), task.Slug)
		if status == spec.ReviewPending {
			hint = "\n" + reviewCommentMarker.Render("  r: open review for "+task.Slug)
		}
	}
	return "\n" + m.viewport.View() + hint
}

func (m Model) specsView() string {
	if len(m.specGroups) == 0 {
		return "\n" + dimStyle.Render("  no specs")
	}

	switch m.specLevel {
	case 1: // file list for selected task
		return m.specFilesView()
	default: // task group list
		return m.specGroupsView()
	}
}

func (m Model) specGroupsView() string {
	var b strings.Builder
	b.WriteString("\n")

	visibleH := m.contentHeight() - 1
	startIdx, endIdx := visibleRange(m.specGroupCursor, len(m.specGroups), visibleH)

	for i := startIdx; i < endIdx; i++ {
		g := m.specGroups[i]
		prefix := "  "
		if i == m.specGroupCursor {
			prefix = "> "
		}

		slug := fmt.Sprintf("%-28s", truncStr(g.Slug, 28))
		info := fmt.Sprintf("%d files  %s", g.FileCount, formatSize(g.TotalSize))

		if i == m.specGroupCursor {
			b.WriteString(titleStyle.Render(prefix+slug) + "  " + dimStyle.Render(info) + "\n")
		} else {
			b.WriteString(prefix + slug + "  " + dimStyle.Render(info) + "\n")
		}
	}

	if len(m.specGroups) > visibleH {
		b.WriteString(dimStyle.Render(fmt.Sprintf("\n  %d/%d", m.specGroupCursor+1, len(m.specGroups))))
	}

	return b.String()
}

func (m Model) specFilesView() string {
	if m.specGroupCursor >= len(m.specGroups) {
		return "\n" + dimStyle.Render("  no files")
	}
	g := m.specGroups[m.specGroupCursor]

	var b strings.Builder
	maxW := m.width - 6

	// Find task status for this spec group.
	taskStatus := ""
	reviewStatus := ""
	for _, t := range m.allTasks {
		if t.Slug == g.Slug {
			taskStatus = t.Status
			break
		}
	}
	rs := spec.ReviewStatusFor(m.ds.ProjectPath(), g.Slug)
	if rs == spec.ReviewPending {
		reviewStatus = reviewCommentMarker.Render(" [review pending]")
	}

	b.WriteString("\n  " + titleStyle.Render(g.Slug))
	if taskStatus != "" {
		b.WriteString("  " + styledStatus(taskStatus))
	}
	b.WriteString(reviewStatus)
	b.WriteString("  " + dimStyle.Render(fmt.Sprintf("%d files  %s", g.FileCount, formatSize(g.TotalSize))))
	b.WriteString("\n")

	// Render rich summary from spec files.
	for i, f := range g.Files {
		content := m.ds.SpecContent(f.TaskSlug, f.File)

		prefix := "  "
		if i == m.specFileCursor {
			prefix = "> "
		}

		// Section header with file name.
		header := specFileLabel(f.File)
		if i == m.specFileCursor {
			b.WriteString("\n" + titleStyle.Render(prefix+header) + "  " + dimStyle.Render(formatSize(f.Size)) + "\n")
		} else {
			b.WriteString("\n" + prefix + sectionHeader.Render(header) + "  " + dimStyle.Render(formatSize(f.Size)) + "\n")
		}

		// Render a summary based on file type.
		switch f.File {
		case "decisions.md":
			renderDecisionsSummary(&b, content, maxW)
		case "session.md":
			renderSessionSummary(&b, content, maxW)
		case "requirements.md":
			renderRequirementsSummary(&b, content, maxW)
		case "design.md":
			renderDesignSummary(&b, content, maxW)
		default:
			if line := firstContentLine(content); line != "" {
				b.WriteString("    " + dimStyle.Render(truncStr(line, maxW-4)) + "\n")
			}
		}
	}

	b.WriteString("\n" + dimStyle.Render("  enter: view full file  esc: back"))

	return b.String()
}

func specFileLabel(file string) string {
	switch file {
	case "requirements.md":
		return "Requirements"
	case "design.md":
		return "Design"
	case "decisions.md":
		return "Decisions"
	case "session.md":
		return "Session"
	default:
		return file
	}
}

func renderDecisionsSummary(b *strings.Builder, content string, maxW int) {
	_, ordered := splitSectionsOrdered(content)
	count := 0
	for _, sec := range ordered {
		if sec.Header == "" {
			continue
		}
		count++
		// Extract "Chosen" and "Alternatives" from decision body.
		chosen := ""
		alternatives := ""
		reason := ""
		for line := range strings.SplitSeq(sec.Body, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "- **Chosen:**") || strings.HasPrefix(trimmed, "**Chosen:**") {
				chosen = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(trimmed, "- "), "**Chosen:**"))
			} else if strings.HasPrefix(trimmed, "- **Alternatives:**") || strings.HasPrefix(trimmed, "**Alternatives:**") {
				alternatives = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(trimmed, "- "), "**Alternatives:**"))
			} else if strings.HasPrefix(trimmed, "- **Reason:**") || strings.HasPrefix(trimmed, "**Reason:**") {
				reason = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(trimmed, "- "), "**Reason:**"))
			}
		}
		// Strip date prefix from header if present (e.g. "[2026-03-15] Title").
		title := sec.Header
		if len(title) > 13 && title[0] == '[' {
			if idx := strings.Index(title, "] "); idx > 0 {
				title = title[idx+2:]
			}
		}
		b.WriteString("    " + truncStr(title, maxW-4) + "\n")
		if chosen != "" {
			b.WriteString("      " + dimStyle.Render("-> "+truncStr(chosen, maxW-9)) + "\n")
		}
		if alternatives != "" {
			b.WriteString("      " + dimStyle.Render("vs "+truncStr(alternatives, maxW-9)) + "\n")
		}
		if reason != "" && chosen == "" {
			b.WriteString("      " + dimStyle.Render(truncStr(reason, maxW-6)) + "\n")
		}
		if count >= 5 {
			remaining := len(ordered) - count
			if remaining > 0 {
				b.WriteString(dimStyle.Render(fmt.Sprintf("    ... +%d more", remaining)) + "\n")
			}
			break
		}
	}
	if count == 0 {
		b.WriteString("    " + dimStyle.Render("(no decisions)") + "\n")
	}
}

func renderSessionSummary(b *strings.Builder, content string, maxW int) {
	parsed := parseSessionSections(content)

	b.WriteString("    " + styledStatus(parsed.status))
	if parsed.focus != "" {
		b.WriteString("  " + truncStr(parsed.focus, maxW-20))
	}
	b.WriteString("\n")

	if parsed.hasBlocker {
		b.WriteString("    " + blockerStyle.Render("! "+truncStr(parsed.blockerText, maxW-6)) + "\n")
	}

	// Progress from next steps.
	done := 0
	for _, s := range parsed.nextSteps {
		if s.Done {
			done++
		}
	}
	if len(parsed.nextSteps) > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("    steps: %d/%d done", done, len(parsed.nextSteps))) + "\n")
	}

	if len(parsed.modFiles) > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("    files: %d modified", len(parsed.modFiles))) + "\n")
	}
}

func renderRequirementsSummary(b *strings.Builder, content string, maxW int) {
	// Show first few non-header, non-empty lines as summary.
	count := 0
	for line := range strings.SplitSeq(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "---") {
			continue
		}
		b.WriteString("    " + dimStyle.Render(truncStr(trimmed, maxW-4)) + "\n")
		count++
		if count >= 3 {
			break
		}
	}
	if count == 0 {
		b.WriteString("    " + dimStyle.Render("(empty)") + "\n")
	}
}

func renderDesignSummary(b *strings.Builder, content string, maxW int) {
	// Show section headers as an outline of the design (document order preserved).
	_, ordered := splitSectionsOrdered(content)
	count := 0
	for _, sec := range ordered {
		if sec.Header == "" {
			continue
		}
		b.WriteString("    " + dimStyle.Render("- "+truncStr(sec.Header, maxW-6)) + "\n")
		count++
		if count >= 5 {
			remaining := len(ordered) - count
			if remaining > 0 {
				b.WriteString(dimStyle.Render(fmt.Sprintf("    ... +%d more sections", remaining)) + "\n")
			}
			break
		}
	}
	if count == 0 {
		b.WriteString("    " + dimStyle.Render("(empty)") + "\n")
	}
}
