---
description: Best practices when creating or editing CLAUDE.md files
paths:
  - "**/CLAUDE.md"
  - "**/.claude/CLAUDE.md"
---

# CLAUDE.md Guidelines

- Keep CLAUDE.md under 200 lines — it loads into context every session
- Move implementation details (schema internals, algorithm specifics) to code comments or rules/ files
- Structure: Project overview → Commands → Rules → Environment (most important first)
- Use tables for structured data (env vars, timeouts, commands)
- Do NOT duplicate content already in rules/ files — CLAUDE.md links to them implicitly
- Avoid verbose prose; use bullet points and concise imperative sentences
- CLAUDE.md is project-level instructions, not documentation — it tells Claude HOW to work, not what the project does
