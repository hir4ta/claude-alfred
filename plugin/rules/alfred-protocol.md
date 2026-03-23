---
paths:
  - ".alfred/**"
---

# Alfred Protocol — Spec-Driven Development

When a `.alfred/specs/` directory exists in the project, follow this protocol strictly.

## Session Start

- Call `dossier action=status` to check for active tasks
- If active, read tasks.md to understand current progress and next steps

## Starting New Work

- Before ANY implementation, call `dossier action=init` to create a spec
- Fill in requirements.md and design.md through conversation with the user
- Never skip spec creation, even for "small" changes

## During Implementation — Task Tracking

- After completing each task, explicitly call `dossier action=check task_id="T-X.Y"`
- Do NOT rely solely on auto-detection — always confirm task completion explicitly
- Record decisions via `ledger action=save sub_type=decision` as they happen

## Wave Completion — Mandatory Review Gate

When all tasks in a Wave are done:

1. **Commit** — Create a git commit with Wave number in message
2. **Self-review** — Run review (delegate to `alfred:code-reviewer` agent or `/alfred:inspect`)
3. **Fix** — If Critical/High findings, fix them before proceeding
4. **Gate clear** — Call `dossier action=gate sub_action=clear reason="<review summary>"` (reason must include: review method, findings count, fix summary, 30+ chars)
5. **Knowledge** — Save learnings via `ledger action=save` (pattern/decision/rule). If nothing to save, state why
6. **Next Wave** — Proceed immediately. Do NOT stop and wait for user input

## Task Lifecycle

- **active**: Being worked on
- **review**: Wave completed, self-review required (Edit/Write blocked until gate cleared)
- **done/completed**: Finished — spec files preserved
- **deferred**: Paused (`dossier action=defer`)
- **cancelled**: Abandoned

## Completing a Spec

- Call `dossier action=complete` to close the spec
- This preserves spec files for future reference
- Prefer complete over delete — completed specs serve as searchable knowledge

## Compact / Session Recovery

After compact or new session, call `dossier action=status` then read:
1. tasks.md (progress + next steps)
2. requirements.md (what am I building?)
3. design.md (how?)
