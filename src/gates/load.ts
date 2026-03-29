import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";
import type { GatesConfig } from "../types.ts";

// Process-scoped cache (read-only, no dirty flag needed)
let _cache: GatesConfig | null | undefined;

/** Load gates.json from .qult/ directory. Returns null if not found (fail-open). */
export function loadGates(): GatesConfig | null {
	if (_cache !== undefined) return _cache;
	try {
		const path = join(process.cwd(), ".qult", "gates.json");
		if (!existsSync(path)) {
			_cache = null;
			return null;
		}
		const parsed: GatesConfig = JSON.parse(readFileSync(path, "utf-8"));
		_cache = parsed;
		return parsed;
	} catch {
		_cache = null;
		return null;
	}
}

/** Reset cache — for testing only. */
export function resetGatesCache(): void {
	_cache = undefined;
}
