package hookhandler

import (
	"regexp"
	"strings"
)

// TaskType represents the classified intent of the user's prompt.
type TaskType string

const (
	TaskBugfix   TaskType = "bugfix"
	TaskFeature  TaskType = "feature"
	TaskRefactor TaskType = "refactor"
	TaskTest     TaskType = "test"
	TaskUnknown  TaskType = ""
)

// classifyIntent classifies user intent using keyword matching.
// Returns TaskUnknown if no clear classification.
func classifyIntent(intent string) TaskType {
	lower := strings.ToLower(intent)

	// Order matters: test before feature (since "add test" should be test, not feature).
	for _, kw := range []string{"test", "coverage", "spec", "テスト", "カバレッジ"} {
		if strings.Contains(lower, kw) {
			return TaskTest
		}
	}
	for _, kw := range []string{"fix", "bug", "error", "broken", "crash", "修正", "バグ", "エラー", "壊れ"} {
		if strings.Contains(lower, kw) {
			return TaskBugfix
		}
	}
	for _, kw := range []string{"refactor", "clean", "reorganize", "simplify", "リファクタ", "整理"} {
		if strings.Contains(lower, kw) {
			return TaskRefactor
		}
	}
	for _, kw := range []string{"add", "implement", "create", "build", "new", "追加", "実装", "作成"} {
		if strings.Contains(lower, kw) {
			return TaskFeature
		}
	}

	return TaskUnknown
}

// testCmdPattern matches common test runner commands.
var testCmdPattern = regexp.MustCompile(`\b(go\s+test|npm\s+test|npx\s+(jest|vitest)|pytest|cargo\s+test|make\s+test|bundle\s+exec\s+rspec)\b`)

// decisionKeywords detects when a user prompt contains a design decision.
var decisionKeywords = []string{
	"decided to", "going with", "opted for", "will use", "instead of",
	"let's go with", "let's use", "choosing", "approach:",
	"に決定", "を採用", "にする", "を使う", "ではなく",
}

// containsDecisionKeyword returns true if the text contains a decision indicator.
func containsDecisionKeyword(text string) bool {
	lower := strings.ToLower(text)
	for _, kw := range decisionKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// checkWorkflowOrder checks if the current action matches the expected workflow
// for the given task type. Returns a suggestion string or "".
func checkWorkflowOrder(taskType TaskType, hasWrite bool, hasTestRun bool, inPlanMode bool) string {
	switch taskType {
	case TaskBugfix:
		// bugfix: test(reproduce) -> fix -> test(verify)
		if hasWrite && !hasTestRun {
			return "Bugfix workflow: consider running the failing test first to reproduce the issue before editing."
		}

	case TaskFeature:
		// feature: plan -> implement -> test
		if hasWrite && !inPlanMode {
			return "Feature workflow: consider using Plan Mode to outline the approach before starting implementation."
		}

	case TaskRefactor:
		// refactor: test(baseline) -> refactor -> test(verify)
		if hasWrite && !hasTestRun {
			return "Refactor workflow: consider running tests first to establish a passing baseline before making changes."
		}
	}

	return ""
}
