---
name: alfred-setup
description: >
  Interactive wizard to set up Claude Code best practices for your project.
  Creates CLAUDE.md, hooks, skills, and other configurations step by step.
user-invocable: true
allowed-tools: Read, Write, Edit, Glob, AskUserQuestion, mcp__claude-alfred__knowledge
---

Project setup wizard for Claude Code.

## Steps

1. Check current setup (same as alfred-audit step 1-2)
2. For each missing configuration, ask the user if they want to set it up:
   - AskUserQuestion with options for each feature

3. If CLAUDE.md is missing or minimal:
   - Detect project stack (look at package.json, go.mod, Cargo.toml, etc.)
   - Call knowledge for CLAUDE.md best practices
   - Generate a template CLAUDE.md with Commands and Rules sections
   - Write it with user approval

4. If hooks are not configured:
   - Explain what hooks can do
   - Offer common hook configurations (pre-commit lint, test on edit)

5. If no skills exist:
   - Explain what skills are
   - Offer to create a starter skill template

6. If no rules exist:
   - Explain file-matched rules
   - Offer to create rules for the project's main language

## Output

For each step, show what was created/modified and why.
At the end, summarize the new setup and suggest next steps.
