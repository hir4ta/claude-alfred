import { recordReviewOutcome } from "../state/metrics.ts";
import { getLatestPlanContent } from "../state/plan-status.ts";
import { recordReview } from "../state/session-state.ts";
import type { HookEvent } from "../types.ts";
import { block } from "./respond.ts";

// [severity] file:line pattern or "No issues found"
const SEVERITY_PATTERN = /\[(critical|high|medium|low)\]/;
const FINDING_RE = new RegExp(SEVERITY_PATTERN.source, "i");
const NO_ISSUES_RE = /no issues found/i;
const REVIEW_PASS_RE = /^Review:\s*PASS/im;
const REVIEW_FAIL_RE = /^Review:\s*FAIL/im;
// Score parsing: strict → colon → loose fallback
const SCORE_STRICT_RE = /Score:\s*Correctness=(\d)\s+Design=(\d)\s+Security=(\d)/i;
const SCORE_COLON_RE = /Correctness[=:]\s*(\d).*?Design[=:]\s*(\d).*?Security[=:]\s*(\d)/i;
const SCORE_LOOSE_RE = /Score:.*?[=:]\s*(\d).*?[=:]\s*(\d).*?[=:]\s*(\d)/i;

export interface ReviewScores {
	correctness: number;
	design: number;
	security: number;
}

interface FindingSummary {
	total: number;
	critical: number;
	high: number;
	medium: number;
	low: number;
}

/** Parse reviewer scores with graduated fallback (strict → colon → loose). */
export function parseScores(output: string): ReviewScores | null {
	for (const re of [SCORE_STRICT_RE, SCORE_COLON_RE, SCORE_LOOSE_RE]) {
		const m = re.exec(output);
		if (m) {
			return {
				correctness: Number.parseInt(m[1]!, 10),
				design: Number.parseInt(m[2]!, 10),
				security: Number.parseInt(m[3]!, 10),
			};
		}
	}
	return null;
}

/** Parse reviewer output for severity-tagged findings. */
export function parseFindings(output: string): FindingSummary {
	const summary: FindingSummary = { total: 0, critical: 0, high: 0, medium: 0, low: 0 };
	const matches = output.matchAll(new RegExp(SEVERITY_PATTERN.source, "gi"));
	for (const m of matches) {
		const sev = m[1]!.toLowerCase() as keyof Omit<FindingSummary, "total">;
		summary[sev]++;
		summary.total++;
	}
	return summary;
}

/** SubagentStop: verify subagent output quality */
export default async function subagentStop(ev: HookEvent): Promise<void> {
	if (ev.stop_hook_active) return;

	const agentType = ev.agent_type;
	const output = ev.last_assistant_message;

	// fail-open: no agent_type or no output → allow
	if (!agentType || !output) return;

	if (agentType === "qult-reviewer") {
		const passed = REVIEW_PASS_RE.test(output);
		const failed = REVIEW_FAIL_RE.test(output);
		validateReviewer(output);
		// Record outcome metric with finding details + scores if verdict is present
		if (passed || failed) {
			try {
				const findings = parseFindings(output);
				const scores = parseScores(output);
				const detail: Record<string, number> = { ...findings };
				if (scores) {
					detail.correctness = scores.correctness;
					detail.design = scores.design;
					detail.security = scores.security;
				}
				recordReviewOutcome(passed, detail);
			} catch {
				/* fail-open */
			}
		}
		// Only clear the review gate on PASS. FAIL requires fixes + re-review.
		if (failed) {
			block("Review: FAIL. Fix the issues found by the reviewer and run /qult:review again.");
		}
		recordReview();
	} else if (agentType === "Plan") {
		validatePlan();
	}
	// Unknown agent_type → allow (fail-open)
}

function validateReviewer(output: string): void {
	const hasVerdict = REVIEW_PASS_RE.test(output) || REVIEW_FAIL_RE.test(output);
	const hasFindings = FINDING_RE.test(output) || NO_ISSUES_RE.test(output);
	const hasScore = parseScores(output) !== null;

	// Accept if: findings present (backward compat) OR verdict + score
	if (hasFindings) return;
	if (hasVerdict && hasScore) return;

	block(
		"Reviewer output must include: (1) 'Review: PASS' or 'Review: FAIL', (2) 'Score: Correctness=N Design=N Security=N', and (3) findings ([severity] file:line) or 'No issues found'. Rerun the review.",
	);
}

function validatePlan(): void {
	const content = getLatestPlanContent();
	if (!content) return; // fail-open: no plan file found

	const hasTasks = content.includes("## Tasks");
	if (hasTasks) return;

	// Review Gates no longer required — review is enforced mechanically by stop.ts and pre-tool.ts
	block("Plan is missing required section: ## Tasks. Add it before exiting.");
}
