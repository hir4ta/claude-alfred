# alfred

Quality butler for [Claude Code](https://docs.anthropic.com/en/docs/claude-code). Monitors Claude Code's actions, enforces quality gates, and learns from past sessions.

**Invisible. Mechanical. Relentless.**

## What alfred does

alfred runs as hooks + MCP server inside Claude Code. It watches every file edit, every bash command, every commit — and enforces quality through walls, not suggestions.

- **Lint/type gates**: PostToolUse runs lint and type checks after every file write. Errors become DIRECTIVE — Claude must fix before continuing
- **Edit blocking**: PreToolUse blocks edits when pending lint/type errors exist (DENY)
- **Test-first enforcement**: UserPromptSubmit detects implementation prompts and injects test-first DIRECTIVE
- **Error resolution cache**: Bash errors are matched against past resolutions via Voyage AI vector search and injected as context
- **Exemplar injection**: Relevant before/after code examples injected for implementation prompts (few-shot, research-backed)
- **Convention enforcement**: Project-specific coding conventions injected at session start
- **Quality scoring**: Every gate pass/fail, every error hit/miss is tracked and scored (0-100)
- **Self-reflection**: Commit gate injects 4-point verification checklist (edge cases, silent failures, simplicity, conventions)

## Architecture

```
User → Claude Code → (alfred hooks: monitor + inject + gate)
              ↓ when needed
           alfred MCP (knowledge DB)
```

| Component | Role | Weight |
|---|---|---|
| Hooks (6 events) | Monitor, inject context, enforce gates | 70% |
| DB + Voyage AI | Knowledge storage, vector search | 20% |
| MCP tool | Claude Code interface to knowledge | 10% |

## Install

```bash
# Build
bun install
bun build.ts
bun link          # Makes 'alfred' command available globally

# Setup (writes to ~/.claude/)
alfred init
```

Requires: `VOYAGE_API_KEY` environment variable (get at https://dash.voyageai.com/)

## Commands

```bash
alfred init          # Setup: MCP, hooks, rules, skills, agents, gates
alfred serve         # Start MCP server (stdio, called by Claude Code)
alfred hook <event>  # Handle hook event (called by Claude Code)
alfred tui           # Quality dashboard in terminal
alfred scan          # Full quality scan (lint/type/test + score)
alfred doctor        # Check installation health
alfred uninstall     # Remove alfred from system
alfred version       # Show version
```

## Skills

- `/alfred:review` — Deep multi-agent code review with Judge filtering (HubSpot 3-criteria pattern)
- `/alfred:conventions` — Scan codebase and discover coding conventions with adoption rates

## TUI Dashboard

```bash
task tui   # or: bun src/tui/main.tsx
```

Displays: Quality Score, Gates pass/fail, Knowledge hits, Recent Events stream, Session info. Press `?` for help (EN/JA toggle with Tab).

## Stack

TypeScript (Bun 1.3+, ESM) / SQLite (bun:sqlite) / Voyage AI (voyage-4-large + rerank-2.5) / MCP SDK / TUI (OpenTUI)

## Design docs

See `design/` for architecture, detailed design, and research references.
