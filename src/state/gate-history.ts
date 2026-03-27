import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";
import { atomicWriteJson } from "./atomic-write.ts";

const STATE_DIR = ".qult/.state";
const FILE = "gate-history.json";
const MAX_ENTRIES = 200;

// Process-scoped cache
let _cache: HistoryState | null = null;
let _dirty = false;

interface GateEntry {
	gate: string;
	passed: boolean;
	error?: string;
	at: string;
	duration_ms?: number;
}

interface CommitEntry {
	at: string;
}

interface HistoryState {
	gates: GateEntry[];
	commits: CommitEntry[];
}

function filePath(): string {
	return join(process.cwd(), STATE_DIR, FILE);
}

function readHistory(): HistoryState {
	if (_cache) return _cache;
	try {
		const path = filePath();
		if (!existsSync(path)) {
			_cache = { gates: [], commits: [] };
			return _cache;
		}
		_cache = JSON.parse(readFileSync(path, "utf-8"));
		return _cache!;
	} catch {
		_cache = { gates: [], commits: [] };
		return _cache;
	}
}

function writeHistory(state: HistoryState): void {
	_cache = state;
	_dirty = true;
}

/** Flush cached history to disk if dirty. */
export function flush(): void {
	if (!_dirty || !_cache) return;
	try {
		atomicWriteJson(filePath(), _cache);
	} catch {
		// fail-open
	}
	_dirty = false;
}

/** Reset cache (for tests). */
export function resetCache(): void {
	_cache = null;
	_dirty = false;
}

/** Record a gate execution result. */
export function recordGateResult(
	gate: string,
	passed: boolean,
	error?: string,
	duration_ms?: number,
): void {
	const history = readHistory();
	const entry: GateEntry = { gate, passed, error, at: new Date().toISOString() };
	if (duration_ms !== undefined) entry.duration_ms = duration_ms;
	history.gates.push(entry);
	if (history.gates.length > MAX_ENTRIES) {
		history.gates = history.gates.slice(-MAX_ENTRIES);
	}
	writeHistory(history);
}

/** Get top N most frequent gate errors. */
export function getTopErrors(n: number): { gate: string; error: string; count: number }[] {
	const history = readHistory();
	const errors = history.gates.filter((e) => !e.passed && e.error);

	const counts = new Map<string, { gate: string; error: string; count: number }>();
	for (const e of errors) {
		const key = `${e.gate}:${e.error}`;
		const existing = counts.get(key);
		if (existing) {
			existing.count++;
		} else {
			counts.set(key, { gate: e.gate, error: e.error!, count: 1 });
		}
	}

	return [...counts.values()].sort((a, b) => b.count - a.count).slice(0, n);
}

/** Record a git commit timestamp. */
export function recordCommit(): void {
	const history = readHistory();
	history.commits.push({ at: new Date().toISOString() });
	if (history.commits.length > MAX_ENTRIES) {
		history.commits = history.commits.slice(-MAX_ENTRIES);
	}
	writeHistory(history);
}

/** Get commit interval statistics. Returns null if < 2 commits. */
export function getCommitStats(): {
	avgMinutes: number;
	medianMinutes: number;
	minMinutes: number;
	maxMinutes: number;
	count: number;
} | null {
	const history = readHistory();
	if (history.commits.length < 2) return null;

	const times = history.commits.map((c) => new Date(c.at).getTime()).sort((a, b) => a - b);
	const intervals: number[] = [];
	for (let i = 1; i < times.length; i++) {
		intervals.push(times[i]! - times[i - 1]!);
	}
	intervals.sort((a, b) => a - b);

	const sum = intervals.reduce((a, b) => a + b, 0);
	const mid = Math.floor(intervals.length / 2);
	const median =
		intervals.length % 2 === 0 ? (intervals[mid - 1]! + intervals[mid]!) / 2 : intervals[mid]!;

	return {
		avgMinutes: Math.round(sum / intervals.length / 60_000),
		medianMinutes: Math.round(median / 60_000),
		minMinutes: Math.round(intervals[0]! / 60_000),
		maxMinutes: Math.round(intervals[intervals.length - 1]! / 60_000),
		count: history.commits.length,
	};
}

/** Get pass rate per gate name. */
export function getGatePassRates(): { gate: string; passRate: number; total: number }[] {
	const history = readHistory();
	const stats = new Map<string, { pass: number; total: number }>();
	for (const e of history.gates) {
		const s = stats.get(e.gate) ?? { pass: 0, total: 0 };
		s.total++;
		if (e.passed) s.pass++;
		stats.set(e.gate, s);
	}
	return [...stats.entries()]
		.map(([gate, s]) => ({ gate, passRate: Math.round((s.pass / s.total) * 100), total: s.total }))
		.sort((a, b) => a.passRate - b.passRate);
}

/** Get average gate execution duration per gate name. */
export function getAvgGateDuration(): { gate: string; avgMs: number; count: number }[] {
	const history = readHistory();
	const stats = new Map<string, { sum: number; count: number }>();
	for (const e of history.gates) {
		if (e.duration_ms === undefined) continue;
		const s = stats.get(e.gate) ?? { sum: 0, count: 0 };
		s.sum += e.duration_ms;
		s.count++;
		stats.set(e.gate, s);
	}
	return [...stats.entries()]
		.map(([gate, s]) => ({ gate, avgMs: Math.round(s.sum / s.count), count: s.count }))
		.sort((a, b) => b.avgMs - a.avgMs);
}

// Patterns to extract specific error codes from gate output
// Biome: lint/category/ruleName, format/category/ruleName, assist/category/ruleName (exactly 3 segments)
const BIOME_RULE_RE = /(?:lint|format|assist)\/\w+\/\w+/g;
const TS_ERROR_RE = /TS\d{4,5}/g;
// ESLint: @scope/rule-name or no-something (must have hyphen to distinguish from paths)
const ESLINT_RULE_RE = /@[\w-]+\/[\w-]+|no-[\w-]+/g;

/** Extract specific error patterns from gate failure output. */
export function getTopErrorPatterns(n: number): { gate: string; pattern: string; count: number }[] {
	const history = readHistory();
	const counts = new Map<string, { gate: string; pattern: string; count: number }>();

	for (const e of history.gates) {
		if (e.passed || !e.error) continue;

		const patterns: string[] = [];
		// Try biome rules first
		const biomeMatches = e.error.match(BIOME_RULE_RE);
		if (biomeMatches) patterns.push(...biomeMatches);
		// Try TS errors
		const tsMatches = e.error.match(TS_ERROR_RE);
		if (tsMatches) patterns.push(...tsMatches);
		// Try eslint rules
		if (patterns.length === 0) {
			const eslintMatches = e.error.match(ESLINT_RULE_RE);
			if (eslintMatches) patterns.push(...eslintMatches);
		}

		// Deduplicate within a single error
		for (const p of new Set(patterns)) {
			const key = `${e.gate}:${p}`;
			const existing = counts.get(key);
			if (existing) {
				existing.count++;
			} else {
				counts.set(key, { gate: e.gate, pattern: p, count: 1 });
			}
		}
	}

	return [...counts.values()].sort((a, b) => b.count - a.count).slice(0, n);
}
