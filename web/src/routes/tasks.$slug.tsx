import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import {
	specContentQueryOptions,
	specsQueryOptions,
	tasksQueryOptions,
	validationQueryOptions,
} from "@/lib/api";
import type { SpecEntry, TaskDetail, ValidationReport } from "@/lib/types";
import { cn } from "@/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";

export const Route = createFileRoute("/tasks/$slug")({
	component: TaskDetailPage,
});

function TaskDetailPage() {
	const { slug } = Route.useParams();
	const { data: tasksData } = useQuery(tasksQueryOptions());
	const { data: specsData } = useQuery(specsQueryOptions(slug));
	const { data: validationData } = useQuery(validationQueryOptions(slug));
	const [selectedFile, setSelectedFile] = useState<string | null>(null);

	const task = tasksData?.tasks.find((t) => t.slug === slug);
	const specs = specsData?.specs ?? [];

	if (!task) {
		return <p className="text-sm text-muted-foreground">Task not found.</p>;
	}

	return (
		<div className="space-y-4">
			<TaskHeader task={task} validation={validationData} />
			<Separator />
			<div className="flex gap-4">
				<SpecFileList specs={specs} selected={selectedFile} onSelect={setSelectedFile} />
				{selectedFile && <SpecContentViewer slug={slug} file={selectedFile} />}
			</div>
		</div>
	);
}

function TaskHeader({ task, validation }: { task: TaskDetail; validation?: ValidationReport }) {
	return (
		<div className="space-y-2">
			<div className="flex items-center gap-3">
				<h2 className="text-lg font-semibold">{task.slug}</h2>
				{task.size && <Badge variant="outline">{task.size}</Badge>}
				{task.spec_type && <Badge variant="outline">{task.spec_type}</Badge>}
				{validation && <ValidationBadge report={validation} />}
			</div>
			{task.focus && <p className="text-sm text-muted-foreground">{task.focus}</p>}
			{task.next_steps && task.next_steps.length > 0 && (
				<div className="space-y-1">
					<p className="text-xs font-medium text-muted-foreground">Next Steps</p>
					{task.next_steps.map((step, i) => (
						<div key={`step-${i}`} className="flex items-center gap-2 text-sm">
							<span className={cn("text-xs", step.done && "line-through text-muted-foreground")}>
								{step.done ? "[x]" : "[ ]"} {step.text}
							</span>
						</div>
					))}
				</div>
			)}
		</div>
	);
}

function ValidationBadge({ report }: { report: ValidationReport }) {
	const passed = report.checks.filter((c) => c.status === "pass").length;
	const failed = report.checks.filter((c) => c.status === "fail").length;
	const color = failed > 0 ? "#c0392b" : "#2d8b7a";

	return (
		<Badge variant="outline" className="text-xs" style={{ borderColor: color, color }}>
			{passed}P / {failed}F
		</Badge>
	);
}

function SpecFileList({
	specs,
	selected,
	onSelect,
}: {
	specs: SpecEntry[];
	selected: string | null;
	onSelect: (file: string) => void;
}) {
	return (
		<div className="w-48 shrink-0 space-y-1">
			{specs.map((spec) => (
				<button
					type="button"
					key={spec.file}
					onClick={() => onSelect(spec.file)}
					className={cn(
						"w-full rounded px-2 py-1.5 text-left text-sm transition-colors",
						"hover:bg-accent",
						selected === spec.file && "bg-accent font-medium",
					)}
				>
					{spec.file}
				</button>
			))}
			{specs.length === 0 && <p className="text-xs text-muted-foreground px-2">No spec files.</p>}
		</div>
	);
}

function SpecContentViewer({ slug, file }: { slug: string; file: string }) {
	const { data, isLoading } = useQuery(specContentQueryOptions(slug, file));

	if (isLoading) {
		return <p className="text-sm text-muted-foreground">Loading...</p>;
	}

	return (
		<Card className="min-w-0 flex-1">
			<CardHeader className="py-2 px-4">
				<CardTitle className="text-sm font-medium">{file}</CardTitle>
			</CardHeader>
			<CardContent className="p-0">
				<ScrollArea className="h-[600px]">
					<pre className="p-4 text-xs leading-relaxed whitespace-pre-wrap break-words font-mono">
						{data?.content ?? "No content."}
					</pre>
				</ScrollArea>
			</CardContent>
		</Card>
	);
}
