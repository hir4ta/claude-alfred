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
	const { width: termW, height: termH } = useTerminalDimensions();

	useEffect(() => {
		setData(loadDashboardData(cwd));
		const interval = setInterval(() => setData(loadDashboardData(cwd)), 2000);
		return () => clearInterval(interval);
	}, []);

	useKeyboard((key) => {
		if (key.name === "q" || key.name === "escape") {
			renderer.destroy();
		}
		if (key.name === "r") setData(loadDashboardData(cwd));
	});

	if (!data) {
		return <box width="100%" height="100%" backgroundColor={C.bg}>
			<text fg={C.dim}>Loading...</text>
		</box>;
	}

	const s = data.score;
	const g = data.gates;
	const k = data.knowledge;
	const kt = data.knowledgeTotals;
	const sess = data.session;

	const scoreDelta = data.previousScore != null ? s.sessionScore - data.previousScore : null;
	const deltaStr = scoreDelta != null
		? scoreDelta >= 0 ? ` ▲ (+${scoreDelta})` : ` ▼ (${scoreDelta})`
		: "";
	const deltaColor = scoreDelta != null && scoreDelta >= 0 ? C.green : C.red;

	const writeTotal = g.onWrite.pass + g.onWrite.fail;
	const commitTotal = g.onCommit.pass + g.onCommit.fail;
	const testTotal = g.test.pass + g.test.fail;
	const errorTotal = k.errorHits + k.errorMisses;
	const convTotal = k.conventionPass + k.conventionWarn;

	const elapsedMins = Math.floor((Date.now() - sess.startedAt) / 60000);
	const elapsedColor = elapsedMins >= 35 ? C.red : elapsedMins >= 25 ? C.yellow : C.fg;

	const eventsHeight = Math.max(termH - 24, 3);

	return (
		<box width="100%" height="100%" backgroundColor={C.bg} flexDirection="column">
			{/* Header: Score */}
			<box height={3} border title=" alfred " style={{ borderStyle: "single" }} backgroundColor={C.bg}>
				<text fg={C.fg}> {data.projectName}</text>
				<text fg={C.dim}> │ Quality Score: </text>
				<text fg={scoreColor(s.sessionScore)}><strong>{s.sessionScore}/100</strong></text>
				{scoreDelta != null && <text fg={deltaColor}>{deltaStr}</text>}
				<text fg={C.dim}> ({s.trend})</text>
			</box>

			{/* Gates */}
			<box height={6} border title="Gates" backgroundColor={C.bg}>
				<box flexDirection="column" padding={1}>
					<text fg={C.fg}>
						{`  on_write   ${pad(g.onWrite.pass, 3)} pass  ${pad(g.onWrite.fail, 3)} fail  ${rate(g.onWrite.pass, writeTotal).padStart(4)}  ${bar(g.onWrite.pass, writeTotal, 10)}`}
					</text>
					<text fg={C.fg}>
						{`  on_commit  ${pad(g.onCommit.pass, 3)} pass  ${pad(g.onCommit.fail, 3)} fail  ${rate(g.onCommit.pass, commitTotal).padStart(4)}  ${bar(g.onCommit.pass, commitTotal, 10)}`}
					</text>
					<text fg={C.fg}>
						{`  test       ${pad(g.test.pass, 3)} pass  ${pad(g.test.fail, 3)} fail  ${rate(g.test.pass, testTotal).padStart(4)}  ${bar(g.test.pass, testTotal, 10)}`}
					</text>
				</box>
			</box>

			{/* Knowledge */}
			<box height={6} border title="Knowledge" backgroundColor={C.bg}>
				<box flexDirection="column" padding={1}>
					<text fg={C.fg}>
						{`  error_resolution  hits: ${k.errorHits}/${errorTotal} (${rate(k.errorHits, errorTotal)})  total: ${kt.errorResolutions}`}
					</text>
					<text fg={C.fg}>
						{`  exemplar          injected: ${k.exemplarInjections}      total: ${kt.exemplars}`}
					</text>
					<text fg={C.fg}>
						{`  convention         adherence: ${rate(k.conventionPass, convTotal)}   total: ${kt.conventions}`}
					</text>
				</box>
			</box>

			{/* Recent Events */}
			<box flexGrow={1} border title="Recent Events" backgroundColor={C.bg}>
				<box flexDirection="column" padding={1}>
					{data.recentEvents.slice(0, eventsHeight).map((e, i) => {
						const icon = e.type.includes("pass") || e.type === "error_hit"
							? "✓" : e.type.includes("fail") || e.type === "error_miss"
							? "✗" : "●";
						const iconColor = icon === "✓" ? C.green : icon === "✗" ? C.red : C.yellow;
						return (
							<text key={String(i)} fg={C.fg}>
								{`  `}<span fg={C.dim}>{e.timestamp}</span>{`  `}<span fg={iconColor}>{icon}</span>{` ${e.type.padEnd(18)}`}<span fg={C.dim}>{e.detail}</span>
							</text>
						);
					})}
					{data.recentEvents.length === 0 && <text fg={C.dim}>  No events recorded yet</text>}
				</box>
			</box>

			{/* Session Info */}
			<box height={3} border title="Session" backgroundColor={C.bg}>
				<text fg={C.dim}>
					{`  `}<span fg={elapsedColor}>{elapsed(sess.startedAt)}</span>
					{` │ Files: `}<span fg={C.fg}>{String(sess.changedFiles)}</span>
					{` │ Commits: `}<span fg={C.fg}>{String(sess.commits)}</span>
					{` │ Pending: `}<span fg={data.pendingFixesCount > 0 ? C.red : C.green}>{String(data.pendingFixesCount)}</span>
					{` │ Directives: `}<span fg={C.fg}>{String(data.directiveCount)}</span>
				</text>
			</box>

			{/* Footer */}
			<box height={1}>
				<text fg={C.dim}> [q] quit  [r] refresh</text>
			</box>
		</box>
	);
}

export function runTui(): Promise<void> {
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
			createRoot(renderer).render(<App />);
		}).catch(reject);
	});
}

if (import.meta.main) {
	runTui();
}
