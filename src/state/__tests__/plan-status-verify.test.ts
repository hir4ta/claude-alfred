import { describe, expect, it } from "vitest";
import { parsePlanTasks, parseVerifyField } from "../plan-status.ts";

describe("parsePlanTasks with verify extraction", () => {
	it("extracts verify field from task block", () => {
		const content = [
			"## Context",
			"Adding auth.",
			"",
			"## Tasks",
			"### Task 1: Add auth [pending]",
			"- **File**: src/auth.ts",
			"- **Change**: Add JWT middleware",
			"- **Boundary**: No route changes",
			"- **Verify**: src/__tests__/auth.test.ts:testAuth",
			"",
			"### Task 2: Add tests [pending]",
			"- **File**: src/__tests__/auth.test.ts",
			"- **Change**: Write test cases",
			"- **Boundary**: Auth test only",
			"- **Verify**: src/__tests__/auth.test.ts:testAuthSuite",
		].join("\n");

		const tasks = parsePlanTasks(content);
		expect(tasks).toHaveLength(2);
		expect(tasks[0]!.name).toBe("Add auth");
		expect(tasks[0]!.verify).toBe("src/__tests__/auth.test.ts:testAuth");
		expect(tasks[1]!.verify).toBe("src/__tests__/auth.test.ts:testAuthSuite");
	});

	it("handles tasks without verify field", () => {
		const content = [
			"## Tasks",
			"### Task 1: Config change [pending]",
			"- **File**: config.json",
			"- **Change**: Update timeout",
		].join("\n");

		const tasks = parsePlanTasks(content);
		expect(tasks).toHaveLength(1);
		expect(tasks[0]!.verify).toBeUndefined();
	});

	it("stops verify search at next task header", () => {
		const content = [
			"## Tasks",
			"### Task 1: First [pending]",
			"- **File**: a.ts",
			"### Task 2: Second [pending]",
			"- **Verify**: b.test.ts:testB",
		].join("\n");

		const tasks = parsePlanTasks(content);
		expect(tasks[0]!.verify).toBeUndefined();
		expect(tasks[1]!.verify).toBe("b.test.ts:testB");
	});
});

describe("parseVerifyField", () => {
	it("parses file:testName format", () => {
		const result = parseVerifyField("src/__tests__/auth.test.ts:testAuth");
		expect(result).toEqual({
			file: "src/__tests__/auth.test.ts",
			testName: "testAuth",
		});
	});

	it("handles paths with colons (Windows drive letters)", () => {
		const result = parseVerifyField("C:/project/test.ts:testFoo");
		expect(result).toEqual({
			file: "C:/project/test.ts",
			testName: "testFoo",
		});
	});

	it("returns null for invalid formats", () => {
		expect(parseVerifyField("no-colon")).toBeNull();
		expect(parseVerifyField(":empty")).toBeNull();
		expect(parseVerifyField("file:")).toBeNull();
	});
});
