import { mkdirSync, rmSync } from "node:fs";
import { join } from "node:path";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { writeHandoff } from "../../state/handoff.ts";

const TEST_DIR = join(import.meta.dirname, ".tmp-session-test");
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

describe("sessionStart hook", () => {
	it("injects handoff context when handoff exists", async () => {
		writeHandoff({
			summary: "Implementing auth middleware",
			changed_files: ["src/middleware.ts"],
			pending_fixes: false,
			next_steps: "Add tests for middleware",
			saved_at: new Date().toISOString(),
		});

		const handler = (await import("../session-start.ts")).default;
		await handler({ hook_type: "SessionStart" });

		const stderr = stderrCapture.join("");
		expect(stderr).toContain("auth middleware");
		expect(stderr).toContain("Add tests");
	});

	it("creates .alfred dir if missing", async () => {
		rmSync(join(TEST_DIR, ".alfred"), { recursive: true, force: true });

		const handler = (await import("../session-start.ts")).default;
		await handler({ hook_type: "SessionStart" });

		const { existsSync } = await import("node:fs");
		expect(existsSync(join(TEST_DIR, ".alfred", ".state"))).toBe(true);
	});
});
