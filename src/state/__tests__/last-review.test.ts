import { mkdirSync, rmSync } from "node:fs";
import { join } from "node:path";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

const TEST_DIR = join(import.meta.dirname, ".tmp-last-review-test");
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

describe("last-review", () => {
	it("returns null when no state exists", async () => {
		const { readLastReview } = await import("../last-review.ts");
		expect(readLastReview()).toBeNull();
	});

	it("records and reads review", async () => {
		const { recordReview, readLastReview } = await import("../last-review.ts");
		recordReview();

		const state = readLastReview();
		expect(state).not.toBeNull();
		expect(state!.reviewed_at).toBeDefined();
	});

	it("clears review state", async () => {
		const { recordReview, readLastReview, clearReview } = await import("../last-review.ts");
		recordReview();
		expect(readLastReview()).not.toBeNull();

		clearReview();
		expect(readLastReview()).toBeNull();
	});
});
