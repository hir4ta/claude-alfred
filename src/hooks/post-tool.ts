import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";
import { runGate } from "../gates/runner.ts";
import { writePace } from "../state/pace.ts";
import { readPendingFixes, writePendingFixes } from "../state/pending-fixes.ts";
import type { GatesConfig, HookEvent, HookResponse, PendingFix } from "../types.ts";

/** PostToolUse: lint/type gate after Edit/Write, test gate after git commit */
export default async function postTool(ev: HookEvent): Promise<void> {
	const tool = ev.tool_name;
	if (!tool) return;

	if (tool === "Edit" || tool === "Write") {
		handleEditWrite(ev);
	} else if (tool === "Bash") {
		handleBash(ev);
	}
}

function handleEditWrite(ev: HookEvent): void {
	const file = typeof ev.tool_input?.file_path === "string" ? ev.tool_input.file_path : null;
	if (!file) return;

	const gates = loadGates();
	if (!gates?.on_write) return;

	// Read existing fixes for OTHER files, merge with new results for THIS file
	const existingFixes = readPendingFixes().filter((f) => f.file !== file);
	const newFixes: PendingFix[] = [];
	const messages: string[] = [];

	for (const [name, gate] of Object.entries(gates.on_write)) {
		const hasPlaceholder = gate.command.includes("{file}");
		const result = runGate(name, gate, hasPlaceholder ? file : undefined);

		if (!result.passed) {
			newFixes.push({ file, errors: [result.output], gate: name });
			messages.push(`[${name}] ${result.output.slice(0, 200)}`);
		}
	}

	writePendingFixes([...existingFixes, ...newFixes]);

	if (newFixes.length > 0) {
		respond(`Fix these errors before continuing:\n${messages.join("\n")}`);
	}
}

function handleBash(ev: HookEvent): void {
	const command = typeof ev.tool_input?.command === "string" ? ev.tool_input.command : null;
	if (!command) return;

	// Detect git commit → reset pace
	if (/\bgit\s+commit\b/.test(command)) {
		writePace({ last_commit_at: new Date().toISOString(), changed_files: 0, tool_calls: 0 });

		const gates = loadGates();
		if (!gates?.on_commit) return;

		const messages: string[] = [];
		for (const [name, gate] of Object.entries(gates.on_commit)) {
			const result = runGate(name, gate);
			if (!result.passed) {
				messages.push(`[${name}] ${result.output.slice(0, 200)}`);
			}
		}

		if (messages.length > 0) {
			respond(`Tests failed after commit:\n${messages.join("\n")}`);
		}
	}
}

function loadGates(): GatesConfig | null {
	try {
		const path = join(process.cwd(), ".alfred", "gates.json");
		if (!existsSync(path)) return null;
		return JSON.parse(readFileSync(path, "utf-8"));
	} catch {
		return null;
	}
}

function respond(context: string): void {
	const response: HookResponse = {
		hookSpecificOutput: {
			additionalContext: context,
		},
	};
	process.stdout.write(JSON.stringify(response));
}
