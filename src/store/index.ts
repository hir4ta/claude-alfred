import { mkdirSync } from "node:fs";
import { homedir } from "node:os";
import { dirname, join } from "node:path";
import { type DbDatabase, openDatabaseSync, pragmaSet } from "./db.js";
import { migrate, SCHEMA_VERSION } from "./schema.js";

export class Store {
	readonly db: DbDatabase;
	readonly dbPath: string;
	expectedDims = 0;

	private constructor(db: DbDatabase, dbPath: string) {
		this.db = db;
		this.dbPath = dbPath;
	}

	static open(dbPath: string): Store {
		mkdirSync(dirname(dbPath), { recursive: true });

		const db = openDatabaseSync(dbPath);

		pragmaSet(db, "journal_mode = WAL");
		pragmaSet(db, "foreign_keys = ON");
		pragmaSet(db, "synchronous = NORMAL");
		pragmaSet(db, "cache_size = -8000");
		pragmaSet(db, "mmap_size = 268435456");
		pragmaSet(db, "temp_store = MEMORY");

		const row = db.prepare("PRAGMA user_version").get() as { user_version: number } | undefined;
		const uv = row?.user_version ?? 0;
		if (uv !== SCHEMA_VERSION) {
			migrate(db);
		}

		return new Store(db, dbPath);
	}

	static openDefault(): Store {
		return Store.open(defaultDBPath());
	}

	close(): void {
		this.db.close();
	}

	schemaVersionCurrent(): number {
		try {
			const row = this.db.prepare("SELECT version FROM schema_version LIMIT 1").get() as
				| { version: number }
				| undefined;
			return row?.version ?? 0;
		} catch {
			return 0;
		}
	}
}

let cachedStore: Store | undefined;

export function openDefaultCached(): Store {
	if (!cachedStore) {
		cachedStore = Store.openDefault();
	}
	return cachedStore;
}

export function defaultDBPath(): string {
	return join(homedir(), ".claude-alfred", "alfred.db");
}
