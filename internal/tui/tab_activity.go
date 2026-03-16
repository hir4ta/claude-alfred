package tui

import (
	"fmt"
	"strings"
)

func (m Model) activityView() string {
	var b strings.Builder
	b.WriteString("\n")
	maxW := m.width - 6

	// Timeline section — rendered as a table.
	if len(m.activityTable.Rows()) > 0 {
		b.WriteString("  " + sectionHeader.Render("Timeline") + "\n")
		b.WriteString(m.activityTable.View())
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
