import { execSync } from "node:child_process";
import { mkdirSync, rmSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { resetAllCaches } from "../../state/flush.ts";

vi.mock("node:child_process", () => ({
	execSync: vi.fn(() => "PASS"),
}));

const mockedExecSync = vi.mocked(execSync);

const TEST_DIR = join(import.meta.dirname, ".tmp-task-completed-test");
const STATE_DIR = join(TEST_DIR, ".qult", ".state");
let stdoutCapture: string[] = [];
const originalCwd = process.cwd();

beforeEach(() => {
	resetAllCaches();
	mkdirSync(STATE_DIR, { recursive: true });
	process.chdir(TEST_DIR);
	stdoutCapture = [];
	mockedExecSync.mockReset();
	mockedExecSync.mockReturnValue("PASS");

	vi.spyOn(process.stdout, "write").mockImplementation((data) => {
		stdoutCapture.push(typeof data === "string" ? data : data.toString());
		return true;
	});
	vi.spyOn(process.stderr, "write").mockImplementation(() => true);
});

afterEach(() => {
	vi.restoreAllMocks();
	process.chdir(originalCwd);
	rmSync(TEST_DIR, { recursive: true, force: true });
});

function writePlan(content: string): void {
	const planDir = join(TEST_DIR, ".claude", "plans");
	mkdirSync(planDir, { recursive: true });
	writeFileSync(join(planDir, "test-plan.md"), content);
}

function writeGates(gates: Record<string, unknown>): void {
	writeFileSync(join(TEST_DIR, ".qult", "gates.json"), JSON.stringify(gates));
}

const PLAN_WITH_VERIFY = `## Context
Test feature

## Tasks
### Task 1: Add handler [pending]
- **File**: src/handler.ts
- **Change**: Add handler
- **Verify**: src/__tests__/handler.test.ts:handlesRequest

### Task 2: Add error handler [pending]
- **File**: src/error.ts
- **Change**: Add error handler
- **Verify**: src/__tests__/error.test.ts:handlesError
`;

describe("taskCompleted: early returns", () => {
	it("returns silently when no task_subject", async () => {
		const taskCompleted = (await import("../task-completed.ts")).default;
		await taskCompleted({});
		expect(stdoutCapture.join("")).toBe("");
		expect(mockedExecSync).not.toHaveBeenCalled();
	});

	it("returns silently when no active plan", async () => {
		const taskCompleted = (await import("../task-completed.ts")).default;
		await taskCompleted({ task_subject: "Task 1: Add handler" });
		expect(stdoutCapture.join("")).toBe("");
		expect(mockedExecSync).not.toHaveBeenCalled();
	});

	it("returns silently when task has no verify field", async () => {
		writePlan(`## Tasks\n### Task 1: Add handler [pending]\n- **File**: src/handler.ts\n`);
		writeGates({ on_commit: { test: { command: "vitest run" } } });

		const taskCompleted = (await import("../task-completed.ts")).default;
		await taskCompleted({ task_subject: "Task 1: Add handler" });
		expect(stdoutCapture.join("")).toBe("");
		expect(mockedExecSync).not.toHaveBeenCalled();
	});

	it("returns silently when no test runner detected", async () => {
		writePlan(PLAN_WITH_VERIFY);
		// No gates → no test runner

		const taskCompleted = (await import("../task-completed.ts")).default;
		await taskCompleted({ task_subject: "Task 1: Add handler" });
		expect(stdoutCapture.join("")).toBe("");
		expect(mockedExecSync).not.toHaveBeenCalled();
	});
});

describe("taskCompleted: task matching", () => {
	it("matches by task number from subject", async () => {
		writePlan(PLAN_WITH_VERIFY);
		writeGates({ on_commit: { test: { command: "vitest run" } } });

		const taskCompleted = (await import("../task-completed.ts")).default;
		await taskCompleted({ task_subject: "Task 2: Add error handler" });

		const output = stdoutCapture.join("");
		expect(output).toContain("Add error handler");
		expect(output).toContain("handlesError");
	});

	it("does NOT match by substring (Add handler should not match Add error handler)", async () => {
		writePlan(PLAN_WITH_VERIFY);
		writeGates({ on_commit: { test: { command: "vitest run" } } });

		const taskCompleted = (await import("../task-completed.ts")).default;
		// Subject is just "Add handler" without task number — should match Task 1 exactly
		await taskCompleted({ task_subject: "Add handler" });

		const output = stdoutCapture.join("");
		expect(output).toContain("Add handler");
		expect(output).toContain("handlesRequest");
		// Should NOT have matched Task 2's verify
		expect(output).not.toContain("handlesError");
	});
});

describe("taskCompleted: verify execution", () => {
	it("runs verify test and responds with success", async () => {
		writePlan(PLAN_WITH_VERIFY);
		writeGates({ on_commit: { test: { command: "vitest run" } } });

		const taskCompleted = (await import("../task-completed.ts")).default;
		await taskCompleted({ task_subject: "Task 1: Add handler" });

		expect(mockedExecSync).toHaveBeenCalledOnce();
		const cmd = mockedExecSync.mock.calls[0]![0] as string;
		expect(cmd).toContain("vitest run");
		expect(cmd).toContain("handler.test.ts");
		expect(cmd).toContain("handlesRequest");

		const output = stdoutCapture.join("");
		expect(output).toContain("Task verified");
		expect(output).toContain("passed");
	});

	it("responds with failure when test throws", async () => {
		writePlan(PLAN_WITH_VERIFY);
		writeGates({ on_commit: { test: { command: "vitest run" } } });

		mockedExecSync.mockImplementation(() => {
			const err = new Error("test failed") as Error & {
				stdout: string;
				stderr: string;
			};
			err.stdout = "FAIL handler.test.ts";
			err.stderr = "AssertionError";
			throw err;
		});

		const taskCompleted = (await import("../task-completed.ts")).default;
		await taskCompleted({ task_subject: "Task 1: Add handler" });

		const output = stdoutCapture.join("");
		expect(output).toContain("verification failed");
		expect(output).toContain("Fix before moving to next task");
	});
});

describe("taskCompleted: shell safety", () => {
	it("rejects verify field with shell metacharacters in file path", async () => {
		writePlan(`## Tasks
### Task 1: Exploit [pending]
- **File**: src/exploit.ts
- **Verify**: src/test.ts;rm+-rf+/:testName
`);
		writeGates({ on_commit: { test: { command: "vitest run" } } });

		const taskCompleted = (await import("../task-completed.ts")).default;
		await taskCompleted({ task_subject: "Task 1: Exploit" });

		expect(stdoutCapture.join("")).toBe("");
		expect(mockedExecSync).not.toHaveBeenCalled();
	});

	it("rejects test name with spaces", async () => {
		writePlan(`## Tasks
### Task 1: Test [pending]
- **File**: src/test.ts
- **Verify**: src/test.ts:test name with spaces
`);
		writeGates({ on_commit: { test: { command: "vitest run" } } });

		const taskCompleted = (await import("../task-completed.ts")).default;
		await taskCompleted({ task_subject: "Task 1: Test" });

		expect(stdoutCapture.join("")).toBe("");
		expect(mockedExecSync).not.toHaveBeenCalled();
	});
});
