import { useState } from "react";
import { Link } from "@tanstack/react-router";
import { motion } from "motion/react";
import { CircleCheck, CircleDot } from "@animated-color-icons/lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { StatusBadge } from "@/components/status-badge";
import { useI18n } from "@/lib/i18n";
import type { TaskDetail } from "@/lib/types";
import { formatDate } from "@/lib/format";
import { cn } from "@/lib/utils";

const SHIMMER_COLORS = [
	{ r: 45, g: 139, b: 122 },
	{ r: 98, g: 129, b: 65 },
	{ r: 123, g: 107, b: 141 },
	{ r: 230, g: 126, b: 34 },
	{ r: 64, g: 81, b: 59 },
];

export function TaskCard({
	task,
	colorIndex,
}: {
	task: TaskDetail;
	colorIndex: number;
}) {
	const { locale } = useI18n();
	const progress = (task.total ?? 0) > 0 ? ((task.completed ?? 0) / (task.total ?? 1)) * 100 : 0;
	const isCompleted = task.status === "completed" || task.status === "done" || task.status === "cancelled";
	const c = SHIMMER_COLORS[colorIndex % SHIMMER_COLORS.length]!;
	const accentColor = `rgb(${c.r},${c.g},${c.b})`;
	const [isRevealed, setIsRevealed] = useState(false);

	return (
		<Link to="/tasks/$slug" params={{ slug: task.slug }} className="block">
			<Card
				className={cn(
					"al-icon-wrapper h-[80px] !gap-0 !py-0 rounded-organic border-stone-200 transition-[border-color,transform] duration-200 hover:border-stone-300 hover:-translate-y-0.5 dark:border-stone-700 dark:hover:border-stone-600",
					isCompleted && "opacity-60",
				)}
				tabIndex={0}
				onMouseEnter={() => setIsRevealed(true)}
				onMouseLeave={() => setIsRevealed(false)}
				onFocus={() => setIsRevealed(true)}
				onBlur={() => setIsRevealed(false)}
			>
				<CardContent className="flex-1 flex flex-col p-4 gap-1 overflow-hidden relative">
					{/* Row 1: Spec name + status */}
					<div className="flex items-center gap-2 min-w-0">
						{isCompleted ? (
							<CircleCheck className="size-4 shrink-0" style={{ color: "#2d8b7a" }} />
						) : (
							<CircleDot className="size-4 shrink-0" style={{ color: accentColor }} />
						)}
						<span className="text-sm font-semibold font-mono truncate">{task.slug}</span>
						<StatusBadge status={task.status ?? "pending"} />
					</div>

					{/* Row 2: Progress bar */}
					<div className="flex items-center gap-2.5">
						<Progress value={progress} className="flex-1" />
						<span className="text-[11px] tabular-nums text-muted-foreground">
							{task.completed}/{task.total}
						</span>
					</div>

					{/* Hover reveal: project + date + badges */}
					<motion.div
						className="absolute bottom-0 left-0 right-0 px-4 pb-2 bg-card"
						initial={{ opacity: 0 }}
						animate={{ opacity: isRevealed ? 1 : 0 }}
						transition={{ type: "spring", damping: 25, stiffness: 200 }}
						aria-hidden={!isRevealed}
					>
						<div className="flex items-center gap-1.5">
							<span className="text-[10px] text-muted-foreground truncate">
								{task.project_name}{task.project_name && task.started_at ? " · " : ""}{task.started_at ? formatDate(task.started_at, locale) : ""}
							</span>
							{task.size && (
								<Badge variant="outline" className="text-[10px] px-1.5 py-0 rounded-full">
									{task.size}
								</Badge>
							)}
							{task.spec_type && task.spec_type !== "feature" && (
								<Badge variant="outline" className="text-[10px] px-1.5 py-0" style={{ borderColor: "rgba(98,129,65,0.4)", color: "#628141" }}>
									{task.spec_type}
								</Badge>
							)}
						</div>
					</motion.div>
				</CardContent>
			</Card>
		</Link>
	);
}
