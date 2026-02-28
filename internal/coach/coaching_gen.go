package coach

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hir4ta/claude-buddy/internal/sessiondb"
)

// GenerateCoaching produces AI-powered coaching text based on task context
// and past patterns. Returns "", nil if the LLM is unavailable (graceful skip).
func GenerateCoaching(ctx context.Context, sdb *sessiondb.SessionDB, taskType, domain string, patterns []string, timeout time.Duration) (string, error) {
	prompt := buildCoachingPrompt(taskType, domain, patterns)
	raw, err := Generate(ctx, sdb, prompt, timeout)
	if err != nil {
		if errors.Is(err, ErrClaudeNotFound) {
			return "", nil
		}
		return "", err
	}

	return strings.TrimSpace(raw), nil
}

// buildCoachingPrompt constructs the LLM prompt from task context.
func buildCoachingPrompt(taskType, domain string, patterns []string) string {
	var b strings.Builder

	b.WriteString("You are a coding coach. Generate a brief, actionable coaching message (2-4 sentences) for the developer.\n\n")

	if taskType != "" {
		fmt.Fprintf(&b, "Task type: %s\n", taskType)
	}
	if domain != "" {
		fmt.Fprintf(&b, "Domain: %s\n", domain)
	}

	if len(patterns) > 0 {
		b.WriteString("\nPast patterns from this project:\n")
		for _, p := range patterns {
			fmt.Fprintf(&b, "- %s\n", p)
		}
	}

	b.WriteString("\nProvide coaching that is specific to the context above. Focus on the most impactful advice. Return only the coaching text, no labels or formatting.")

	return b.String()
}
