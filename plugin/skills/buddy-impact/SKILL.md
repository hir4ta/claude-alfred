---
name: buddy-impact
description: >
  Analyze the blast radius of planned changes before editing. Shows importers,
  test coverage, and co-change history for a file.
user-invocable: true
allowed-tools: mcp__claude-buddy__buddy_patterns, mcp__claude-buddy__buddy_decisions, Read, Grep, Glob, Bash
---

Analyze the impact of changing a specific file or module.

## Steps

1. Identify the target file from the user's request
2. Call buddy_patterns to find related past changes and issues
3. Call buddy_decisions to check for architectural constraints
4. Use Grep to find importers/references to this file
5. Use Glob to find related test files

## Output

- Blast radius: number of files that reference this module
- Test coverage: which test files exist for this code
- Past issues: any known problems from pattern DB
- Recommendations: suggested approach for the change

Keep output under 10 lines. Be specific about file paths.
