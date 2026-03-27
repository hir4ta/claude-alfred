import { mkdirSync, rmSync } from "node:fs";
import { join } from "node:path";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const TEST_DIR = join(import.meta.dirname, ".tmp-user-prompt-test");
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

function getResponse(): Record<string, unknown> | null {
	const output = stdoutCapture.join("");
	if (!output) return null;
	return JSON.parse(output);
}

describe("userPrompt", () => {
	it("injects full template for large plan prompt (300+ chars)", async () => {
		const handler = (await import("../user-prompt.ts")).default;
		const longPrompt =
			"implement authentication with JWT tokens, add login endpoint, signup endpoint, " +
			"middleware for protected routes, update user model, add password hashing, " +
			"create refresh token logic, update the database schema, add tests for all endpoints, " +
			"implement rate limiting, add CORS configuration, create API documentation";

		await handler({
			hook_type: "UserPromptSubmit",
			permission_mode: "plan",
			prompt: longPrompt,
		});

		const response = getResponse();
		expect(response).not.toBeNull();
		const context = (response?.hookSpecificOutput as Record<string, string>)?.additionalContext;
		expect(context).toContain("## Tasks");
		expect(context).toContain("Review Gates");
	});

	it("injects compact template for medium plan prompt (100-300 chars)", async () => {
		const handler = (await import("../user-prompt.ts")).default;
		// 100-300 chars
		const mediumPrompt =
			"add a helper function to parse dates and validate format, with tests for edge cases including invalid inputs and timezone handling";

		await handler({
			hook_type: "UserPromptSubmit",
			permission_mode: "plan",
			prompt: mediumPrompt,
		});

		const response = getResponse();
		expect(response).not.toBeNull();
		const context = (response?.hookSpecificOutput as Record<string, string>)?.additionalContext;
		expect(context).toContain("## Tasks");
		expect(context).not.toContain("Review Gates");
	});

	it("does not inject template for short plan prompt (< 100 chars)", async () => {
		const handler = (await import("../user-prompt.ts")).default;

		await handler({
			hook_type: "UserPromptSubmit",
			permission_mode: "plan",
			prompt: "fix the typo in README",
		});

		const output = stdoutCapture.join("");
		expect(output).toBe("");
	});

	it("does not inject template in normal mode", async () => {
		const handler = (await import("../user-prompt.ts")).default;

		await handler({
			hook_type: "UserPromptSubmit",
			prompt: "fix a typo",
		});

		const response = getResponse();
		// No output or no plan template
		if (response) {
			const context = (response?.hookSpecificOutput as Record<string, string>)?.additionalContext;
			expect(context ?? "").not.toContain("## Tasks");
		}
	});

	it("advises plan mode for large tasks in normal mode (no block)", async () => {
		const handler = (await import("../user-prompt.ts")).default;

		// Very long prompt (>500 chars) to trigger advisory
		const longPrompt =
			"implement authentication with JWT tokens, add login endpoint, signup endpoint, " +
			"middleware for protected routes, update user model, add password hashing, " +
			"create refresh token logic, update the database schema, add tests for all endpoints, " +
			"implement rate limiting on all auth endpoints with sliding window algorithm, " +
			"add CORS configuration for frontend origin, create API documentation with OpenAPI spec, " +
			"set up email verification flow with confirmation links, add two-factor authentication support, " +
			"implement password reset with secure token generation and expiry";

		await handler({
			hook_type: "UserPromptSubmit",
			prompt: longPrompt,
		});

		const response = getResponse();
		expect(response).not.toBeNull();
		const context = (response?.hookSpecificOutput as Record<string, string>)?.additionalContext;
		expect(context).toBeDefined();
		expect(context?.toLowerCase()).toContain("plan mode");
	});
});
