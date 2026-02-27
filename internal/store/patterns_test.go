package store

import (
	"testing"

	"github.com/hir4ta/claude-buddy/internal/parser"
)

func TestExtractPatterns_EventType(t *testing.T) {
	t.Parallel()

	events := []EventRow{
		{
			ID:        1,
			SessionID: "s1",
			EventType: int(parser.EventUserMessage),
			UserText:  "Fix the database connection timeout",
		},
		{
			// This is the correct event type for assistant text.
			ID:            2,
			SessionID:     "s1",
			EventType:     int(parser.EventAssistantText),
			AssistantText: "The timeout was caused by connection pool exhaustion. I fixed by increasing the max pool size.",
		},
		{
			// ToolUse should NOT be scanned for patterns.
			ID:            3,
			SessionID:     "s1",
			EventType:     int(parser.EventToolUse),
			ToolName:      "Edit",
			AssistantText: "", // tool_use has no assistant text
		},
	}

	patterns := ExtractPatterns(events, "s1", "go")

	if len(patterns) == 0 {
		t.Fatal("ExtractPatterns returned 0 patterns; expected at least 1 from assistant text")
	}

	// The assistant text contains "caused by" (error_solution signal) and "fixed by" (error keyword).
	foundError := false
	for _, p := range patterns {
		if p.PatternType == "error_solution" {
			foundError = true
			break
		}
	}
	if !foundError {
		t.Errorf("expected error_solution pattern from 'caused by' / 'fixed by' text, got types: %v",
			patternTypes(patterns))
	}
}

func TestExtractPatterns_SkipsToolUse(t *testing.T) {
	t.Parallel()

	// Only ToolUse events — should produce no patterns.
	events := []EventRow{
		{
			ID:        1,
			SessionID: "s1",
			EventType: int(parser.EventToolUse),
			ToolName:  "Read",
		},
		{
			ID:        2,
			SessionID: "s1",
			EventType: int(parser.EventToolUse),
			ToolName:  "Edit",
		},
	}

	patterns := ExtractPatterns(events, "s1", "go")
	if len(patterns) != 0 {
		t.Errorf("ExtractPatterns(tool_use only) = %d patterns, want 0", len(patterns))
	}
}

func TestExtractPatterns_MultipleTypes(t *testing.T) {
	t.Parallel()

	events := []EventRow{
		{
			ID:        1,
			SessionID: "s1",
			EventType: int(parser.EventUserMessage),
			UserText:  "Design the API gateway",
		},
		{
			ID:            2,
			SessionID:     "s1",
			EventType:     int(parser.EventAssistantText),
			AssistantText: "The gateway is responsible for rate limiting all incoming traffic. We chose REST instead of GraphQL for simplicity.",
		},
	}

	patterns := ExtractPatterns(events, "s1", "go")

	types := make(map[string]bool)
	for _, p := range patterns {
		types[p.PatternType] = true
	}

	if !types["architecture"] {
		t.Error("expected architecture pattern from 'responsible for' signal")
	}
	if !types["decision"] {
		t.Error("expected decision pattern from 'instead of' signal")
	}
}

func patternTypes(patterns []PatternRow) []string {
	var types []string
	for _, p := range patterns {
		types = append(types, p.PatternType)
	}
	return types
}
