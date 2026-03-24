import { CircleCheck, CircleDot, AlertTriangle, Search, Lightbulb, Wrench, Shield } from "@animated-color-icons/lucide-react";
import { Badge } from "@/components/ui/badge";
import { useI18n } from "@/lib/i18n";

// ---- Types (mirror src/spec/types.ts, frontend-only) ----

interface SpecTask {
	id: string;
	title: string;
	checked: boolean;
	requirements?: string[];
	depends?: string[];
	files?: string[];
	verify?: string;
	subtasks?: string[];
}

interface SpecWave {
	key: number | "closing";
	title: string;
	tasks: SpecTask[];
}

interface TasksFile {
	slug: string;
	waves: SpecWave[];
	dependency_graph?: Record<string, string[]>;
}

interface GherkinStep {
	type: string;
	text: string;
}

interface TestScenario {
	name: string;
	steps: string[];
}

interface TestSpec {
	id: string;
	name?: string;
	title?: string;
	source?: string;
	// Actual format: Gherkin steps directly on spec
	steps?: GherkinStep[];
	// Legacy format: scenarios with nested steps
	scenarios?: TestScenario[];
}

interface CoverageEntry {
	req_id: string;
	test_ids: string[];
	type: string;
	priority: string;
}

interface TestSpecsFile {
	specs: TestSpec[];
	coverage_matrix?: CoverageEntry[];
}

interface BugfixFile {
	summary: string;
	severity: "P0" | "P1" | "P2" | "P3";
	impact?: string;
	reproduction_steps: string[];
	root_cause: string;
	five_whys?: string[];
	fix_strategy: string;
	regression_prevention?: string;
	confidence?: number;
}

// ---- Shared ----

const SEVERITY_COLORS: Record<string, { bg: string; fg: string }> = {
	P0: { bg: "#c0392b", fg: "#fff" },
	P1: { bg: "#e67e22", fg: "#fff" },
	P2: { bg: "#f1c40f", fg: "#1c1917" },
	P3: { bg: "#7b6b8d", fg: "#fff" },
};

function SectionLabel({ children }: { children: React.ReactNode }) {
	return (
		<h3 className="text-[13px] font-semibold text-foreground mt-3 mb-1.5 first:mt-0">
			{children}
		</h3>
	);
}

function OrderedList({ items }: { items: string[] }) {
	return (
		<ol className="list-decimal list-inside space-y-0.5">
			{items.map((item, i) => (
				<li key={i} className="text-[13px] text-muted-foreground leading-relaxed">
					{item}
				</li>
			))}
		</ol>
	);
}

// ---- Tasks Renderer ----

function TaskRow({ task }: { task: SpecTask }) {
	return (
		<li className="flex items-start gap-2 py-0.5">
			{task.checked ? (
				<CircleCheck className="size-3.5 shrink-0 mt-0.5" style={{ color: "#2d8b7a" }} />
			) : (
				<CircleDot className="size-3.5 shrink-0 mt-0.5" style={{ color: "#7b6b8d" }} />
			)}
			<span className="font-mono text-[12px] shrink-0" style={{ color: "#e67e22" }}>
				{task.id}
			</span>
			<span className={`text-[13px] leading-snug ${task.checked ? "text-muted-foreground" : "text-foreground"}`}>
				{task.title}
			</span>
		</li>
	);
}

function WaveSection({ wave }: { wave: SpecWave }) {
	const done = wave.tasks.filter((t) => t.checked).length;
	const total = wave.tasks.length;
	const isClosing = wave.key === "closing";
	const label = isClosing ? "Closing" : `Wave ${wave.key}`;
	const progressPct = total > 0 ? (done / total) * 100 : 0;

	return (
		<div className="mb-3 last:mb-0">
			<div className="flex items-center gap-2 mb-1.5">
				<span className="text-[12px] font-semibold font-mono" style={{ color: isClosing ? "#7b6b8d" : "#628141" }}>
					{label}
				</span>
				<span className="text-[13px] text-foreground font-medium">{wave.title}</span>
				<span className="text-[11px] text-muted-foreground ml-auto">
					{done}/{total}
				</span>
			</div>
			<div className="h-1 rounded-full bg-muted mb-2">
				<div
					className="h-1 rounded-full transition-all"
					style={{
						width: `${progressPct}%`,
						backgroundColor: progressPct === 100 ? "#2d8b7a" : "#628141",
					}}
				/>
			</div>
			<ul className="space-y-0.5 pl-1">
				{wave.tasks.map((task) => (
					<TaskRow key={task.id} task={task} />
				))}
			</ul>
		</div>
	);
}

function TasksRenderer({ data }: { data: TasksFile }) {
	return (
		<div className="space-y-1">
			{data.waves.map((wave) => (
				<WaveSection key={String(wave.key)} wave={wave} />
			))}
		</div>
	);
}

// ---- Test Specs Renderer ----

const STEP_TYPE_COLORS: Record<string, string> = {
	Given: "#628141",
	When: "#e67e22",
	Then: "#2d8b7a",
	And: "#7b6b8d",
};

function TestSpecCard({ spec }: { spec: TestSpec }) {
	const displayName = spec.name ?? spec.title ?? spec.id;

	return (
		<div className="mb-3 last:mb-0">
			<div className="flex items-start gap-2 mb-1">
				<span className="font-mono text-[12px] shrink-0 mt-0.5" style={{ color: "#7b6b8d" }}>
					{spec.id}
				</span>
				<span className="text-[13px] font-medium text-foreground">{displayName}</span>
				{spec.source && (
					<span className="text-[11px] text-muted-foreground shrink-0">({spec.source})</span>
				)}
			</div>
			{/* Gherkin steps (actual format) */}
			{spec.steps && spec.steps.length > 0 && (
				<div className="ml-6 space-y-0.5">
					{spec.steps.map((step, j) => (
						<div key={j} className="flex items-start gap-2">
							<span
								className="font-mono text-[11px] shrink-0 w-12 text-right"
								style={{ color: STEP_TYPE_COLORS[step.type] ?? "#6b7280" }}
							>
								{step.type}
							</span>
							<span className="text-[12px] text-muted-foreground leading-relaxed">{step.text}</span>
						</div>
					))}
				</div>
			)}
			{/* Scenario format (legacy) */}
			{spec.scenarios && spec.scenarios.map((scenario, i) => (
				<div key={i} className="ml-4 mb-2 last:mb-0">
					<div className="flex items-center gap-1.5 mb-0.5">
						<Search className="size-3 shrink-0" style={{ color: "#2d8b7a" }} />
						<span className="text-[12px] font-medium text-foreground">{scenario.name}</span>
					</div>
					<ol className="list-decimal list-inside ml-4 space-y-0">
						{scenario.steps.map((step, j) => (
							<li key={j} className="text-[12px] text-muted-foreground leading-relaxed">
								{step}
							</li>
						))}
					</ol>
				</div>
			))}
		</div>
	);
}

function CoverageMatrix({ matrix }: { matrix: CoverageEntry[] }) {
	const { t } = useI18n();
	return (
		<div className="mb-4">
			<SectionLabel>{t("spec.coverageMatrix")}</SectionLabel>
			<div className="overflow-x-auto">
				<table className="text-[12px] w-full border-collapse">
					<thead>
						<tr className="border-b border-border">
							<th className="text-left px-2 py-1 text-muted-foreground font-medium">Req</th>
							<th className="text-left px-2 py-1 text-muted-foreground font-medium">Tests</th>
							<th className="text-left px-2 py-1 text-muted-foreground font-medium">Type</th>
							<th className="text-left px-2 py-1 text-muted-foreground font-medium">Priority</th>
						</tr>
					</thead>
					<tbody>
						{matrix.map((entry) => (
							<tr key={entry.req_id} className="border-b border-border/50">
								<td className="px-2 py-1 font-mono" style={{ color: "#40513b" }}>{entry.req_id}</td>
								<td className="px-2 py-1 font-mono text-muted-foreground">{entry.test_ids.join(", ")}</td>
								<td className="px-2 py-1 text-muted-foreground">{entry.type}</td>
								<td className="px-2 py-1">
									<span style={{ color: entry.priority === "P0" ? "#c0392b" : "#e67e22" }}>{entry.priority}</span>
								</td>
							</tr>
						))}
					</tbody>
				</table>
			</div>
		</div>
	);
}

function TestSpecsRenderer({ data }: { data: TestSpecsFile }) {
	const { t } = useI18n();
	if ((!data.specs || data.specs.length === 0) && !data.coverage_matrix) {
		return <p className="text-[13px] text-muted-foreground">{t("spec.noTestSpecs")}</p>;
	}
	return (
		<div className="space-y-1">
			{data.coverage_matrix && data.coverage_matrix.length > 0 && (
				<CoverageMatrix matrix={data.coverage_matrix} />
			)}
			{data.specs?.map((spec) => (
				<TestSpecCard key={spec.id} spec={spec} />
			))}
		</div>
	);
}

// ---- Bugfix Renderer ----

function BugfixRenderer({ data }: { data: BugfixFile }) {
	const { t } = useI18n();
	const sevColor = SEVERITY_COLORS[data.severity] ?? SEVERITY_COLORS.P2;

	return (
		<div className="space-y-1">
			{/* Summary + Severity */}
			<div className="flex items-start gap-2">
				<AlertTriangle className="size-4 shrink-0 mt-0.5" style={{ color: sevColor.bg }} />
				<div className="flex-1 min-w-0">
					<div className="flex items-center gap-2 mb-1">
						<Badge
							className="text-[10px] px-1.5 py-0"
							style={{ backgroundColor: sevColor.bg, color: sevColor.fg, borderColor: sevColor.bg }}
						>
							{data.severity}
						</Badge>
						<span className="text-[13px] font-medium text-foreground">{data.summary}</span>
					</div>
				</div>
			</div>

			{/* Reproduction Steps */}
			{data.reproduction_steps?.length > 0 && (
				<div>
					<SectionLabel>{t("spec.reproSteps")}</SectionLabel>
					<OrderedList items={data.reproduction_steps} />
				</div>
			)}

			{/* Root Cause */}
			<div>
				<SectionLabel>{t("spec.rootCause")}</SectionLabel>
				<p className="text-[13px] text-muted-foreground leading-relaxed">{data.root_cause}</p>
			</div>

			{/* Five Whys */}
			{data.five_whys && data.five_whys.length > 0 && (
				<div>
					<SectionLabel>
						<span className="inline-flex items-center gap-1.5">
							<Lightbulb className="size-3.5" style={{ color: "#e67e22" }} />
							{t("spec.fiveWhys")}
						</span>
					</SectionLabel>
					<OrderedList items={data.five_whys} />
				</div>
			)}

			{/* Fix Strategy */}
			<div>
				<SectionLabel>
					<span className="inline-flex items-center gap-1.5">
						<Wrench className="size-3.5" style={{ color: "#628141" }} />
						{t("spec.fixStrategy")}
					</span>
				</SectionLabel>
				<p className="text-[13px] text-muted-foreground leading-relaxed">{data.fix_strategy}</p>
			</div>

			{/* Regression Prevention */}
			{data.regression_prevention && (
				<div>
					<SectionLabel>
						<span className="inline-flex items-center gap-1.5">
							<Shield className="size-3.5" style={{ color: "#2d8b7a" }} />
							{t("spec.regressionPrevention")}
						</span>
					</SectionLabel>
					<p className="text-[13px] text-muted-foreground leading-relaxed">{data.regression_prevention}</p>
				</div>
			)}
		</div>
	);
}

// ---- Dispatcher ----

export function JsonSpecRenderer({ filename, content }: { filename: string; content: string }) {
	try {
		const parsed = JSON.parse(content);

		if (filename === "tasks.json") {
			return <TasksRenderer data={parsed as TasksFile} />;
		}
		if (filename === "test-specs.json") {
			return <TestSpecsRenderer data={parsed as TestSpecsFile} />;
		}
		if (filename === "bugfix.json") {
			return <BugfixRenderer data={parsed as BugfixFile} />;
		}
	} catch {
		// Fall through to raw display
	}

	// Fallback: pretty-printed JSON
	let formatted = content;
	try {
		formatted = JSON.stringify(JSON.parse(content), null, 2);
	} catch { /* keep original */ }

	return (
		<pre className="text-[12px] leading-relaxed text-muted-foreground bg-muted rounded-lg p-3 overflow-x-auto whitespace-pre-wrap break-all">
			{formatted}
		</pre>
	);
}
