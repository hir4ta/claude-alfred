import { mkdirSync, rmSync } from "node:fs";
import { join } from "node:path";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

const TEST_DIR = join(import.meta.dirname, ".tmp-context-budget-test");
const STATE_DIR = join(TEST_DIR, ".alfred", ".state");
const originalCwd = process.cwd();

beforeEach(() => {
	mkdirSync(STATE_DIR, { recursive: true });
	process.chdir(TEST_DIR);
});

afterEach(() => {
	process.chdir(originalCwd);
	rmSync(TEST_DIR, { recursive: true, force: true });
});

describe("context-budget", () => {
	it("allows injection within budget", async () => {
		const { checkBudget, resetBudget } = await import("../context-budget.ts");
		resetBudget("session-1");
		expect(checkBudget(100)).toBe(true);
	});

	it("blocks injection when budget exceeded", async () => {
		const { checkBudget, recordInjection, resetBudget } = await import("../context-budget.ts");
		resetBudget("session-1");
		recordInjection(1900);
		// Budget is 2000, already used 1900, trying to add 200 → over
		expect(checkBudget(200)).toBe(false);
	});

	it("tracks cumulative usage", async () => {
		const { checkBudget, recordInjection, resetBudget } = await import("../context-budget.ts");
		resetBudget("session-1");
		recordInjection(500);
		recordInjection(500);
		recordInjection(500);
		// 1500 used, 600 more → over 2000
		expect(checkBudget(600)).toBe(false);
		// 1500 used, 400 more → within 2000
		expect(checkBudget(400)).toBe(true);
	});

	it("resets on new session", async () => {
		const { checkBudget, recordInjection, resetBudget } = await import("../context-budget.ts");
		resetBudget("session-1");
		recordInjection(1900);
		expect(checkBudget(200)).toBe(false);

		// New session → budget resets
		resetBudget("session-2");
		expect(checkBudget(200)).toBe(true);
	});

	it("returns true when no state exists (fail-open)", async () => {
		const { checkBudget } = await import("../context-budget.ts");
		// No resetBudget called → no state file
		expect(checkBudget(100)).toBe(true);
	});
});
