import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";
import type { GatesConfig } from "../types.ts";

/** Auto-detect gates from project configuration files */
export function detectGates(projectRoot: string): GatesConfig {
	const gates: GatesConfig = { on_write: {}, on_commit: {} };

	// Linter
	if (existsSync(join(projectRoot, "biome.json")) || existsSync(join(projectRoot, "biome.jsonc"))) {
		gates.on_write!.lint = {
			command: "biome check {file} --no-errors-on-unmatched",
			timeout: 3000,
		};
	} else if (
		existsSync(join(projectRoot, ".eslintrc.json")) ||
		existsSync(join(projectRoot, ".eslintrc.js")) ||
		existsSync(join(projectRoot, "eslint.config.js")) ||
		existsSync(join(projectRoot, "eslint.config.mjs"))
	) {
		gates.on_write!.lint = { command: "eslint {file}", timeout: 5000 };
	}

	// TypeScript
	if (existsSync(join(projectRoot, "tsconfig.json"))) {
		gates.on_write!.typecheck = {
			command: "tsc --noEmit",
			timeout: 10000,
			run_once_per_batch: true,
		};
	}

	// Test framework
	const pkgPath = join(projectRoot, "package.json");
	if (existsSync(pkgPath)) {
		try {
			const pkg = JSON.parse(readFileSync(pkgPath, "utf-8"));
			const deps = { ...pkg.dependencies, ...pkg.devDependencies };

			if (deps.vitest) {
				gates.on_commit!.test = {
					command: "bunx --bun vitest --changed --reporter=verbose",
					timeout: 30000,
				};
			} else if (deps.jest) {
				gates.on_commit!.test = {
					command: "jest --changedSince=HEAD~1",
					timeout: 30000,
				};
			}
		} catch {
			// ignore parse errors
		}
	}

	// Python
	const isPython =
		existsSync(join(projectRoot, "pyproject.toml")) ||
		existsSync(join(projectRoot, "uv.lock")) ||
		existsSync(join(projectRoot, "setup.py"));
	if (isPython) {
		const hasUv = existsSync(join(projectRoot, "uv.lock"));
		const prefix = hasUv ? "uv run " : "";

		// Linter: ruff (fast, Astral)
		if (!gates.on_write!.lint) {
			if (existsSync(join(projectRoot, "ruff.toml")) || hasPyprojectKey(projectRoot, "ruff")) {
				gates.on_write!.lint = { command: `${prefix}ruff check {file}`, timeout: 3000 };
			}
		}
		// Type checker: pyright (fast) > mypy
		if (
			hasPyprojectKey(projectRoot, "pyright") ||
			existsSync(join(projectRoot, "pyrightconfig.json"))
		) {
			gates.on_write!.typecheck = {
				command: `${prefix}pyright`,
				timeout: 30000,
				run_once_per_batch: true,
			};
		} else if (existsSync(join(projectRoot, "mypy.ini")) || hasPyprojectKey(projectRoot, "mypy")) {
			gates.on_write!.typecheck = {
				command: `${prefix}mypy .`,
				timeout: 30000,
				run_once_per_batch: true,
			};
		}
		// Test: pytest
		if (!gates.on_commit!.test) {
			gates.on_commit!.test = { command: `${prefix}pytest`, timeout: 30000 };
		}
	}

	// Go
	if (existsSync(join(projectRoot, "go.mod"))) {
		if (!gates.on_write!.lint) {
			gates.on_write!.lint = { command: "go vet ./...", timeout: 5000, run_once_per_batch: true };
		}
		if (!gates.on_commit!.test) {
			gates.on_commit!.test = { command: "go test ./...", timeout: 30000 };
		}
	}

	// Rust
	if (existsSync(join(projectRoot, "Cargo.toml"))) {
		if (!gates.on_write!.lint) {
			gates.on_write!.lint = {
				command: "cargo clippy -- -D warnings",
				timeout: 30000,
				run_once_per_batch: true,
			};
		}
		if (!gates.on_commit!.test) {
			gates.on_commit!.test = { command: "cargo test", timeout: 60000 };
		}
	}

	return gates;
}

function hasPyprojectKey(root: string, key: string): boolean {
	try {
		const content = readFileSync(join(root, "pyproject.toml"), "utf-8");
		return content.includes(`[tool.${key}]`);
	} catch {
		return false;
	}
}
