---
name: rules-audit
description: >
  Audit .claude/rules/*.md files for quality, structure, and effectiveness.
  Validates one-topic-per-file, path glob syntax, duplication with CLAUDE.md,
  instruction specificity, and file organization. Checks both project
  (.claude/rules/) and user (~/.claude/rules/) rule files. Use when adding
  new rules, reorganizing rules, or checking rule quality after Claude ignores
  instructions. NOT for CLAUDE.md itself (use /claude-md-audit). NOT for
  hooks configuration (use /hooks-audit).
user-invocable: true
argument-hint: "[rules-dir or file] [--all]"
allowed-tools: Read, Glob, Grep, WebFetch
context: fork
model: sonnet
---

# /rules-audit — Rules Quality Auditor

## Phase 0: Refresh Knowledge

1. **WebFetch** `https://code.claude.com/docs/en/memory` with prompt:
   "Extract .claude/rules/ best practices: path-specific rules frontmatter,
   glob patterns, file organization, priority order, symlink support."

## Phase 1: Discover Rules

- Path given → audit that file or directory
- `--all` → scan: `.claude/rules/`, `~/.claude/rules/`
- No arguments → `.claude/rules/`

Also read `./CLAUDE.md` for duplication checking.

## Phase 2: Evaluate Each Rule File

### Structure (HIGH)

| # | Check | Rule |
|---|---|---|
| S1 | One topic per file | Filename should describe the single concern |
| S2 | Descriptive filename | `testing.md`, `api-design.md` — not `rules.md` or `misc.md` |
| S3 | Concise | Rules are loaded into context — every line costs tokens |
| S4 | Markdown formatted | Headers, bullets for scannability |

### Path Frontmatter (HIGH)

| # | Check | Rule |
|---|---|---|
| P1 | paths glob syntax valid | Test each pattern compiles |
| P2 | paths patterns match files | `src/**/*.ts` — do matching files actually exist? |
| P3 | Brace expansion valid | `*.{ts,tsx}` format correct |
| P4 | No overly broad patterns | `**/*` matches everything — probably unintended |

### Content Quality (MEDIUM)

| # | Check | Rule |
|---|---|---|
| C1 | Instructions are actionable | "Use X" not "consider X" |
| C2 | No vague guidance | "write good code" = useless |
| C3 | Rules are testable | Can verify if Claude followed them |

### Duplication (MEDIUM)

| # | Check | Rule |
|---|---|---|
| D1 | No duplication with CLAUDE.md | Same instruction in both places |
| D2 | No duplication between rule files | Same rule in two different files |
| D3 | No contradiction with CLAUDE.md | Conflicting instructions |

### Security (CRITICAL)

| # | Check | Rule |
|---|---|---|
| F1 | No secrets in rule content | API keys, tokens, paths with credentials |

## Phase 3: Score & Report

```
## Rules Audit: .claude/rules/

**Files: {N}** | **Path-scoped: {N}** | **Global: {N}** | **Score: {X}/100**

| File | Lines | Scoped? | Issues |
|------|-------|---------|--------|
| testing.md | 12 | Yes (*.test.ts) | OK |
| misc.md | 45 | No | [HIGH] S2: Vague filename |
```

## Example

User: `/rules-audit`

```
## Rules Audit: .claude/rules/

**Files: 5** | **Path-scoped: 3** | **Global: 2** | **Score: 78/100**

| File | Lines | Issues |
|------|-------|--------|
| go-style.md | 28 | OK |
| testing.md | 15 | OK |
| general.md | 52 | [HIGH] S1: Multiple topics (style + testing + deployment) |
| api.md | 8 | [MEDIUM] P2: paths "src/api/**" — directory doesn't exist |
| security.md | 12 | [MEDIUM] D1: Line 3 duplicates CLAUDE.md line 45 |
```

## Troubleshooting

- **Path patterns seem wrong**: Glob patterns use `*` (single dir) and `**` (recursive). Brace expansion: `*.{ts,tsx}`.
- **Rules not loading**: Rules without `paths` frontmatter load at startup. Path-scoped rules load when Claude reads matching files.
- **WebFetch fails**: Evaluate using the embedded checklist (Phase 2 tables).
