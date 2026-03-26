import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { getCommitStats } from "./gate-history.ts";

const STATE_DIR = ".alfred/.state";
const PACE_FILE = "session-pace.json";

// Default thresholds (used when no commit history available)
const DEFAULT_RED_MINUTES = 35;
const DEFAULT_FILES = 5;

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

/** Get adaptive red zone threshold in minutes. Uses commit history if available. */
export function getRedThreshold(): number {
	try {
		const stats = getCommitStats();
		if (stats && stats.count >= 3) {
			// Red = 2x average commit interval (min 10 min, max 60 min)
			return Math.max(10, Math.min(60, stats.avgMinutes * 2));
		}
	} catch {
		// fail-open
	}
	return DEFAULT_RED_MINUTES;
}

/** Check if pace is in red zone. Uses adaptive threshold when history available. */
export function isPaceRed(state: PaceState | null): boolean {
	if (!state) return false;
	const elapsed = Date.now() - new Date(state.last_commit_at).getTime();
	const minutes = elapsed / 60_000;
	const threshold = getRedThreshold();
	return minutes >= threshold && state.changed_files >= DEFAULT_FILES;
}
