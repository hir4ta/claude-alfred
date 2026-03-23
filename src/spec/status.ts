import {
	type TaskStatus,
	effectiveStatus,
	readActiveState,
	transitionStatus,
	writeActiveState,
} from "./types.js";

export type StatusTrigger =
	| "manual"
	| "auto:first-edit"
	| "auto:wave-complete"
	| "auto:gate-clear"
	| "dossier:complete"
	| "dossier:defer"
	| "dossier:cancel";

/**
 * Transition a task's status with validation, persistence, and audit trail.
 * Throws on invalid transition.
 */
export function updateTaskStatus(
	projectPath: string,
	slug: string,
	newStatus: TaskStatus,
	trigger: StatusTrigger,
): void {
	const state = readActiveState(projectPath);
	const task = state.tasks.find((t) => t.slug === slug);
	if (!task) return;

	const current = effectiveStatus(task.status);
	if (current === newStatus) return; // no-op

	transitionStatus(current, newStatus); // throws on invalid

	const oldStatus = current;
	task.status = newStatus;
	if (newStatus === "done") {
		task.completed_at = new Date().toISOString();
	}
	writeActiveState(projectPath, state);

}
