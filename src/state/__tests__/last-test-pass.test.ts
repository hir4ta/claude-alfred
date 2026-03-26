import { mkdirSync, rmSync } from "node:fs";
import { join } from "node:path";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

const TEST_DIR = join(import.meta.dirname, ".tmp-last-test-pass-test");
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

describe("last-test-pass", () => {
	it("returns null when no state exists", async () => {
		const { readLastTestPass } = await import("../last-test-pass.ts");
		expect(readLastTestPass()).toBeNull();
	});

	it("records and reads test pass", async () => {
		const { recordTestPass, readLastTestPass } = await import("../last-test-pass.ts");
		recordTestPass("vitest run");

		const state = readLastTestPass();
		expect(state).not.toBeNull();
		expect(state!.command).toBe("vitest run");
		expect(state!.passed_at).toBeDefined();
	});

	it("clears test pass state", async () => {
		const { recordTestPass, readLastTestPass, clearTestPass } = await import(
			"../last-test-pass.ts"
		);
		recordTestPass("vitest run");
		expect(readLastTestPass()).not.toBeNull();

		clearTestPass();
		expect(readLastTestPass()).toBeNull();
	});

	it("overwrites previous state", async () => {
		const { recordTestPass, readLastTestPass } = await import("../last-test-pass.ts");
		recordTestPass("jest");
		recordTestPass("vitest run");

		const state = readLastTestPass();
		expect(state!.command).toBe("vitest run");
	});
});
