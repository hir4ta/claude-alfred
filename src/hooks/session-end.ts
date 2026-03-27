import { existsSync, readFileSync, rmSync } from "node:fs";
import { join } from "node:path";
import { writeHandoff } from "../state/handoff.ts";
import { readPendingFixes } from "../state/pending-fixes.ts";
import { recordOutcome } from "../state/session-outcomes.ts";
import { readSessionState } from "../state/session-state.ts";
import type { HookEvent } from "../types.ts";

/** SessionEnd: save state on any exit (complement to PreCompact for non-normal exits) */
export default async function sessionEnd(_ev: HookEvent): Promise<void> {
	try {
		const fixes = readPendingFixes();
		const changedFiles = getChangedFiles();

		writeHandoff({
			summary: "Session ended",
			changed_files: changedFiles,
			pending_fixes: fixes.length > 0,
			next_steps:
				fixes.length > 0
					? `Fix pending errors: ${fixes.map((f) => f.file).join(", ")}`
					: "Continue from where you left off",
			saved_at: new Date().toISOString(),
		});
	} catch {
		// fail-open: session is ending, don't crash
	}

	// Record session outcome
	try {
		const startFile = join(process.cwd(), ".alfred", ".state", "_session-start.json");
		if (!existsSync(startFile)) return;

		const startData = JSON.parse(readFileSync(startFile, "utf-8"));
		const endingPending = readPendingFixes().length;
		const sessionState = readSessionState();

		recordOutcome({
			session_id: startData.session_id ?? "",
			started_at: startData.started_at,
			ended_at: new Date().toISOString(),
			starting_pending: startData.starting_pending ?? 0,
			ending_pending: endingPending,
			deny_count: sessionState.session_deny_count,
			block_count: sessionState.session_block_count,
			respond_count: sessionState.session_respond_count,
			commits: 0,
			clean_exit: endingPending === 0,
		});

		rmSync(startFile, { force: true });
	} catch {
		// fail-open
	}
}

function getChangedFiles(): string[] {
	try {
		const result = Bun.spawnSync(["git", "diff", "--name-only", "HEAD"], {
			cwd: process.cwd(),
			timeout: 3000,
			stdio: ["ignore", "pipe", "pipe"],
		});
		if (result.exitCode !== 0) return [];
		return result.stdout
			.toString()
			.split("\n")
			.filter((line) => line.trim().length > 0);
	} catch {
		return [];
	}
}
