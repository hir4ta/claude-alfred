package tui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// dmp is the shared DiffMatchPatch instance for diff computation.
var dmp = diffmatchpatch.New()

// termResponseRe matches terminal responses that leak as visible text.
// DECRPM: [?2026;2$y  CPR: [8;1R  DA: [?65;1c
var termResponseRe = regexp.MustCompile(`\[\??\d+[;\d]*(?:\$y|[Rc])`)

// stripDECRPM removes leaked terminal capability responses from rendered output.
func stripDECRPM(s string) string {
	return termResponseRe.ReplaceAllString(s, "")
}

func truncStr(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen < 2 {
		return string(runes[:1])
	}
	return string(runes[:maxLen-1]) + "~"
}

func formatSize(bytes int64) string {
	switch {
	case bytes >= 1024*1024:
		return fmt.Sprintf("%.1fM", float64(bytes)/1024/1024)
	case bytes >= 1024:
		return fmt.Sprintf("%.1fK", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

func formatDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

func formatAuditAction(action string) string {
	switch action {
	case "spec.init":
		return "created"
	case "spec.delete":
		return "deleted"
	case "spec.complete":
		return "completed"
	case "review.submit":
		return "reviewed"
	case "epic.link":
		return "linked"
	default:
		return action
	}
}

func simplifyKnowledgeLabel(label string) (title, context string) {
	parts := strings.Split(label, " > ")
	if len(parts) <= 1 {
		return label, ""
	}
	title = parts[len(parts)-1]
	// Skip numeric ID prefix (first part).
	start := 0
	if len(parts[0]) > 0 && parts[0][0] >= '0' && parts[0][0] <= '9' {
		start = 1
	}
	if start < len(parts)-1 {
		context = strings.Join(parts[start:len(parts)-1], " / ")
	}
	return
}

func firstContentLine(s string) string {
	for line := range strings.SplitSeq(s, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func visibleRange(cursor, total, visibleH int) (start, end int) {
	if visibleH < 1 {
		visibleH = 1
	}
	start = 0
	if cursor >= visibleH {
		start = cursor - visibleH + 1
	}
	end = start + visibleH
	if end > total {
		end = total
	}
	return start, end
}
