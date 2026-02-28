package coach

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/hir4ta/claude-buddy/internal/sessiondb"
)

// PatternResult represents a single extracted knowledge pattern.
type PatternResult struct {
	Type    string `json:"type"`    // error_solution, architecture, decision
	Title   string `json:"title"`   // short summary
	Content string `json:"content"` // reusable knowledge
}

const extractPrompt = `Analyze the following session events and extract reusable knowledge patterns.
Return a JSON array of objects with these fields:
- "type": one of "error_solution", "architecture", "decision"
- "title": short summary (under 80 chars)
- "content": the reusable knowledge (1-3 sentences)

Only include genuinely reusable patterns. If nothing is worth extracting, return an empty array [].
Return ONLY the JSON array, no other text.

Events:
`

// ExtractPatterns uses the LLM to extract reusable knowledge from session events.
// Returns nil, nil if the LLM is unavailable (graceful skip).
func ExtractPatterns(ctx context.Context, sdb *sessiondb.SessionDB, events string, timeout time.Duration) ([]PatternResult, error) {
	if events == "" {
		return nil, nil
	}

	prompt := extractPrompt + events
	raw, err := Generate(ctx, sdb, prompt, timeout)
	if err != nil {
		if errors.Is(err, ErrClaudeNotFound) {
			return nil, nil
		}
		return nil, err
	}

	if raw == "" {
		return nil, nil
	}

	// Strip markdown code fences if present.
	raw = stripCodeFence(raw)

	var results []PatternResult
	if err := json.Unmarshal([]byte(raw), &results); err != nil {
		return nil, nil
	}

	return results, nil
}

// stripCodeFence removes ```json ... ``` wrapping from LLM output.
func stripCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		// Remove opening fence line.
		if idx := strings.Index(s, "\n"); idx >= 0 {
			s = s[idx+1:]
		}
		// Remove closing fence.
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
		s = strings.TrimSpace(s)
	}
	return s
}
