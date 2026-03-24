import { createCliRenderer } from "@opentui/core";
import { createRoot, useKeyboard, useRenderer, useTerminalDimensions } from "@opentui/react";
import { useState, useEffect, useCallback, createElement } from "react";
import { openDefaultCached } from "../store/index.js";
import { type TaskInfo, loadTasks, resolveAllProjects } from "./data.js";

// --- Gruvbox Material Dark (medium) ---
const C = {
	bg: "#282828",
	bg1: "#32302f",
	bg3: "#45403d",
	bg5: "#5a524c",
	fg: "#d4be98",
	fg1: "#ddc7a1",
	red: "#ea6962",
	orange: "#e78a4e",
	yellow: "#d8a657",
	green: "#a9b665",
	aqua: "#89b482",
	blue: "#7daea3",
	purple: "#d3869b",
	grey0: "#7c6f64",
	grey1: "#928374",
	grey2: "#a89984",
};

// --- Shimmer ---

function useShimmer(speed = 150) {
	const [frame, setFrame] = useState(0);
	useEffect(() => {
		const id = setInterval(() => setFrame((f) => f + 1), speed);
		return () => clearInterval(id);
	}, [speed]);
	return frame;
}

function ShimmerText({ text, baseColor = C.orange, brightColor = C.yellow, speed = 120, width = 3 }: {
	text: string; baseColor?: string; brightColor?: string; speed?: number; width?: number;
}) {
	const frame = useShimmer(speed);
	const len = text.length;
	const cycle = len + width + 4;
	const pos = frame % cycle;
	const hlStart = Math.max(0, pos - width);
	const hlEnd = Math.min(len, pos);
	const before = text.slice(0, hlStart);
	const highlight = text.slice(hlStart, hlEnd);
	const after = text.slice(hlEnd);

	return (
		<box style={{ flexDirection: "row", height: 1 }}>
			{before && <text content={before} fg={baseColor} />}
			{highlight && <text content={highlight} fg={brightColor} />}
			{after && <text content={after} fg={baseColor} />}
		</box>
	);
}

// --- Components ---

function ProgressBar({ value, total, width = 20, showPercent = false, color: overrideColor }: {
	value: number; total: number; width?: number; showPercent?: boolean; color?: string;
}) {
	const pct = total > 0 ? value / total : 0;
	const filled = Math.round(pct * width);
	const color = overrideColor ?? (pct >= 1 ? C.green : C.purple);
	const label = showPercent ? `${Math.round(pct * 100)}%` : `${value}/${total}`;

	return (
		<text>
			<span fg={color}>{"━".repeat(filled)}</span>
			<span fg={C.bg5}>{"─".repeat(width - filled)}</span>
			<span fg={C.grey1}> {label}</span>
		</text>
	);
}

function StatusBadge({ status }: { status: string }) {
	const map: Record<string, { label: string; fg: string; bg: string }> = {
		active: { label: " active ", fg: C.bg, bg: C.green },
		"in-progress": { label: " active ", fg: C.bg, bg: C.green },
		completed: { label: " done ", fg: C.fg, bg: C.bg5 },
		cancelled: { label: " cancel ", fg: C.bg, bg: C.red },
		deferred: { label: " defer ", fg: C.bg, bg: C.yellow },
	};
	const { label, fg, bg } = map[status] ?? { label: ` ${status} `, fg: C.bg, bg: C.grey0 };
	return <text content={label} fg={fg} bg={bg} />;
}

// --- Header with ASCII title ---

function Header({ projCount, taskCount, filterMode, filterText }: {
	projCount: number; taskCount: number; filterMode: boolean; filterText: string;
}) {
	const { width: cols } = useTerminalDimensions();
	const wide = cols >= 60;

	return (
		<box style={{ flexDirection: "column", paddingX: 1 }}>
			{wide ? (
				/* ASCII Font title + stats */
				<box style={{ flexDirection: "row" }}>
					<ascii-font text="alfred" font="tiny" color={C.aqua} />
					<box style={{ flexDirection: "column", paddingLeft: 2, justifyContent: "flex-end" }}>
						<text>
							<span fg={C.grey1}>{projCount} project{projCount !== 1 ? "s" : ""} · </span>
							<span fg={C.fg}>{taskCount} spec{taskCount !== 1 ? "s" : ""}</span>
						</text>
					</box>
				</box>
			) : (
				/* Compact fallback for narrow terminals */
				<box style={{ height: 1 }}>
					<text>
						<b fg={C.aqua}>alfred</b>
						<span fg={C.bg5}>{" │ "}</span>
						<span fg={C.grey1}>{projCount} proj · </span>
						<span fg={C.fg}>{taskCount} spec{taskCount !== 1 ? "s" : ""}</span>
					</text>
				</box>
			)}
			{/* Filter bar (only shown when active) */}
			{filterMode && (
				<box style={{ height: 1, flexDirection: "row" }}>
					<text>
						<span fg={C.yellow}> / </span>
						<span fg={C.fg1}>{filterText}</span>
						<span fg={C.grey0}>▎</span>
					</text>
				</box>
			)}
		</box>
	);
}

// --- Project Tabs ---

function ProjectTabs({ projects, selectedIdx, onSelect }: {
	projects: Array<{ name: string }>; selectedIdx: number; onSelect: (i: number) => void;
}) {
	if (projects.length <= 1) return null;
	return (
		<box style={{ flexDirection: "row", paddingX: 1, height: 1, gap: 1 }}>
			{projects.map((p, i) => {
				const isActive = i === selectedIdx;
				return (
					<text
						key={p.name}
						content={isActive ? ` ${p.name} ` : ` ${p.name} `}
						fg={isActive ? C.bg : C.grey1}
						bg={isActive ? C.aqua : C.bg3}
					/>
				);
			})}
		</box>
	);
}

// --- Spec List (left panel) ---

function SpecList({ tasks, selectedIdx }: { tasks: TaskInfo[]; selectedIdx: number }) {
	if (tasks.length === 0) {
		return (
			<box style={{ borderStyle: "rounded", borderColor: C.bg5, flexDirection: "column", flexGrow: 1, padding: 1 }}>
				<text fg={C.grey1}>No specs found.</text>
			</box>
		);
	}

	return (
		<box style={{ borderStyle: "rounded", borderColor: C.bg5, flexDirection: "column", flexGrow: 1, overflow: "hidden" }}>
			{tasks.map((task, i) => {
				const isSelected = i === selectedIdx;
				const bg = isSelected ? C.bg3 : undefined;
				const indicator = isSelected ? "▸" : " ";

				return (
					<box key={`${task.projectName}/${task.slug}`} style={{ paddingX: 1, backgroundColor: bg, flexDirection: "column" }}>
						{/* Line 1: indicator + spec name */}
						<box style={{ flexDirection: "row", height: 1 }}>
							<text content={`${indicator} ${task.slug}`} fg={isSelected ? C.fg1 : C.fg} />
						</box>
						{/* Line 2: project name */}
						<box style={{ paddingLeft: 2, height: 1 }}>
							<text content={task.projectName} fg={C.grey0} />
						</box>
						{/* Line 3: progress */}
						<box style={{ paddingLeft: 2, height: 1 }}>
							<ProgressBar value={task.completed} total={task.total} width={10} showPercent />
						</box>
						{/* Line 4: status badge */}
						<box style={{ paddingLeft: 2, height: 1 }}>
							<StatusBadge status={task.status} />
						</box>
					</box>
				);
			})}
		</box>
	);
}

// --- Spec Detail (right panel) ---

function SpecDetail({ task, focused }: { task: TaskInfo; focused: boolean }) {
	return (
		<box style={{ borderStyle: "rounded", borderColor: focused ? C.aqua : C.bg5, flexDirection: "column", flexGrow: 1, overflow: "hidden" }} title={task.slug}>
			<scrollbox focused={focused} style={{ contentOptions: { flexDirection: "column", padding: 1, gap: 1 } }}>
				{/* Spec header */}
				<box style={{ flexDirection: "row", gap: 1, height: 1 }}>
					<text content={task.slug} fg={C.fg1} />
					<StatusBadge status={task.status} />
					<text content={`${task.size}`} fg={C.grey0} />
					{task.startedAt && <text content={`started ${task.startedAt.slice(0, 10)}`} fg={C.bg5} />}
				</box>

				{/* Overall progress */}
				<ProgressBar value={task.completed} total={task.total} width={30} showPercent />

				{/* Separator */}
				<text content={"─".repeat(50)} fg={C.bg5} />

				{task.waves.length === 0 && (
					<text fg={C.grey1}>
						{task.status === "completed" ? "✓ Completed" : task.status === "cancelled" ? "✗ Cancelled" : "No tasks.json"}
					</text>
				)}

				{task.waves.map((wave) => {
					const done = wave.total > 0 && wave.checked === wave.total;
					const isCur = wave.isCurrent;
					const isClosing = wave.key === "closing";
					const waveLabel = isClosing ? "Closing" : `Wave ${wave.key}`;
					const labelColor = isClosing ? C.red : done ? C.green : isCur ? C.orange : C.grey1;
					const titleColor = isClosing ? C.grey1 : done ? C.grey1 : isCur ? C.fg1 : C.grey0;
					const barColor = isClosing ? C.red : done ? C.green : C.purple;

					return (
						<box key={wave.key} style={{ flexDirection: "column" }}>
							{isCur
								? <ShimmerText text={`▸ ${waveLabel}: ${wave.title}`} speed={100} width={5} />
								: (
									<text>
										<span fg={labelColor}>{`  ${waveLabel}`}</span>
										<span fg={titleColor}>{`: ${wave.title}`}</span>
									</text>
								)
							}
							<box style={{ paddingLeft: 4 }}>
								<ProgressBar value={wave.checked} total={wave.total} width={20} color={barColor} />
							</box>
							{wave.tasks?.map((t) => {
								let icon: string;
								let iconColor: string;
								let idColor: string;
								let titleColor: string;
								if (t.checked) {
									icon = "✓";
									iconColor = isClosing ? C.grey1 : C.green;
									idColor = isClosing ? C.grey0 : C.aqua;
									titleColor = isClosing ? C.grey1 : C.fg;
								} else if (isCur) {
									icon = "○";
									iconColor = C.yellow;
									idColor = C.aqua;
									titleColor = C.fg1;
								} else {
									icon = "·";
									iconColor = C.bg5;
									idColor = C.bg5;
									titleColor = C.bg5;
								}
								const paddedId = t.id.padEnd(7);
								return (
									<box key={t.id} style={{ paddingLeft: 4, flexDirection: "row" }}>
										<text>
											<span fg={iconColor}>{`${icon} `}</span>
											<span fg={idColor}>{paddedId}</span>
										</text>
										<box style={{ flexShrink: 1 }}>
											<text content={t.title} fg={titleColor} />
										</box>
									</box>
								);
							})}
						</box>
					);
				})}
			</scrollbox>
		</box>
	);
}

// --- App ---

export { App };

function App({ showAll = false }: { showAll?: boolean }) {
	const [allProjects, setAllProjects] = useState<Array<{ path: string; name: string }>>([]);
	const [projIdx, setProjIdx] = useState(-1); // -1 = all projects
	const [tasks, setTasks] = useState<TaskInfo[]>([]);
	const [selectedIdx, setSelectedIdx] = useState(0);
	const [detailFocused, setDetailFocused] = useState(false);
	const [filterMode, setFilterMode] = useState(false);
	const [filterText, setFilterText] = useState("");

	const refresh = useCallback(() => {
		const store = openDefaultCached();
		const projects = resolveAllProjects(store);
		setAllProjects(projects);

		const targetProjects = projIdx === -1 ? projects : projects[projIdx] ? [projects[projIdx]] : projects;
		const allTasks: TaskInfo[] = [];
		for (const proj of targetProjects) {
			allTasks.push(...loadTasks(proj.path, proj.name, { showAll }));
		}

		// Sort by startedAt descending (newest first)
		allTasks.sort((a, b) => (b.startedAt || "").localeCompare(a.startedAt || ""));

		let filtered = showAll ? allTasks : allTasks.filter((t) => t.status !== "done" && t.status !== "completed" && t.status !== "cancelled");

		if (filterText) {
			const q = filterText.toLowerCase();
			filtered = filtered.filter((t) => t.slug.toLowerCase().includes(q) || t.projectName.toLowerCase().includes(q));
		}

		setTasks(filtered);
	}, [showAll, projIdx, filterText]);

	useEffect(() => {
		refresh();
		const interval = setInterval(refresh, 3000);
		return () => clearInterval(interval);
	}, [refresh]);

	const renderer = useRenderer();
	useKeyboard((key) => {
		if (key.ctrl && key.name === "c") {
			renderer.destroy();
			process.exit(0);
		}

		// Filter mode
		if (filterMode) {
			if (key.name === "escape" || key.name === "return") {
				setFilterMode(false);
				return;
			}
			if (key.name === "backspace") {
				setFilterText((t) => t.slice(0, -1));
				return;
			}
			if (key.raw && key.raw.length === 1 && key.raw.charCodeAt(0) >= 32) {
				setFilterText((t) => t + key.raw);
				return;
			}
			return;
		}

		// Detail scroll mode
		if (detailFocused) {
			if (key.name === "escape" || key.name === "q") {
				setDetailFocused(false);
			}
			return;
		}

		// Normal mode
		if (key.name === "q") {
			renderer.destroy();
			process.exit(0);
		}
		if (key.name === "j" || key.name === "down") {
			setSelectedIdx((prev) => Math.min(prev + 1, tasks.length - 1));
		} else if (key.name === "k" || key.name === "up") {
			setSelectedIdx((prev) => Math.max(prev - 1, 0));
		} else if (key.name === "return" && tasks.length > 0) {
			setDetailFocused(true);
		} else if (key.raw === "/") {
			setFilterMode(true);
		} else if (key.name === "tab" && !key.shift) {
			// Next project tab
			setProjIdx((i) => {
				const max = allProjects.length - 1;
				return i >= max ? -1 : i + 1;
			});
			setSelectedIdx(0);
		} else if (key.name === "tab" && key.shift) {
			// Prev project tab
			setProjIdx((i) => {
				const max = allProjects.length - 1;
				return i === -1 ? max : i - 1;
			});
			setSelectedIdx(0);
		}
	});

	const selected = tasks[selectedIdx];
	const projTabs = [{ name: "all" }, ...allProjects];
	const activeTabIdx = projIdx + 1; // -1 → 0 (all)

	const helpParts: string[] = [];
	if (!detailFocused) {
		helpParts.push("j/k select");
		helpParts.push("enter detail");
		helpParts.push("/ filter");
		if (allProjects.length > 1) helpParts.push("tab project");
		helpParts.push("q quit");
	}
	const helpText = helpParts.join(" · ");

	return (
		<box style={{ flexDirection: "column", width: "100%", height: "100%" }}>
			{/* Header — fixed, never shrinks */}
			<box style={{ flexShrink: 0 }}>
				<Header
					projCount={allProjects.length}
					taskCount={tasks.length}
					filterMode={filterMode}
					filterText={filterText}
				/>
			</box>

			{/* Project tabs — fixed */}
			{allProjects.length > 1 && (
				<box style={{ flexShrink: 0 }}>
					<ProjectTabs
						projects={projTabs}
						selectedIdx={activeTabIdx}
						onSelect={(i) => { setProjIdx(i - 1); setSelectedIdx(0); }}
					/>
				</box>
			)}

			{/* Main content — fills remaining space, height:0 prevents content from expanding */}
			<box style={{ height: 0, flexGrow: 1, paddingX: 1, paddingBottom: 1, flexDirection: "row", gap: 1, overflow: "hidden" }}>
				<box style={{ width: "30%", flexDirection: "column", overflow: "hidden" }}>
					<SpecList tasks={tasks} selectedIdx={selectedIdx} />
				</box>
				<box style={{ width: "70%", flexDirection: "column", overflow: "hidden" }}>
					{selected
						? <SpecDetail task={selected} focused={detailFocused} />
						: (
							<box style={{ borderStyle: "rounded", borderColor: C.bg5, flexGrow: 1, padding: 2, justifyContent: "center", alignItems: "center" }}>
								<text fg={C.grey1}>No specs to display.</text>
								{filterText && <text fg={C.grey0}>Filter: "{filterText}" — press / to search</text>}
							</box>
						)
					}
				</box>
			</box>

			{/* Footer — fixed, never shrinks */}
			<box style={{ flexShrink: 0, paddingX: 1, height: 1 }}>
				<text content={helpText} fg={C.grey0} />
			</box>
		</box>
	);
}

// --- Entry point ---
export function runTui(opts?: { showAll?: boolean }) {
	const showAll = opts?.showAll ?? false;
	return new Promise<void>((resolve, reject) => {
		createCliRenderer({
			exitOnCtrlC: true,
			onDestroy: () => {
				process.stdout.write("\x1b[?1000l\x1b[?1003l\x1b[?1006l");
				resolve();
			},
		}).then((renderer) => {
			process.once("uncaughtException", (err) => {
				renderer.destroy();
				console.error(err);
				process.exit(1);
			});
			process.once("unhandledRejection", (err) => {
				renderer.destroy();
				console.error(err);
				process.exit(1);
			});

			createRoot(renderer).render(<App showAll={showAll} />);
		}).catch(reject);
	});
}

// Auto-run when executed directly
if (import.meta.main) {
	const showAll = process.argv.includes("--all");
	runTui({ showAll });
}
