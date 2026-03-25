/**
 * alfred TUI — Quality Dashboard
 */
import { createCliRenderer } from "@opentui/core";
import { createRoot, useKeyboard, useRenderer, useTerminalDimensions } from "@opentui/react";
import { useState, useEffect, createElement } from "react";
import { loadDashboardData, type QualityDashboardData } from "./data.js";

// Gruvbox Material Dark
const C = {
	fg: "#d4be98",
	dim: "#7c6f64",
	accent: "#7daea3",
	green: "#a9b665",
	yellow: "#d8a657",
	orange: "#e78a4e",
	red: "#ea6962",
	aqua: "#89b482",
	border: "#5a524c",
};

function scoreColor(n: number) { return n >= 80 ? C.green : n >= 60 ? C.yellow : C.red; }

function bar(pass: number, total: number, w = 10) {
	if (total === 0) return "─".repeat(w);
	const f = Math.round((pass / total) * w);
	return "━".repeat(f) + "─".repeat(w - f);
}

function pct(pass: number, total: number) {
	return total === 0 ? " -" : `${Math.round((pass / total) * 100)}%`;
}

function App() {
	const cwd = process.cwd();
	const [data, setData] = useState<QualityDashboardData | null>(null);
	const renderer = useRenderer();
	const { height: termH } = useTerminalDimensions();

	useEffect(() => {
		setData(loadDashboardData(cwd));
		const id = setInterval(() => setData(loadDashboardData(cwd)), 2000);
		return () => clearInterval(id);
	}, []);

	useKeyboard((key) => {
		if (key.name === "q" || key.name === "escape") renderer.destroy();
		if (key.name === "r") setData(loadDashboardData(cwd));
	});

	if (!data) return <text fg={C.dim}>Loading...</text>;

	const s = data.score;
	const g = data.gates;
	const k = data.knowledge;
	const kt = data.knowledgeTotals;
	const sess = data.session;

	const delta = data.previousScore != null ? s.sessionScore - data.previousScore : null;
	const deltaStr = delta != null ? (delta >= 0 ? `▲+${delta}` : `▼${delta}`) : "";

	const wT = g.onWrite.pass + g.onWrite.fail;
	const cT = g.onCommit.pass + g.onCommit.fail;
	const tT = g.test.pass + g.test.fail;
	const eT = k.errorHits + k.errorMisses;
	const cvT = k.conventionPass + k.conventionWarn;

	const mins = Math.floor((Date.now() - sess.startedAt) / 60000);
	const minsColor = mins >= 35 ? C.red : mins >= 25 ? C.yellow : C.fg;

	const evMax = Math.max(termH - 22, 2);

	return (
		<box style={{ flexDirection: "column" }}>
			{/* Header */}
			<box style={{ flexDirection: "row", paddingX: 1 }}>
				<ascii-font text="alfred" font="tiny" color={C.accent} />
				<box style={{ flexDirection: "column", paddingLeft: 2, justifyContent: "flex-end" }}>
					<text>
						<span fg={C.fg}>{data.projectName}</span>
						<span fg={C.dim}> │ Score: </span>
						<span fg={scoreColor(s.sessionScore)}>{String(s.sessionScore)}/100</span>
						{delta != null && <span fg={delta >= 0 ? C.green : C.red}> {deltaStr}</span>}
						<span fg={C.dim}> ({s.trend})</span>
					</text>
				</box>
			</box>

			{/* Gates */}
			<box style={{ borderStyle: "rounded", borderColor: C.border, flexDirection: "column", paddingX: 1, marginTop: 1 }} title="Gates">
				<text fg={C.fg}>{`on_write   ${String(g.onWrite.pass).padStart(2)} pass  ${String(g.onWrite.fail).padStart(2)} fail  ${pct(g.onWrite.pass, wT).padStart(4)}  `}<span fg={wT > 0 && g.onWrite.pass === wT ? C.green : C.fg}>{bar(g.onWrite.pass, wT)}</span></text>
				<text fg={C.fg}>{`on_commit  ${String(g.onCommit.pass).padStart(2)} pass  ${String(g.onCommit.fail).padStart(2)} fail  ${pct(g.onCommit.pass, cT).padStart(4)}  `}<span fg={cT > 0 && g.onCommit.pass === cT ? C.green : C.fg}>{bar(g.onCommit.pass, cT)}</span></text>
				<text fg={C.fg}>{`test       ${String(g.test.pass).padStart(2)} pass  ${String(g.test.fail).padStart(2)} fail  ${pct(g.test.pass, tT).padStart(4)}  `}<span fg={tT > 0 && g.test.pass === tT ? C.green : C.fg}>{bar(g.test.pass, tT)}</span></text>
			</box>

			{/* Knowledge */}
			<box style={{ borderStyle: "rounded", borderColor: C.border, flexDirection: "column", paddingX: 1 }} title="Knowledge">
				<text fg={C.fg}>{`error_resolution  hits: ${k.errorHits}/${eT} (${pct(k.errorHits, eT)})  total: ${kt.errorResolutions}`}</text>
				<text fg={C.fg}>{`exemplar          injected: ${k.exemplarInjections}      total: ${kt.exemplars}`}</text>
				<text fg={C.fg}>{`convention        adherence: ${pct(k.conventionPass, cvT)}   total: ${kt.conventions}`}</text>
			</box>

			{/* Recent Events */}
			<box style={{ borderStyle: "rounded", borderColor: C.border, flexDirection: "column", paddingX: 1, flexGrow: 1, overflow: "hidden" }} title="Recent Events">
				{data.recentEvents.length === 0 && <text fg={C.dim}>No events recorded yet</text>}
				{data.recentEvents.slice(0, evMax).map((e, i) => {
					const ok = e.type.includes("pass") || e.type === "error_hit";
					const bad = e.type.includes("fail") || e.type === "error_miss";
					const icon = ok ? "✓" : bad ? "✗" : "●";
					const ic = ok ? C.green : bad ? C.red : C.yellow;
					return (
						<text key={String(i)} fg={C.fg}>
							<span fg={C.dim}>{e.timestamp}</span>{`  `}<span fg={ic}>{icon}</span>{` ${e.type.padEnd(18)}`}<span fg={C.dim}>{e.detail}</span>
						</text>
					);
				})}
			</box>

			{/* Session */}
			<box style={{ height: 1, paddingX: 1 }}>
				<text>
					<span fg={C.dim}>Session: </span><span fg={minsColor}>{String(mins)}min</span>
					<span fg={C.dim}> │ Files: </span><span fg={C.fg}>{String(sess.changedFiles)}</span>
					<span fg={C.dim}> │ Commits: </span><span fg={C.fg}>{String(sess.commits)}</span>
					<span fg={C.dim}> │ Pending: </span><span fg={data.pendingFixesCount > 0 ? C.red : C.green}>{String(data.pendingFixesCount)}</span>
					<span fg={C.dim}> │ Directives: </span><span fg={C.fg}>{String(data.directiveCount)}</span>
				</text>
			</box>

			{/* Footer */}
			<box style={{ height: 1, paddingX: 1 }}>
				<text fg={C.dim}>[q] quit  [r] refresh</text>
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
