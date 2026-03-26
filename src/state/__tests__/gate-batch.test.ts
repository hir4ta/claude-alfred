import { mkdirSync, rmSync } from "node:fs";
import { join } from "node:path";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

const TEST_DIR = join(import.meta.dirname, ".tmp-gate-batch-test");
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

describe("shouldSkip", () => {
	it("returns false when no batch state exists", async () => {
		const { shouldSkip } = await import("../gate-batch.ts");
		expect(shouldSkip("typecheck", "session-1")).toBe(false);
	});

	it("returns false for different session_id", async () => {
		const { shouldSkip, markRan } = await import("../gate-batch.ts");
		markRan("typecheck", "session-1");
		expect(shouldSkip("typecheck", "session-2")).toBe(false);
	});

	it("returns true for same gate + same session_id", async () => {
		const { shouldSkip, markRan } = await import("../gate-batch.ts");
		markRan("typecheck", "session-1");
		expect(shouldSkip("typecheck", "session-1")).toBe(true);
	});

	it("returns false for different gate name in same session", async () => {
		const { shouldSkip, markRan } = await import("../gate-batch.ts");
		markRan("typecheck", "session-1");
		expect(shouldSkip("lint", "session-1")).toBe(false);
	});
});

describe("clearBatch", () => {
	it("clears all batch state", async () => {
		const { shouldSkip, markRan, clearBatch } = await import("../gate-batch.ts");
		markRan("typecheck", "session-1");
		expect(shouldSkip("typecheck", "session-1")).toBe(true);

		clearBatch();
		expect(shouldSkip("typecheck", "session-1")).toBe(false);
	});
});

describe("markRan", () => {
	it("records gate execution with timestamp", async () => {
		const { markRan, readBatch } = await import("../gate-batch.ts");
		markRan("typecheck", "session-1");

		const batch = readBatch();
		expect(batch).not.toBeNull();
		expect(batch!.typecheck).toBeDefined();
		expect(batch!.typecheck!.session_id).toBe("session-1");
		expect(batch!.typecheck!.ran_at).toBeDefined();
	});
});
