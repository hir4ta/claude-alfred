import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";

const STATE_DIR = ".alfred/.state";
const FILE = "context-budget.json";
const BUDGET = 2000; // tokens

interface BudgetState {
	session_id: string;
	used: number;
}

function filePath(): string {
	return join(process.cwd(), STATE_DIR, FILE);
}

function readState(): BudgetState | null {
	try {
		const path = filePath();
		if (!existsSync(path)) return null;
		return JSON.parse(readFileSync(path, "utf-8"));
	} catch {
		return null;
	}
}

function writeState(state: BudgetState): void {
	try {
		const dir = join(process.cwd(), STATE_DIR);
		if (!existsSync(dir)) mkdirSync(dir, { recursive: true });
		writeFileSync(filePath(), JSON.stringify(state, null, 2));
	} catch {
		// fail-open
	}
}

/** Check if adding `tokens` would exceed the budget. Returns true if within budget. */
export function checkBudget(tokens: number): boolean {
	const state = readState();
	if (!state) return true; // fail-open: no state → allow
	return state.used + tokens <= BUDGET;
}

/** Record tokens injected. */
export function recordInjection(tokens: number): void {
	const state = readState();
	if (!state) return;
	state.used += tokens;
	writeState(state);
}

/** Reset budget for a new session. */
export function resetBudget(sessionId: string): void {
	const state = readState();
	if (state && state.session_id === sessionId) return; // same session, no reset
	writeState({ session_id: sessionId, used: 0 });
}
