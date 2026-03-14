---
name: alfred-workflow
description: Proactive workflow guidance for development tasks
globs:
  - "**/*"
---

# Development Workflow

## Judging task scale — spec is NOT always needed

**Use /alfred:plan (spec-driven) when:**
- Multiple files or components need coordinated changes
- Design decisions with trade-offs need to be made
- The task will span multiple sessions or survive compaction
- The user explicitly asks for planning or structured work

**Skip spec, just implement when:**
- Single file fix or small change
- Clear requirements with no design ambiguity
- Quick bug fix, config change, docs update
- The user says "just do it" or similar

Use judgment. When in doubt, ask: "This looks substantial — want me to create a spec with /alfred:plan, or dive straight in?"

## When to use alfred skills proactively

**Large task (new feature, major refactor, multi-file change):**
1. `/alfred:plan <task-slug>` — Create a structured spec with multi-agent deliberation
2. Implement following the spec
3. `/alfred:review` — Multi-agent code review before committing
4. Update spec session.md with final status

**Design exploration (unclear direction, multiple options):**
1. `/alfred:brainstorm <theme>` — Divergent thinking with 3 agents
2. `/alfred:refine` — Converge on a decision
3. Optionally `/alfred:plan` — Create spec from the decision

**Quick fix (bug fix, small change):**
- No plan needed
- `/alfred:review` only for non-trivial changes

## Proactive behavior — be JARVIS, not a passive tool

- When the user describes a large task, **suggest** `/alfred:plan` (don't just start coding)
- When implementation is complete, **suggest** `/alfred:review`
- When a spec is active, keep session.md updated via the `spec` MCP tool
- After review findings, record key decisions to spec's decisions.md
- When the user is stuck or exploring options, **suggest** `/alfred:brainstorm`
- Always explain WHY you're suggesting a skill — don't just mention it
