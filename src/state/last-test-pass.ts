import { existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { join } from "node:path";

const STATE_DIR = ".alfred/.state";
const FILE = "last-test-pass.json";

interface TestPassState {
	passed_at: string;
	command: string;
}

function filePath(): string {
	return join(process.cwd(), STATE_DIR, FILE);
}

/** Read last test pass state. Returns null if not found or on error. */
export function readLastTestPass(): TestPassState | null {
	try {
		const path = filePath();
		if (!existsSync(path)) return null;
		return JSON.parse(readFileSync(path, "utf-8"));
	} catch {
		return null;
	}
}

/** Record that tests passed successfully. */
export function recordTestPass(command: string): void {
	try {
		const dir = join(process.cwd(), STATE_DIR);
		if (!existsSync(dir)) mkdirSync(dir, { recursive: true });
		const state: TestPassState = { passed_at: new Date().toISOString(), command };
		writeFileSync(filePath(), JSON.stringify(state, null, 2));
	} catch {
		// fail-open
	}
}

/** Clear test pass state (e.g. after git commit). */
export function clearTestPass(): void {
	try {
		const path = filePath();
		if (existsSync(path)) rmSync(path);
	} catch {
		// fail-open
	}
}
