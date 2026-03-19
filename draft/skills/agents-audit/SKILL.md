---
name: agents-audit
description: >
  Audit .claude/agents/*.md files for correctness and security. Validates
  required fields (name, description), tool restrictions, model validity,
  bypassPermissions warnings, description quality (3+ usage examples
  recommended), and YAML syntax. Checks both project (.claude/agents/) and
  user (~/.claude/agents/) agent definitions. Use when creating agents,
  auditing agent security, or debugging agent delegation failures. NOT for
  skill review (use /skill-audit). NOT for hooks (use /hooks-audit).
user-invocable: true
argument-hint: "[agents-dir or file] [--all]"
allowed-tools: Read, Glob, Grep, WebFetch
context: fork
model: sonnet
---

# /agents-audit — Custom Agents Auditor

## Phase 0: Refresh Knowledge

1. **WebFetch** `https://code.claude.com/docs/en/sub-agents` with prompt:
   "Extract custom agent frontmatter fields (required and optional), name
   format, tool allowlisting, model options, permissionMode values,
   bypassPermissions implications, and best practices."

## Phase 1: Discover Agents

- Path given → audit that file or directory
- `--all` → scan: `.claude/agents/`, `~/.claude/agents/`
- No arguments → `.claude/agents/`

## Phase 2: Evaluate Each Agent

### Required Fields (CRITICAL)

| # | Check | Rule |
|---|---|---|
| R1 | name field present | Required |
| R2 | name is lowercase-hyphen | No spaces, capitals, underscores |
| R3 | description field present | Required — Claude uses this to decide when to delegate |
| R4 | YAML frontmatter valid | Opens and closes with `---` |

### Description Quality (HIGH)

| # | Check | Rule |
|---|---|---|
| DQ1 | Description explains WHEN to delegate | Not just what the agent does |
| DQ2 | 3+ usage examples | Help Claude recognize delegation opportunities |
| DQ3 | Action-oriented language | "Use when..." not "This agent..." |

### Tool Restrictions (HIGH)

| # | Check | Rule |
|---|---|---|
| T1 | tools field specified | Omitting = inherits ALL tools (warn) |
| T2 | Least privilege | Only tools the agent actually needs |
| T3 | MCP tools use full names | `mcp__server__tool` format |
| T4 | Agent() syntax correct | `Agent(name)` for subagent restriction |

### Model & Permissions (HIGH)

| # | Check | Rule |
|---|---|---|
| MP1 | model is valid | sonnet, opus, haiku, full model ID, or inherit |
| MP2 | bypassPermissions = warning | CRITICAL security risk if true |
| MP3 | permissionMode documented | If set, explain why in description |
| MP4 | maxTurns reasonable | Prevent infinite loops |

### Security (CRITICAL)

| # | Check | Rule |
|---|---|---|
| S1 | No bypassPermissions: true | Unless explicitly justified |
| S2 | No secrets in content | API keys, tokens |

## Phase 3: Score & Report

```
## Agents Audit: .claude/agents/

**Agents: {N}** | **Score: {X}/100**

| Agent | Model | Tools | Issues |
|-------|-------|-------|--------|
| explorer | haiku | Read, Glob, Grep | OK |
| deployer | sonnet | ALL (inherited) | [HIGH] T1: No tools restriction |
```

## Example

User: `/agents-audit`

```
## Agents Audit: .claude/agents/

**Agents: 3** | **Score: 75/100**

| Agent | Model | Issues |
|-------|-------|--------|
| explorer | haiku | OK |
| reviewer | sonnet | [MEDIUM] DQ2: Only 1 usage example (recommend 3+) |
| admin | opus | [CRITICAL] S1: bypassPermissions: true |
```

## Troubleshooting

- **Agent not loading**: Check filename matches name field. File must be `.md` in `.claude/agents/`.
- **Agent inherits too many tools**: Add explicit `tools` field to restrict.
- **WebFetch fails**: Evaluate using the embedded checklist (Phase 2 tables).
