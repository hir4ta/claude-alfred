---
name: buddy-review
description: >
  Review recent changes against pattern DB knowledge. Checks for known
  anti-patterns, past failures with similar code, and architectural decisions.
user-invocable: true
allowed-tools: mcp__claude-buddy__buddy_skill_context, mcp__claude-buddy__buddy_patterns, mcp__claude-buddy__buddy_alerts, mcp__claude-buddy__buddy_decisions, Bash, Read
---

Review recent code changes using pattern database knowledge.

## Steps

1. Call buddy_skill_context with skill_name="buddy-review" to get modified files, test status, and related patterns in one call
2. Run 'git diff --stat' to see detailed changes
3. Call buddy_patterns for specific files if more detail needed
4. Call buddy_decisions to check if changes align with past architectural choices

## Output

- List specific issues found (max 5)
- Reference past patterns or decisions that are relevant
- Suggest concrete fixes if issues found
- If clean, say "No issues found" and summarize what was reviewed
