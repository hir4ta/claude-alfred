import { mkdirSync, rmSync } from "node:fs";
import { join } from "node:path";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { resetAllCaches } from "../flush.ts";

const TEST_DIR = join(import.meta.dirname, ".tmp-gate-history-test");
const originalCwd = process.cwd();

beforeEach(() => {
	resetAllCaches();
	mkdirSync(join(TEST_DIR, ".qult", ".state"), { recursive: true });
	mkdirSync(join(TEST_DIR, ".qult", "gate-history"), { recursive: true });
	process.chdir(TEST_DIR);
});

afterEach(() => {
	process.chdir(originalCwd);
	rmSync(TEST_DIR, { recursive: true, force: true });
});

describe("gate results", () => {
	it("records gate results and retrieves top errors", async () => {
		const { recordGateResult, flush, getTopErrors } = await import("../gate-history.ts");
		recordGateResult("lint", false, "unused import");
		recordGateResult("lint", false, "unused import");
		recordGateResult("lint", false, "unused import");
		recordGateResult("typecheck", false, "type error");
		recordGateResult("lint", true);
		flush();

		const top = getTopErrors(3);
		expect(top.length).toBeGreaterThan(0);
		expect(top[0]!.gate).toBe("lint");
		expect(top[0]!.count).toBe(3);
	});

	it("returns empty for no errors", async () => {
		const { recordGateResult, flush, getTopErrors } = await import("../gate-history.ts");
		recordGateResult("lint", true);
		recordGateResult("typecheck", true);
		flush();

		const top = getTopErrors(3);
		expect(top).toHaveLength(0);
	});

	it("returns empty array for getTopErrors(0)", async () => {
		const { recordGateResult, flush, getTopErrors } = await import("../gate-history.ts");
		recordGateResult("lint", false, "unused import");
		flush();

		const top = getTopErrors(0);
		expect(top).toHaveLength(0);
		expect(Array.isArray(top)).toBe(true);
	});

	it("returns all available when n > actual count", async () => {
		const { recordGateResult, flush, getTopErrors } = await import("../gate-history.ts");
		recordGateResult("lint", false, "unused import");
		flush();

		const top = getTopErrors(100);
		expect(top).toHaveLength(1);
		expect(top[0]!.gate).toBe("lint");
	});

	it("daily files have no entry cap", async () => {
		const { recordGateResult, flush, getTopErrors } = await import("../gate-history.ts");
		for (let i = 0; i < 210; i++) {
			recordGateResult("lint", false, `error-${i}`);
		}
		flush();

		const top = getTopErrors(1);
		expect(top).toHaveLength(1);
	});
});

describe("commit stats", () => {
	it("returns null when no commits recorded", async () => {
		const { getCommitStats } = await import("../gate-history.ts");
		expect(getCommitStats()).toBeNull();
	});

	it("calculates average commit interval", async () => {
		const { recordCommit, flush, getCommitStats } = await import("../gate-history.ts");
		recordCommit();
		recordCommit();
		flush();

		const stats = getCommitStats();
		expect(stats).not.toBeNull();
		expect(stats!.count).toBe(2);
		expect(stats!.avgMinutes).toBeLessThan(1);
	});
});
