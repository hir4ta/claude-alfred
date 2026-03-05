# Butler Protocol — Autonomous Spec Management

When a `.alfred/specs/` directory exists in the project, follow this protocol:

## Session Start
- Call `butler-status` with project_path to check for an active task
- If active, read the session state to understand current position and next steps
- If session.md mentions "Compact Marker", you are resuming after context compaction — read all spec files to restore full context

## Starting New Work
- Before implementation, call `butler-init` to create a spec
- Fill in requirements.md and design.md through conversation with the user
- Break work into tasks in tasks.md

## During Implementation
Record these autonomously — do not wait for user instruction:

**decisions.md** — When you make or recommend a design choice:
```
## [date] Decision Title
- **Chosen:** what was selected
- **Alternatives:** what was considered
- **Reason:** why this option
```

**knowledge.md** — When you discover something:
```
## Discovery Title
- **Finding:** what you learned
- **Context:** when/where this matters
- **Dead ends:** what didn't work and why (CRITICAL — prevents re-exploration)
```

**tasks.md** — Update checkboxes as you complete work

**session.md** — Update when:
- Starting a new sub-task (Current Position)
- Completing a milestone
- Encountering a blocker (Unresolved Issues)

## Compact/Session Recovery
After compact or new session, butler-status provides session.md.
Read spec files in this order to rebuild context:
1. session.md (where am I?)
2. requirements.md (what am I building?)
3. design.md (how?)
4. tasks.md (what's done/remaining?)
5. decisions.md (why these choices?)
6. knowledge.md (what did I learn?)

## Review
- Before committing, call `butler-review` to check changes against specs and accumulated knowledge
