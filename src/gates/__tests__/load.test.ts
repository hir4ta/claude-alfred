import { mkdirSync, rmSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { loadGates, resetGatesCache } from "../load.ts";

const TEST_DIR = join(import.meta.dirname, ".tmp-load-test");
const QULT_DIR = join(TEST_DIR, ".qult");
const originalCwd = process.cwd();

beforeEach(() => {
	resetGatesCache();
	mkdirSync(QULT_DIR, { recursive: true });
	process.chdir(TEST_DIR);
});

afterEach(() => {
	process.chdir(originalCwd);
	rmSync(TEST_DIR, { recursive: true, force: true });
});

describe("loadGates", () => {
	it("returns null when gates.json does not exist", () => {
		const result = loadGates();
		expect(result).toBeNull();
	});

	it("returns parsed config from valid gates.json", () => {
		const gates = {
			on_write: { lint: { command: "biome check {file}", timeout: 3000 } },
		};
		writeFileSync(join(QULT_DIR, "gates.json"), JSON.stringify(gates));

		const result = loadGates();
		expect(result).not.toBeNull();
		expect(result?.on_write?.lint?.command).toBe("biome check {file}");
	});

	it("returns null on corrupted JSON (fail-open)", () => {
		writeFileSync(join(QULT_DIR, "gates.json"), "not valid json{{{");

		const result = loadGates();
		expect(result).toBeNull();
	});

	it("caches after first read", () => {
		const gates = {
			on_write: { lint: { command: "biome check {file}", timeout: 3000 } },
		};
		writeFileSync(join(QULT_DIR, "gates.json"), JSON.stringify(gates));

		const first = loadGates();
		// Delete file — second call should still return cached value
		rmSync(join(QULT_DIR, "gates.json"));
		const second = loadGates();

		expect(first).not.toBeNull();
		expect(second).toBe(first);
	});

	it("returns fresh data after resetGatesCache", () => {
		const v1 = { on_write: { lint: { command: "v1", timeout: 1000 } } };
		const v2 = { on_write: { lint: { command: "v2", timeout: 1000 } } };

		writeFileSync(join(QULT_DIR, "gates.json"), JSON.stringify(v1));
		const first = loadGates();

		writeFileSync(join(QULT_DIR, "gates.json"), JSON.stringify(v2));
		resetGatesCache();
		const second = loadGates();

		expect(first?.on_write?.lint?.command).toBe("v1");
		expect(second?.on_write?.lint?.command).toBe("v2");
	});
});
