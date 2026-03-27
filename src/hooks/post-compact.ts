import { readPendingFixes } from "../state/pending-fixes.ts";
import { getActivePlan } from "../state/plan-status.ts";
import type { HookEvent } from "../types.ts";

/** PostCompact: remind about pending fixes + plan progress after context compaction */
export default async function postCompact(_ev: HookEvent): Promise<void> {
	const fixes = readPendingFixes();
	if (fixes.length > 0) {
		process.stderr.write(
			`[alfred] WARNING: ${fixes.length} pending lint/type fix(es). Fix them before continuing.\n`,
		);
	}

	// Inject plan progress so Claude knows where it was after compaction
	const plan = getActivePlan();
	if (plan && plan.tasks.length > 0) {
		const done = plan.tasks.filter((t) => t.status === "done");
		const inProgress = plan.tasks.filter((t) => t.status === "in-progress");
		const pending = plan.tasks.filter((t) => t.status === "pending");
		const lines = [`[alfred] Plan progress (${plan.path}):`];
		if (done.length > 0) lines.push(`  Done: ${done.map((t) => t.name).join(", ")}`);
		if (inProgress.length > 0)
			lines.push(`  In progress: ${inProgress.map((t) => t.name).join(", ")}`);
		if (pending.length > 0) lines.push(`  Remaining: ${pending.map((t) => t.name).join(", ")}`);
		process.stderr.write(`${lines.join("\n")}\n`);
	}
}
