---
name: buddy-estimate
description: >
  Estimate task complexity based on historical workflow data. Shows expected
  tool count, success rate, and recommended workflow.
user-invocable: true
allowed-tools: mcp__claude-buddy__buddy_estimate, mcp__claude-buddy__buddy_patterns
---

Estimate the complexity of a task using historical data.

## Steps

1. Determine the task type from the user's description (bugfix, feature, refactor, research, review)
2. Call buddy_estimate with the task type
3. Call buddy_patterns to find similar past tasks if available

## Output

- Task type classification
- Expected tool count (median from past sessions)
- Success rate for this type of task
- Recommended workflow steps
- Any relevant patterns from past sessions

Keep it concise — 5-8 lines max.
