package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/key"
)

// activityFilters defines the cycle order for action type filtering.
var activityFilters = []string{"", "spec.init", "spec.complete", "review.submit"}

func (m *Model) handleActivityKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "f":
		// Cycle filter.
		current := 0
		for i, f := range activityFilters {
			if f == m.activityFilter {
				current = i
				break
			}
		}
		m.activityFilter = activityFilters[(current+1)%len(activityFilters)]
		return m, nil
	case key.Matches(msg, keys.Enter):
		// Drilldown: if on an epic task, jump to Tasks tab.
		// For now, just switch to Tasks tab.
		if len(m.epics) > 0 {
			m.switchTab(tabTasks)
			return m, nil
		}
	default:
		m.viewport, _ = m.viewport.Update(msg)
	}
	return m, nil
}

func (m Model) activityView() string {
	var b strings.Builder
	b.WriteString("\n")
	maxW := m.width - 6

	// Timeline section — rendered as a table, with optional filter.
	if len(m.activityTable.Rows()) > 0 {
		filterLabel := "all"
		if m.activityFilter != "" {
			filterLabel = m.activityFilter
		}
		b.WriteString("  " + sectionHeader.Render("Timeline") + "  " + dimStyle.Render("("+filterLabel+")  f: cycle filter") + "\n")

		// If filter is active, show filtered rows.
		if m.activityFilter != "" {
			for _, a := range m.activity {
				if a.Action != m.activityFilter {
					continue
				}
				b.WriteString(fmt.Sprintf("  %s  %-12s %-24s %s\n",
					a.Timestamp.Format("15:04"),
					formatAuditAction(a.Action),
					truncStr(a.Target, 24),
					truncStr(a.Detail, maxW-48)))
			}
		} else {
			b.WriteString(m.activityTable.View())
		}
		b.WriteString("\n\n")
	}

	// Epic progress section.
	if len(m.epics) > 0 {
		b.WriteString("  " + sectionHeader.Render("Epics") + "\n")
		for _, e := range m.epics {
			status := styledStatus(e.Status)
			progBar := ""
			if e.Total > 0 {
				pct := float64(e.Completed) / float64(e.Total)
				barW := 10
				filled := int(pct * float64(barW))
				progBar = strings.Repeat("#", filled) + strings.Repeat("-", barW-filled)
				progBar += fmt.Sprintf(" %d%%", int(pct*100))
			}
			b.WriteString(fmt.Sprintf("  %-20s %s  %s\n", truncStr(e.Name, 20), progBar, status))
			// Show epic tasks.
			for _, t := range e.Tasks {
				taskStatus := styledStatus(t.Status)
				b.WriteString(fmt.Sprintf("    - %-16s %s\n", truncStr(t.Slug, 16), taskStatus))
			}
		}
		b.WriteString("\n")
	}

	// Cross-task decisions section.
	if len(m.decisions) > 0 {
		b.WriteString("  " + sectionHeader.Render("Recent Decisions") + "\n")
		for _, d := range m.decisions {
			title := truncStr(d.Title, maxW-30)
			task := dimStyle.Render(d.TaskSlug)
			b.WriteString("  " + title + "  " + task + "\n")
			if d.Chosen != "" {
				b.WriteString("    " + dimStyle.Render("-> "+truncStr(d.Chosen, maxW-7)) + "\n")
			}
		}
		b.WriteString("\n")
	}

	// Stats summary — count from both _active.md tasks and audit timeline.
	b.WriteString("  " + sectionHeader.Render("Stats") + "\n")
	activeCount := 0
	completedCount := 0
	for _, t := range m.allTasks {
		if t.Status == "completed" || t.Status == "done" {
			completedCount++
		} else {
			activeCount++
		}
	}
	// Also count completions from audit timeline (covers tasks already removed from _active.md).
	auditCompleted := 0
	for _, a := range m.activity {
		if a.Action == "spec.complete" {
			auditCompleted++
		}
	}
	if auditCompleted > completedCount {
		completedCount = auditCompleted
	}
	b.WriteString(fmt.Sprintf("  Tasks: %d active, %d completed\n", activeCount, completedCount))

	if b.Len() < 3 {
		return "\n" + dimStyle.Render("  no activity yet")
	}

	return b.String()
}
