import { mkdirSync, rmSync } from "node:fs";
import { join } from "node:path";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { writePendingFixes } from "../../state/pending-fixes.ts";

const TEST_DIR = join(import.meta.dirname, ".tmp-post-compact-test");
const STATE_DIR = join(TEST_DIR, ".alfred", ".state");
let stderrCapture: string[] = [];
const originalCwd = process.cwd();

beforeEach(() => {
	mkdirSync(STATE_DIR, { recursive: true });
	process.chdir(TEST_DIR);
	stderrCapture = [];
	vi.spyOn(process.stderr, "write").mockImplementation((data) => {
		stderrCapture.push(typeof data === "string" ? data : data.toString());
		return true;
	});
});

afterEach(() => {
	vi.restoreAllMocks();
	process.chdir(originalCwd);
	rmSync(TEST_DIR, { recursive: true, force: true });
});

describe("postCompact", () => {
	it("warns about pending fixes via stderr", async () => {
		writePendingFixes([{ file: "src/auth.ts", errors: ["type error"], gate: "typecheck" }]);

		const handler = (await import("../post-compact.ts")).default;
		await handler({ hook_type: "PostCompact" });

		const stderr = stderrCapture.join("");
		expect(stderr).toContain("pending lint/type fix");
	});

	it("does nothing when no pending fixes", async () => {
		const handler = (await import("../post-compact.ts")).default;
		await handler({ hook_type: "PostCompact" });

		expect(stderrCapture.join("")).toBe("");
	});

	it("injects plan progress after compaction", async () => {
		const planDir = join(TEST_DIR, ".claude", "plans");
		mkdirSync(planDir, { recursive: true });
		const { writeFileSync } = await import("node:fs");
		writeFileSync(
			join(planDir, "plan.md"),
			"## Tasks\n### Task 1: Setup [done]\n- done\n### Task 2: Implement [in-progress]\n- wip\n### Task 3: Test [pending]\n- todo",
		);

		const handler = (await import("../post-compact.ts")).default;
		await handler({ hook_type: "PostCompact" });

		const stderr = stderrCapture.join("");
		expect(stderr).toContain("Plan progress");
		expect(stderr).toContain("Done: Setup");
		expect(stderr).toContain("Remaining: Test");
	});
});
