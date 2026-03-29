import { existsSync, mkdirSync, readdirSync, statSync, unlinkSync } from "node:fs";
import { join } from "node:path";
import { writePendingFixes } from "../state/pending-fixes.ts";
import type { HookEvent } from "../types.ts";

const STALE_MS = 24 * 60 * 60 * 1000; // 24 hours
const SCOPED_FILE_RE = /^(session-state|pending-fixes)-.+\.json$/;

/** SessionStart: ensure .qult/ exists, prompt gate detection if empty */
export default async function sessionStart(_ev: HookEvent): Promise<void> {
	const qultDir = join(process.cwd(), ".qult");
	const stateDir = join(qultDir, ".state");
	if (!existsSync(stateDir)) {
		mkdirSync(stateDir, { recursive: true });
	}

	// Clean up stale session-scoped state files (>24h old)
	cleanupStaleScopedFiles(stateDir);

	// Clear this session's pending-fixes. Gates will re-detect on edit.
	writePendingFixes([]);

	// Gate detection prompt moved to MCP server instructions + rules
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
