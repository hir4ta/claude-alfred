import { describe, it, expect, beforeEach } from "vitest";
import { Store } from "../index.js";
import { insertAuditLog, queryAuditLog } from "../audit.js";
import { SCHEMA_VERSION } from "../schema.js";

describe("Schema V10", () => {
	let store: Store;

	beforeEach(() => {
		store = Store.open(":memory:");
	});

	it("has SCHEMA_VERSION 10", () => {
		expect(SCHEMA_VERSION).toBe(10);
	});

	it("creates audit_log table", () => {
		const row = store.db
			.prepare("SELECT name FROM sqlite_master WHERE type='table' AND name='audit_log'")
			.get() as { name: string } | undefined;
		expect(row?.name).toBe("audit_log");
	});

	it("knowledge_index has author column", () => {
		const info = store.db.prepare("PRAGMA table_info(knowledge_index)").all() as Array<{ name: string }>;
		const colNames = info.map((c) => c.name);
		expect(colNames).toContain("author");
	});
});

describe("audit_log CRUD", () => {
	let store: Store;

	beforeEach(() => {
		store = Store.open(":memory:");
		// Insert a test project
		store.db.prepare(`
			INSERT INTO projects (id, name, path, registered_at, last_seen_at)
			VALUES ('proj-1', 'Test Project', '/tmp/test', '2026-01-01', '2026-01-01')
		`).run();
	});

	it("insertAuditLog inserts entry", () => {
		insertAuditLog(store, {
			projectId: "proj-1",
			timestamp: "2026-03-21T10:00:00Z",
			event: "spec.init",
			actor: "alice",
			slug: "my-feature",
			action: "",
			detail: "{}",
		});

		const result = queryAuditLog(store, { projectId: "proj-1" });
		expect(result.total).toBe(1);
		expect(result.entries[0]!.event).toBe("spec.init");
		expect(result.entries[0]!.actor).toBe("alice");
	});

	it("UNIQUE constraint deduplicates", () => {
		const entry = {
			projectId: "proj-1",
			timestamp: "2026-03-21T10:00:00Z",
			event: "spec.init",
			actor: "alice",
			slug: "my-feature",
			action: "",
			detail: "{}",
		};

		insertAuditLog(store, entry);
		insertAuditLog(store, entry); // duplicate

		const result = queryAuditLog(store);
		expect(result.total).toBe(1);
	});

	it("filters by actor", () => {
		insertAuditLog(store, {
			projectId: "proj-1",
			timestamp: "2026-03-21T10:00:00Z",
			event: "spec.init",
			actor: "alice",
			slug: "",
			action: "",
			detail: "{}",
		});
		insertAuditLog(store, {
			projectId: "proj-1",
			timestamp: "2026-03-21T11:00:00Z",
			event: "spec.init",
			actor: "bob",
			slug: "",
			action: "",
			detail: "{}",
		});

		const result = queryAuditLog(store, { actor: "alice" });
		expect(result.total).toBe(1);
		expect(result.entries[0]!.actor).toBe("alice");
	});

	it("supports pagination", () => {
		for (let i = 0; i < 5; i++) {
			insertAuditLog(store, {
				projectId: "proj-1",
				timestamp: `2026-03-21T1${i}:00:00Z`,
				event: `event-${i}`,
				actor: "alice",
				slug: "",
				action: "",
				detail: "{}",
			});
		}

		const page1 = queryAuditLog(store, { limit: 2, offset: 0 });
		expect(page1.entries).toHaveLength(2);
		expect(page1.total).toBe(5);

		const page2 = queryAuditLog(store, { limit: 2, offset: 2 });
		expect(page2.entries).toHaveLength(2);
	});
});
