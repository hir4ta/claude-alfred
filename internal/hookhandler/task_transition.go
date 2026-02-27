package hookhandler

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/hir4ta/claude-buddy/internal/sessiondb"
)

// generateTaskTransitionBriefing builds a compact briefing when task type
// changes or is first classified. Returns "" if on cooldown or no useful data.
func generateTaskTransitionBriefing(sdb *sessiondb.SessionDB, prevType, newType, cwd string) string {
	on, _ := sdb.IsOnCooldown("task_transition")
	if on {
		return ""
	}

	var parts []string

	// Header: show transition direction when switching tasks.
	if prevType != "" {
		parts = append(parts, fmt.Sprintf("Task switch: %s → %s", prevType, newType))
	}

	// Previous task's unresolved issues (only on transition).
	if prevType != "" {
		failures, _ := sdb.RecentFailures(3)
		var unresolved []string
		for _, f := range failures {
			if f.FilePath == "" {
				continue
			}
			if ur, failType, _ := sdb.HasUnresolvedFailure(f.FilePath); ur {
				unresolved = append(unresolved, fmt.Sprintf("%s in %s", failType, filepath.Base(f.FilePath)))
			}
			if len(unresolved) >= 3 {
				break
			}
		}
		if len(unresolved) > 0 {
			parts = append(parts, fmt.Sprintf("Previous: %d unresolved — %s", len(unresolved), strings.Join(unresolved, ", ")))
		}
	}

	// Estimate for the new task type.
	if est := autoEstimate(cwd, newType); est != "" {
		parts = append(parts, "Estimate: "+est)
	}

	// Recommended first action.
	if next := autoNextStep(sdb); next != "" {
		parts = append(parts, "Start with: "+next)
	}

	if len(parts) == 0 {
		return ""
	}

	_ = sdb.SetCooldown("task_transition", 3*time.Minute)
	return strings.Join(parts, "\n")
}
