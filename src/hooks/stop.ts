import type { DirectiveItem } from "./directives.js";
import { emitDirectives } from "./directives.js";
import type { HookEvent } from "./dispatcher.js";
import { hasPendingFixes, readPendingFixes, formatPendingFixes } from "./pending-fixes.js";
import { writeStateJSON } from "./state.js";
import { openDefaultCached } from "../store/index.js";
import { resolveOrRegisterProject } from "../store/project.js";
import { getSessionSummary, calculateQualityScore } from "../store/quality-events.js";

/**
 * Stop handler: soft reminders + final quality summary save.
 */
export async function stop(ev: HookEvent): Promise<void> {
	if (!ev.cwd) return;

	const items: DirectiveItem[] = [];

	// 1. Check pending-fixes → WARNING if unresolved
	if (hasPendingFixes(ev.cwd)) {
		const fixes = readPendingFixes(ev.cwd);
		const formatted = formatPendingFixes(fixes);
		items.push({
			level: "WARNING",
			message: `Unresolved lint/type errors remain:\n${formatted}`,
		});
	}

	// 2. Save final quality summary
	saveFinalQualitySummary(ev.cwd);

	// TODO (Phase 4): git diff → untested file check

	emitDirectives("Stop", items);
}

function saveFinalQualitySummary(cwd: string): void {
	try {
		const store = openDefaultCached();
		const project = resolveOrRegisterProject(store, cwd);
		const sessionId = `session-${Date.now()}`;

		const summary = getSessionSummary(store, sessionId);
		const score = calculateQualityScore(store, sessionId);

		writeStateJSON(cwd, "session-summary.json", {
			...summary,
			score: score.sessionScore,
			saved_at: new Date().toISOString(),
		});
	} catch {
		/* fail-open */
	}
}
