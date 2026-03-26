import { existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { join } from "node:path";

const STATE_DIR = ".alfred/.state";
const FILE = "last-review.json";

interface ReviewState {
	reviewed_at: string;
}

function filePath(): string {
	return join(process.cwd(), STATE_DIR, FILE);
}

/** Read last review state. Returns null if not found or on error. */
export function readLastReview(): ReviewState | null {
	try {
		const path = filePath();
		if (!existsSync(path)) return null;
		return JSON.parse(readFileSync(path, "utf-8"));
	} catch {
		return null;
	}
}

/** Record that a review was completed. */
export function recordReview(): void {
	try {
		const dir = join(process.cwd(), STATE_DIR);
		if (!existsSync(dir)) mkdirSync(dir, { recursive: true });
		const state: ReviewState = { reviewed_at: new Date().toISOString() };
		writeFileSync(filePath(), JSON.stringify(state, null, 2));
	} catch {
		// fail-open
	}
}

/** Clear review state (e.g. after git commit). */
export function clearReview(): void {
	try {
		const path = filePath();
		if (existsSync(path)) rmSync(path);
	} catch {
		// fail-open
	}
}
