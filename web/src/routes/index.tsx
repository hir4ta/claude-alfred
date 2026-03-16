import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
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
import { AlertTriangle, Brain, CheckCircle2, Clock, Zap } from "lucide-react";

export const Route = createFileRoute("/")({
	component: OverviewPage,
});

function OverviewPage() {
	const { data: tasksData, isLoading: tasksLoading } = useQuery(tasksQueryOptions());
	const { data: healthData } = useQuery(healthQueryOptions());
	const { data: epicsData } = useQuery(epicsQueryOptions());
	const { data: decisionsData } = useQuery(decisionsQueryOptions(5));

	const tasks = tasksData?.tasks ?? [];
	const activeSlug = tasksData?.active ?? "";

	return (
		<div className="space-y-8">
			{/* Stats row */}
			<div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
				<StatCard
					label="Total Tasks"
					value={tasks.length}
					icon={<ListIcon />}
					loading={tasksLoading}
				/>
				<StatCard
					label="Active"
					value={tasks.filter((t) => t.status === "active").length}
					icon={<Clock className="size-4" style={{ color: "#40513b" }} />}
					loading={tasksLoading}
				/>
				<StatCard
					label="Completed"
					value={tasks.filter((t) => t.status === "completed").length}
					icon={<CheckCircle2 className="size-4" style={{ color: "#2d8b7a" }} />}
					loading={tasksLoading}
				/>
				<StatCard
					label="Memories"
					value={healthData?.total ?? 0}
					icon={<Brain className="size-4" style={{ color: "#628141" }} />}
				/>
			</div>

			{/* Task cards */}
			{tasks.length > 0 && (
				<section className="space-y-3">
					<h2
						className="text-sm font-semibold uppercase tracking-wider text-muted-foreground"
						style={{ fontFamily: "var(--font-display)" }}
					>
						Tasks
					</h2>
					<div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
						{tasks.map((task) => (
							<TaskCard key={task.slug} task={task} isActive={task.slug === activeSlug} />
						))}
					</div>
				</section>
			)}

			{/* Bottom row: Health + Epics + Decisions */}
			<div className="grid gap-6 lg:grid-cols-3">
				<HealthCard stats={healthData} />
				<EpicProgressCard epics={epicsData?.epics} />
				<RecentDecisionsCard decisions={decisionsData?.decisions} />
			</div>
		</div>
	);
}

function ListIcon() {
	return <Zap className="size-4" style={{ color: "#e67e22" }} />;
}

function StatCard({
	label,
	value,
	icon,
	loading,
}: { label: string; value: number; icon: React.ReactNode; loading?: boolean }) {
	return (
		<Card className="border-stone-200 dark:border-stone-700">
			<CardContent className="flex items-center gap-4 py-4">
				<div className="flex size-10 items-center justify-center rounded-lg bg-accent/80">
					{icon}
				</div>
				<div>
					{loading ? (
						<Skeleton className="h-7 w-12" />
					) : (
						<p className="text-2xl font-bold" style={{ fontFamily: "var(--font-display)" }}>
							{value}
						</p>
					)}
					<p className="text-xs text-muted-foreground">{label}</p>
				</div>
			</CardContent>
		</Card>
	);
}

function TaskCard({ task, isActive }: { task: TaskDetail; isActive: boolean }) {
	const progress = task.total > 0 ? (task.completed / task.total) * 100 : 0;

	return (
		<Link to="/tasks/$slug" params={{ slug: task.slug }}>
			<Card
				className={cn(
					"border-stone-200 transition-all hover:shadow-md hover:border-stone-300 dark:border-stone-700 dark:hover:border-stone-600",
					isActive && "ring-1 ring-brand-session/30",
				)}
			>
				<CardHeader className="pb-2">
					<div className="flex items-center justify-between gap-2">
						<CardTitle className="text-sm font-semibold truncate">{task.slug}</CardTitle>
						<div className="flex shrink-0 gap-1.5">
							{task.size && (
								<Badge variant="outline" className="text-[10px] px-1.5 py-0 rounded-full">
									{task.size}
								</Badge>
							)}
							<StatusBadge status={task.status} />
						</div>
					</div>
				</CardHeader>
				<CardContent className="space-y-2.5">
					{task.focus && (
						<p className="text-xs text-muted-foreground line-clamp-2 leading-relaxed">
							{task.focus}
						</p>
					)}
					<div className="flex items-center gap-2.5">
						<Progress value={progress} className="h-1.5 flex-1" />
						<span className="text-[11px] tabular-nums text-muted-foreground">
							{task.completed}/{task.total}
						</span>
					</div>
					{task.has_blocker && (
						<div className="flex items-center gap-1.5 text-xs" style={{ color: "#c0392b" }}>
							<AlertTriangle className="size-3" />
							<span className="line-clamp-1">{task.blocker_text}</span>
						</div>
					)}
				</CardContent>
			</Card>
		</Link>
	);
}

function StatusBadge({ status }: { status: string }) {
	const config: Record<string, { bg: string; text: string }> = {
		active: { bg: "#40513b20", text: "#40513b" },
		completed: { bg: "#2d8b7a20", text: "#2d8b7a" },
	};
	const c = config[status] ?? { bg: "#6b728020", text: "#6b7280" };
	return (
		<span
			className="inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium"
			style={{ backgroundColor: c.bg, color: c.text }}
		>
			{status}
		</span>
	);
}

function HealthCard({ stats }: { stats?: MemoryHealthStats }) {
	if (!stats) return null;
	return (
		<Card className="border-stone-200 dark:border-stone-700">
			<CardHeader className="pb-3">
				<CardTitle className="text-sm font-semibold" style={{ fontFamily: "var(--font-display)" }}>
					Memory Health
				</CardTitle>
			</CardHeader>
			<CardContent className="space-y-4">
				<div className="grid grid-cols-3 gap-3 text-center">
					<MetricBlock value={stats.total} label="Total" />
					<MetricBlock
						value={stats.stale_count}
						label="Stale"
						warn={stats.stale_count > 0}
						warnColor="#e67e22"
					/>
					<MetricBlock
						value={stats.conflict_count}
						label="Conflicts"
						warn={stats.conflict_count > 0}
						warnColor="#c0392b"
					/>
				</div>
				{stats.vitality_dist && <VitalityDist dist={stats.vitality_dist} />}
			</CardContent>
		</Card>
	);
}

function MetricBlock({
	value,
	label,
	warn,
	warnColor,
}: { value: number; label: string; warn?: boolean; warnColor?: string }) {
	return (
		<div className="rounded-lg bg-accent/50 px-2 py-2.5">
			<p
				className="text-xl font-bold"
				style={{ color: warn ? warnColor : undefined, fontFamily: "var(--font-display)" }}
			>
				{value}
			</p>
			<p className="text-[10px] text-muted-foreground">{label}</p>
		</div>
	);
}

function VitalityDist({ dist }: { dist: [number, number, number, number, number] }) {
	const labels = ["0-20", "21-40", "41-60", "61-80", "81-100"];
	const max = Math.max(...dist, 1);
	return (
		<div className="space-y-1.5">
			<p className="text-[10px] font-medium text-muted-foreground uppercase tracking-wider">
				Vitality
			</p>
			<div className="flex items-end gap-1.5 h-10">
				{dist.map((count, i) => (
					<div key={labels[i]} className="flex-1 flex flex-col items-center gap-1">
						<div
							className="w-full rounded-sm transition-all"
							style={{
								height: `${Math.max((count / max) * 100, count > 0 ? 8 : 0)}%`,
								backgroundColor: "#2d8b7a",
								opacity: 0.25 + (i / 4) * 0.75,
							}}
						/>
						<span className="text-[9px] text-muted-foreground">{labels[i]}</span>
					</div>
				))}
			</div>
		</div>
	);
}

function EpicProgressCard({ epics }: { epics?: EpicSummary[] }) {
	if (!epics || epics.length === 0) return null;
	return (
		<Card className="border-stone-200 dark:border-stone-700">
			<CardHeader className="pb-3">
				<CardTitle className="text-sm font-semibold" style={{ fontFamily: "var(--font-display)" }}>
					Epics
				</CardTitle>
			</CardHeader>
			<CardContent className="space-y-3">
				{epics.map((epic) => {
					const progress = epic.total > 0 ? (epic.completed / epic.total) * 100 : 0;
					return (
						<div key={epic.slug} className="space-y-1.5">
							<div className="flex items-center justify-between gap-2">
								<span className="text-sm font-medium truncate">{epic.name}</span>
								<span className="text-xs tabular-nums text-muted-foreground shrink-0">
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

function RecentDecisionsCard({ decisions }: { decisions?: DecisionEntry[] }) {
	if (!decisions || decisions.length === 0) return null;
	return (
		<Card className="border-stone-200 dark:border-stone-700">
			<CardHeader className="pb-3">
				<CardTitle className="text-sm font-semibold" style={{ fontFamily: "var(--font-display)" }}>
					Recent Decisions
				</CardTitle>
			</CardHeader>
			<CardContent>
				<div className="space-y-3">
					{decisions.map((dec, i) => (
						<div key={`${dec.task_slug}-${dec.title}-${i}`}>
							{i > 0 && <Separator className="mb-3" />}
							<div className="flex items-start gap-3">
								<div
									className="mt-1.5 size-2 shrink-0 rounded-full"
									style={{ backgroundColor: "#628141" }}
								/>
								<div className="min-w-0 flex-1">
									<p className="text-sm font-medium leading-snug">{dec.title}</p>
									{dec.chosen && (
										<p className="mt-0.5 text-xs text-muted-foreground line-clamp-1">
											{dec.chosen}
										</p>
									)}
								</div>
								<Badge variant="outline" className="shrink-0 text-[10px] rounded-full">
									{dec.task_slug}
								</Badge>
							</div>
						</div>
					))}
				</div>
			</CardContent>
		</Card>
	);
}
