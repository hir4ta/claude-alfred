import { existsSync, readFileSync } from "node:fs";
import { join, resolve } from "node:path";
import { readActiveState } from "../spec/types.js";

export interface SpecState {
	slug: string;
	size: string;
	status: string;
}

const VALID_SIZES = new Set(["S", "M", "L", ""]);

/**
 * Read active spec state from _active.json via proper YAML parsing.
 * Returns null on any error (NFR-2: fail-open).
 */
export function tryReadActiveSpec(cwd: string | undefined): SpecState | null {
	if (!cwd) return null;
	try {
		return parseSpecState(cwd);
	} catch {
		return null; // NFR-2: fail-open
	}
}

/**
 * Check if _active.json exists but cannot be parsed or has invalid enum values.
 * Used by PreToolUse to deny edits instead of silently allowing.
 * Returns false when primary is empty (no active spec = valid state).
 */
export function isActiveSpecMalformed(cwd: string | undefined): boolean {
	if (!cwd) return false;
	const path = join(cwd, ".alfred", "specs", "_active.json");
	if (!existsSync(path)) return false;
	try {
		const state = readActiveState(cwd);
		// No active spec (primary is empty) is a valid state, not malformed.
		if (!state.primary) return false;
		return parseSpecState(cwd) === null;
	} catch {
		return true; // file exists but can't be read/parsed
	}
}

/** Shared parsing logic — single readActiveState call for both functions. */
function parseSpecState(cwd: string): SpecState | null {
	const state = readActiveState(cwd);
	if (!state.primary) return null;
	const task = state.tasks.find((t) => t.slug === state.primary);
	if (!task) return null;
	const size = task.size ?? "";
	const status = task.status ?? "pending";
	if (!VALID_SIZES.has(size)) return null;
	return { slug: task.slug, size, status };
}

/**
 * Check if file_path is under .alfred/ directory (spec/config files should not be blocked).
 */
export function isSpecFilePath(cwd: string | undefined, filePath: string): boolean {
	if (!cwd || !filePath) return false;
	const resolved = resolve(cwd, filePath);
	const alfredDir = join(cwd, ".alfred");
	return resolved.startsWith(`${alfredDir}/`) || resolved === alfredDir;
}

/**
 * Count unchecked task checkboxes (`- [ ]`) in tasks.md.
 */
export function countUncheckedTasks(cwd: string | undefined, slug: string): number {
	if (!cwd) return 0;
	try {
		const tasks = readFileSync(join(cwd, ".alfred", "specs", slug, "tasks.md"), "utf-8");
		return (tasks.match(/^- \[ \] /gm) ?? []).length;
	} catch {
		return 0;
	}
}

/**
 * Check if tasks.md has unchecked self-review items.
 */
export function hasUncheckedSelfReview(cwd: string | undefined, slug: string): boolean {
	if (!cwd) return false;
	try {
		const tasks = readFileSync(join(cwd, ".alfred", "specs", slug, "tasks.md"), "utf-8");
		return tasks.split("\n").some(
			(line) =>
				line.startsWith("- [ ] ") && (/セルフレビュー/i.test(line) || /self-review/i.test(line)),
		);
	} catch {
		return false;
	}
}

/**
 * PreToolUse: explicitly allow tool via permissionDecision JSON (exit 0).
 * Signals to Claude Code that this edit is permitted, so subsequent hooks
 * (e.g. prompt-based spec-first judge) can be skipped.
 */
export function allowTool(reason: string): void {
	const out = {
		hookSpecificOutput: {
			hookEventName: "PreToolUse",
			permissionDecision: "allow",
			permissionDecisionReason: reason,
		},
	};
	process.stdout.write(`${JSON.stringify(out)}\n`);
}

/**
 * PreToolUse: deny tool via permissionDecision JSON (exit 0).
 */
export function denyTool(reason: string): void {
	const out = {
		hookSpecificOutput: {
			hookEventName: "PreToolUse",
			permissionDecision: "deny",
			permissionDecisionReason: reason,
		},
	};
	process.stdout.write(`${JSON.stringify(out)}\n`);
}

/**
 * Stop: block Claude from stopping via decision JSON.
 */
export function blockStop(reason: string): void {
	const out = { decision: "block", reason };
	process.stdout.write(`${JSON.stringify(out)}\n`);
}
