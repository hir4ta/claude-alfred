import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { renderForSize } from "../templates.js";
import type { SpecFile, SpecSize, SpecType } from "../types.js";
import { extractIDs, validateGherkin, validateSpec } from "../validate.js";

let tmpDir: string;

beforeEach(() => {
	tmpDir = mkdtempSync(join(tmpdir(), "validate-"));
});

afterEach(() => {
	rmSync(tmpDir, { recursive: true, force: true });
});

function initSpec(
	slug: string,
	size: SpecSize,
	specType: SpecType,
	overrides?: Partial<Record<SpecFile, string>>,
): void {
	const specDir = join(tmpDir, ".alfred", "specs", slug);
	mkdirSync(specDir, { recursive: true });

	const rendered = renderForSize(size, specType, {
		taskSlug: slug,
		description: "Test spec",
		date: "2026-03-18",
		specType,
	});

	for (const [file, content] of rendered) {
		writeFileSync(join(specDir, file), overrides?.[file] ?? content);
	}
}

describe("extractIDs", () => {
	it("extracts FR-N IDs", () => {
		expect(extractIDs("FR-1 and FR-2 and FR-1 again", /FR-\d+/g)).toEqual(["FR-1", "FR-2"]);
	});

	it("extracts T-N.N IDs", () => {
		expect(extractIDs("T-1.1 and T-2.3", /T-\d+\.\d+/g)).toEqual(["T-1.1", "T-2.3"]);
	});

	it("returns empty for no matches", () => {
		expect(extractIDs("no ids here", /FR-\d+/g)).toEqual([]);
	});
});

describe("validateGherkin", () => {
	it("passes valid gherkin", () => {
		const content = "```gherkin\nGiven x\nWhen y\nThen z\n```";
		expect(validateGherkin(content).valid).toBe(true);
	});

	it("warns on missing Then", () => {
		const content = "```gherkin\nGiven x\nWhen y\n```";
		const result = validateGherkin(content);
		expect(result.valid).toBe(false);
		expect(result.issues[0]).toContain("Then");
	});

	it("passes when no gherkin blocks", () => {
		expect(validateGherkin("no gherkin here").valid).toBe(true);
	});
});

describe("validateSpec — template defaults", () => {
	it("S spec defaults have no fail checks (NFR-3)", () => {
		initSpec("test-s", "S", "feature");
		const result = validateSpec(tmpDir, "test-s", "S", "feature");
		const fails = result.checks.filter((c) => c.status === "fail");
		expect(fails).toEqual([]);
	});

	it("M spec defaults have no fail checks (NFR-3)", () => {
		initSpec("test-m", "M", "feature");
		const result = validateSpec(tmpDir, "test-m", "M", "feature");
		const fails = result.checks.filter((c) => c.status === "fail");
		expect(fails).toEqual([]);
	});

	it("L spec defaults have no fail checks (NFR-3)", () => {
		initSpec("test-l", "L", "feature");
		const result = validateSpec(tmpDir, "test-l", "L", "feature");
		const fails = result.checks.filter((c) => c.status === "fail");
		expect(fails).toEqual([]);
	});

	it("D spec defaults have no fail checks (NFR-3)", () => {
		initSpec("test-d", "D", "delta");
		const result = validateSpec(tmpDir, "test-d", "D", "delta");
		const fails = result.checks.filter((c) => c.status === "fail");
		expect(fails).toEqual([]);
	});

	it("bugfix spec defaults have no fail checks (NFR-3)", () => {
		initSpec("test-bug", "M", "bugfix");
		const result = validateSpec(tmpDir, "test-bug", "M", "bugfix");
		const fails = result.checks.filter((c) => c.status === "fail");
		expect(fails).toEqual([]);
	});
});

describe("validateSpec — traceability", () => {
	it("detects unreferenced FR in tasks (fr_to_task)", () => {
		initSpec("test-trace", "L", "feature", {
			"requirements.md": `# Req\n## Functional Requirements\n### FR-1: A\n### FR-2: B\n## Non-Functional Requirements\n`,
			"tasks.md": `# Tasks\n## Wave 1\n### T-1.1: Do A\n- Requirements: FR-1\n## Wave: Closing\n- [ ] Review\n`,
		});
		const result = validateSpec(tmpDir, "test-trace", "L", "feature");
		const frToTask = result.checks.find((c) => c.name === "fr_to_task");
		expect(frToTask?.status).toBe("fail");
		expect(frToTask?.message).toContain("FR-2");
	});

	it("passes when all FR referenced", () => {
		initSpec("test-trace2", "L", "feature", {
			"requirements.md": `# Req\n## Functional Requirements\n### FR-1: A\n## Non-Functional Requirements\n`,
			"tasks.md": `# Tasks\n## Wave 1\n### T-1.1: Do A\n- Requirements: FR-1\n## Wave: Closing\n- [ ] Review\n`,
		});
		const result = validateSpec(tmpDir, "test-trace2", "L", "feature");
		const frToTask = result.checks.find((c) => c.name === "fr_to_task");
		expect(frToTask?.status).toBe("pass");
	});
});

describe("validateSpec — size-conditional checks", () => {
	it("XL includes xl_wave_count and xl_nfr_required", () => {
		initSpec("test-xl", "XL", "feature");
		const result = validateSpec(tmpDir, "test-xl", "XL", "feature");
		const checkNames = result.checks.map((c) => c.name);
		expect(checkNames).toContain("xl_wave_count");
		expect(checkNames).toContain("xl_nfr_required");
	});

	it("S does not include XL or L/XL checks", () => {
		initSpec("test-s2", "S", "feature");
		const result = validateSpec(tmpDir, "test-s2", "S", "feature");
		const checkNames = result.checks.map((c) => c.name);
		expect(checkNames).not.toContain("xl_wave_count");
		expect(checkNames).not.toContain("decisions_completeness");
		expect(checkNames).not.toContain("nfr_traceability");
	});
});

describe("validateSpec — min_fr_count", () => {
	it("fails S spec with 0 FRs", () => {
		initSpec("test-nofr", "S", "feature", {
			"requirements.md": "# Requirements\n## Goal\n## Functional Requirements\n\nNothing here\n",
		});
		const result = validateSpec(tmpDir, "test-nofr", "S", "feature");
		const check = result.checks.find((c) => c.name === "min_fr_count");
		expect(check?.status).toBe("fail");
	});
});

describe("validateSpec — performance (NFR-1)", () => {
	it("validates L spec in under 100ms", () => {
		initSpec("test-perf", "L", "feature");
		const t0 = performance.now();
		validateSpec(tmpDir, "test-perf", "L", "feature");
		const elapsed = performance.now() - t0;
		expect(elapsed).toBeLessThan(100);
	});
});
