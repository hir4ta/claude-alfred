import { useQueries, useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { CircleCheck, CircleDot } from "@animated-color-icons/lucide-react";
import { SectionCard } from "@/components/section-card";
import { TraceabilityMatrix } from "@/components/traceability";
import { WaveTimeline } from "@/components/wave-timeline";
import { ButlerEmpty } from "@/components/butler-empty";
import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
	specContentQueryOptions,
	specsQueryOptions,
	tasksQueryOptions,
	validationQueryOptions,
} from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import type { ValidationReport } from "@/lib/types";

export const Route = createFileRoute("/tasks/$slug")({
	component: TaskDetailPage,
});

function TaskDetailPage() {
	const { t } = useI18n();
	const { slug } = Route.useParams();
	const search = Route.useSearch() as { project?: string };
	const projectId = search.project;
	const { data: tasksData } = useQuery(tasksQueryOptions(projectId));
	const { data: specsData } = useQuery(specsQueryOptions(slug, projectId));
	const { data: validationData } = useQuery(validationQueryOptions(slug, projectId));

	const task = tasksData?.tasks.find((t) => t.slug === slug);
	const specs = specsData?.specs ?? [];

	const specContents = useQueries({
		queries: specs.map((spec) => specContentQueryOptions(slug, spec.file, projectId)),
	});

	if (!task) {
		return <p className="text-sm text-muted-foreground">{t("task.notFound")}</p>;
	}

	return (
		<div className="flex flex-col h-[calc(100vh-120px)]">
			{/* Header */}
			<div className="shrink-0 pb-5 space-y-4">
				<div className="flex items-center gap-2.5 flex-wrap">
					{(task.status === "completed" || task.status === "done") ? (
						<CircleCheck className="size-4 shrink-0" style={{ color: "#2d8b7a" }} />
					) : (
						<CircleDot className="size-4 shrink-0" style={{ color: "#40513b" }} />
					)}
					<h2 className="text-lg font-semibold font-mono truncate">{task.slug}</h2>
					{task.size && <Badge variant="outline" style={{ borderColor: "rgba(123,107,141,0.4)", color: "#7b6b8d" }}>{task.size}</Badge>}
					{task.spec_type && <Badge variant="outline" style={{ borderColor: "rgba(98,129,65,0.4)", color: "#628141" }}>{task.spec_type}</Badge>}
					{validationData && <ValidationBadge report={validationData} />}
				</div>
				{task.waves && task.waves.length > 0 && (
					<div className="pt-1">
						<WaveTimeline waves={task.waves} />
					</div>
				)}
			</div>

			{/* Scrollable content */}
			<div className="flex-1 overflow-y-auto space-y-4 pb-8">
				{specs.map((spec, i) => {
					const content = specContents[i]?.data?.content ?? "";
					if (!content) return null;
					return (
						<SectionCard
							key={spec.file}
							title={spec.file}
							content={content}
							defaultOpen={spec.file === "session.md"}
							slug={slug}
						/>
					);
				})}
				{specs.length === 0 && (
					<ButlerEmpty scene="empty-tray" messageKey="empty.noSpecs" />
				)}

				{/* Traceability */}
				{(() => {
					const contentMap: Record<string, string> = {};
					for (let i = 0; i < specs.length; i++) {
						const c = specContents[i]?.data?.content;
						if (c) contentMap[specs[i]!.file] = c;
					}
					return Object.keys(contentMap).length > 0 ? <TraceabilityMatrix specContents={contentMap} /> : null;
				})()}
			</div>
		</div>
	);
}

function ValidationBadge({ report }: { report: ValidationReport }) {
	const { t } = useI18n();
	const passed = report.checks.filter((c) => c.status === "pass").length;
	const failed = report.checks.filter((c) => c.status === "fail").length;
	const color = failed > 0 ? "#c0392b" : "#2d8b7a";
	return (
		<Tooltip>
			<TooltipTrigger asChild>
				<Badge
					variant="outline"
					className="text-xs cursor-help"
					style={{ borderColor: color, color }}
				>
					{passed}P / {failed}F
				</Badge>
			</TooltipTrigger>
			<TooltipContent>
				<p>
					{t("task.validation")}: {passed} {t("task.passed")}, {failed} {t("task.failed")}
				</p>
			</TooltipContent>
		</Tooltip>
	);
}
