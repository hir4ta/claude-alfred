import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { writeFileSync, mkdtempSync, rmSync, readFileSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { mergeKnowledgeFiles } from "../merge-driver.js";

describe("mergeKnowledgeFiles", () => {
	let dir: string;

	beforeEach(() => {
		dir = mkdtempSync(join(tmpdir(), "merge-test-"));
	});

	afterEach(() => {
		rmSync(dir, { recursive: true, force: true });
	});

	function writeJson(name: string, obj: Record<string, unknown>): string {
		const path = join(dir, name);
		writeFileSync(path, JSON.stringify(obj));
		return path;
	}

	function readResult(path: string): Record<string, unknown> {
		return JSON.parse(readFileSync(path, "utf-8"));
	}

	it("takes max for numeric fields", () => {
		const base = writeJson("base.json", { id: "test", title: "T", hitCount: 5 });
		const ours = writeJson("ours.json", { id: "test", title: "T", hitCount: 10 });
		const theirs = writeJson("theirs.json", { id: "test", title: "T", hitCount: 8 });

		const code = mergeKnowledgeFiles(base, ours, theirs);
		expect(code).toBe(0);
		const result = readResult(ours);
		expect(result.hitCount).toBe(10);
	});

	it("takes theirs when ours unchanged", () => {
		const base = writeJson("base.json", { id: "test", title: "Original" });
		const ours = writeJson("ours.json", { id: "test", title: "Original" });
		const theirs = writeJson("theirs.json", { id: "test", title: "Modified" });

		const code = mergeKnowledgeFiles(base, ours, theirs);
		expect(code).toBe(0);
		expect(readResult(ours).title).toBe("Modified");
	});

	it("unions array fields", () => {
		const base = writeJson("base.json", { id: "test", title: "T", tags: ["auth"] });
		const ours = writeJson("ours.json", { id: "test", title: "T", tags: ["auth", "security"] });
		const theirs = writeJson("theirs.json", { id: "test", title: "T", tags: ["auth", "jwt"] });

		const code = mergeKnowledgeFiles(base, ours, theirs);
		expect(code).toBe(0);
		const tags = readResult(ours).tags as string[];
		expect(tags).toContain("auth");
		expect(tags).toContain("security");
		expect(tags).toContain("jwt");
	});

	it("returns 1 for both-changed conflict", () => {
		const base = writeJson("base.json", { id: "test", title: "Original" });
		const ours = writeJson("ours.json", { id: "test", title: "Our change" });
		const theirs = writeJson("theirs.json", { id: "test", title: "Their change" });

		const code = mergeKnowledgeFiles(base, ours, theirs);
		expect(code).toBe(1);
	});

	it("returns 1 for invalid JSON", () => {
		const base = join(dir, "base.json");
		writeFileSync(base, "not json");
		const ours = writeJson("ours.json", { id: "test", title: "T" });
		const theirs = writeJson("theirs.json", { id: "test", title: "T" });

		const code = mergeKnowledgeFiles(base, ours, theirs);
		expect(code).toBe(1);
	});

	it("returns 1 for non-knowledge JSON", () => {
		const base = writeJson("base.json", { foo: "bar" });
		const ours = writeJson("ours.json", { foo: "bar" });
		const theirs = writeJson("theirs.json", { foo: "bar" });

		const code = mergeKnowledgeFiles(base, ours, theirs);
		expect(code).toBe(1);
	});
});
