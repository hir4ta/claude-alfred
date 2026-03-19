---
name: permissions-audit
description: >
  Audit Claude Code permissions across all settings files for security issues.
  Checks settings.json and settings.local.json at project and user levels for
  allow/deny conflicts, over-permissive patterns (Bash(*), Read without scope),
  cross-file conflicts, dangerous feature flags, and sandbox configuration.
  Validates against official permission rule syntax including gitignore patterns
  for Read/Edit and glob patterns for Bash. Use when reviewing security posture,
  after settings changes, or before deploying to a team. NOT for hooks
  (use /hooks-audit). NOT for code review.
user-invocable: true
argument-hint: "[settings-path] [--all]"
allowed-tools: Read, Glob, Grep, WebFetch
context: fork
model: sonnet
---

# /permissions-audit — Permissions & Settings Security Auditor

## Phase 0: Refresh Knowledge

1. **WebFetch** `https://code.claude.com/docs/en/permissions` with prompt:
   "Extract permission rule syntax: allow/deny/ask evaluation order, tool
   patterns (Bash glob, Read/Edit gitignore, WebFetch domain), wildcards,
   absolute vs relative paths, and dangerous patterns."

2. **WebFetch** `https://code.claude.com/docs/en/settings` with prompt:
   "Extract settings file locations, precedence order, array merge behavior,
   managed-only settings, sandbox configuration, and security-relevant fields."

## Phase 1: Discover Settings

- Path given → read that file
- `--all` or no arguments → scan all:
  - `~/.claude/settings.json` (user)
  - `.claude/settings.json` (project shared)
  - `.claude/settings.local.json` (project local)

## Phase 2: Evaluate

### Permission Conflicts (CRITICAL)

| # | Check | Rule |
|---|---|---|
| PC1 | No same-pattern in allow AND deny | Same file: deny wins, but indicates confusion |
| PC2 | No cross-file conflicts | Project allows what user denies (deny always wins) |
| PC3 | Evaluation order understood | deny > ask > allow (first match) |

### Over-Permissive (HIGH)

| # | Check | Rule |
|---|---|---|
| OP1 | No Bash(*) in allow | Allows ALL shell commands |
| OP2 | No Read without scope | Unrestricted file access |
| OP3 | No Edit without scope | Unrestricted file modification |
| OP4 | No mcp__*__* wildcard | Allows all MCP tools from all servers |

### Path Syntax (HIGH)

| # | Check | Rule |
|---|---|---|
| PS1 | Read/Edit use correct prefix | `//` absolute, `~/` home, `/` project root, bare = cwd |
| PS2 | Common mistake: `/Users/x` | This is project-root-relative, NOT absolute. Use `//Users/x` |
| PS3 | Bash patterns space-aware | `Bash(ls *)` != `Bash(ls*)` (word boundary) |

### Dangerous Features (CRITICAL)

| # | Check | Rule |
|---|---|---|
| DF1 | bypassPermissions mode | Skips ALL checks — production danger |
| DF2 | enableAllProjectMcpServers | Auto-approves untrusted MCP servers |
| DF3 | disableAllHooks | Disables safety hooks |
| DF4 | sandbox disabled | No filesystem/network isolation |

### Sandbox Config (MEDIUM)

| # | Check | Rule |
|---|---|---|
| SB1 | allowWrite paths reasonable | Not overly broad |
| SB2 | denyRead covers secrets | ~/.ssh, credentials files |
| SB3 | allowedDomains scoped | Not `*` for all domains |

## Phase 3: Score & Report

```
## Permissions Audit

**Files scanned: {N}** | **Score: {X}/100** | **Status: {PASS|NEEDS_FIXES}**

### Settings Precedence
| Scope | File | Exists |
|-------|------|--------|
| User | ~/.claude/settings.json | Yes/No |
| Project | .claude/settings.json | Yes/No |
| Local | .claude/settings.local.json | Yes/No |

### Findings
| # | Check | Scope | Status | Detail |
...
```

## Example

User: `/permissions-audit`

```
## Permissions Audit

**Files: 2** | **Score: 70/100** | **Status: NEEDS_FIXES**

### Findings
- [CRITICAL] OP1: `.claude/settings.json` allow contains `Bash(*)` — allows all commands
- [HIGH] PC1: `.claude/settings.json` has `Bash(npm *)` in both allow and deny
- [CRITICAL] DF2: `enableAllProjectMcpServers: true` — auto-approves untrusted servers
- [MEDIUM] SB2: No denyRead for `~/.ssh` or credential files
```

## Troubleshooting

- **Settings not found**: Claude Code reads settings from multiple locations. Use `--all` to scan all.
- **Deny not working**: Check if managed settings override. Managed settings cannot be overridden.
- **WebFetch fails**: Evaluate using the embedded rules (Phase 2 tables).
