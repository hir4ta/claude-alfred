package hookhandler

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/hir4ta/claude-alfred/internal/sessiondb"
)

type stopInput struct {
	CommonInput
	LastAssistantMessage string `json:"last_assistant_message,omitempty"`
}

// Patterns indicating incomplete work in Claude's final message.
var incompletePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bnext step\b`),
	regexp.MustCompile(`(?i)\bremaining\b`),
	regexp.MustCompile(`(?i)\bnot yet\b`),
	regexp.MustCompile(`(?i)\bincomplete\b`),
	regexp.MustCompile(`後で`),
	regexp.MustCompile(`残り`),
	regexp.MustCompile(`未完了`),
}

// Patterns indicating unresolved errors in Claude's final message.
var stopErrorPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\btest.*fail`),
	regexp.MustCompile(`(?i)\bfailing test`),
	regexp.MustCompile(`(?i)\bbuild.*fail`),
	regexp.MustCompile(`(?i)\bcompilation.*fail`),
}

// handleStop analyzes Claude's final message for incomplete work and unresolved errors.
// Blocks Claude from stopping when there is high-confidence evidence of incomplete tasks
// (multiple text signals or sessiondb-confirmed failures).
// Single text signals produce a soft warning without blocking.
func handleStop(input []byte) (*HookOutput, error) {
	var in stopInput
	if err := json.Unmarshal(input, &in); err != nil {
		return nil, fmt.Errorf("parse input: %w", err)
	}

	sdb, err := sessiondb.Open(in.SessionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[alfred] Stop: open session db: %v\n", err)
		return nil, nil
	}
	defer sdb.Close()

	_ = sdb.SetContext("stop_event_seen", "true")

	if in.LastAssistantMessage == "" {
		return nil, nil
	}

	var issues []string
	msg := in.LastAssistantMessage

	// Check for TODO/FIXME (reuse existing patterns from subagent_stop.go).
	for _, p := range placeholderPatterns {
		if p.MatchString(msg) {
			issues = append(issues, "incomplete marker (TODO/FIXME)")
			break
		}
	}

	// Check for incomplete task indicators.
	for _, p := range incompletePatterns {
		if p.MatchString(msg) {
			issues = append(issues, "incomplete task indicator")
			break
		}
	}

	// Check tail of message for unresolved error mentions.
	tail := msg
	if runeLen := len([]rune(tail)); runeLen > 500 {
		tail = string([]rune(tail)[runeLen-500:])
	}
	for _, p := range stopErrorPatterns {
		if p.MatchString(tail) {
			issues = append(issues, "unresolved error mentioned")
			break
		}
	}

	// Check sessiondb for unresolved failures.
	unresolvedCount := countUnresolvedFailures(sdb)
	if unresolvedCount > 0 {
		issues = append(issues, fmt.Sprintf("%d unresolved failure(s) in session", unresolvedCount))
	}

	if len(issues) == 0 {
		return nil, nil
	}

	// Block only with high confidence: sessiondb-confirmed failures OR multiple text signals.
	shouldBlock := unresolvedCount > 0 || len(issues) >= 2
	if shouldBlock {
		reason := fmt.Sprintf("[alfred] Incomplete work detected: %s. Please resolve before completing.",
			strings.Join(issues, "; "))
		return &HookOutput{
			Decision: "block",
			Reason:   reason,
		}, nil
	}

	// Soft warning for single text signal.
	SetDeliveryContext(sdb)
	detail := fmt.Sprintf("Stop check: %s", strings.Join(issues, "; "))
	Deliver(sdb, "stop-quality", "warning",
		"Session ending with potential incomplete work", detail, PriorityHigh)

	return nil, nil
}

// countUnresolvedFailures returns the number of unresolved failures in the session.
func countUnresolvedFailures(sdb *sessiondb.SessionDB) int {
	failures, _ := sdb.RecentFailures(5)
	count := 0
	for _, f := range failures {
		if f.FilePath == "" {
			continue
		}
		unresolved, _, _ := sdb.HasUnresolvedFailure(f.FilePath)
		if unresolved {
			count++
		}
	}
	return count
}
