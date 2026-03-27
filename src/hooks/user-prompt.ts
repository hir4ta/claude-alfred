import type { HookEvent } from "../types.ts";
import { respond } from "./respond.ts";

const SHORT_THRESHOLD = 200;
const LARGE_TASK_ADVISORY = 400;
const FULL_TEMPLATE_THRESHOLD = 500;
const LARGE_TASK_THRESHOLD = 800;

const COMPACT_TEMPLATE = `Structure your plan:

## Context
Why this change is needed.

## Tasks
Each task should be focused (1-2 files) with a Verify field.

### Task N: <name> [pending]
- **File**: <path>
- **Change**: <what to do>
- **Verify**: <test file : test function>

## Success Criteria
Concrete, testable conditions that define "done" for this plan.
- [ ] \`<specific command>\` — expected outcome

Update status to [done] as you complete each task.`;

const FULL_TEMPLATE = `${COMPACT_TEMPLATE}

Note: Independent review (/alfred:review) is automatically required before each commit. You don't need to plan for it — it's enforced by the harness.`;

/** UserPromptSubmit: dynamic Plan template injection + large task detection */
export default async function userPrompt(ev: HookEvent): Promise<void> {
	const prompt = typeof ev.prompt === "string" ? ev.prompt : "";

	if (ev.permission_mode === "plan") {
		if (prompt.length < SHORT_THRESHOLD) return; // short task → no template
		if (prompt.length >= FULL_TEMPLATE_THRESHOLD) {
			respond(FULL_TEMPLATE);
		} else {
			respond(COMPACT_TEMPLATE);
		}
		return;
	}

	if (prompt.length > LARGE_TASK_THRESHOLD) {
		respond(
			"Large task detected. Consider using Plan mode (Shift+Tab twice) to break it into small, verified tasks.",
		);
	} else if (prompt.length > LARGE_TASK_ADVISORY) {
		respond(
			"This looks like a large task. Consider using Plan mode (Shift+Tab twice) to break it into small, verified tasks before implementing.",
		);
	}
}
