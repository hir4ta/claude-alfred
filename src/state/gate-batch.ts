import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";

const STATE_DIR = ".alfred/.state";
const BATCH_FILE = "gate-batch.json";

interface BatchEntry {
	session_id: string;
	ran_at: string;
}

type BatchState = Record<string, BatchEntry>;

function batchPath(): string {
	return join(process.cwd(), STATE_DIR, BATCH_FILE);
}

/** Read batch state. Returns null on error (fail-open). */
export function readBatch(): BatchState | null {
	try {
		const path = batchPath();
		if (!existsSync(path)) return null;
		return JSON.parse(readFileSync(path, "utf-8"));
	} catch {
		return null;
	}
}

function writeBatch(state: BatchState): void {
	try {
		const dir = join(process.cwd(), STATE_DIR);
		if (!existsSync(dir)) mkdirSync(dir, { recursive: true });
		writeFileSync(batchPath(), JSON.stringify(state, null, 2));
	} catch {
		// fail-open
	}
}

/** Check if a gate should be skipped (already ran in this session). */
export function shouldSkip(gateName: string, sessionId: string): boolean {
	const batch = readBatch();
	if (!batch) return false;
	const entry = batch[gateName];
	if (!entry) return false;
	return entry.session_id === sessionId;
}

/** Record that a gate was executed in this session. */
export function markRan(gateName: string, sessionId: string): void {
	const batch = readBatch() ?? {};
	batch[gateName] = { session_id: sessionId, ran_at: new Date().toISOString() };
	writeBatch(batch);
}

/** Clear all batch state (e.g. after git commit). */
export function clearBatch(): void {
	writeBatch({});
}
