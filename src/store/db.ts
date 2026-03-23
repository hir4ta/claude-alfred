/**
 * Database adapter — abstracts better-sqlite3 (Node.js) and bun:sqlite (Bun).
 *
 * API surface used by the store:
 *   .prepare(sql) → Statement { .run(), .get(), .all() }
 *   .exec(sql)
 *   .transaction(fn)
 *   .close()
 *
 * Differences handled:
 *   - `.pragma()` — only in better-sqlite3; bun:sqlite uses db.run("PRAGMA ...")
 *   - BLOB return type — better-sqlite3: Buffer, bun:sqlite: Uint8Array
 *   - Import path — "better-sqlite3" vs "bun:sqlite"
 */

/** Minimal statement interface covering what the store needs. */
export interface DbStatement {
	run(...params: unknown[]): { changes: number; lastInsertRowid: number | bigint };
	get(...params: unknown[]): unknown;
	all(...params: unknown[]): unknown[];
}

/** Minimal database interface covering what the store needs. */
export interface DbDatabase {
	prepare(sql: string): DbStatement;
	exec(sql: string): void;
	transaction<T>(fn: (...args: unknown[]) => T): (...args: unknown[]) => T;
	close(): void;
}

// biome-ignore lint/suspicious/noExplicitAny: globalThis.Bun is Bun-only
export const isBun = typeof (globalThis as any).Bun !== "undefined";

/**
 * Open a SQLite database synchronously.
 * On Bun: uses bun:sqlite (built-in). On Node.js: uses better-sqlite3.
 */
export function openDatabaseSync(dbPath: string): DbDatabase {
	if (isBun) {
		return openBunSync(dbPath);
	}
	return openNodeSync(dbPath);
}

// --- Bun implementation ---

function openBunSync(dbPath: string): DbDatabase {
	// Dynamic require to avoid bundler resolution on Node.js
	// biome-ignore lint/suspicious/noExplicitAny: bun:sqlite is Bun-only
	const { Database } = require("bun:sqlite") as any;
	return wrapAnyDatabase(new Database(dbPath), /* isBun */ true);
}

// --- Node.js (better-sqlite3) implementation ---

function openNodeSync(dbPath: string): DbDatabase {
	// biome-ignore lint/suspicious/noExplicitAny: dynamic require for better-sqlite3
	const BetterSqlite3 = require("better-sqlite3") as any;
	return wrapAnyDatabase(new BetterSqlite3(dbPath), /* isBun */ false);
}

/**
 * Wrap any SQLite database instance (better-sqlite3 or bun:sqlite) into DbDatabase.
 * Both have nearly identical APIs; the only exec() difference is handled here.
 */
// biome-ignore lint/suspicious/noExplicitAny: wrapping two different DB drivers
function wrapAnyDatabase(db: any, bun: boolean): DbDatabase {
	return {
		prepare(sql: string): DbStatement {
			const stmt = db.prepare(sql);
			return {
				run(...params: unknown[]) {
					return stmt.run(...params);
				},
				get(...params: unknown[]) {
					return stmt.get(...params);
				},
				all(...params: unknown[]) {
					return stmt.all(...params);
				},
			};
		},
		exec(sql: string): void {
			// bun:sqlite: db.exec is alias for db.run (returns value)
			// better-sqlite3: db.exec runs multi-statement SQL (void)
			if (bun) {
				db.run(sql);
			} else {
				db.exec(sql);
			}
		},
		transaction<T>(fn: (...args: unknown[]) => T): (...args: unknown[]) => T {
			return db.transaction(fn);
		},
		close(): void {
			db.close();
		},
	};
}

/**
 * Run a PRAGMA and return the result.
 */
export function pragma(db: DbDatabase, statement: string): unknown {
	return db.prepare(`PRAGMA ${statement}`).get();
}

/**
 * Set a PRAGMA (fire-and-forget, no return value needed).
 */
export function pragmaSet(db: DbDatabase, statement: string): void {
	db.exec(`PRAGMA ${statement}`);
}
