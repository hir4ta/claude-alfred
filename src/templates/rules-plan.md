# Plan Rules (qult)

## Plan Structure

When writing a plan, use this structure:

```
## Context
Why this change is needed.

## Tasks
### Task N: <name> [pending]
- **File**: <path> (include consumer files: tests, docs, registries)
- **Change**: <what to do>
- **Boundary**: <what NOT to change>
- **Verify**: <test file : test function>

## Success Criteria
- [ ] `<specific command>` -- expected outcome
```

Update task status to [done] as you complete each task.

## Task Registration

When transitioning from Plan mode to implementation:
1. Read the approved plan file
2. Create a task (TaskCreate) for each `### Task N:` entry in the plan
3. Update each task status as you work (in_progress when starting, completed when done)
