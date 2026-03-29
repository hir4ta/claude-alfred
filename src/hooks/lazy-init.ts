import { existsSync, mkdirSync, readdirSync, statSync, unlinkSync } from "node:fs";
import { join } from "node:path";
import { writePendingFixes } from "../state/pending-fixes.ts";

const STALE_MS = 24 * 60 * 60 * 1000; // 24 hours
const SCOPED_FILE_RE = /^(session-state|pending-fixes)-.+\.json$/;

let _initialized = false;

/**
 * Lazy initialization: replaces SessionStart hook.
 * Called at the start of every dispatch(), idempotent.
 *
 * - Ensures .qult/.state/ exists
 * - Cleans up stale session-scoped files (>24h)
 * - Clears pending-fixes for fresh session start
 */
export function lazyInit(): void {
	if (_initialized) return;
	_initialized = true;

	try {
		const stateDir = join(process.cwd(), ".qult", ".state");
		if (!existsSync(stateDir)) {
			mkdirSync(stateDir, { recursive: true });
		}
		cleanupStaleScopedFiles(stateDir);
		writePendingFixes([]);
	} catch {
		/* fail-open */
	}
}

/** Remove session-scoped state files older than 24h. */
function cleanupStaleScopedFiles(stateDir: string): void {
	try {
		const now = Date.now();
		for (const file of readdirSync(stateDir)) {
			if (!SCOPED_FILE_RE.test(file)) continue;
			const filePath = join(stateDir, file);
			const age = now - statSync(filePath).mtimeMs;
			if (age > STALE_MS) {
				unlinkSync(filePath);
			}
		}
	} catch {
		/* fail-open */
	}
}

/** Reset for testing. */
export function resetLazyInit(): void {
	_initialized = false;
}
