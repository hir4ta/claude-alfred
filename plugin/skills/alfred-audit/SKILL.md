---
name: alfred-audit
description: >
  Review your project's Claude Code setup and suggest improvements.
  Checks CLAUDE.md, hooks, skills, agents, rules, MCP, and memory
  against best practices from the alfred knowledge base.
user-invocable: true
allowed-tools: Read, Glob, Grep, mcp__claude-alfred__knowledge, mcp__claude-alfred__state
context: fork
agent: general-purpose
---

Project setup reviewer and improvement advisor.

## Steps

1. Detect project root (look for .git or CLAUDE.md)
2. Scan for Claude Code configuration:
   - CLAUDE.md — exists? size? has Commands/Rules sections?
   - .claude/hooks.json — exists? which events are hooked?
   - .claude/skills/ — count and names
   - .claude/agents/ — count and names
   - .claude/rules/ — count and names
   - .mcp.json or .claude/mcp.json — MCP servers configured?
   - .claude/memory/ or MEMORY.md — memory in use?
3. For each found configuration, Read the file and assess quality
4. Call knowledge to get best practices for each feature area
5. Compare current setup vs best practices
6. Determine proficiency level (beginner/intermediate/advanced)

## Output

Report format:
- **Level**: [Beginner/Intermediate/Advanced]
- **Features in use**: [list]
- **Missing features**: [list with brief explanation of value]
- **Improvement suggestions**: [max 5, ordered by impact]
  - For each: what to do, why it helps, one-line example

Keep under 20 lines. Be specific and actionable.
