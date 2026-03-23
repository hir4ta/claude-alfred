import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useSearch } from "@tanstack/react-router";
import { Brain, CheckCircle2, Clock, FolderOpen, Zap } from "@animated-color-icons/lucide-react";
import { useState } from "react";
import {
	Pagination,
	PaginationContent,
	PaginationItem,
	PaginationLink,
	PaginationNext,
	PaginationPrevious,
} from "@/components/ui/pagination";
import { StaggerContainer } from "@/components/stagger-container";
import {
	knowledgeStatsQueryOptions,
	projectsQueryOptions,
	tasksQueryOptions,
} from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { StatCard } from "@/components/overview/stat-card";
import { TaskCard } from "@/components/overview/task-card";
import { HeroTile } from "@/components/overview/hero-tile";

export const Route = createFileRoute("/")({
	component: OverviewPage,
});

const ITEMS_PER_PAGE = 6;

function OverviewPage() {
	const { t } = useI18n();
	const [taskPage, setTaskPage] = useState(1);
	const search = useSearch({ strict: false }) as { project?: string };
	const projectId = search.project;
	const { data: tasksData, isLoading: tasksLoading } = useQuery(tasksQueryOptions(projectId));
	const { data: statsData } = useQuery(knowledgeStatsQueryOptions(projectId));
	const { data: projectsData } = useQuery(projectsQueryOptions());

	const tasks = [...(tasksData?.tasks ?? [])].sort((a, b) => {
		const aTime = a.started_at ?? "";
		const bTime = b.started_at ?? "";
		return bTime.localeCompare(aTime);
	});

	const activeTasks = tasks.filter((task) => {
		const s = task.status;
		return s !== "completed" && s !== "done" && s !== "cancelled";
	});
	const heroTask = activeTasks[0];
	const remainingTasks = heroTask ? tasks.filter((t) => t.slug !== heroTask.slug) : tasks;

	return (
		<div className="space-y-6">
			{/* Bento Grid */}
			<div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
				{/* Stats row */}
				<StatCard
					label={t("overview.totalTasks")}
					value={tasks.length}
					icon={<Zap className="size-4" style={{ color: "#e67e22" }} />}
					loading={tasksLoading}
				/>
				<StatCard
					label={t("overview.active")}
					value={activeTasks.length}
					icon={<Clock className="size-4" style={{ color: "#40513b" }} />}
					loading={tasksLoading}
				/>
				<StatCard
					label={t("overview.completed")}
					value={tasks.filter((t) => t.status === "completed" || t.status === "done").length}
					icon={<CheckCircle2 className="size-4" style={{ color: "#2d8b7a" }} />}
					loading={tasksLoading}
				/>
				<StatCard
					label={t("overview.knowledge")}
					value={statsData?.total ?? 0}
					icon={<Brain className="size-4" style={{ color: "#628141" }} />}
				/>

				{/* Hero tile — 2 col span, only if active spec exists */}
				{heroTask && (
					<div className="sm:col-span-2">
						<HeroTile task={heroTask} />
					</div>
				)}
			</div>

			{/* Projects */}
			{(projectsData?.projects?.length ?? 0) > 0 && (
				<section className="space-y-3">
					<h2
						className="text-2xl font-semibold tracking-tight text-foreground"
						style={{ fontFamily: "var(--font-display)" }}
					>
						{t("overview.projects")}
					</h2>
					<div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
						{projectsData!.projects.filter((p) => p.status === "active").map((project) => (
							<div
								key={project.id}
								className="al-icon-wrapper flex items-center gap-3 rounded-organic border border-border/60 bg-card px-4 py-3 hover:border-border hover:-translate-y-0.5 transition-transform"
							>
								<FolderOpen className="size-4 shrink-0" style={{ color: "#40513b" }} />
								<div className="min-w-0 flex-1">
									<p className="text-sm font-medium truncate">{project.name}</p>
									<p className="text-[11px] text-muted-foreground truncate font-mono">{project.path}</p>
								</div>
							</div>
						))}
					</div>
				</section>
			)}

			{/* Remaining task cards */}
			{remainingTasks.length > 0 && (() => {
				const totalPages = Math.ceil(remainingTasks.length / ITEMS_PER_PAGE);
				const paged = remainingTasks.slice((taskPage - 1) * ITEMS_PER_PAGE, taskPage * ITEMS_PER_PAGE);
				return (
					<section className="space-y-3">
						<h2
							className="text-2xl font-semibold tracking-tight text-foreground"
							style={{ fontFamily: "var(--font-display)" }}
						>
							{t("overview.tasks")}
						</h2>
						<StaggerContainer className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
							{paged.map((task, i) => (
								<TaskCard
									key={task.slug}
									task={task}
									colorIndex={(taskPage - 1) * ITEMS_PER_PAGE + i}
								/>
							))}
						</StaggerContainer>
						{totalPages > 1 && (
							<SimplePagination page={taskPage} totalPages={totalPages} onPageChange={setTaskPage} />
						)}
					</section>
				);
			})()}
		</div>
	);
}

function SimplePagination({
	page,
	totalPages,
	onPageChange,
}: {
	page: number;
	totalPages: number;
	onPageChange: (page: number) => void;
}) {
	return (
		<Pagination>
			<PaginationContent>
				<PaginationItem>
					<PaginationPrevious
						onClick={() => onPageChange(Math.max(1, page - 1))}
						aria-disabled={page <= 1}
						className={page <= 1 ? "pointer-events-none opacity-50" : "cursor-pointer"}
					/>
				</PaginationItem>
				{Array.from({ length: totalPages }, (_, i) => i + 1).map((p) => (
					<PaginationItem key={p}>
						<PaginationLink
							isActive={p === page}
							onClick={() => onPageChange(p)}
							className="cursor-pointer"
						>
							{p}
						</PaginationLink>
					</PaginationItem>
				))}
				<PaginationItem>
					<span className="text-xs text-muted-foreground tabular-nums px-2">
						{page} / {totalPages}
					</span>
				</PaginationItem>
				<PaginationItem>
					<PaginationNext
						onClick={() => onPageChange(Math.min(totalPages, page + 1))}
						aria-disabled={page >= totalPages}
						className={page >= totalPages ? "pointer-events-none opacity-50" : "cursor-pointer"}
					/>
				</PaginationItem>
			</PaginationContent>
		</Pagination>
	);
}
