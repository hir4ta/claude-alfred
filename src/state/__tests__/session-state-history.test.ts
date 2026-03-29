import { mkdirSync, rmSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { resetAllCaches } from "../flush.ts";
import {
	clearOnCommit,
	getPlanEvalScoreHistory,
	getReviewIteration,
	getReviewScoreHistory,
	readSessionState,
	recordPlanEvalIteration,
	recordReviewIteration,
	resetPlanEvalIteration,
	resetReviewIteration,
} from "../session-state.ts";

const TEST_DIR = join(import.meta.dirname, ".tmp-history");
const STATE_DIR = join(TEST_DIR, ".qult", ".state");
const originalCwd = process.cwd();

beforeEach(() => {
	resetAllCaches();
	mkdirSync(STATE_DIR, { recursive: true });
	process.chdir(TEST_DIR);
});

afterEach(() => {
	process.chdir(originalCwd);
	rmSync(TEST_DIR, { recursive: true, force: true });
});

describe("review score history", () => {
	it("pushes scores and tracks iteration count", () => {
		recordReviewIteration(9);
		expect(getReviewIteration()).toBe(1);
		expect(getReviewScoreHistory()).toEqual([9]);

		recordReviewIteration(11);
		expect(getReviewIteration()).toBe(2);
		expect(getReviewScoreHistory()).toEqual([9, 11]);

		recordReviewIteration(13);
		expect(getReviewIteration()).toBe(3);
		expect(getReviewScoreHistory()).toEqual([9, 11, 13]);
	});

	it("resets history on resetReviewIteration", () => {
		recordReviewIteration(9);
		recordReviewIteration(10);
		resetReviewIteration();

		expect(getReviewIteration()).toBe(0);
		expect(getReviewScoreHistory()).toEqual([]);
	});

	it("resets history on clearOnCommit", () => {
		recordReviewIteration(9);
		clearOnCommit();

		expect(getReviewScoreHistory()).toEqual([]);
		const state = readSessionState();
		expect(state.review_iteration).toBe(0);
	});
});

describe("plan eval score history", () => {
	it("pushes scores and resets", () => {
		recordPlanEvalIteration(8);
		recordPlanEvalIteration(10);
		expect(getPlanEvalScoreHistory()).toEqual([8, 10]);

		resetPlanEvalIteration();
		expect(getPlanEvalScoreHistory()).toEqual([]);
	});
});

describe("legacy migration", () => {
	it("migrates scalar review_last_aggregate to array", () => {
		writeFileSync(
			join(STATE_DIR, "session-state.json"),
			JSON.stringify({
				review_iteration: 1,
				review_last_aggregate: 9,
				plan_eval_iteration: 0,
				plan_eval_last_aggregate: 0,
			}),
		);
		resetAllCaches();

		const state = readSessionState();
		expect(state.review_score_history).toEqual([9]);
		expect(state.plan_eval_score_history).toEqual([]);
	});
});
