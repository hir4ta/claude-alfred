---
name: skill-audit
description: >
  Audit SKILL.md files against Anthropic's official skill design guide and the
  Agent Skills open standard. Evaluates 21 checks: frontmatter compliance (name
  format 1-64 chars kebab-case no consecutive hyphens, description 1-1024 chars
  with WHAT+WHEN, no XML brackets, no reserved prefixes), structure (SKILL.md
  exact naming, folder kebab-case, under 500 lines, no README.md), progressive
  disclosure (supporting files for 200+ line skills), best practices (actionable
  instructions, error handling, examples, negative triggers), and security
  (allowed-tools restriction, no embedded secrets). Produces scored report with
  auto-fix via --fix. Use when auditing skills before publishing, checking
  plugin quality, or after guideline updates. NOT for code review (use a code
  review tool). NOT for CLAUDE.md review (use /claude-md-audit).
user-invocable: true
argument-hint: "[skill-path or --all] [--fix]"
allowed-tools: Read, Edit, Glob, Grep, WebFetch
context: fork
model: sonnet
---

# /skill-audit — Skill Best Practices Auditor

Audit skills against the latest official guidelines. Self-updating: fetches
current best practices before every evaluation.

## Phase 0: Refresh Knowledge (ALWAYS do this first)

Fetch the latest skill guidelines from these authoritative sources:

1. **WebFetch** `https://code.claude.com/docs/en/skills` with prompt:
   "Extract all skill frontmatter fields, their constraints (required/optional,
   character limits, format rules), forbidden items, and description best practices."

2. **WebFetch** `https://code.claude.com/docs/en/best-practices` with prompt:
   "Extract skill-related best practices: description structure, progressive
   disclosure rules, line limits, and content guidelines."

3. Merge fetched rules with the baseline [checklist.md](checklist.md).
   If fetched docs contain newer rules not in checklist, **use the newer version**.

## Phase 1: Discover Skills

Parse `$ARGUMENTS`:
- Path given → review that single skill
- `--all` → scan: `.claude/skills/*/SKILL.md`, `~/.claude/skills/*/SKILL.md`
- No arguments → scan current project `.claude/skills/`

## Phase 2: Evaluate Each Skill

Read the checklist from [checklist.md](checklist.md) and evaluate all checks.

### A. Frontmatter Compliance (CRITICAL)

| # | Check | Rule |
|---|---|---|
| A1 | name exists, kebab-case | `^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`, no `--`, 1-64 chars, matches folder name |
| A2 | description exists | Non-empty, string type (not YAML list) |
| A3 | No XML angle brackets in ANY frontmatter field | `<` `>` forbidden (security: frontmatter in system prompt) |
| A4 | No reserved prefixes | "claude" or "anthropic" in name = reserved |

### B. Description Quality (HIGH)

| # | Check | Rule |
|---|---|---|
| B1 | Includes WHAT | First sentence describes the action |
| B2 | Includes WHEN | Trigger phrases: "Use when...", specific tasks users might say |
| B3 | Under 1024 characters | Claude Code truncation limit |
| B4 | Specific and actionable | Not vague ("Helps with projects" = FAIL) |

### C. Structure (MIXED)

| # | Check | Severity | Rule |
|---|---|---|---|
| C1 | SKILL.md exact filename | CRITICAL | Case-sensitive, no variations |
| C2 | Folder name kebab-case | CRITICAL | Same format as A1 |
| C3 | Under 500 lines | MEDIUM | Move detailed docs to references/ |
| C4 | No README.md inside skill folder | LOW | Docs go in SKILL.md or references/ |

### D. Progressive Disclosure (MEDIUM)

| # | Check | Rule |
|---|---|---|
| D1 | 200+ line skills use supporting files | scripts/, references/, examples/ |
| D2 | Supporting files referenced from SKILL.md | Links with relative paths |
| D3 | SKILL.md focused on core instructions | Reference material in separate files |

### E. Best Practices (MIXED)

| # | Check | Severity | Rule |
|---|---|---|---|
| E1 | Actionable instructions | MEDIUM | "Run X" not "validate things" |
| E2 | Error handling documented | MEDIUM | Troubleshooting section with 2+ items |
| E3 | Examples provided | LOW | At least one Example section |
| E4 | Negative triggers | LOW | "NOT for X — use Y" in description |

### F. Security (MIXED)

| # | Check | Severity | Rule |
|---|---|---|---|
| F1 | allowed-tools restricts access | MEDIUM | Least privilege |
| F2 | No secrets in content | CRITICAL | No API keys, tokens, passwords |

## Phase 3: Score & Report

Scoring:
- CRITICAL: gate (any failure = NEEDS_FIXES)
- HIGH: 2 points per pass
- MEDIUM: 1 point per pass
- LOW: 0.5 points per pass
- N/A: excluded from denominator

### Output Format

```
## Skill Audit: {name}

**Score: {X}/{max} ({pct}%)**
**Status: {PERFECT | PASS | NEEDS_FIXES}**

| # | Check | Status | Evidence |
|---|-------|--------|----------|
| A1 | name kebab-case | OK/NG | "my-skill" |
...

### Fixes Required
1. [CRITICAL] description (line N)

### Summary (--all)
| Skill | Score | Status | Issues |
|-------|-------|--------|--------|
```

## Phase 4: Auto-Fix (--fix)

Only fix frontmatter issues:
- A3: Remove `<` `>` from argument-hint
- B2: Append trigger phrases (propose, then apply)
- C4: Remove README.md (warn first)

Never modify instructions or semantic content.

## Example

User: `/skill-audit --all`

```
## Skill Audit Summary

| Skill | Score | Status |
|-------|-------|--------|
| deploy | 100% | PERFECT |
| review | 96% | PASS |
| my-tool | 72% | NEEDS_FIXES |

my-tool issues:
1. [CRITICAL] A3: argument-hint contains "<url>" — remove angle brackets
2. [HIGH] B2: No "Use when" trigger phrases in description
```

## Troubleshooting

- **WebFetch fails**: Use checklist.md baseline — it contains all essential rules.
- **No skills found**: Check path; try `--all` to scan both project and user directories.
- **--fix breaks a skill**: Show the diff and offer to revert.

## Guardrails

- ALWAYS fetch latest docs in Phase 0 before evaluating
- NEVER modify skill body/instructions in --fix mode
- ALWAYS show changes before applying
- Report but don't score N/A items
