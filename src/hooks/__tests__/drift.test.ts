import { describe, expect, it } from "vitest";
import { parseTasksFileRefs } from "../drift.js";

describe("parseTasksFileRefs", () => {
	it("extracts backtick-quoted file paths", () => {
		const content = "- [ ] Edit `src/hooks/post-tool.ts` and `src/hooks/drift.ts`";
		const refs = parseTasksFileRefs(content);
		expect(refs).toContain("src/hooks/post-tool.ts");
		expect(refs).toContain("src/hooks/drift.ts");
	});

	it("extracts Files: comma-separated paths", () => {
		const content = "- Files: src/api/server.ts, src/hooks/stop.ts";
		const refs = parseTasksFileRefs(content);
		expect(refs).toContain("src/api/server.ts");
		expect(refs).toContain("src/hooks/stop.ts");
	});

	it("ignores non-path backtick content", () => {
		const content = "Use `dossier action=init` to create";
		const refs = parseTasksFileRefs(content);
		expect(refs).toEqual([]);
	});

	it("handles mixed content", () => {
		const content = [
			"### T-1.1: Fix `src/hooks/pre-tool.ts`",
			"- Requirements: FR-1",
			"- Files: src/hooks/spec-guard.ts, src/spec/validate.ts",
			"- [ ] Update logic",
		].join("\n");
		const refs = parseTasksFileRefs(content);
		expect(refs).toContain("src/hooks/pre-tool.ts");
		expect(refs).toContain("src/hooks/spec-guard.ts");
		expect(refs).toContain("src/spec/validate.ts");
		expect(refs).toHaveLength(3);
	});

	it("returns empty for no file references", () => {
		const content = "# Tasks\n## Wave 1\n- [ ] Do something";
		expect(parseTasksFileRefs(content)).toEqual([]);
	});

	it("deduplicates paths", () => {
		const content = "Edit `src/foo.ts` and Files: src/foo.ts";
		const refs = parseTasksFileRefs(content);
		expect(refs.filter((r) => r === "src/foo.ts")).toHaveLength(1);
	});
});
