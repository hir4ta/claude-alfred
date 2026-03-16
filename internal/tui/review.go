package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/hir4ta/claude-alfred/internal/spec"
)

// Review mode styles.
var (
	reviewCursorStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#2e3430")).
				Foreground(fgWarm)

	reviewLineNumStyle = lipgloss.NewStyle().
				Foreground(gray)

	reviewLineNumCursorStyle = lipgloss.NewStyle().
					Background(lipgloss.Color("#2e3430")).
					Foreground(aqua).
					Bold(true)

	reviewCommentMarker = lipgloss.NewStyle().
				Foreground(gold).
				Bold(true)

	reviewCommentStyle = lipgloss.NewStyle().
				Foreground(gold)

	reviewInputBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(gold).
				Padding(0, 1).
				MarginTop(1)
)

// enterReviewMode initializes review mode for the currently viewed spec file.
func (m *Model) enterReviewMode() {
	if m.specGroupCursor >= len(m.specGroups) {
		return
	}
	g := m.specGroups[m.specGroupCursor]
	if m.specFileCursor >= len(g.Files) {
		return
	}
	f := g.Files[m.specFileCursor]
	content := m.ds.SpecContent(f.TaskSlug, f.File)

	m.reviewMode = true
	m.reviewFile = f.File
	m.reviewTaskSlug = f.TaskSlug
	m.reviewLines = strings.Split(content, "\n")
	m.reviewCursor = 0
	m.reviewComments = make(map[int]string)
	m.reviewEditing = false
	m.reviewConfirmPending = false

	// Load all review rounds.
	sd := &spec.SpecDir{ProjectPath: m.ds.ProjectPath(), TaskSlug: f.TaskSlug}
	rounds, _ := sd.AllReviews()
	m.reviewRounds = rounds
	m.reviewRoundIdx = len(rounds) // new round (beyond existing)

	ta := textarea.New()
	ta.Placeholder = "comment..."
	ta.CharLimit = 500
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	m.reviewInput = ta

	m.rebuildReviewOverlay()
}

// updateReviewMode handles keys in review navigation mode.
func (m *Model) updateReviewMode(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	isLatestRound := m.reviewRoundIdx >= len(m.reviewRounds)

	switch {
	case key.Matches(msg, keys.Back), key.Matches(msg, keys.Quit):
		m.reviewMode = false
		// Restore normal overlay content.
		if m.specGroupCursor < len(m.specGroups) {
			g := m.specGroups[m.specGroupCursor]
			if m.specFileCursor < len(g.Files) {
				f := g.Files[m.specFileCursor]
				content := m.ds.SpecContent(f.TaskSlug, f.File)
				m.overlayVP.SetContent(m.renderMarkdown(content))
				m.overlayTitle = f.File
			}
		}
		return m, nil

	case key.Matches(msg, keys.Down):
		if m.reviewCursor < len(m.reviewLines)-1 {
			m.reviewCursor++
			m.rebuildReviewOverlay()
		}
		return m, nil

	case key.Matches(msg, keys.Up):
		if m.reviewCursor > 0 {
			m.reviewCursor--
			m.rebuildReviewOverlay()
		}
		return m, nil

	case msg.String() == "left":
		// Navigate to previous review round.
		if m.reviewRoundIdx > 0 {
			m.reviewRoundIdx--
			m.reviewComments = m.commentsForRound(m.reviewRoundIdx)
			m.rebuildReviewOverlay()
		}
		return m, nil

	case msg.String() == "right":
		// Navigate to next review round / new round.
		if m.reviewRoundIdx < len(m.reviewRounds) {
			m.reviewRoundIdx++
			if m.reviewRoundIdx >= len(m.reviewRounds) {
				// New round: clear comments for fresh editing.
				m.reviewComments = make(map[int]string)
			} else {
				m.reviewComments = m.commentsForRound(m.reviewRoundIdx)
			}
			m.rebuildReviewOverlay()
		}
		return m, nil

	case msg.String() == "c" && isLatestRound:
		// Start commenting on current line (only in new/latest round).
		m.reviewEditing = true
		m.reviewInputLine = m.reviewCursor
		m.reviewInput.SetValue("")
		if existing, ok := m.reviewComments[m.reviewCursor]; ok {
			m.reviewInput.SetValue(existing)
		}
		m.reviewInput.Focus()
		return m, nil

	case msg.String() == "a" && isLatestRound && !m.reviewConfirmPending:
		// Approve — require confirmation.
		m.reviewConfirmPending = true
		m.reviewConfirmStatus = spec.ReviewApproved
		m.rebuildReviewOverlay()
		return m, nil

	case msg.String() == "x" && isLatestRound && !m.reviewConfirmPending:
		// Request Changes — require confirmation.
		m.reviewConfirmPending = true
		m.reviewConfirmStatus = spec.ReviewChangesRequested
		m.rebuildReviewOverlay()
		return m, nil

	case msg.String() == "y" && m.reviewConfirmPending:
		// Confirm submission.
		m.reviewConfirmPending = false
		m.submitReview(m.reviewConfirmStatus)
		return m, nil

	case msg.String() == "n" && m.reviewConfirmPending:
		// Cancel submission.
		m.reviewConfirmPending = false
		m.rebuildReviewOverlay()
		return m, nil

	case msg.String() == "d" && isLatestRound:
		// Delete comment on current line (only in new round).
		delete(m.reviewComments, m.reviewCursor)
		m.rebuildReviewOverlay()
		return m, nil

	default:
		var cmd tea.Cmd
		m.overlayVP, cmd = m.overlayVP.Update(msg)
		return m, cmd
	}
}

// commentsForRound extracts comments from a historical review round as a line→body map.
// Only includes comments for the currently viewed file.
func (m *Model) commentsForRound(idx int) map[int]string {
	comments := make(map[int]string)
	if idx >= len(m.reviewRounds) {
		return comments
	}
	r := m.reviewRounds[idx]
	for _, c := range r.Comments {
		if c.File == m.reviewFile {
			comments[c.Line-1] = c.Body // convert 1-based to 0-based
		}
	}
	return comments
}

// carriedComments returns unresolved comments from all previous rounds
// (before the current round) as a line→body map. Used to highlight
// comments that haven't been addressed yet.
func (m *Model) carriedComments() map[int]string {
	carried := make(map[int]string)
	// Only show carried comments in the latest (new) round.
	if m.reviewRoundIdx < len(m.reviewRounds) {
		return carried
	}
	for _, r := range m.reviewRounds {
		for _, c := range r.Comments {
			if c.File == m.reviewFile && !c.Resolved {
				carried[c.Line-1] = c.Body
			}
		}
	}
	return carried
}

// updateReviewInput handles keys while typing a comment.
// ctrl+s saves, esc cancels. Enter inserts newlines in the textarea.
func (m *Model) updateReviewInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+s":
		// Save comment.
		body := strings.TrimSpace(m.reviewInput.Value())
		if body != "" {
			m.reviewComments[m.reviewInputLine] = body
		}
		m.reviewEditing = false
		m.reviewInput.Blur()
		m.rebuildReviewOverlay()
		return m, nil
	case "esc":
		// Cancel editing.
		m.reviewEditing = false
		m.reviewInput.Blur()
		m.rebuildReviewOverlay()
		return m, nil
	default:
		var cmd tea.Cmd
		m.reviewInput, cmd = m.reviewInput.Update(msg)
		return m, cmd
	}
}

// submitReview saves the review and updates the active task's review status.
func (m *Model) submitReview(status spec.ReviewStatus) {
	sd := &spec.SpecDir{
		ProjectPath: m.ds.ProjectPath(),
		TaskSlug:    m.reviewTaskSlug,
	}

	// Build review comments.
	var comments []spec.ReviewComment
	for line, body := range m.reviewComments {
		comments = append(comments, spec.ReviewComment{
			File: m.reviewFile,
			Line: line + 1, // 1-based
			Body: body,
		})
	}

	review := &spec.Review{
		Status:   status,
		Comments: comments,
	}
	if len(m.reviewComments) > 0 {
		review.Summary = fmt.Sprintf("%d comments on %s", len(comments), m.reviewFile)
	}

	_ = sd.SaveReview(review) // best-effort
	_ = spec.SetReviewStatus(m.ds.ProjectPath(), m.reviewTaskSlug, status)
	spec.AppendAudit(m.ds.ProjectPath(), spec.AuditEntry{
		Action: "review.submit",
		Target: m.reviewTaskSlug + "/" + m.reviewFile,
		Detail: fmt.Sprintf("status=%s comments=%d", status, len(comments)),
		User:   "tui",
	})

	// Exit review mode.
	m.reviewMode = false
	m.overlayTitle = fmt.Sprintf("Review submitted: %s", status)
	m.overlayVP.SetContent(fmt.Sprintf(
		"\n  Review saved for %s/%s\n  Status: %s\n  Comments: %d\n\n  %s",
		m.reviewTaskSlug, m.reviewFile, status, len(comments),
		dimStyle.Render("press esc to close"),
	))
}

// rebuildReviewOverlay renders the line-numbered review view into the overlay.
func (m *Model) rebuildReviewOverlay() {
	var b strings.Builder
	w := m.overlayVP.Width() - 2
	lineW := w - 8 // gutter(4 digits + marker + space) = 7, plus margin

	isLatestRound := m.reviewRoundIdx >= len(m.reviewRounds)

	// Round navigation bar.
	totalRounds := len(m.reviewRounds) + 1 // +1 for new round
	roundLabel := fmt.Sprintf("Round %d/%d", m.reviewRoundIdx+1, totalRounds)
	if isLatestRound {
		roundLabel += " (new)"
	} else {
		r := m.reviewRounds[m.reviewRoundIdx]
		roundLabel += fmt.Sprintf(" [%s]", r.Status)
	}
	navHint := ""
	if m.reviewRoundIdx > 0 {
		navHint += "<- "
	}
	navHint += roundLabel
	if m.reviewRoundIdx < len(m.reviewRounds) {
		navHint += " ->"
	}
	b.WriteString("  " + reviewRoundStyle.Render(navHint) + "\n")

	// Status bar.
	commentCount := len(m.reviewComments)
	statusLine := dimStyle.Render(fmt.Sprintf("  %s  ", m.reviewFile))
	if commentCount > 0 {
		statusLine += reviewCommentMarker.Render(fmt.Sprintf("  %d comment(s)", commentCount))
	}
	if !isLatestRound {
		statusLine += dimStyle.Render("  (read-only)")
	}
	b.WriteString(statusLine + "\n\n")

	// Collect carried-over (unresolved) comments from previous rounds for highlight.
	carried := m.carriedComments()

	for i, line := range m.reviewLines {
		lineNum := fmt.Sprintf("%4d", i+1)
		isCursor := i == m.reviewCursor
		_, hasComment := m.reviewComments[i]

		// Gutter: line number + comment marker.
		marker := " "
		if hasComment {
			marker = reviewCommentMarker.Render("*")
		}

		// Truncate line content.
		text := line
		runes := []rune(text)
		if len(runes) > lineW {
			text = string(runes[:lineW])
		}

		if isCursor {
			// Active line: full background highlight.
			// Pad text to fill width for continuous background.
			padded := text
			if len([]rune(padded)) < lineW {
				padded += strings.Repeat(" ", lineW-len([]rune(padded)))
			}
			b.WriteString(reviewLineNumCursorStyle.Render(lineNum) + marker + " " + reviewCursorStyle.Render(padded) + "\n")
		} else {
			b.WriteString(reviewLineNumStyle.Render(lineNum) + marker + " " + text + "\n")
		}

		// Show inline comment below the line.
		if hasComment {
			comment := m.reviewComments[i]
			b.WriteString("      " + reviewCommentStyle.Render("  "+comment) + "\n")
		}

		// Show carried-over unresolved comments from previous rounds (dimmer).
		if carriedComment, ok := carried[i]; ok && !hasComment {
			b.WriteString("      " + reviewCarriedStyle.Render("  [prev] "+carriedComment) + "\n")
		}
	}

	// Confirmation prompt for review submission.
	if m.reviewConfirmPending {
		action := "Approve"
		if m.reviewConfirmStatus == spec.ReviewChangesRequested {
			action = "Request Changes"
		}
		prompt := reviewCommentMarker.Render(fmt.Sprintf("  %s? (y/n)", action))
		b.WriteString("\n" + prompt + "\n")
	}

	// Comment input area — fixed at bottom, clearly separated.
	if m.reviewEditing {
		inputLabel := fmt.Sprintf(" Line %d ", m.reviewInputLine+1)
		inputContent := reviewInputBorder.Width(min(w-4, 80)).Render(
			reviewCommentMarker.Render(inputLabel) + "\n" + m.reviewInput.View(),
		)
		b.WriteString("\n" + inputContent + "\n")
	}

	m.overlayTitle = fmt.Sprintf("Review: %s", m.reviewFile)
	m.overlayVP.SetContent(b.String())

	// Scroll position: center on cursor, but force to bottom when confirming.
	if m.reviewConfirmPending {
		m.overlayVP.GotoBottom()
	} else {
		targetOffset := max(0, m.reviewCursor-m.overlayVP.Height()/2)
		m.overlayVP.SetYOffset(targetOffset)
	}
}
