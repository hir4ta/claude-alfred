import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import type { PendingFix } from "../types.ts";

const STATE_DIR = ".alfred/.state";
const FIXES_FILE = "pending-fixes.json";

function stateDir(): string {
	return join(process.cwd(), STATE_DIR);
}

function fixesPath(): string {
	return join(stateDir(), FIXES_FILE);
}

/** Read current pending fixes. Returns empty array on any error (fail-open). */
export function readPendingFixes(): PendingFix[] {
	try {
		const path = fixesPath();
		if (!existsSync(path)) return [];
		const raw = readFileSync(path, "utf-8");
		return JSON.parse(raw);
	} catch {
		return [];
	}
}

/** Write pending fixes to state file. */
export function writePendingFixes(fixes: PendingFix[]): void {
	try {
		const dir = stateDir();
		if (!existsSync(dir)) mkdirSync(dir, { recursive: true });
		writeFileSync(fixesPath(), JSON.stringify(fixes, null, 2));
	} catch {
		// fail-open
	}
}

/** Remove all fixes for a specific file. */
export function clearFixesForFile(file: string): void {
	const fixes = readPendingFixes();
	const filtered = fixes.filter((f) => f.file !== file);
	writePendingFixes(filtered);
}
