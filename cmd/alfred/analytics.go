package main

import (
	"context"
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/hir4ta/claude-alfred/internal/store"
)

func runAnalytics() error {
	st, err := store.OpenDefault()
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	ctx := context.Background()

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7571F9"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Width(24)
	valStyle := lipgloss.NewStyle().Bold(true)
	okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4672"))
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))

	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("  " + headerStyle.Render("alfred analytics") + "\n")
	b.WriteString("  " + mutedStyle.Render(strings.Repeat("─", 42)) + "\n\n")

	// Feedback summary.
	fs, err := st.GetFeedbackSummary(ctx)
	if err != nil {
		return fmt.Errorf("feedback summary: %w", err)
	}

	b.WriteString("  " + headerStyle.Render("Feedback Loop") + "\n\n")
	b.WriteString("  " + labelStyle.Render("Docs tracked") + valStyle.Render(fmt.Sprintf("%d", fs.TotalTracked)) + "\n")
	b.WriteString("  " + labelStyle.Render("Total positive signals") + okStyle.Render(fmt.Sprintf("+%d", fs.TotalPositive)) + "\n")
	b.WriteString("  " + labelStyle.Render("Total negative signals") + warnStyle.Render(fmt.Sprintf("-%d", fs.TotalNegative)) + "\n")
	b.WriteString("  " + labelStyle.Render("Boosted docs") + valStyle.Render(fmt.Sprintf("%d", fs.BoostedCount)) + "\n")
	b.WriteString("  " + labelStyle.Render("Penalized docs") + valStyle.Render(fmt.Sprintf("%d", fs.PenalizedCount)) + "\n")

	// Recent injection stats.
	injected7, unique7, _ := st.RecentInjectionStats(ctx, 7)
	injected30, unique30, _ := st.RecentInjectionStats(ctx, 30)

	b.WriteString("\n  " + headerStyle.Render("Injection Activity") + "\n\n")
	b.WriteString("  " + labelStyle.Render("Last 7 days") + valStyle.Render(fmt.Sprintf("%d injections, %d unique docs", injected7, unique7)) + "\n")
	b.WriteString("  " + labelStyle.Render("Last 30 days") + valStyle.Render(fmt.Sprintf("%d injections, %d unique docs", injected30, unique30)) + "\n")

	// Top boosted docs.
	topBoosted, _ := st.TopFeedbackDocs(ctx, 5, false)
	if len(topBoosted) > 0 {
		b.WriteString("\n  " + headerStyle.Render("Top Boosted Docs") + "\n\n")
		for _, d := range topBoosted {
			net := d.Positive - d.Negative
			label := truncateStr(d.SectionPath, 40)
			b.WriteString(fmt.Sprintf("  %-42s %s  (boost: %.2f)\n",
				label,
				okStyle.Render(fmt.Sprintf("+%d/-%d net=%d", d.Positive, d.Negative, net)),
				d.BoostFactor))
		}
	}

	// Top penalized docs.
	topPenalized, _ := st.TopFeedbackDocs(ctx, 5, true)
	if len(topPenalized) > 0 {
		b.WriteString("\n  " + headerStyle.Render("Top Penalized Docs") + "\n\n")
		for _, d := range topPenalized {
			net := d.Positive - d.Negative
			label := truncateStr(d.SectionPath, 40)
			b.WriteString(fmt.Sprintf("  %-42s %s  (boost: %.2f)\n",
				label,
				warnStyle.Render(fmt.Sprintf("+%d/-%d net=%d", d.Positive, d.Negative, net)),
				d.BoostFactor))
		}
	}

	b.WriteString("\n")
	fmt.Print(b.String())
	return nil
}
