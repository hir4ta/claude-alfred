/**
 * Data layer for TUI — reads directly from filesystem.
 */
import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";
import { parseTasksFile } from "../spec/types.js";
import { Store } from "../store/index.js";

export interface TaskItem {
	id: string;
	title: string;
	label: string;
	checked: boolean;
}

export interface WaveInfo {
	key: string;
	title: string;
	total: number;
	checked: number;
	isCurrent: boolean;
	tasks: TaskItem[];
}

export interface TaskInfo {
	slug: string;
	status: string;
	size: string;
	specType: string;
	startedAt: string;
	focus: string;
	completed: number;
	total: number;
	waves: WaveInfo[];
	projectName: string;
}

// --- State file readers ---

type TaskEntry = { slug: string; status?: string; started_at?: string; size?: string; spec_type?: string };
type StateFile = { primary?: string; tasks: TaskEntry[] };

function readJsonState(path: string): StateFile {
	try {
		return JSON.parse(readFileSync(path, "utf-8"));
	} catch {
		return { tasks: [] };
	}
}

function readActiveState(projPath: string): StateFile & { primary: string } {
	const p = join(projPath, ".alfred", "specs", "_active.json");
	const state = readJsonState(p);
	return { primary: state.primary ?? "", tasks: state.tasks };
}

function readCompleteState(projPath: string): StateFile {
	return readJsonState(join(projPath, ".alfred", "specs", "_complete.json"));
}

function readCancelState(projPath: string): StateFile {
	return readJsonState(join(projPath, ".alfred", "specs", "_cancel.json"));
}

// --- Wave + task parser ---

// --- JSON → WaveInfo conversion ---

function jsonToWaves(data: { waves: Array<{ key: number | string; title: string; tasks: Array<{ id: string; title: string; checked: boolean }> }> }): WaveInfo[] {
	const allWaves = data.waves;
	const result: WaveInfo[] = allWaves.map(w => {
		const tasks = w.tasks.map(t => ({
			id: t.id,
			title: t.title,
			label: `${t.id} ${t.title}`,
			checked: t.checked,
		}));
		const checked = tasks.filter(t => t.checked).length;
		return {
			key: String(w.key),
			title: w.title,
			total: tasks.length,
			checked,
			isCurrent: false,
			tasks,
		};
	});

	// Determine current wave
	const nonClosing = result.filter(w => w.key !== "closing");
	const firstIncomplete = nonClosing.find(w => w.checked < w.total);
	if (firstIncomplete) {
		firstIncomplete.isCurrent = true;
	} else {
		const closing = result.find(w => w.key === "closing");
		if (closing && closing.checked < closing.total) closing.isCurrent = true;
	}

	return result;
}

// --- Load tasks ---

export function loadTasks(projPath: string, projName: string, opts?: { showAll?: boolean }): TaskInfo[] {
	const state = readActiveState(projPath);
	const tasks: TaskInfo[] = state.tasks.map(task => buildTaskInfo(projPath, projName, task));

	if (opts?.showAll) {
		for (const task of readCompleteState(projPath).tasks) {
			tasks.push(buildTaskInfo(projPath, projName, task));
		}
		for (const task of readCancelState(projPath).tasks) {
			tasks.push(buildTaskInfo(projPath, projName, task));
		}
	}

	return tasks;
}

function buildTaskInfo(projPath: string, projName: string, task: TaskEntry): TaskInfo {
	let waves: WaveInfo[] = [];
	let focus = "";
	let completed = 0;
	let total = 0;

	try {
		const raw = readFileSync(join(projPath, ".alfred", "specs", task.slug, "tasks.json"), "utf-8");
		const data = parseTasksFile(raw);
		waves = jsonToWaves(data);
		for (const w of waves) { completed += w.checked; total += w.total; }
		const cur = waves.find(w => w.isCurrent);
		if (cur) focus = cur.title;
	} catch {
		// No tasks.json — for completed/cancelled specs, show as 100%
		if (task.status === "completed" || task.status === "cancelled") {
			completed = 1;
			total = 1;
		}
	}

	return {
		slug: task.slug,
		status: task.status ?? "active",
		size: task.size ?? "M",
		specType: task.spec_type ?? "feature",
		startedAt: task.started_at ?? "",
		focus,
		completed,
		total,
		waves,
		projectName: projName,
	};
}

// --- Project resolution (cross-project) ---

export function resolveAllProjects(store: Store): Array<{ path: string; name: string }> {
	const results: Array<{ path: string; name: string }> = [];
	const seen = new Set<string>();

	const cwd = process.cwd();
	// CWD first if it has .alfred/specs/
	if (existsSync(join(cwd, ".alfred", "specs"))) {
		const row = store.db.prepare("SELECT name FROM projects WHERE path = ? LIMIT 1").get(cwd) as { name: string } | undefined;
		results.push({ path: cwd, name: row?.name ?? cwd.split("/").pop() ?? "project" });
		seen.add(cwd);
	}

	// All active DB projects that have .alfred/specs/
	const rows = store.db
		.prepare("SELECT name, path FROM projects WHERE status = 'active' ORDER BY last_seen_at DESC")
		.all() as Array<{ name: string; path: string }>;
	for (const row of rows) {
		if (seen.has(row.path)) continue;
		if (existsSync(join(row.path, ".alfred", "specs"))) {
			results.push({ path: row.path, name: row.name });
			seen.add(row.path);
		}
	}

	return results;
}

/** @deprecated Use resolveAllProjects for cross-project support */
export function resolveProject(store: Store): { path: string; name: string } {
	const all = resolveAllProjects(store);
	return all[0] ?? { path: process.cwd(), name: process.cwd().split("/").pop() ?? "unknown" };
}
