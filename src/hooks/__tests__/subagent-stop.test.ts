import { mkdirSync, rmSync } from "node:fs";
import { join } from "node:path";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const TEST_DIR = join(import.meta.dirname, ".tmp-subagent-stop-test");
let stdoutCapture: string[] = [];
let exitCode: number | null = null;
const originalCwd = process.cwd();

beforeEach(() => {
	mkdirSync(join(TEST_DIR, ".alfred", ".state"), { recursive: true });
	process.chdir(TEST_DIR);
	stdoutCapture = [];
	exitCode = null;
	vi.spyOn(process.stdout, "write").mockImplementation((data) => {
		stdoutCapture.push(typeof data === "string" ? data : data.toString());
		return true;
	});
	vi.spyOn(process.stderr, "write").mockImplementation(() => true);
	vi.spyOn(process, "exit").mockImplementation((code) => {
		exitCode = code as number;
		throw new Error(`process.exit(${code})`);
	});
});

afterEach(() => {
	vi.restoreAllMocks();
	process.chdir(originalCwd);
	rmSync(TEST_DIR, { recursive: true, force: true });
});

describe("subagentStop", () => {
	it("allows normal subagent completion", async () => {
		const handler = (await import("../subagent-stop.ts")).default;
		await handler({
			hook_type: "SubagentStop",
			stop_hook_active: false,
		});
		expect(exitCode).toBeNull();
	});

	it("does not block when stop_hook_active is true", async () => {
		const handler = (await import("../subagent-stop.ts")).default;
		await handler({
			hook_type: "SubagentStop",
			stop_hook_active: true,
		});
		expect(exitCode).toBeNull();
	});

	it("allows unknown agent_type (fail-open)", async () => {
		const handler = (await import("../subagent-stop.ts")).default;
		await handler({
			hook_type: "SubagentStop",
			agent_type: "Explore",
			last_assistant_message: "some output",
		});
		expect(exitCode).toBeNull();
	});

	it("allows when last_assistant_message is missing (fail-open)", async () => {
		const handler = (await import("../subagent-stop.ts")).default;
		await handler({
			hook_type: "SubagentStop",
			agent_type: "alfred-reviewer",
		});
		expect(exitCode).toBeNull();
	});

	it("blocks alfred-reviewer without findings", async () => {
		const handler = (await import("../subagent-stop.ts")).default;
		try {
			await handler({
				hook_type: "SubagentStop",
				agent_type: "alfred-reviewer",
				last_assistant_message: "I looked at the code and it seems fine.",
			});
		} catch {
			// process.exit(2)
		}
		expect(exitCode).toBe(2);
	});

	it("allows alfred-reviewer with severity findings", async () => {
		const handler = (await import("../subagent-stop.ts")).default;
		await handler({
			hook_type: "SubagentStop",
			agent_type: "alfred-reviewer",
			last_assistant_message:
				"- [high] src/foo.ts:42 — missing null check\n  Fix: add if (!x) return;",
		});
		expect(exitCode).toBeNull();
	});

	it("allows alfred-reviewer with 'No issues found'", async () => {
		const handler = (await import("../subagent-stop.ts")).default;
		await handler({
			hook_type: "SubagentStop",
			agent_type: "alfred-reviewer",
			last_assistant_message: "No issues found from correctness perspective.",
		});
		expect(exitCode).toBeNull();
	});

	it("blocks Plan agent without required sections", async () => {
		const handler = (await import("../subagent-stop.ts")).default;
		try {
			await handler({
				hook_type: "SubagentStop",
				agent_type: "Plan",
				last_assistant_message: "Here is my plan:\n- Do stuff\n- Do more stuff",
			});
		} catch {
			// process.exit(2)
		}
		expect(exitCode).toBe(2);
	});

	it("allows Plan agent with Tasks and Review Gates", async () => {
		const handler = (await import("../subagent-stop.ts")).default;
		await handler({
			hook_type: "SubagentStop",
			agent_type: "Plan",
			last_assistant_message:
				"## Context\nAdding auth\n\n## Tasks\n### Task 1: Add middleware [pending]\n\n## Review Gates\n- [ ] Final Review",
		});
		expect(exitCode).toBeNull();
	});
});
