---
name: buddy-test-guidance
description: >
  Use automatically when tests have failed 2+ times consecutively.
  Analyzes failure patterns and suggests targeted debugging strategies
  based on test output correlation and past failure solutions.
user-invocable: false
allowed-tools: mcp__claude-buddy__buddy_skill_context, mcp__claude-buddy__buddy_patterns, mcp__claude-buddy__buddy_recall
---

Test failure debugging advisor.

## Steps

1. Call buddy_skill_context with skill_name="buddy-test-guidance" to get test status, recent failures, and correlated files
2. Call buddy_patterns with the failing test name or error message
3. If the failure involves a specific file, call buddy_recall to find past fixes for that file

## Output

Provide a targeted debugging strategy:
- Root cause hypothesis (one sentence based on error pattern)
- Specific debugging step to try next (one actionable instruction)
- If a past similar failure exists, reference the resolution

Keep it under 5 lines. Be specific about which file/function to investigate.
