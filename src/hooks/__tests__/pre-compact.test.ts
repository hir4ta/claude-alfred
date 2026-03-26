import { mkdirSync, rmSync } from "node:fs";
import { join } from "node:path";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { readHandoff } from "../../state/handoff.ts";
import { writePendingFixes } from "../../state/pending-fixes.ts";

const TEST_DIR = join(import.meta.dirname, ".tmp-precompact-test");
const STATE_DIR = join(TEST_DIR, ".alfred", ".state");
let stdoutCapture: string[] = [];
const originalCwd = process.cwd();

beforeEach(() => {
	mkdirSync(STATE_DIR, { recursive: true });
	process.chdir(TEST_DIR);
	stdoutCapture = [];

	vi.spyOn(process.stdout, "write").mockImplementation((data) => {
		stdoutCapture.push(typeof data === "string" ? data : data.toString());
		return true;
	});
});

afterEach(() => {
	vi.restoreAllMocks();
	process.chdir(originalCwd);
	rmSync(TEST_DIR, { recursive: true, force: true });
});

describe("preCompact hook", () => {
	it("saves handoff state", async () => {
		writePendingFixes([{ file: "src/a.ts", errors: ["err"], gate: "lint" }]);

		const handler = (await import("../pre-compact.ts")).default;
		await handler({ hook_type: "PreCompact" });

		const handoff = readHandoff();
		expect(handoff).not.toBeNull();
		expect(handoff!.pending_fixes).toBe(true);
		expect(handoff!.saved_at).toBeDefined();
	});

	it("saves handoff with no pending fixes", async () => {
		const handler = (await import("../pre-compact.ts")).default;
		await handler({ hook_type: "PreCompact" });

		const handoff = readHandoff();
		expect(handoff).not.toBeNull();
		expect(handoff!.pending_fixes).toBe(false);
	});
});
