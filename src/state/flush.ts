import { flush as flushGateHistory, resetCache as resetGateHistory } from "./gate-history.ts";
import { flush as flushMetrics, resetCache as resetMetrics } from "./metrics.ts";
import { flush as flushPendingFixes, resetCache as resetPendingFixes } from "./pending-fixes.ts";
import { flush as flushSessionState, resetCache as resetSessionState } from "./session-state.ts";

/** Flush all dirty state caches to disk. Called once at end of hook dispatch. */
export function flushAll(): void {
	try {
		flushSessionState();
	} catch {
		/* fail-open */
	}
	try {
		flushMetrics();
	} catch {
		/* fail-open */
	}
	try {
		flushPendingFixes();
	} catch {
		/* fail-open */
	}
	try {
		flushGateHistory();
	} catch {
		/* fail-open */
	}
}

/** Reset all caches (for tests). */
export function resetAllCaches(): void {
	resetSessionState();
	resetMetrics();
	resetPendingFixes();
	resetGateHistory();
}
