# Skill Audit Baseline Checklist

Authoritative sources (fetched at runtime for latest):
- https://code.claude.com/docs/en/skills
- https://code.claude.com/docs/en/best-practices
- https://agentskills.io/specification
- https://resources.anthropic.com/hubfs/The-Complete-Guide-to-Building-Skill-for-Claude.pdf (33p guide)
- https://github.com/anthropics/skills (official examples + quick_validate.py)

## Name Rules
- Required field
- 1-64 characters
- Lowercase letters, numbers, hyphens only
- Must match parent directory name
- No start/end with hyphen
- No consecutive hyphens (`--`)
- No reserved prefixes: "claude", "anthropic"

## Description Rules
- Required field (recommended in Claude Code, required in Agent Skills spec)
- 1-1024 characters (Claude Code truncates beyond 1024)
- Must be a string (not YAML list — known parsing bug)
- Structure: [What it does] + [When to use it] + [Key capabilities]
- Include specific trigger phrases users would actually say
- No XML angle brackets (< >)

### Good Descriptions
```
# Specific + trigger phrases
description: Analyzes Figma design files and generates developer handoff
  documentation. Use when user uploads .fig files, asks for "design specs",
  "component documentation", or "design-to-code handoff".
```

### Bad Descriptions
```
# Too vague
description: Helps with projects.

# Missing triggers
description: Creates sophisticated multi-page documentation systems.

# Too technical
description: Implements the Project entity model with hierarchical relationships.
```

## Frontmatter Security
- No XML angle brackets (< >) in ANY field (injected into system prompt)
- No code execution in YAML
- `allowed-tools` for least privilege

## Structure Rules
- File must be exactly `SKILL.md` (case-sensitive)
- Folder must be kebab-case
- Under 500 lines (move docs to references/)
- Under 5000 words recommended
- No README.md inside skill folder
- Supporting files referenced with relative links

## Progressive Disclosure
- Level 1 (frontmatter): ~100 tokens, loaded at startup
- Level 2 (SKILL.md body): < 5000 tokens, loaded when activated
- Level 3 (references/scripts/assets): loaded on demand

## Description Budget
- All skill descriptions share ~2% of context window (~16,000 chars fallback)
- Override via SLASH_COMMAND_TOOL_CHAR_BUDGET env var
- At 250 chars avg: ~64 skills fit in budget
- Compress to 130 chars to fit 120+ skills
- `disable-model-invocation: true` = 0 budget cost (removed from list)

## String Substitutions
- `$ARGUMENTS`, `$ARGUMENTS[N]`, `$N`
- `${CLAUDE_SESSION_ID}`, `${CLAUDE_SKILL_DIR}`
- `!`command`` for dynamic context injection
