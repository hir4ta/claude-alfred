import { readPace } from "../state/pace.ts";
import { readPendingFixes } from "../state/pending-fixes.ts";
import type { HookEvent, HookResponse } from "../types.ts";

const PACE_YELLOW_MINUTES = 20;

/** Stop: block if pending-fixes, warn on pace */
export default async function stop(ev: HookEvent): Promise<void> {
	// Prevent infinite loop: if stop_hook_active, we're already in a forced continue
	if (ev.stop_hook_active) return;

	// Block if pending fixes remain
	const fixes = readPendingFixes();
	if (fixes.length > 0) {
		const fileList = fixes.map((f) => `  ${f.file}`).join("\n");
		block(`Pending lint/type errors remain. Fix these before completing:\n${fileList}`);
		return;
	}

	// Pace warning (soft — additionalContext only, no block)
	const pace = readPace();
	if (pace) {
		const elapsed = (Date.now() - new Date(pace.last_commit_at).getTime()) / 60_000;
		if (elapsed >= PACE_YELLOW_MINUTES) {
			respond(
				`${Math.round(elapsed)} minutes since last commit. Consider committing your current progress.`,
			);
		}
	}
}

function block(reason: string): void {
	const response: HookResponse = {
		hookSpecificOutput: {
			decision: "block",
			reason,
		},
	};
	process.stdout.write(JSON.stringify(response));
	process.exit(2);
}

function respond(context: string): void {
	const response: HookResponse = {
		hookSpecificOutput: {
			additionalContext: context,
		},
	};
	process.stdout.write(JSON.stringify(response));
}
