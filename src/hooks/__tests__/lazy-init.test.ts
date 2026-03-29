import { existsSync, mkdirSync, rmSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { resetAllCaches } from "../../state/flush.ts";
import { lazyInit, resetLazyInit } from "../lazy-init.ts";

const TEST_DIR = join(import.meta.dirname, ".tmp-lazy-init");
const STATE_DIR = join(TEST_DIR, ".qult", ".state");
const originalCwd = process.cwd();

beforeEach(() => {
	resetAllCaches();
	resetLazyInit();
	rmSync(TEST_DIR, { recursive: true, force: true });
	mkdirSync(TEST_DIR, { recursive: true });
	process.chdir(TEST_DIR);
});

afterEach(() => {
	process.chdir(originalCwd);
	rmSync(TEST_DIR, { recursive: true, force: true });
});

describe("lazyInit", () => {
	it("creates .qult/.state/ if missing", () => {
		expect(existsSync(STATE_DIR)).toBe(false);
		lazyInit();
		expect(existsSync(STATE_DIR)).toBe(true);
	});

	it("is idempotent — second call is a no-op", () => {
		lazyInit();
		expect(existsSync(STATE_DIR)).toBe(true);

		// Remove dir and call again — should not recreate (already initialized)
		rmSync(STATE_DIR, { recursive: true });
		lazyInit();
		expect(existsSync(STATE_DIR)).toBe(false);
	});

	it("cleans up stale session-scoped files (>24h)", () => {
		mkdirSync(STATE_DIR, { recursive: true });

		const stalePath = join(STATE_DIR, "pending-fixes-old-session.json");
		writeFileSync(stalePath, "[]");
		// Set mtime to 25h ago
		const { utimesSync } = require("node:fs");
		const past = (Date.now() - 25 * 60 * 60 * 1000) / 1000;
		utimesSync(stalePath, past, past);

		const freshPath = join(STATE_DIR, "session-state-fresh.json");
		writeFileSync(freshPath, "{}");

		lazyInit();

		expect(existsSync(stalePath)).toBe(false);
		expect(existsSync(freshPath)).toBe(true);
	});

	it("does not remove non-scoped files", () => {
		mkdirSync(STATE_DIR, { recursive: true });

		const nonScoped = join(STATE_DIR, "pending-fixes.json");
		writeFileSync(nonScoped, "[]");
		const { utimesSync } = require("node:fs");
		const past = (Date.now() - 48 * 60 * 60 * 1000) / 1000;
		utimesSync(nonScoped, past, past);

		lazyInit();

		expect(existsSync(nonScoped)).toBe(true);
	});
});
