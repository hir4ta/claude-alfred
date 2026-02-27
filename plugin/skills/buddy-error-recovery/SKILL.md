---
name: buddy-error-recovery
description: >
  Invoke after any tool returns an error, especially compilation errors,
  import failures, or permission denied. Searches cross-session knowledge
  for exact resolution diffs. Invoke BEFORE attempting a manual fix.
user-invocable: false
allowed-tools: mcp__claude-buddy__buddy_skill_context, mcp__claude-buddy__buddy_patterns, mcp__claude-buddy__buddy_recall
---

Automatic error recovery advisor. Triggered after tool failures.

## Steps

1. Call buddy_skill_context with skill_name="buddy-error-recovery" to get failure context, recent errors, and related solutions
2. If a past solution with resolution diff exists, present the exact fix
3. If no direct match, call buddy_patterns with the error message to find similar past solutions

## Output

- If an exact past fix exists: "Past fix found: change X to Y in file Z"
- If a similar pattern exists: "Similar error was resolved by: [approach]"
- If no past knowledge: "No past solutions found. Suggested approach: [one sentence]"

Keep it under 4 lines. Include file paths and concrete changes when available.
