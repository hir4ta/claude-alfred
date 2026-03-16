import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Separator } from "@/components/ui/separator";
import {
	decisionsQueryOptions,
	epicsQueryOptions,
	healthQueryOptions,
	tasksQueryOptions,
} from "@/lib/api";
import type { DecisionEntry, EpicSummary, MemoryHealthStats, TaskDetail } from "@/lib/types";
import { cn } from "@/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { Link, createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/")({
	component: OverviewPage,
});

function OverviewPage() {
	const { data: tasksData } = useQuery(tasksQueryOptions());
	const { data: healthData } = useQuery(healthQueryOptions());
	const { data: epicsData } = useQuery(epicsQueryOptions());
	const { data: decisionsData } = useQuery(decisionsQueryOptions(5));

	const tasks = tasksData?.tasks ?? [];
	const activeSlug = tasksData?.active ?? "";

	return (
		<div className="space-y-6">
			<TaskSummarySection tasks={tasks} activeSlug={activeSlug} />
			<div className="grid gap-6 md:grid-cols-2">
				<HealthCard stats={healthData} />
				<EpicProgressCard epics={epicsData?.epics} />
			</div>
			<RecentDecisionsCard decisions={decisionsData?.decisions} />
		</div>
	);
}

// --- Task Summary ---

function TaskSummarySection({ tasks, activeSlug }: { tasks: TaskDetail[]; activeSlug: string }) {
	const active = tasks.filter((t) => t.status === "active");
	const completed = tasks.filter((t) => t.status === "completed");

	return (
		<div className="space-y-3">
			<div className="flex items-center gap-3 text-sm text-muted-foreground">
				<span>{tasks.length} tasks</span>
				<Separator orientation="vertical" className="h-4" />
				<span>{active.length} active</span>
				<Separator orientation="vertical" className="h-4" />
				<span>{completed.length} completed</span>
			</div>
			<div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
				{tasks.map((task) => (
					<TaskCard key={task.slug} task={task} isActive={task.slug === activeSlug} />
				))}
			</div>
		</div>
	);
}

function TaskCard({ task, isActive }: { task: TaskDetail; isActive: boolean }) {
	const progress = task.total > 0 ? (task.completed / task.total) * 100 : 0;

	return (
		<Link to="/tasks/$slug" params={{ slug: task.slug }}>
			<Card
				className={cn(
					"transition-colors hover:border-brand-pattern/30",
					isActive && "border-brand-session/40",
				)}
			>
				<CardHeader className="pb-2">
					<div className="flex items-center justify-between">
						<CardTitle className="text-sm font-medium">{task.slug}</CardTitle>
						<div className="flex gap-1">
							{task.size && (
								<Badge variant="outline" className="text-xs">
									{task.size}
								</Badge>
							)}
							<StatusBadge status={task.status} />
						</div>
					</div>
				</CardHeader>
				<CardContent className="space-y-2">
					{task.focus && <p className="text-xs text-muted-foreground line-clamp-2">{task.focus}</p>}
					<div className="flex items-center gap-2">
						<Progress value={progress} className="h-1.5 flex-1" />
						<span className="text-xs text-muted-foreground">
							{task.completed}/{task.total}
						</span>
					</div>
					{task.has_blocker && (
						<p className="text-xs" style={{ color: "#c0392b" }}>
							Blocked: {task.blocker_text}
						</p>
					)}
				</CardContent>
			</Card>
		</Link>
	);
}

function StatusBadge({ status }: { status: string }) {
	const styles: Record<string, { bg: string; text: string }> = {
		active: { bg: "rgba(64,81,59,0.15)", text: "#40513b" },
		completed: { bg: "rgba(45,139,122,0.15)", text: "#2d8b7a" },
	};
	const s = styles[status] ?? { bg: "rgba(107,114,128,0.15)", text: "#6b7280" };

	return (
		<span
			className="inline-flex items-center rounded px-1.5 py-0.5 text-xs font-medium"
			style={{ backgroundColor: s.bg, color: s.text }}
		>
			{status}
		</span>
	);
}

// --- Health Card ---

function HealthCard({ stats }: { stats?: MemoryHealthStats }) {
	if (!stats) return null;

	return (
		<Card>
			<CardHeader className="pb-2">
				<CardTitle className="text-sm font-medium">Memory Health</CardTitle>
			</CardHeader>
			<CardContent className="space-y-3">
				<div className="grid grid-cols-3 gap-4 text-center">
					<div>
						<p className="text-2xl font-semibold text-foreground">{stats.total}</p>
						<p className="text-xs text-muted-foreground">Total</p>
					</div>
					<div>
						<p
							className="text-2xl font-semibold"
							style={{ color: stats.stale_count > 0 ? "#e67e22" : undefined }}
						>
							{stats.stale_count}
						</p>
						<p className="text-xs text-muted-foreground">Stale</p>
					</div>
					<div>
						<p
							className="text-2xl font-semibold"
							style={{ color: stats.conflict_count > 0 ? "#c0392b" : undefined }}
						>
							{stats.conflict_count}
						</p>
						<p className="text-xs text-muted-foreground">Conflicts</p>
					</div>
				</div>
				{stats.vitality_dist && <VitalityDist dist={stats.vitality_dist} />}
			</CardContent>
		</Card>
	);
}

function VitalityDist({ dist }: { dist: [number, number, number, number, number] }) {
	const labels = ["0-20", "21-40", "41-60", "61-80", "81-100"];
	const max = Math.max(...dist, 1);

	return (
		<div className="space-y-1">
			<p className="text-xs text-muted-foreground">Vitality Distribution</p>
			<div className="flex items-end gap-1 h-8">
				{dist.map((count, i) => (
					<div key={labels[i]} className="flex-1 flex flex-col items-center gap-0.5">
						<div
							className="w-full rounded-sm"
							style={{
								height: `${(count / max) * 100}%`,
								minHeight: count > 0 ? "2px" : 0,
								backgroundColor: "#2d8b7a",
								opacity: 0.3 + (i / 4) * 0.7,
							}}
						/>
						<span className="text-[10px] text-muted-foreground">{labels[i]}</span>
					</div>
				))}
			</div>
		</div>
	);
}

// --- Epic Progress ---

function EpicProgressCard({ epics }: { epics?: EpicSummary[] }) {
	if (!epics || epics.length === 0) return null;

	return (
		<Card>
			<CardHeader className="pb-2">
				<CardTitle className="text-sm font-medium">Epics</CardTitle>
			</CardHeader>
			<CardContent className="space-y-3">
				{epics.map((epic) => {
					const progress = epic.total > 0 ? (epic.completed / epic.total) * 100 : 0;
					return (
						<div key={epic.slug} className="space-y-1">
							<div className="flex items-center justify-between">
								<span className="text-sm">{epic.name}</span>
								<span className="text-xs text-muted-foreground">
									{epic.completed}/{epic.total}
								</span>
							</div>
							<Progress value={progress} className="h-1.5" />
						</div>
					);
				})}
			</CardContent>
		</Card>
	);
}

// --- Recent Decisions ---

function RecentDecisionsCard({ decisions }: { decisions?: DecisionEntry[] }) {
	if (!decisions || decisions.length === 0) return null;

	return (
		<Card>
			<CardHeader className="pb-2">
				<CardTitle className="text-sm font-medium">Recent Decisions</CardTitle>
			</CardHeader>
			<CardContent>
				<div className="space-y-2">
					{decisions.map((dec, i) => (
						<div key={`${dec.task_slug}-${dec.title}-${i}`} className="flex items-start gap-2">
							<span
								className="mt-0.5 inline-block h-2 w-2 shrink-0 rounded-full"
								style={{ backgroundColor: "#628141" }}
							/>
							<div className="min-w-0">
								<p className="text-sm font-medium text-foreground">{dec.title}</p>
								{dec.chosen && (
									<p className="text-xs text-muted-foreground line-clamp-1">{dec.chosen}</p>
								)}
							</div>
							<Badge variant="outline" className="ml-auto shrink-0 text-xs">
								{dec.task_slug}
							</Badge>
						</div>
					))}
				</div>
			</CardContent>
		</Card>
	);
}
