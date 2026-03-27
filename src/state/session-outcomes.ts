import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";

const STATE_DIR = ".alfred/.state";
const FILE = "session-outcomes.json";
const MAX_ENTRIES = 50;

export interface SessionOutcome {
	session_id: string;
	started_at: string;
	ended_at: string;
	starting_pending: number;
	ending_pending: number;
	deny_count: number;
	block_count: number;
	respond_count: number;
	commits: number;
	clean_exit: boolean;
}

function filePath(): string {
	return join(process.cwd(), STATE_DIR, FILE);
}

function readState(): SessionOutcome[] {
	try {
		const path = filePath();
		if (!existsSync(path)) return [];
		return JSON.parse(readFileSync(path, "utf-8"));
	} catch {
		return [];
	}
}

function writeState(entries: SessionOutcome[]): void {
	try {
		const dir = join(process.cwd(), STATE_DIR);
		if (!existsSync(dir)) mkdirSync(dir, { recursive: true });
		writeFileSync(filePath(), JSON.stringify(entries, null, 2));
	} catch {
		// fail-open
	}
}

/** Record a completed session outcome. */
export function recordOutcome(outcome: SessionOutcome): void {
	const entries = readState();
	entries.push(outcome);
	if (entries.length > MAX_ENTRIES) {
		entries.splice(0, entries.length - MAX_ENTRIES);
	}
	writeState(entries);
}

/** Read all recorded outcomes (up to 50). */
export function readOutcomes(): SessionOutcome[] {
	return readState();
}

/** Get summary statistics from outcomes. */
export function getOutcomeSummary(): {
	total: number;
	clean: number;
	cleanRate: number;
	avgStartingPending: number;
	avgEndingPending: number;
} | null {
	const entries = readState();
	if (entries.length === 0) return null;

	const clean = entries.filter((e) => e.clean_exit).length;
	const totalStarting = entries.reduce((sum, e) => sum + e.starting_pending, 0);
	const totalEnding = entries.reduce((sum, e) => sum + e.ending_pending, 0);

	return {
		total: entries.length,
		clean,
		cleanRate: Math.round((clean / entries.length) * 100),
		avgStartingPending: Math.round((totalStarting / entries.length) * 10) / 10,
		avgEndingPending: Math.round((totalEnding / entries.length) * 10) / 10,
	};
}
