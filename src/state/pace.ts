import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";

const STATE_DIR = ".alfred/.state";
const PACE_FILE = "session-pace.json";

interface PaceState {
	last_commit_at: string;
	changed_files: number;
	tool_calls: number;
}

function pacePath(): string {
	return join(process.cwd(), STATE_DIR, PACE_FILE);
}

/** Read pace state. Returns null on error (fail-open). */
export function readPace(): PaceState | null {
	try {
		const path = pacePath();
		if (!existsSync(path)) return null;
		return JSON.parse(readFileSync(path, "utf-8"));
	} catch {
		return null;
	}
}

/** Write pace state. */
export function writePace(state: PaceState): void {
	try {
		const dir = join(process.cwd(), STATE_DIR);
		if (!existsSync(dir)) mkdirSync(dir, { recursive: true });
		writeFileSync(pacePath(), JSON.stringify(state, null, 2));
	} catch {
		// fail-open
	}
}

/** Check if pace is in red zone (35+ min without commit on many files). */
export function isPaceRed(state: PaceState | null): boolean {
	if (!state) return false;
	const elapsed = Date.now() - new Date(state.last_commit_at).getTime();
	const minutes = elapsed / 60_000;
	return minutes >= 35 && state.changed_files >= 5;
}
