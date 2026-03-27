import { mkdirSync, rmSync } from "node:fs";
import { join } from "node:path";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import {
	checkBudget,
	clearOnCommit,
	isPaceRed,
	markGateRan,
	readPace,
	readSessionState,
	recordFailure,
	recordInjection,
	recordReview,
	recordTestPass,
	resetBudget,
	shouldSkipGate,
	writePace,
} from "../state/session-state.ts";

const TEST_DIR = join(import.meta.dirname, ".tmp-session-state");
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

describe("session-state: pace tracking", () => {
	it("read returns null when no state", () => {
		expect(readPace()).toBeNull();
	});

	it("write and read pace", () => {
		const now = new Date().toISOString();
		writePace({ last_commit_at: now, changed_files: 3, tool_calls: 5 });
		const pace = readPace();
		expect(pace).not.toBeNull();
		expect(pace!.changed_files).toBe(3);
		expect(pace!.tool_calls).toBe(5);
	});
});

describe("session-state: isPaceRed", () => {
	it("returns false at 50 min with 7 files (below new default)", () => {
		const pace = {
			last_commit_at: new Date(Date.now() - 50 * 60_000).toISOString(),
			changed_files: 7,
		};
		expect(isPaceRed(pace)).toBe(false);
	});

	it("returns true at 65 min with 9 files", () => {
		const pace = {
			last_commit_at: new Date(Date.now() - 65 * 60_000).toISOString(),
			changed_files: 9,
		};
		expect(isPaceRed(pace)).toBe(true);
	});

	it("hasPlan gives more headroom (90 min / 12 files)", () => {
		const pace = {
			last_commit_at: new Date(Date.now() - 65 * 60_000).toISOString(),
			changed_files: 9,
		};
		// Without plan: 65 min >= 60, 9 >= 8 → red
		expect(isPaceRed(pace)).toBe(true);
		// With plan: 65 min < 90, threshold not met → not red
		expect(isPaceRed(pace, true)).toBe(false);
	});
});

describe("session-state: test pass tracking", () => {
	it("recordTestPass and read back", () => {
		recordTestPass("vitest run");
		const state = readSessionState();
		expect(state.test_passed_at).not.toBeNull();
		expect(state.test_command).toBe("vitest run");
	});

	it("clearOnCommit clears test pass", () => {
		recordTestPass("vitest run");
		clearOnCommit();
		const state = readSessionState();
		expect(state.test_passed_at).toBeNull();
	});
});

describe("session-state: review tracking", () => {
	it("recordReview and read back", () => {
		recordReview();
		const state = readSessionState();
		expect(state.review_completed_at).not.toBeNull();
	});

	it("clearOnCommit clears review", () => {
		recordReview();
		clearOnCommit();
		const state = readSessionState();
		expect(state.review_completed_at).toBeNull();
	});
});

describe("session-state: gate batch tracking", () => {
	it("shouldSkipGate returns false when not ran", () => {
		expect(shouldSkipGate("lint", "session-1")).toBe(false);
	});

	it("markGateRan then shouldSkipGate returns true for same session", () => {
		markGateRan("typecheck", "session-1");
		expect(shouldSkipGate("typecheck", "session-1")).toBe(true);
	});

	it("shouldSkipGate returns false for different session", () => {
		markGateRan("typecheck", "session-1");
		expect(shouldSkipGate("typecheck", "session-2")).toBe(false);
	});

	it("clearOnCommit clears gate batch", () => {
		markGateRan("typecheck", "session-1");
		clearOnCommit();
		expect(shouldSkipGate("typecheck", "session-1")).toBe(false);
	});
});

describe("session-state: fail count tracking", () => {
	it("recordFailure increments on same signature", () => {
		expect(recordFailure("err:timeout")).toBe(1);
		expect(recordFailure("err:timeout")).toBe(2);
		expect(recordFailure("err:timeout")).toBe(3);
	});

	it("recordFailure resets on different signature", () => {
		recordFailure("err:timeout");
		recordFailure("err:timeout");
		expect(recordFailure("err:other")).toBe(1);
	});

	it("clearOnCommit resets fail count", () => {
		recordFailure("err:timeout");
		recordFailure("err:timeout");
		clearOnCommit();
		expect(recordFailure("err:timeout")).toBe(1);
	});
});

describe("session-state: context budget", () => {
	it("resetBudget initializes budget", () => {
		resetBudget("session-1");
		expect(checkBudget(100)).toBe(true);
	});

	it("budget exceeded returns false", () => {
		resetBudget("session-1");
		recordInjection(1900);
		expect(checkBudget(200)).toBe(false);
	});

	it("same session does not reset", () => {
		resetBudget("session-1");
		recordInjection(500);
		resetBudget("session-1"); // same session — no reset
		expect(checkBudget(1600)).toBe(false); // still 500 used
	});

	it("new session resets budget", () => {
		resetBudget("session-1");
		recordInjection(1900);
		resetBudget("session-2"); // new session — reset
		expect(checkBudget(100)).toBe(true);
	});
});

describe("session-state: clearOnCommit preserves budget", () => {
	it("budget survives clearOnCommit", () => {
		resetBudget("session-1");
		recordInjection(500);
		clearOnCommit();
		// Budget should still have 500 used
		const state = readSessionState();
		expect(state.context_used).toBe(500);
	});
});
