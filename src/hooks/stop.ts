import type { HookEvent } from "./dispatcher.js";
import { isGateActive } from "./review-gate.js";
import {
	blockStop,
	countUncheckedNextSteps,
	hasUncheckedSelfReview,
	tryReadActiveSpec,
} from "./spec-guard.js";

/**
 * Stop handler: block Claude from stopping when review gate is active or spec has incomplete items.
 * DEC-4: stop_hook_active=true → always allow (infinite loop prevention).
 */
export async function stop(ev: HookEvent): Promise<void> {
	// DEC-4: Prevent infinite loop — if Stop already triggered once, let Claude stop.
	// This intentionally overrides both review-gate and Next Steps checks.
	if (ev.stop_hook_active) return;

	// Review gate check — blocks stop when spec/wave review is pending.
	const gate = isGateActive(ev.cwd);
	if (gate) {
		const gateLabel =
			gate.gate === "wave-review" ? `Wave ${gate.wave ?? "?"} review` : "Spec self-review";
		blockStop(
			`${gateLabel} not completed for spec '${gate.slug}'. Run review, then: dossier action=gate sub_action=clear reason="<review summary>"`,
		);
		return;
	}

	const spec = tryReadActiveSpec(ev.cwd);
	// No spec or already completed → allow stop.
	if (!spec || spec.status === "completed") return;

	const issues: string[] = [];

	// Unchecked Next Steps.
	const unchecked = countUncheckedNextSteps(ev.cwd, spec.slug);
	if (unchecked > 0) {
		issues.push(`${unchecked} unchecked Next Steps remaining`);
	}

	// Wave self-review not done.
	const selfReviewPending = hasUncheckedSelfReview(ev.cwd, spec.slug);
	if (selfReviewPending) {
		issues.push(
			"Wave self-review not completed — run /alfred:inspect or delegate to alfred:code-reviewer",
		);
	}

	// Spec not completed — only show when all other items are done.
	if (issues.length === 0) {
		issues.push("All tasks done. Run `dossier action=complete` to close the spec");
	}

	if (issues.length > 0) {
		blockStop(
			`Active spec '${spec.slug}' incomplete: ${issues.join("; ")}. Complete all items before stopping.`,
		);
	}
}
