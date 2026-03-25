/**
 * alfred TUI — Quality Dashboard
 *
 * Full layout per design/remaining-design.md:
 * - Quality Score + previous session comparison
 * - Gates (on_write / on_commit / test)
 * - Knowledge (error_resolution hits, exemplar injections, convention adherence, DB totals)
 * - Recent Events (real-time stream)
 * - Session Info (elapsed, files, commits, pending-fixes, directives)
 */
import { createCliRenderer } from "@opentui/core";
import { createRoot, useKeyboard, useRenderer, useTerminalDimensions } from "@opentui/react";
import { useState, useEffect, createElement } from "react";
import { loadDashboardData, type QualityDashboardData } from "./data.js";

// Gruvbox Material Dark palette (per design spec)
const C = {
	bg: "#1c1917",
	fg: "#d3c6aa",
	dim: "#5c6a72",
	accent: "#7fbbb3",
	green: "#a9b665",
	yellow: "#d8a657",
	orange: "#e78a4e",
	red: "#ea6962",
	aqua: "#89b482",
	border: "#414b50",
};

function scoreColor(score: number): string {
	if (score >= 80) return C.green;
	if (score >= 60) return C.yellow;
	return C.red;
}

function bar(pass: number, total: number, width: number): string {
	if (total === 0) return "─".repeat(width);
	const filled = Math.round((pass / total) * width);
	return "█".repeat(filled) + "░".repeat(width - filled);
}

function rate(pass: number, total: number): string {
	if (total === 0) return "  -";
	return `${Math.round((pass / total) * 100)}%`;
}

function pad(s: string | number, n: number): string {
	return String(s).padStart(n);
}

function elapsed(startMs: number): string {
	const mins = Math.floor((Date.now() - startMs) / 60000);
	return `${mins}min`;
}

function App() {
	const cwd = process.cwd();
	const [data, setData] = useState<QualityDashboardData | null>(null);
	const renderer = useRenderer();
	const [width, height] = useTerminalDimensions();

	useEffect(() => {
		setData(loadDashboardData(cwd));
		const interval = setInterval(() => setData(loadDashboardData(cwd)), 2000);
		return () => clearInterval(interval);
	}, []);

	useKeyboard((key) => {
		if (key === "q" || key === "escape") {
			renderer.close();
			process.exit(0);
		}
		if (key === "r") setData(loadDashboardData(cwd));
	});

	if (!data) {
		return <box width="100%" height="100%" backgroundColor={C.bg}>
			<text color={C.dim}>Loading...</text>
		</box>;
	}

	const s = data.score;
	const g = data.gates;
	const k = data.knowledge;
	const kt = data.knowledgeTotals;
	const sess = data.session;

	const scoreDelta = data.previousScore != null ? s.sessionScore - data.previousScore : null;
	const deltaStr = scoreDelta != null
		? scoreDelta >= 0 ? `▲ (+${scoreDelta})` : `▼ (${scoreDelta})`
		: "";
	const deltaColor = scoreDelta != null && scoreDelta >= 0 ? C.green : C.red;

	const writeTotal = g.onWrite.pass + g.onWrite.fail;
	const commitTotal = g.onCommit.pass + g.onCommit.fail;
	const testTotal = g.test.pass + g.test.fail;
	const errorTotal = k.errorHits + k.errorMisses;
	const convTotal = k.conventionPass + k.conventionWarn;

	const elapsedMins = Math.floor((Date.now() - sess.startedAt) / 60000);
	// Research #7: task time 2x = failure rate 4x. 35min threshold.
	const elapsedColor = elapsedMins >= 35 ? C.red : elapsedMins >= 25 ? C.yellow : C.fg;

	return (
		<box width="100%" height="100%" backgroundColor={C.bg} flexDirection="column">
			{/* ── Header: Score ── */}
			<box height={3} width="100%" borderStyle="single" borderColor={C.border}>
				<text color={C.accent} bold> alfred </text>
				<text color={C.dim}>│ </text>
				<text color={C.fg}>{data.projectName}</text>
				<text color={C.dim}> │ Quality Score: </text>
				<text color={scoreColor(s.sessionScore)} bold>{s.sessionScore}/100</text>
				<text color={C.dim}> </text>
				{scoreDelta != null && <text color={deltaColor}>{deltaStr}</text>}
			</box>

			{/* ── Gates ── */}
			<box height={6} width="100%" borderStyle="single" borderColor={C.border}>
				<box flexDirection="column" paddingLeft={1}>
					<text color={C.accent} bold>Gates</text>
					<text color={C.fg}>
						{"  on_write   "}{pad(g.onWrite.pass, 3)} pass  {pad(g.onWrite.fail, 3)} fail  {rate(g.onWrite.pass, writeTotal).padStart(4)}  {bar(g.onWrite.pass, writeTotal, 10)}
					</text>
					<text color={C.fg}>
						{"  on_commit  "}{pad(g.onCommit.pass, 3)} pass  {pad(g.onCommit.fail, 3)} fail  {rate(g.onCommit.pass, commitTotal).padStart(4)}  {bar(g.onCommit.pass, commitTotal, 10)}
					</text>
					<text color={C.fg}>
						{"  test       "}{pad(g.test.pass, 3)} pass  {pad(g.test.fail, 3)} fail  {rate(g.test.pass, testTotal).padStart(4)}  {bar(g.test.pass, testTotal, 10)}
					</text>
				</box>
			</box>

			{/* ── Knowledge ── */}
			<box height={6} width="100%" borderStyle="single" borderColor={C.border}>
				<box flexDirection="column" paddingLeft={1}>
					<text color={C.accent} bold>Knowledge</text>
					<text color={C.fg}>
						{"  error_resolution  hits: "}{k.errorHits}/{errorTotal} ({rate(k.errorHits, errorTotal)}){"  total: "}{kt.errorResolutions}
					</text>
					<text color={C.fg}>
						{"  exemplar          injected: "}{k.exemplarInjections}{"      total: "}{kt.exemplars}
					</text>
					<text color={C.fg}>
						{"  convention         adherence: "}{rate(k.conventionPass, convTotal)}{"   total: "}{kt.conventions}
					</text>
				</box>
			</box>

			{/* ── Recent Events ── */}
			<box flexGrow={1} width="100%" borderStyle="single" borderColor={C.border}>
				<box flexDirection="column" paddingLeft={1}>
					<text color={C.accent} bold>Recent Events</text>
					{data.recentEvents.slice(0, Math.max(height - 24, 3)).map((e, i) => {
						const icon = e.type.includes("pass") || e.type === "error_hit"
							? "✓" : e.type.includes("fail") || e.type === "error_miss"
							? "✗" : "●";
						const color = icon === "✓" ? C.green : icon === "✗" ? C.red : C.yellow;
						return (
							<text key={i} color={C.fg}>
								<text color={C.dim}>{e.timestamp}</text>{"  "}<text color={color}>{icon}</text>{" "}{e.type.padEnd(18)}<text color={C.dim}>{e.detail}</text>
							</text>
						);
					})}
					{data.recentEvents.length === 0 && <text color={C.dim}>  No events recorded yet</text>}
				</box>
			</box>

			{/* ── Session Info ── */}
			<box height={3} width="100%" borderStyle="single" borderColor={C.border}>
				<text color={C.dim}>  Session: </text>
				<text color={elapsedColor}>{elapsed(sess.startedAt)}</text>
				<text color={C.dim}> │ Files: </text><text color={C.fg}>{sess.changedFiles}</text>
				<text color={C.dim}> │ Commits: </text><text color={C.fg}>{sess.commits}</text>
				<text color={C.dim}> │ Pending: </text>
				<text color={data.pendingFixesCount > 0 ? C.red : C.green}>{data.pendingFixesCount}</text>
				<text color={C.dim}> │ Directives: </text><text color={C.fg}>{data.directiveCount}</text>
			</box>

			{/* ── Footer ── */}
			<box height={1} width="100%">
				<text color={C.dim}> [q] quit  [r] refresh</text>
			</box>
		</box>
	);
}

export async function runTui(): Promise<void> {
	const renderer = createCliRenderer();
	const root = createRoot(renderer);
	root.render(<App />);
}
