---
name: buddy-analyze
description: >
  Analyze blast radius of planned changes and review recent modifications.
  Shows importers, test coverage, co-change history, anti-patterns, and
  architectural alignment.
user-invocable: true
allowed-tools: mcp__claude-buddy__buddy_skill_context, mcp__claude-buddy__buddy_patterns, mcp__claude-buddy__buddy_decisions, mcp__claude-buddy__buddy_alerts, Read, Grep, Glob, Bash
context: fork
agent: Explore
---

Impact analysis and change review.

## Steps

1. Identify target files from the user's request or recent git diff
2. Call buddy_skill_context with skill_name="buddy-analyze" for modified files, test status, and patterns
3. Use Grep to find importers/references, Glob for related test files
4. Call buddy_decisions to check architectural constraints
5. Call buddy_patterns for known issues with these files

## Output

- Blast radius: files referencing the target module
- Test coverage: existing test files for this code
- Past issues: known problems from pattern DB
- Alignment: whether changes match past architectural decisions
- Recommendations: suggested approach

Keep under 10 lines. Be specific about file paths.
