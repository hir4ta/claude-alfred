import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { tasksQueryOptions } from "@/lib/api";
import type { StepItem, TaskDetail } from "@/lib/types";
import { cn } from "@/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { Link, Outlet, createFileRoute } from "@tanstack/react-router";
import { CircleCheck, CircleDot, Circle } from "lucide-react";

export const Route = createFileRoute("/tasks")({
	component: TasksLayout,
});

function TasksLayout() {
	const { data } = useQuery(tasksQueryOptions());
	const tasks = data?.tasks ?? [];
	const activeSlug = data?.active ?? "";

	return (
		<div className="flex gap-6">
			<div className="w-80 shrink-0 space-y-3">
				{tasks.map((task) => (
					<TaskListCard key={task.slug} task={task} isActive={task.slug === activeSlug} />
				))}
				{tasks.length === 0 && <p className="text-sm text-muted-foreground">No tasks found.</p>}
			</div>
			<div className="min-w-0 flex-1">
				<Outlet />
			</div>
		</div>
	);
}

function TaskListCard({ task, isActive }: { task: TaskDetail; isActive: boolean }) {
	const progress = task.total > 0 ? (task.completed / task.total) * 100 : 0;
	const isCompleted = task.status === "completed";
	const firstUnchecked = task.next_steps?.find((s) => !s.done);

	return (
		<Link to="/tasks/$slug" params={{ slug: task.slug }}>
			<Card
				className={cn(
					"transition-colors hover:border-brand-pattern/30 cursor-pointer",
					isActive && "border-brand-session/40 bg-brand-session/[0.04]",
					isCompleted && "opacity-60",
				)}
			>
				<CardHeader className="p-3 pb-1.5">
					<div className="flex items-center justify-between">
						<div className="flex items-center gap-2 min-w-0">
							{isCompleted ? (
								<CircleCheck className="size-3.5 shrink-0 text-brand-pattern" />
							) : isActive ? (
								<CircleDot className="size-3.5 shrink-0 text-brand-session" />
							) : (
								<Circle className="size-3.5 shrink-0 text-muted-foreground/40" />
							)}
							<CardTitle className="text-sm font-medium truncate">{task.slug}</CardTitle>
						</div>
						<div className="flex gap-1">
							{task.size && (
								<Badge variant="outline" className="text-[10px] px-1 py-0">
									{task.size}
								</Badge>
							)}
						</div>
					</div>
				</CardHeader>
				<CardContent className="p-3 pt-0 space-y-1.5">
					{task.focus && (
						<p className="text-[11px] text-muted-foreground line-clamp-1">{task.focus}</p>
					)}
					<div className="flex items-center gap-2">
						<Progress value={progress} className="h-1 flex-1" />
						<span className="text-[10px] tabular-nums text-muted-foreground">
							{task.completed}/{task.total}
						</span>
					</div>
					{firstUnchecked && !isCompleted && <NextStepHighlight step={firstUnchecked} />}
				</CardContent>
			</Card>
		</Link>
	);
}

function NextStepHighlight({ step }: { step: StepItem }) {
	return (
		<div className="relative overflow-hidden rounded-md px-2 py-1">
			<div
				className="absolute inset-0 animate-shimmer"
				style={{
					background:
						"linear-gradient(90deg, rgba(45,139,122,0.03) 0%, rgba(45,139,122,0.10) 50%, rgba(45,139,122,0.03) 100%)",
					backgroundSize: "200% 100%",
				}}
			/>
			<p className="relative text-[10px] text-muted-foreground line-clamp-1">→ {step.text}</p>
		</div>
	);
}
