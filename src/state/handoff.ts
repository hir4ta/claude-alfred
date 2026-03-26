import { existsSync, mkdirSync, readFileSync, unlinkSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import type { HandoffState } from "../types.ts";

const STATE_DIR = ".alfred/.state";
const HANDOFF_FILE = "handoff.json";

function handoffPath(): string {
	return join(process.cwd(), STATE_DIR, HANDOFF_FILE);
}

/** Read handoff state from previous compaction. Returns null if none. */
export function readHandoff(): HandoffState | null {
	try {
		const path = handoffPath();
		if (!existsSync(path)) return null;
		return JSON.parse(readFileSync(path, "utf-8"));
	} catch {
		return null;
	}
}

/** Save handoff state for next session/compaction. */
export function writeHandoff(state: HandoffState): void {
	try {
		const dir = join(process.cwd(), STATE_DIR);
		if (!existsSync(dir)) mkdirSync(dir, { recursive: true });
		writeFileSync(handoffPath(), JSON.stringify(state, null, 2));
	} catch {
		// fail-open
	}
}

/** Clear handoff after it's been consumed. */
export function clearHandoff(): void {
	try {
		const path = handoffPath();
		if (existsSync(path)) unlinkSync(path);
	} catch {
		// fail-open
	}
}
