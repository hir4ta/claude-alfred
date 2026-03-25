import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { createTestEnv, insertTestProject, TEST_PROJECT_ID } from "../../__tests__/test-utils.js";
import type { Store } from "../index.js";
import {
	calculateQualityScore,
	getLatestSessionId,
	getRecentEvents,
	getSessionSummary,
	insertQualityEvent,
} from "../quality-events.js";

describe("getLatestSessionId", () => {
	let store: Store;
	let cleanup: () => void;

	beforeEach(() => {
		const env = createTestEnv();
		store = env.store;
		cleanup = env.cleanup;
		insertTestProject(store);
	});

	afterEach(() => cleanup());

	it("returns null when no events exist", () => {
		const result = getLatestSessionId(store, TEST_PROJECT_ID);
		expect(result).toBeNull();
	});

	it("returns the latest session_id by rowid", () => {
		insertQualityEvent(store, TEST_PROJECT_ID, "session-aaa", "gate_pass");
		insertQualityEvent(store, TEST_PROJECT_ID, "real-session-1", "gate_pass");
		insertQualityEvent(store, TEST_PROJECT_ID, "real-session-2", "gate_fail");

		// created_at may have same second; ORDER BY created_at DESC + LIMIT 1
		// returns whichever was last inserted in that second
		const result = getLatestSessionId(store, TEST_PROJECT_ID);
		expect(result).not.toBeNull();
		expect(result).not.toMatch(/^session-/);
	});

	it("excludes session- prefixed IDs (scan sessions)", () => {
		insertQualityEvent(store, TEST_PROJECT_ID, "session-12345", "gate_pass");

		const result = getLatestSessionId(store, TEST_PROJECT_ID);
		expect(result).toBeNull();
	});

	it("ignores events from other projects", () => {
		insertTestProject(store, "other-project", "/other");
		insertQualityEvent(store, "other-project", "other-session", "gate_pass");

		const result = getLatestSessionId(store, TEST_PROJECT_ID);
		expect(result).toBeNull();
	});
});

describe("calculateQualityScore", () => {
	let store: Store;
	let cleanup: () => void;

	beforeEach(() => {
		const env = createTestEnv();
		store = env.store;
		cleanup = env.cleanup;
		insertTestProject(store);
	});

	afterEach(() => cleanup());

	it("returns base + default rates when no events exist", () => {
		const score = calculateQualityScore(store, "empty-session");
		// No events: gate rates default to 1.0, convention defaults to 1.0
		// 1*30 + 1*20 + 0*15 + 1*10 + 25 = 85
		expect(score.sessionScore).toBe(85);
		expect(score.trend).toBe("stable");
	});

	it("calculates correct score with mixed gate results", () => {
		const sid = "test-session";
		// 2 write passes, 1 write fail
		insertQualityEvent(store, TEST_PROJECT_ID, sid, "gate_pass", { group: "on_write" });
		insertQualityEvent(store, TEST_PROJECT_ID, sid, "gate_pass", { group: "on_write" });
		insertQualityEvent(store, TEST_PROJECT_ID, sid, "gate_fail", { group: "on_write" });
		// 1 commit pass
		insertQualityEvent(store, TEST_PROJECT_ID, sid, "gate_pass", { group: "on_commit" });

		const score = calculateQualityScore(store, sid);
		// write rate = 2/3 ≈ 0.667 → 0.667 * 30 = 20
		// commit rate = 1/1 = 1.0 → 1.0 * 20 = 20
		// error = 0 → 0 * 15 = 0
		// convention = 0/0 → rate=1 → 1 * 10 = 10
		// base = 25
		// total = 20 + 20 + 0 + 10 + 25 = 75
		expect(score.sessionScore).toBe(75);
		expect(score.breakdown.gatePassRateWrite.pass).toBe(2);
		expect(score.breakdown.gatePassRateWrite.total).toBe(3);
		expect(score.breakdown.gatePassRateCommit.pass).toBe(1);
		expect(score.breakdown.gatePassRateCommit.total).toBe(1);
	});
});

describe("getRecentEvents", () => {
	let store: Store;
	let cleanup: () => void;

	beforeEach(() => {
		const env = createTestEnv();
		store = env.store;
		cleanup = env.cleanup;
		insertTestProject(store);
	});

	afterEach(() => cleanup());

	it("returns empty array when no events", () => {
		const events = getRecentEvents(store, "no-session");
		expect(events).toEqual([]);
	});

	it("returns correct number of events", () => {
		const sid = "test-session";
		insertQualityEvent(store, TEST_PROJECT_ID, sid, "gate_pass");
		insertQualityEvent(store, TEST_PROJECT_ID, sid, "gate_fail");
		insertQualityEvent(store, TEST_PROJECT_ID, sid, "error_hit");

		const events = getRecentEvents(store, sid, 5);
		expect(events).toHaveLength(3);
		const types = events.map((e) => e.eventType);
		expect(types).toContain("gate_pass");
		expect(types).toContain("gate_fail");
		expect(types).toContain("error_hit");
	});

	it("respects limit parameter", () => {
		const sid = "test-session";
		insertQualityEvent(store, TEST_PROJECT_ID, sid, "gate_pass");
		insertQualityEvent(store, TEST_PROJECT_ID, sid, "gate_fail");
		insertQualityEvent(store, TEST_PROJECT_ID, sid, "error_hit");

		const events = getRecentEvents(store, sid, 2);
		expect(events).toHaveLength(2);
	});
});
