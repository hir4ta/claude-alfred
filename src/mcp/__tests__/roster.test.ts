import { existsSync, mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { Store } from "../../store/index.js";
import { handleRoster } from "../roster.js";

let tmpDir: string;
let store: Store;

function parseResult(result: { content: Array<{ type: string; text: string }> }) {
	return JSON.parse(result.content[0]!.text);
}

beforeEach(() => {
	tmpDir = mkdtempSync(join(tmpdir(), "roster-test-"));
	store = Store.open(join(tmpDir, "test.db"));
	mkdirSync(join(tmpDir, ".alfred", "epics"), { recursive: true });
});

afterEach(() => {
	store.close();
	rmSync(tmpDir, { recursive: true, force: true });
});

function createSpec(slug: string) {
	const specsDir = join(tmpDir, ".alfred", "specs", slug);
	mkdirSync(specsDir, { recursive: true });
	writeFileSync(join(specsDir, "requirements.md"), "# Requirements");
}

describe("roster init", () => {
	it("creates an epic", async () => {
		const result = await handleRoster(store, {
			action: "init", epic_slug: "my-epic", name: "My Epic",
			project_path: tmpDir,
		});
		const data = parseResult(result);
		expect(data.epic_slug).toBe("my-epic");
		expect(data.name).toBe("My Epic");
		expect(data.status).toBe("draft");
	});

	it("uses epic_slug as name when name not provided", async () => {
		const result = await handleRoster(store, {
			action: "init", epic_slug: "auto-name", project_path: tmpDir,
		});
		const data = parseResult(result);
		expect(data.name).toBe("auto-name");
	});

	it("rejects missing epic_slug", async () => {
		const result = await handleRoster(store, {
			action: "init", project_path: tmpDir,
		});
		const data = parseResult(result);
		expect(data.error).toContain("epic_slug");
	});
});

describe("roster status", () => {
	it("returns epic status with progress", async () => {
		await handleRoster(store, {
			action: "init", epic_slug: "status-epic", name: "Status Test",
			project_path: tmpDir,
		});

		const result = await handleRoster(store, {
			action: "status", epic_slug: "status-epic", project_path: tmpDir,
		});
		const data = parseResult(result);
		expect(data.epic_slug).toBe("status-epic");
		expect(data.completed).toBe(0);
		expect(data.total).toBe(0);
		expect(data.progress_pct).toBe(0);
	});

	it("returns error for non-existent epic", async () => {
		const result = await handleRoster(store, {
			action: "status", epic_slug: "nonexistent", project_path: tmpDir,
		});
		const data = parseResult(result);
		expect(data.error).toContain("not found");
	});
});

describe("roster link/unlink", () => {
	it("links a task to an epic", async () => {
		await handleRoster(store, {
			action: "init", epic_slug: "link-epic", project_path: tmpDir,
		});
		createSpec("task-a");

		const result = await handleRoster(store, {
			action: "link", epic_slug: "link-epic", task_slug: "task-a",
			project_path: tmpDir,
		});
		const data = parseResult(result);
		expect(data.task_slug).toBe("task-a");

		// Verify status shows the task
		const status = await handleRoster(store, {
			action: "status", epic_slug: "link-epic", project_path: tmpDir,
		});
		const statusData = parseResult(status);
		expect(statusData.total).toBe(1);
	});

	it("links with dependencies", async () => {
		await handleRoster(store, {
			action: "init", epic_slug: "dep-epic", project_path: tmpDir,
		});
		createSpec("task-b");
		createSpec("task-c");

		await handleRoster(store, {
			action: "link", epic_slug: "dep-epic", task_slug: "task-b",
			project_path: tmpDir,
		});
		const result = await handleRoster(store, {
			action: "link", epic_slug: "dep-epic", task_slug: "task-c",
			depends_on: "task-b", project_path: tmpDir,
		});
		const data = parseResult(result);
		expect(data.depends_on).toEqual(["task-b"]);
	});

	it("rejects link without spec", async () => {
		await handleRoster(store, {
			action: "init", epic_slug: "no-spec-epic", project_path: tmpDir,
		});

		const result = await handleRoster(store, {
			action: "link", epic_slug: "no-spec-epic", task_slug: "no-spec",
			project_path: tmpDir,
		});
		const data = parseResult(result);
		expect(data.error).toContain("no spec");
	});

	it("unlinks a task", async () => {
		await handleRoster(store, {
			action: "init", epic_slug: "unlink-epic", project_path: tmpDir,
		});
		createSpec("task-d");
		await handleRoster(store, {
			action: "link", epic_slug: "unlink-epic", task_slug: "task-d",
			project_path: tmpDir,
		});

		const result = await handleRoster(store, {
			action: "unlink", epic_slug: "unlink-epic", task_slug: "task-d",
			project_path: tmpDir,
		});
		const data = parseResult(result);
		expect(data.message).toContain("unlinked");
	});
});

describe("roster order", () => {
	it("returns topological order", async () => {
		await handleRoster(store, {
			action: "init", epic_slug: "order-epic", project_path: tmpDir,
		});
		createSpec("task-e");
		createSpec("task-f");
		await handleRoster(store, {
			action: "link", epic_slug: "order-epic", task_slug: "task-e",
			project_path: tmpDir,
		});
		await handleRoster(store, {
			action: "link", epic_slug: "order-epic", task_slug: "task-f",
			depends_on: "task-e", project_path: tmpDir,
		});

		const result = await handleRoster(store, {
			action: "order", epic_slug: "order-epic", project_path: tmpDir,
		});
		const data = parseResult(result);
		expect(data.recommended_order.indexOf("task-e")).toBeLessThan(
			data.recommended_order.indexOf("task-f"),
		);
	});
});

describe("roster list", () => {
	it("lists all epics", async () => {
		await handleRoster(store, {
			action: "init", epic_slug: "list-a", project_path: tmpDir,
		});
		await handleRoster(store, {
			action: "init", epic_slug: "list-b", project_path: tmpDir,
		});

		const result = await handleRoster(store, {
			action: "list", project_path: tmpDir,
		});
		const data = parseResult(result);
		expect(data.count).toBe(2);
		expect(data.epics.map((e: any) => e.epic_slug).sort()).toEqual(["list-a", "list-b"]);
	});

	it("returns empty list when no epics", async () => {
		const result = await handleRoster(store, {
			action: "list", project_path: tmpDir,
		});
		const data = parseResult(result);
		expect(data.count).toBe(0);
	});
});

describe("roster update", () => {
	it("updates epic name", async () => {
		await handleRoster(store, {
			action: "init", epic_slug: "update-epic", project_path: tmpDir,
		});

		const result = await handleRoster(store, {
			action: "update", epic_slug: "update-epic", name: "New Name",
			project_path: tmpDir,
		});
		const data = parseResult(result);
		expect(data.name).toBe("New Name");
	});

	it("updates epic status", async () => {
		await handleRoster(store, {
			action: "init", epic_slug: "status-update", project_path: tmpDir,
		});

		const result = await handleRoster(store, {
			action: "update", epic_slug: "status-update", status: "in-progress",
			project_path: tmpDir,
		});
		const data = parseResult(result);
		expect(data.status).toBe("in-progress");
	});
});

describe("roster delete", () => {
	it("previews delete without confirm", async () => {
		await handleRoster(store, {
			action: "init", epic_slug: "del-epic", project_path: tmpDir,
		});

		const result = await handleRoster(store, {
			action: "delete", epic_slug: "del-epic", project_path: tmpDir,
		});
		const data = parseResult(result);
		expect(data.warning).toContain("confirm=true");
	});

	it("deletes with confirm", async () => {
		await handleRoster(store, {
			action: "init", epic_slug: "del-confirm", project_path: tmpDir,
		});

		const result = await handleRoster(store, {
			action: "delete", epic_slug: "del-confirm", confirm: true,
			project_path: tmpDir,
		});
		const data = parseResult(result);
		expect(data.deleted).toBe(true);
	});

	it("returns error for non-existent epic", async () => {
		const result = await handleRoster(store, {
			action: "delete", epic_slug: "ghost", project_path: tmpDir,
		});
		const data = parseResult(result);
		expect(data.error).toContain("not found");
	});
});

describe("roster unknown action", () => {
	it("returns error", async () => {
		const result = await handleRoster(store, {
			action: "unknown", project_path: tmpDir,
		});
		const data = parseResult(result);
		expect(data.error).toContain("unknown");
	});
});
