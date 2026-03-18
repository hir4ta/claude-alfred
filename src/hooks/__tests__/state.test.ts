import { mkdirSync, mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import {
	addWorkedSlug,
	ensureStateDir,
	parseWaveProgress,
	readStateJSON,
	readStateText,
	readWaveProgress,
	readWorkedSlugs,
	resetWorkedSlugs,
	stateDir,
	writeStateJSON,
	writeStateText,
	writeWaveProgress,
} from "../state.js";
import type { WaveProgress } from "../state.js";

let tmpDir: string;

beforeEach(() => {
	tmpDir = mkdtempSync(join(tmpdir(), "state-"));
	mkdirSync(join(tmpDir, ".alfred"), { recursive: true });
});

afterEach(() => {
	rmSync(tmpDir, { recursive: true, force: true });
});

describe("stateDir", () => {
	it("returns .alfred/.state path", () => {
		expect(stateDir(tmpDir)).toBe(join(tmpDir, ".alfred", ".state"));
	});
});

describe("ensureStateDir", () => {
	it("creates directory if missing", () => {
		ensureStateDir(tmpDir);
		const { existsSync } = require("node:fs");
		expect(existsSync(stateDir(tmpDir))).toBe(true);
	});

	it("is idempotent", () => {
		ensureStateDir(tmpDir);
		ensureStateDir(tmpDir); // no throw
	});
});

describe("readStateJSON / writeStateJSON", () => {
	it("round-trips JSON data", () => {
		const data = { count: 3, label: "test" };
		writeStateJSON(tmpDir, "test.json", data);
		expect(readStateJSON(tmpDir, "test.json", {})).toEqual(data);
	});

	it("returns fallback when file missing", () => {
		expect(readStateJSON(tmpDir, "missing.json", { default: true })).toEqual({ default: true });
	});

	it("returns fallback on invalid JSON", () => {
		writeStateText(tmpDir, "bad.json", "not json");
		expect(readStateJSON(tmpDir, "bad.json", [])).toEqual([]);
	});
});

describe("readStateText / writeStateText", () => {
	it("round-trips text data", () => {
		writeStateText(tmpDir, "counter", "42");
		expect(readStateText(tmpDir, "counter", "0")).toBe("42");
	});

	it("returns fallback when file missing", () => {
		expect(readStateText(tmpDir, "missing", "default")).toBe("default");
	});
});

describe("worked-slugs", () => {
	it("returns empty array when no file exists", () => {
		expect(readWorkedSlugs(tmpDir)).toEqual([]);
	});

	it("adds slugs with deduplication", () => {
		addWorkedSlug(tmpDir, "task-a");
		addWorkedSlug(tmpDir, "task-b");
		addWorkedSlug(tmpDir, "task-a"); // duplicate
		expect(readWorkedSlugs(tmpDir)).toEqual(["task-a", "task-b"]);
	});

	it("resets to empty array", () => {
		addWorkedSlug(tmpDir, "task-a");
		resetWorkedSlugs(tmpDir);
		expect(readWorkedSlugs(tmpDir)).toEqual([]);
	});
});

describe("parseWaveProgress", () => {
	it("counts per-wave tasks correctly (TS-1.1)", () => {
		const content = `# Tasks: test-slug

## Wave 1: Setup

- [x] T-1.1: First task
- [x] T-1.2: Second task
- [ ] T-1.3: Third task

## Wave 2: Implementation

- [ ] T-2.1: Fourth task
- [ ] T-2.2: Fifth task
`;
		const result = parseWaveProgress(content, "test-slug");
		expect(result.waves["1"]).toEqual({ total: 3, checked: 2, reviewed: false });
		expect(result.waves["2"]).toEqual({ total: 2, checked: 0, reviewed: false });
		expect(result.slug).toBe("test-slug");
	});

	it("parses Closing Wave (TS-1.5)", () => {
		const content = `# Tasks: test

## Wave 1: Core

- [x] T-1.1: Task

## Wave: Closing

- [ ] Self-review
- [x] Build check
`;
		const result = parseWaveProgress(content, "test");
		expect(result.waves["closing"]).toEqual({ total: 2, checked: 1, reviewed: false });
	});

	it("determines current_wave from first incomplete wave", () => {
		const content = `# Tasks: test

## Wave 1: Done

- [x] T-1.1: Done task

## Wave 2: In Progress

- [ ] T-2.1: Pending
`;
		const result = parseWaveProgress(content, "test");
		expect(result.current_wave).toBe(2);
	});
});

describe("wave progress persistence (TS-1.4)", () => {
	it("round-trips wave progress", () => {
		const progress: WaveProgress = {
			slug: "test-slug",
			current_wave: 2,
			waves: {
				"1": { total: 3, checked: 3, reviewed: true },
				"2": { total: 2, checked: 0, reviewed: false },
			},
		};
		writeWaveProgress(tmpDir, progress);
		const read = readWaveProgress(tmpDir);
		expect(read).toEqual(progress);
	});

	it("returns null when no file exists (TS-1.6)", () => {
		expect(readWaveProgress(tmpDir)).toBeNull();
	});
});
