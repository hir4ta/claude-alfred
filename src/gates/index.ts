/**
 * Quality gates framework — CI-style checks triggered by hooks.
 *
 * gates.json defines commands to run:
 * - on_write: after Edit/Write (lint, typecheck)
 * - on_commit: after git commit (test, typecheck)
 */
import { existsSync, readFileSync } from "node:fs";
import { spawnSync } from "node:child_process";
import { join } from "node:path";

export interface GateCheck {
	command: string;
	timeout?: number; // ms, default 5000
	run_once_per_batch?: boolean;
}

export interface GatesConfig {
	on_write: Record<string, GateCheck | string>;
	on_commit: Record<string, GateCheck | string>;
}

export interface GateResult {
	name: string;
	passed: boolean;
	output: string; // stderr + stdout on failure
	duration: number; // ms
}

const DEFAULT_TIMEOUT = 5000;

/**
 * Load gates.json from project directory.
 * Returns null if not found.
 */
export function loadGates(cwd: string): GatesConfig | null {
	const gatesPath = join(cwd, ".alfred", "gates.json");
	if (!existsSync(gatesPath)) return null;
	try {
		return JSON.parse(readFileSync(gatesPath, "utf-8")) as GatesConfig;
	} catch {
		return null;
	}
}

/**
 * Run a single gate check.
 */
export function runGate(
	cwd: string,
	name: string,
	gate: GateCheck | string,
	file?: string,
): GateResult {
	const start = Date.now();
	const check = typeof gate === "string" ? { command: gate } : gate;
	const timeout = check.timeout ?? DEFAULT_TIMEOUT;

	// Replace {file} placeholder
	const command = file ? check.command.replace(/\{file\}/g, file) : check.command;

	const result = spawnSync("sh", ["-c", command], {
		cwd,
		timeout,
		stdio: ["ignore", "pipe", "pipe"],
		env: { ...process.env },
	});

	const duration = Date.now() - start;
	const stdout = result.stdout?.toString("utf-8") ?? "";
	const stderr = result.stderr?.toString("utf-8") ?? "";
	const passed = result.status === 0;

	return {
		name,
		passed,
		output: passed ? "" : `${stderr}\n${stdout}`.trim().slice(0, 1000),
		duration,
	};
}

/**
 * Run all gates in a group (on_write or on_commit).
 */
export function runGateGroup(
	cwd: string,
	group: Record<string, GateCheck | string>,
	file?: string,
): GateResult[] {
	const results: GateResult[] = [];
	for (const [name, gate] of Object.entries(group)) {
		results.push(runGate(cwd, name, gate, file));
	}
	return results;
}

/**
 * Auto-detect gates from project toolchain.
 */
export function detectGates(cwd: string): GatesConfig {
	const gates: GatesConfig = { on_write: {}, on_commit: {} };

	// Read package.json
	const pkgPath = join(cwd, "package.json");
	let allDeps: Record<string, string> = {};
	if (existsSync(pkgPath)) {
		try {
			const pkg = JSON.parse(readFileSync(pkgPath, "utf-8"));
			allDeps = { ...pkg.dependencies, ...pkg.devDependencies };
		} catch { /* ignore */ }
	}

	// Linter
	if (existsSync(join(cwd, "biome.json")) || existsSync(join(cwd, "biome.jsonc"))) {
		gates.on_write.lint = { command: "biome check {file} --no-errors-on-unmatched", timeout: 3000 };
	} else if (allDeps.eslint) {
		gates.on_write.lint = { command: "eslint {file}", timeout: 5000 };
	}

	// TypeScript
	if (existsSync(join(cwd, "tsconfig.json"))) {
		gates.on_write.typecheck = { command: "tsc --noEmit", timeout: 10000, run_once_per_batch: true };
	}

	// Test runner
	if (allDeps.vitest) {
		gates.on_commit.test_changed = { command: "vitest --changed --reporter=verbose", timeout: 30000 };
	} else if (allDeps.jest) {
		gates.on_commit.test_changed = { command: "jest --changedSince=HEAD~1", timeout: 30000 };
	} else if (existsSync(join(cwd, "go.mod"))) {
		gates.on_commit.test_changed = { command: "go test ./...", timeout: 60000 };
	} else if (existsSync(join(cwd, "Cargo.toml"))) {
		gates.on_commit.test_changed = { command: "cargo test", timeout: 60000 };
	}

	return gates;
}
