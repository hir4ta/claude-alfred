package hookhandler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hir4ta/claude-buddy/internal/analyzer"
	"github.com/hir4ta/claude-buddy/internal/sessiondb"
	"github.com/hir4ta/claude-buddy/internal/store"
)

type preToolUseInput struct {
	CommonInput
	ToolName  string          `json:"tool_name"`
	ToolInput json.RawMessage `json:"tool_input"`
	ToolUseID string          `json:"tool_use_id"`
}

func handlePreToolUse(input []byte) (*HookOutput, error) {
	var in preToolUseInput
	if err := json.Unmarshal(input, &in); err != nil {
		return nil, fmt.Errorf("parse input: %w", err)
	}

	// Destructive command gate for Bash.
	if in.ToolName == "Bash" {
		var toolInput struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal(in.ToolInput, &toolInput); err == nil && toolInput.Command != "" {
			obs, sugg, matched := analyzer.MatchDestructiveCommand(toolInput.Command)
			if matched {
				reason := fmt.Sprintf("[buddy] %s\n→ %s", obs, sugg)
				return makeDenyOutput(reason), nil
			}
		}
	}

	// Open session DB for context-aware checks and nudge delivery.
	sdb, err := sessiondb.Open(in.SessionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[buddy] PreToolUse: open session db: %v\n", err)
		return nil, nil
	}
	defer sdb.Close()

	// --- JARVIS advisor signals (proactive, before action) ---

	// Bash: past failure warning.
	if in.ToolName == "Bash" {
		if warning := pastFailureWarning(sdb, in.ToolInput); warning != "" {
			return makeOutput("PreToolUse", warning), nil
		}
	}

	// Edit/Write: stale read check + related decision surfacing.
	if in.ToolName == "Edit" || in.ToolName == "Write" {
		if guidance := staleReadCheck(sdb, in.ToolInput); guidance != "" {
			return makeOutput("PreToolUse", guidance), nil
		}
		if decision := relatedDecisionSurfacing(sdb, in.ToolInput); decision != "" {
			return makeOutput("PreToolUse", decision), nil
		}
	}

	// Dequeue pending nudges as additionalContext.
	nudges, _ := sdb.DequeueNudges(1)
	if len(nudges) == 0 {
		return nil, nil
	}

	entries := make([]nudgeEntry, len(nudges))
	for i, n := range nudges {
		entries[i] = nudgeEntry{
			Pattern:     n.Pattern,
			Level:       n.Level,
			Observation: n.Observation,
			Suggestion:  n.Suggestion,
		}
	}
	return makeOutput("PreToolUse", formatNudges(entries)), nil
}

// staleReadCheck warns when an Edit/Write targets a file whose last Read
// was many tool calls ago, suggesting the content may be stale.
func staleReadCheck(sdb *sessiondb.SessionDB, toolInput json.RawMessage) string {
	var ei struct {
		FilePath string `json:"file_path"`
	}
	if json.Unmarshal(toolInput, &ei) != nil || ei.FilePath == "" {
		return ""
	}

	lastSeq, _ := sdb.FileLastReadSeq(ei.FilePath)
	if lastSeq == 0 {
		// File was never Read in this session — warn.
		key := "stale_read:" + ei.FilePath
		on, _ := sdb.IsOnCooldown(key)
		if on {
			return ""
		}
		_ = sdb.SetCooldown(key, 10*time.Minute)
		return "[buddy] This file has not been Read in this session. Consider reading it first to ensure old_string matches current content."
	}

	currentSeq, _ := sdb.CurrentEventSeq()
	distance := currentSeq - lastSeq
	if distance < 8 {
		return ""
	}

	key := "stale_read:" + ei.FilePath
	on, _ := sdb.IsOnCooldown(key)
	if on {
		return ""
	}
	_ = sdb.SetCooldown(key, 10*time.Minute)

	return fmt.Sprintf(
		"[buddy] This file was last Read %d tool calls ago. Content may have changed — consider re-reading before editing.",
		distance,
	)
}

// pastFailureWarning checks if a similar Bash command failed recently in this session.
func pastFailureWarning(sdb *sessiondb.SessionDB, toolInput json.RawMessage) string {
	var bi struct {
		Command string `json:"command"`
	}
	if json.Unmarshal(toolInput, &bi) != nil || bi.Command == "" {
		return ""
	}

	sig := extractCmdSignature(bi.Command)
	if sig == "" {
		return ""
	}

	key := "past_failure:" + sig
	on, _ := sdb.IsOnCooldown(key)
	if on {
		return ""
	}

	summary, _ := sdb.FindSimilarFailure(sig)
	if summary == "" {
		return ""
	}

	_ = sdb.SetCooldown(key, 5*time.Minute)

	if len([]rune(summary)) > 100 {
		summary = string([]rune(summary)[:100]) + "..."
	}
	return fmt.Sprintf("[buddy] A similar command failed earlier in this session: %s", summary)
}

// extractCmdSignature extracts the base command pattern from a Bash command.
// "go test ./internal/store/..." → "go test"
// "npm install lodash" → "npm install"
func extractCmdSignature(command string) string {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return ""
	}
	if len(parts) >= 2 {
		return parts[0] + " " + parts[1]
	}
	return parts[0]
}

// relatedDecisionSurfacing checks for past design decisions related to the target file.
func relatedDecisionSurfacing(sdb *sessiondb.SessionDB, toolInput json.RawMessage) string {
	var ei struct {
		FilePath string `json:"file_path"`
	}
	if json.Unmarshal(toolInput, &ei) != nil || ei.FilePath == "" {
		return ""
	}

	key := "decision_surfacing:" + filepath.Base(ei.FilePath)
	on, _ := sdb.IsOnCooldown(key)
	if on {
		return ""
	}

	st, err := store.OpenDefault()
	if err != nil {
		return ""
	}
	defer st.Close()

	decisions, _ := st.SearchDecisionsByFile(ei.FilePath, 2)
	if len(decisions) == 0 {
		return ""
	}

	_ = sdb.SetCooldown(key, 15*time.Minute)

	var b strings.Builder
	b.WriteString("[buddy] Past decisions for this file:\n")
	for _, d := range decisions {
		text := d.DecisionText
		if len([]rune(text)) > 120 {
			text = string([]rune(text)[:120]) + "..."
		}
		fmt.Fprintf(&b, "  - %s\n", text)
	}
	return b.String()
}
