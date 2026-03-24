import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { addWorkedSlug, resetWorkedSlugs } from "../state.js";
import { checkSpecRequired } from "../user-prompt.js";

let tmpDir: string;

beforeEach(() => {
	tmpDir = mkdtempSync(join(tmpdir(), "user-prompt-"));
});

afterEach(() => {
	rmSync(tmpDir, { recursive: true, force: true });
});

function setupAlfred(): void {
	mkdirSync(join(tmpDir, ".alfred"), { recursive: true });
}

function setupSpec(opts: { size?: string }): void {
	setupAlfred();
	const specsDir = join(tmpDir, ".alfred", "specs");
	mkdirSync(specsDir, { recursive: true });
	const state = {
		primary: "test-task",
		tasks: [{
			slug: "test-task",
			started_at: "2026-01-01T00:00:00Z",
			...(opts.size ? { size: opts.size } : {}),
		}],
	};
	writeFileSync(join(specsDir, "_active.json"), JSON.stringify(state));
}

describe("checkSpecRequired (prompt-based, FR-2)", () => {
	it("returns DIRECTIVE when no spec and implementation prompt", () => {
		setupAlfred();
		const result = checkSpecRequired(tmpDir, "ログイン機能を実装してください");
		expect(result).not.toBeNull();
		expect(result!.level).toBe("DIRECTIVE");
	});

	it("returns DIRECTIVE for bugfix prompt", () => {
		setupAlfred();
		const result = checkSpecRequired(tmpDir, "fix the authentication bug");
		expect(result).not.toBeNull();
		expect(result!.level).toBe("DIRECTIVE");
	});

	it("returns DIRECTIVE for tdd prompt", () => {
		setupAlfred();
		const result = checkSpecRequired(tmpDir, "TDDでユニットテストを書いて");
		expect(result).not.toBeNull();
		expect(result!.level).toBe("DIRECTIVE");
	});

	it("returns null for review/research prompts (no impl keywords)", () => {
		setupAlfred();
		expect(checkSpecRequired(tmpDir, "review the code changes")).toBeNull();
		expect(checkSpecRequired(tmpDir, "research design patterns")).toBeNull();
	});

	it("returns null for general chat", () => {
		setupAlfred();
		expect(checkSpecRequired(tmpDir, "hello, how are you?")).toBeNull();
	});

	it("returns WARNING when spec exists (parallel dev guard, Stage 1.5)", () => {
		setupSpec({ size: "M" });
		const result = checkSpecRequired(tmpDir, "implement login feature");
		expect(result).not.toBeNull();
		expect(result!.level).toBe("WARNING");
		expect(result!.message).toContain("test-task");
	});

	it("returns WARNING for any size spec (Stage 1.5)", () => {
		setupSpec({ size: "S" });
		const result = checkSpecRequired(tmpDir, "バグを修正して");
		expect(result).not.toBeNull();
		expect(result!.level).toBe("WARNING");
		expect(result!.message).toContain("test-task");
	});

	it("returns null for non-impl prompt when spec exists", () => {
		setupSpec({ size: "S" });
		expect(checkSpecRequired(tmpDir, "explain how this works")).toBeNull();
	});

	it("suppresses WARNING when slug is in worked-slugs", () => {
		setupSpec({ size: "S" });
		mkdirSync(join(tmpDir, ".alfred", ".state"), { recursive: true });
		resetWorkedSlugs(tmpDir);
		addWorkedSlug(tmpDir, "test-task");
		expect(checkSpecRequired(tmpDir, "implement the feature")).toBeNull();
	});

	it("returns null when no .alfred/ directory", () => {
		expect(checkSpecRequired(tmpDir, "implement something")).toBeNull();
	});

	it("guards against repeated prompting (spec-prompt.json)", () => {
		setupAlfred();
		// First call: DIRECTIVE
		const first = checkSpecRequired(tmpDir, "implement login");
		expect(first).not.toBeNull();
		expect(first!.level).toBe("DIRECTIVE");
		// Second call: null (guard prevents re-prompting)
		const second = checkSpecRequired(tmpDir, "implement login");
		expect(second).toBeNull();
	});
});
