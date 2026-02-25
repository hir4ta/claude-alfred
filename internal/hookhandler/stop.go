package hookhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hir4ta/claude-buddy/internal/sessiondb"
)

type stopInput struct {
	CommonInput
	StopHookActive       bool   `json:"stop_hook_active"`
	LastAssistantMessage string `json:"last_assistant_message"`
}

func handleStop(input []byte) (*HookOutput, error) {
	var in stopInput
	if err := json.Unmarshal(input, &in); err != nil {
		return nil, fmt.Errorf("parse input: %w", err)
	}

	// Prevent infinite loops: if stop_hook_active, allow stop immediately.
	if in.StopHookActive {
		return nil, nil
	}

	issues := checkCompleteness(in.LastAssistantMessage)

	// Check for uncommitted changes (informational, adds to issues if any exist).
	if gitIssue := checkUncommittedChanges(in.SessionID, in.CWD); gitIssue != "" {
		issues = append(issues, gitIssue)
	}

	if len(issues) > 0 {
		return makeBlockStopOutput(strings.Join(issues, "; ")), nil
	}

	return nil, nil
}

// checkCompleteness scans assistant message for signs of incomplete work.
// Only checks for high-signal deterministic patterns. Error detection is
// left to the LLM prompt hook to avoid false positives on explanatory text.
func checkCompleteness(msg string) []string {
	if msg == "" {
		return nil
	}

	lower := strings.ToLower(msg)
	var issues []string

	// TODO/FIXME markers.
	for _, p := range []string{"todo:", "fixme:", "hack:", "xxx:"} {
		if strings.Contains(lower, p) {
			issues = append(issues, "TODO/FIXME marker found in last response")
			break
		}
	}

	// Explicit incomplete work.
	for _, p := range []string{
		"i'll finish", "i'll complete", "remaining work",
		"not yet implemented", "placeholder",
		"まだ完了していません", "残りの作業", "未実装",
	} {
		if strings.Contains(lower, p) {
			issues = append(issues, "Incomplete work mentioned in last response")
			break
		}
	}

	// Test failures mentioned without resolution.
	for _, p := range []string{
		"test fail", "tests fail", "test failed", "tests failed", "failing test",
		"テストが失敗", "テスト失敗",
	} {
		if strings.Contains(lower, p) {
			issues = append(issues, "Unresolved test failure mentioned in last response")
			break
		}
	}

	// Build failures mentioned without resolution.
	for _, p := range []string{
		"build failed", "compilation error", "compile error", "does not compile",
		"ビルド失敗", "コンパイルエラー",
	} {
		if strings.Contains(lower, p) {
			issues = append(issues, "Unresolved build failure mentioned in last response")
			break
		}
	}

	return issues
}

// checkUncommittedChanges checks if there are uncommitted git changes when stopping.
// Returns an informational message, or "" if clean or not in a git repo.
func checkUncommittedChanges(sessionID, cwd string) string {
	if cwd == "" {
		// Try to get CWD from session DB.
		sdb, err := sessiondb.Open(sessionID)
		if err != nil {
			return ""
		}
		defer sdb.Close()
		cwd, _ = sdb.GetContext("cwd")
		if cwd == "" {
			return ""
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	status, err := execGit(ctx, cwd, "status", "--porcelain")
	if err != nil || strings.TrimSpace(status) == "" {
		return ""
	}

	lines := strings.Split(strings.TrimSpace(status), "\n")
	return fmt.Sprintf("%d uncommitted file(s) in working directory", len(lines))
}
