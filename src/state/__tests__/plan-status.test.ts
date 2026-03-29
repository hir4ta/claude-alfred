import { describe, expect, it } from "vitest";
import { parsePlanTasks } from "../plan-status.ts";

describe("parsePlanTasks", () => {
	it("parses tasks with status markers and taskNumber", () => {
		const plan = `## Context
Adding auth feature

## Tasks
### Task 1: Add middleware [done]
- File: src/middleware.ts

### Task 2: Add routes [pending]
- File: src/routes.ts

### Task 3: Update config [in-progress]
- File: src/config.ts

## Review Gates
- [x] Design Review
- [ ] Final Review`;

		const tasks = parsePlanTasks(plan);
		expect(tasks).toHaveLength(5); // 3 tasks + 2 review gates
		expect(tasks[0]).toEqual({ name: "Add middleware", status: "done", taskNumber: 1 });
		expect(tasks[1]).toEqual({ name: "Add routes", status: "pending", taskNumber: 2 });
		expect(tasks[2]).toEqual({ name: "Update config", status: "in-progress", taskNumber: 3 });
		expect(tasks[3]).toEqual({ name: "Design Review", status: "done" });
		expect(tasks[4]).toEqual({ name: "Final Review", status: "pending" });
	});

	it("populates taskNumber from header", () => {
		const plan = `## Tasks
### Task 5: Fix bug [pending]
- **File**: src/fix.ts
### Task 12: Add feature [done]
- **File**: src/feature.ts`;

		const tasks = parsePlanTasks(plan);
		expect(tasks).toHaveLength(2);
		expect(tasks[0]!.taskNumber).toBe(5);
		expect(tasks[1]!.taskNumber).toBe(12);
	});

	it("defaults to pending when no status marker", () => {
		const plan = `## Tasks
### Task 1: Add helper
- File: src/helper.ts`;

		const tasks = parsePlanTasks(plan);
		expect(tasks).toHaveLength(1);
		expect(tasks[0]).toEqual({ name: "Add helper", status: "pending", taskNumber: 1 });
	});

	it("returns empty for plan without tasks", () => {
		const plan = `## Context
Just a note`;

		const tasks = parsePlanTasks(plan);
		expect(tasks).toHaveLength(0);
	});

	it("parses review gates as checkboxes", () => {
		const plan = `## Review Gates
- [x] Design Review
- [ ] Phase Review
- [x] Final Review`;

		const tasks = parsePlanTasks(plan);
		expect(tasks).toHaveLength(3);
		expect(tasks[0]).toEqual({ name: "Design Review", status: "done" });
		expect(tasks[1]).toEqual({ name: "Phase Review", status: "pending" });
		expect(tasks[2]).toEqual({ name: "Final Review", status: "done" });
	});
});
