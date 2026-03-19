---
name: claude-md-audit
description: >
  Audit CLAUDE.md files for size, structure, specificity, contradictions, and
  @import validity. Checks all hierarchy levels: project root, .claude/,
  parent directories, user-level (~/.claude/CLAUDE.md). Validates against
  official guidelines: under 200 lines, markdown headers and bullets, concrete
  verifiable instructions, no self-evident rules, no conflicts with .claude/rules/.
  Use when editing CLAUDE.md, checking project instruction quality, or wanting
  to ensure Claude follows your rules consistently. NOT for rules/ files (use
  /rules-audit). NOT for code review.
user-invocable: true
argument-hint: "[path] [--all]"
allowed-tools: Read, Glob, Grep, WebFetch
context: fork
model: sonnet
---

# /claude-md-audit — CLAUDE.md Quality Auditor

## Phase 0: Refresh Knowledge

1. **WebFetch** `https://code.claude.com/docs/en/memory` with prompt:
   "Extract CLAUDE.md best practices: size limits, structure rules, @import
   syntax, file locations, priority order, what to include vs exclude."

2. **WebFetch** `https://code.claude.com/docs/en/best-practices` with prompt:
   "Extract CLAUDE.md writing guidelines: specificity, structure, include/exclude
   table, emphasis rules, pruning advice."

## Phase 1: Discover Files

- Path given → audit that file
- `--all` → scan all CLAUDE.md locations:
  - `./CLAUDE.md`, `./.claude/CLAUDE.md`
  - Parent directories up to git root
  - `~/.claude/CLAUDE.md`
- No arguments → `./CLAUDE.md`

Also discover `.claude/rules/*.md` for contradiction checking.

## Phase 2: Evaluate

### Size & Structure (HIGH)

| # | Check | Rule |
|---|---|---|
| S1 | Under 200 lines | Longer files reduce adherence |
| S2 | Uses markdown headers | Scannable sections, not wall of text |
| S3 | Uses bullets/lists | Not dense paragraphs |
| S4 | Has required sections | Stack, Commands, Structure, or Rules recommended |

### Content Quality (HIGH)

| # | Check | Rule |
|---|---|---|
| C1 | Instructions are specific | "Use 2-space indentation" not "Format code properly" |
| C2 | Instructions are verifiable | Can check if Claude followed them |
| C3 | Commands are copy-pasteable | `npm test` not "run the tests" |
| C4 | No self-evident rules | "write clean code" adds no value |
| C5 | No stale/outdated info | References to files/commands that don't exist |

### @Imports (MEDIUM)

| # | Check | Rule |
|---|---|---|
| I1 | @imported files exist | Verify each `@path` resolves |
| I2 | Import depth under 5 | No deeply nested chains |
| I3 | No circular imports | A→B→A detected |

### Contradictions (HIGH)

| # | Check | Rule |
|---|---|---|
| X1 | No self-contradictions | Two rules in same file that conflict |
| X2 | No CLAUDE.md vs rules/ conflicts | Same instruction with different content |
| X3 | No parent vs child conflicts | Monorepo CLAUDE.md inconsistencies |

### Security (CRITICAL)

| # | Check | Rule |
|---|---|---|
| F1 | No secrets | API keys, tokens, passwords |
| F2 | No absolute user paths | Hardcoded paths break for other team members |

## Phase 3: Score & Report

Same scoring as skill-audit. Output format:

```
## CLAUDE.md Audit: {path}

**Lines: {N}** | **Score: {X}/100** | **Status: {PASS|NEEDS_FIXES}**

### Findings
| # | Check | Status | Detail |
...

### Include/Exclude Analysis
| Should include | Found? |
|----------------|--------|
| Build commands | Yes/No |
| Test commands | Yes/No |
| Code style rules | Yes/No |
```

## Example

User: `/claude-md-audit`

```
## CLAUDE.md Audit: ./CLAUDE.md

**Lines: 87** | **Score: 92/100** | **Status: PASS**

### Findings
- [HIGH] C4: Line 23 "Write clean code" — self-evident, remove
- [MEDIUM] S4: Missing "## Commands" section — add build/test commands

### Include/Exclude Analysis
| Should include | Found? |
|----------------|--------|
| Build commands | No — add `## Commands` |
| Test commands | Yes (line 45) |
| Code style rules | Yes (line 12-18) |
```

## Troubleshooting

- **Multiple CLAUDE.md files**: Use `--all` to audit the full hierarchy and check for contradictions.
- **@import resolution fails**: Verify paths are relative to the containing file, not cwd.
- **WebFetch fails**: Evaluate using the rules embedded in this skill (Phase 2 tables).
