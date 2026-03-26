import type { HookEvent } from "../types.ts";
import { block } from "./respond.ts";

// [severity] file:line pattern or "No issues found"
const FINDING_RE = /\[(critical|high|medium|low)\]/i;
const NO_ISSUES_RE = /no issues found/i;

/** SubagentStop: verify subagent output quality */
export default async function subagentStop(ev: HookEvent): Promise<void> {
	if (ev.stop_hook_active) return;

	const agentType = ev.agent_type;
	const output = ev.last_assistant_message;

	// fail-open: no agent_type or no output → allow
	if (!agentType || !output) return;

	if (agentType === "alfred-reviewer") {
		validateReviewer(output);
	} else if (agentType === "Plan") {
		validatePlan(output);
	}
	// Unknown agent_type → allow (fail-open)
}

function validateReviewer(output: string): void {
	if (FINDING_RE.test(output) || NO_ISSUES_RE.test(output)) return;
	block(
		"Reviewer output must contain findings ([severity] file:line) or 'No issues found'. Rerun the review with structured output.",
	);
}

function validatePlan(output: string): void {
	const hasTasks = output.includes("## Tasks");
	const hasReview = /review/i.test(output) && /gates?/i.test(output);
	if (hasTasks && hasReview) return;

	const missing: string[] = [];
	if (!hasTasks) missing.push("## Tasks");
	if (!hasReview) missing.push("Review Gates");
	block(`Plan is missing required sections: ${missing.join(", ")}. Add them before exiting.`);
}
