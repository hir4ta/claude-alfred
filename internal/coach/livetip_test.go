package coach

import (
	"strings"
	"testing"
	"time"

	"github.com/hir4ta/claude-buddy/internal/analyzer"
	"github.com/hir4ta/claude-buddy/internal/parser"
)

func TestParseFeedbackOutput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  analyzer.Feedback
	}{
		{
			name: "full output",
			input: `SITUATION: Implementing REST API endpoints
OBSERVATION: No tests written after 10 turns. Bash grep used 3 times instead of Grep tool.
SUGGESTION: Run go test ./... before proceeding to catch regressions early
LEVEL: warning`,
			want: analyzer.Feedback{
				Situation:   "Implementing REST API endpoints",
				Observation: "No tests written after 10 turns. Bash grep used 3 times instead of Grep tool.",
				Suggestion:  "Run go test ./... before proceeding to catch regressions early",
				Level:       analyzer.LevelWarning,
			},
		},
		{
			name: "partial output - missing LEVEL",
			input: `SITUATION: Refactoring auth module
OBSERVATION: Good use of Plan Mode for complex changes
SUGGESTION: Add CLAUDE.md rules for the new module structure`,
			want: analyzer.Feedback{
				Situation:   "Refactoring auth module",
				Observation: "Good use of Plan Mode for complex changes",
				Suggestion:  "Add CLAUDE.md rules for the new module structure",
				Level:       analyzer.LevelLow,
			},
		},
		{
			name: "partial output - only SUGGESTION",
			input: `SUGGESTION: Use /compact to free up context`,
			want: analyzer.Feedback{
				Situation:   "Analyzing session...",
				Observation: "Gathering session data",
				Suggestion:  "Use /compact to free up context",
				Level:       analyzer.LevelLow,
			},
		},
		{
			name:  "empty input",
			input: "",
			want: analyzer.Feedback{
				Situation:   "Analyzing session...",
				Observation: "Gathering session data",
				Suggestion:  "Include specific file paths in your instructions for better accuracy",
				Level:       analyzer.LevelLow,
			},
		},
		{
			name:  "garbage input",
			input: "This is not in the expected format at all.",
			want: analyzer.Feedback{
				Situation:   "Analyzing session...",
				Observation: "Gathering session data",
				Suggestion:  "Include specific file paths in your instructions for better accuracy",
				Level:       analyzer.LevelLow,
			},
		},
		{
			name: "level action",
			input: `SITUATION: Long session without compaction
OBSERVATION: 50+ turns, context likely degraded
SUGGESTION: Run /compact immediately
LEVEL: action`,
			want: analyzer.Feedback{
				Situation:   "Long session without compaction",
				Observation: "50+ turns, context likely degraded",
				Suggestion:  "Run /compact immediately",
				Level:       analyzer.LevelAction,
			},
		},
		{
			name: "level insight",
			input: `SITUATION: Building UI components
OBSERVATION: Consistent use of subagents for parallel research
SUGGESTION: Consider adding .claude/agents/ for custom agent definitions
LEVEL: insight`,
			want: analyzer.Feedback{
				Situation:   "Building UI components",
				Observation: "Consistent use of subagents for parallel research",
				Suggestion:  "Consider adding .claude/agents/ for custom agent definitions",
				Level:       analyzer.LevelInsight,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseFeedbackOutput(tt.input)
			if got.Situation != tt.want.Situation {
				t.Errorf("Situation = %q, want %q", got.Situation, tt.want.Situation)
			}
			if got.Observation != tt.want.Observation {
				t.Errorf("Observation = %q, want %q", got.Observation, tt.want.Observation)
			}
			if got.Suggestion != tt.want.Suggestion {
				t.Errorf("Suggestion = %q, want %q", got.Suggestion, tt.want.Suggestion)
			}
			if got.Level != tt.want.Level {
				t.Errorf("Level = %d, want %d", got.Level, tt.want.Level)
			}
		})
	}
}

func TestBuildSummaryIncludesAlerts(t *testing.T) {
	t.Parallel()
	now := time.Now()
	events := []parser.SessionEvent{
		{Type: parser.EventUserMessage, UserText: "fix the bug", Timestamp: now},
		{Type: parser.EventToolUse, ToolName: "Read", ToolInput: "main.go", Timestamp: now},
	}
	stats := analyzer.NewStats()
	for _, ev := range events {
		stats.Update(ev)
	}
	alerts := []analyzer.Alert{
		{Pattern: analyzer.PatternRetryLoop, Level: analyzer.LevelWarning, Observation: "same tool repeated 3 times"},
	}

	summary := buildSummary(events, stats, alerts, 0.8, nil)

	if !strings.Contains(summary, "Anti-Pattern Alerts") {
		t.Error("summary should contain Anti-Pattern Alerts section")
	}
	if !strings.Contains(summary, "retry-loop") {
		t.Error("summary should contain pattern name 'retry-loop'")
	}
	if !strings.Contains(summary, "same tool repeated 3 times") {
		t.Error("summary should contain alert observation")
	}
	if !strings.Contains(summary, "Session Health: 80%") {
		t.Errorf("summary should contain Session Health: 80%%, got:\n%s", summary)
	}
}

func TestBuildSummaryTurnNumbers(t *testing.T) {
	t.Parallel()
	now := time.Now()
	events := []parser.SessionEvent{
		{Type: parser.EventUserMessage, UserText: "first task", Timestamp: now},
		{Type: parser.EventToolUse, ToolName: "Read", ToolInput: "file.go", Timestamp: now},
		{Type: parser.EventUserMessage, UserText: "second task", Timestamp: now},
		{Type: parser.EventToolUse, ToolName: "Edit", ToolInput: "file.go", Timestamp: now},
	}
	stats := analyzer.NewStats()
	for _, ev := range events {
		stats.Update(ev)
	}

	summary := buildSummary(events, stats, nil, 1.0, nil)

	if !strings.Contains(summary, "T1 U: first task") {
		t.Errorf("summary should contain 'T1 U: first task', got:\n%s", summary)
	}
	if !strings.Contains(summary, "T2 U: second task") {
		t.Errorf("summary should contain 'T2 U: second task', got:\n%s", summary)
	}
}

func TestComputeUsageHintsVagueInstructions(t *testing.T) {
	t.Parallel()
	now := time.Now()
	events := []parser.SessionEvent{
		{Type: parser.EventUserMessage, UserText: "fix", Timestamp: now},
		{Type: parser.EventUserMessage, UserText: "do it", Timestamp: now},
		{Type: parser.EventUserMessage, UserText: "ok", Timestamp: now},
	}
	stats := analyzer.NewStats()
	for _, ev := range events {
		stats.Update(ev)
	}

	hints := computeUsageHints(events, stats)
	if !strings.Contains(hints, "under 20 chars") {
		t.Errorf("should detect vague instructions, got: %q", hints)
	}
}

func TestComputeUsageHintsMultiFileNoPlan(t *testing.T) {
	t.Parallel()
	now := time.Now()
	events := []parser.SessionEvent{
		{Type: parser.EventUserMessage, UserText: "refactor everything", Timestamp: now},
		{Type: parser.EventToolUse, ToolName: "Edit", ToolInput: "a.go", Timestamp: now},
		{Type: parser.EventToolUse, ToolName: "Edit", ToolInput: "b.go", Timestamp: now},
		{Type: parser.EventToolUse, ToolName: "Edit", ToolInput: "c.go", Timestamp: now},
		{Type: parser.EventToolUse, ToolName: "Edit", ToolInput: "d.go", Timestamp: now},
		{Type: parser.EventToolUse, ToolName: "Edit", ToolInput: "e.go", Timestamp: now},
		{Type: parser.EventUserMessage, UserText: "done?", Timestamp: now},
	}
	stats := analyzer.NewStats()
	for _, ev := range events {
		stats.Update(ev)
	}

	hints := computeUsageHints(events, stats)
	if !strings.Contains(hints, "files modified without Plan Mode") {
		t.Errorf("should detect multi-file without plan mode, got: %q", hints)
	}
}

func TestComputeUsageHintsCleanSession(t *testing.T) {
	t.Parallel()
	now := time.Now()
	events := []parser.SessionEvent{
		{Type: parser.EventUserMessage, UserText: "Please implement user authentication with JWT tokens in the auth module", Timestamp: now},
		{Type: parser.EventToolUse, ToolName: "EnterPlanMode", ToolInput: "", Timestamp: now},
		{Type: parser.EventToolUse, ToolName: "Read", ToolInput: "auth.go", Timestamp: now},
		{Type: parser.EventToolUse, ToolName: "Edit", ToolInput: "auth.go", Timestamp: now},
	}
	stats := analyzer.NewStats()
	for _, ev := range events {
		stats.Update(ev)
	}

	hints := computeUsageHints(events, stats)
	if hints != "" {
		t.Errorf("clean session should produce no hints, got: %q", hints)
	}
}
