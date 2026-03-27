import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";

const STATE_DIR = ".alfred/.state";
const FILE = "metrics.json";
const MAX_ENTRIES = 50;

interface MetricEntry {
	action: string; // "event:type" e.g. "pre-tool:deny"
	reason: string;
	at: string;
}

function filePath(): string {
	return join(process.cwd(), STATE_DIR, FILE);
}

function readState(): MetricEntry[] {
	try {
		const path = filePath();
		if (!existsSync(path)) return [];
		return JSON.parse(readFileSync(path, "utf-8"));
	} catch {
		return [];
	}
}

function writeState(entries: MetricEntry[]): void {
	try {
		const dir = join(process.cwd(), STATE_DIR);
		if (!existsSync(dir)) mkdirSync(dir, { recursive: true });
		writeFileSync(filePath(), JSON.stringify(entries, null, 2));
	} catch {
		// fail-open
	}
}

/** Record a DENY/block/respond action with event name and reason. */
export function recordAction(
	event: string,
	type: "deny" | "block" | "respond",
	reason: string,
): void {
	const entries = readState();
	entries.push({ action: `${event}:${type}`, reason, at: new Date().toISOString() });
	if (entries.length > MAX_ENTRIES) {
		entries.splice(0, entries.length - MAX_ENTRIES);
	}
	writeState(entries);
}

/** Record a DENY resolution (pending-fix cleared after DENY). */
export function recordResolution(event: string, reason: string): void {
	const entries = readState();
	entries.push({ action: `${event}:resolution`, reason, at: new Date().toISOString() });
	if (entries.length > MAX_ENTRIES) {
		entries.splice(0, entries.length - MAX_ENTRIES);
	}
	writeState(entries);
}

/** Read all recorded metrics (up to 50). */
export function readMetrics(): MetricEntry[] {
	return readState();
}

/** Get summary: counts by action type + top reasons. */
export function getMetricsSummary(): {
	deny: number;
	block: number;
	respond: number;
	resolution: number;
	denyResolutionRate: number;
	topReasons: { reason: string; count: number }[];
} {
	const entries = readState();
	let deny = 0;
	let block = 0;
	let respond = 0;
	let resolution = 0;
	const reasonCounts = new Map<string, number>();

	for (const e of entries) {
		if (e.action.endsWith(":deny")) deny++;
		else if (e.action.endsWith(":block")) block++;
		else if (e.action.endsWith(":respond")) respond++;
		else if (e.action.endsWith(":resolution")) resolution++;

		reasonCounts.set(e.reason, (reasonCounts.get(e.reason) ?? 0) + 1);
	}

	const topReasons = [...reasonCounts.entries()]
		.map(([reason, count]) => ({ reason, count }))
		.sort((a, b) => b.count - a.count)
		.slice(0, 5);

	return {
		deny,
		block,
		respond,
		resolution,
		denyResolutionRate: deny > 0 ? Math.min(100, Math.round((resolution / deny) * 100)) : 0,
		topReasons,
	};
}
