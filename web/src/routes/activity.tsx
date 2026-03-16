import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Skeleton } from "@/components/ui/skeleton";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { activityQueryOptions, epicsQueryOptions } from "@/lib/api";
import type { ActivityEntry, EpicSummary } from "@/lib/types";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";

export const Route = createFileRoute("/activity")({
	component: ActivityPage,
});

const FILTERS = ["all", "spec.init", "spec.complete", "review.submit"] as const;

function ActivityPage() {
	const [filter, setFilter] = useState<string>("all");
	const { data: activityData, isLoading } = useQuery(
		activityQueryOptions(100, filter === "all" ? undefined : filter),
	);
	const { data: epicsData } = useQuery(epicsQueryOptions());

	const entries = activityData?.entries ?? [];
	const epics = epicsData?.epics ?? [];

	return (
		<div className="space-y-6">
			<Tabs value={filter} onValueChange={setFilter}>
				<TabsList>
					{FILTERS.map((f) => (
						<TabsTrigger key={f} value={f} className="text-xs">
							{f === "all" ? "All" : f}
						</TabsTrigger>
					))}
				</TabsList>
			</Tabs>

			{isLoading ? (
				<div className="space-y-2">
					{Array.from({ length: 5 }).map((_, i) => (
						<Skeleton key={`skel-${i}`} className="h-10 w-full" />
					))}
				</div>
			) : (
				<ActivityTable entries={entries} />
			)}

			{epics.length > 0 && <EpicSection epics={epics} />}
		</div>
	);
}

function ActivityTable({ entries }: { entries: ActivityEntry[] }) {
	return (
		<Table>
			<TableHeader>
				<TableRow>
					<TableHead className="w-44">Timestamp</TableHead>
					<TableHead className="w-32">Action</TableHead>
					<TableHead>Target</TableHead>
					<TableHead>Detail</TableHead>
				</TableRow>
			</TableHeader>
			<TableBody>
				{entries.map((entry, i) => (
					<TableRow key={`${entry.timestamp}-${i}`}>
						<TableCell className="text-xs text-muted-foreground font-mono">
							{formatTimestamp(entry.timestamp)}
						</TableCell>
						<TableCell>
							<ActionBadge action={entry.action} />
						</TableCell>
						<TableCell className="text-sm">{entry.target}</TableCell>
						<TableCell className="text-xs text-muted-foreground max-w-xs truncate">
							{entry.detail}
						</TableCell>
					</TableRow>
				))}
				{entries.length === 0 && (
					<TableRow>
						<TableCell colSpan={4} className="text-center text-sm text-muted-foreground">
							No activity found.
						</TableCell>
					</TableRow>
				)}
			</TableBody>
		</Table>
	);
}

const ACTION_COLORS: Record<string, string> = {
	"spec.init": "#40513b",
	"spec.complete": "#2d8b7a",
	"spec.delete": "#c0392b",
	"review.submit": "#628141",
	"living-spec.update": "#7b6b8d",
};

function ActionBadge({ action }: { action: string }) {
	const color = ACTION_COLORS[action] ?? "#6b7280";
	return (
		<Badge variant="outline" className="text-xs" style={{ borderColor: `${color}40`, color }}>
			{action}
		</Badge>
	);
}

function EpicSection({ epics }: { epics: EpicSummary[] }) {
	return (
		<div className="space-y-3">
			<h3 className="text-sm font-medium text-foreground">Epics</h3>
			<div className="grid gap-3 sm:grid-cols-2">
				{epics.map((epic) => {
					const progress = epic.total > 0 ? (epic.completed / epic.total) * 100 : 0;
					return (
						<Card key={epic.slug}>
							<CardHeader className="pb-2">
								<div className="flex items-center justify-between">
									<CardTitle className="text-sm">{epic.name}</CardTitle>
									<Badge variant="outline" className="text-xs">
										{epic.status}
									</Badge>
								</div>
							</CardHeader>
							<CardContent className="space-y-2">
								<div className="flex items-center gap-2">
									<Progress value={progress} className="h-1.5 flex-1" />
									<span className="text-xs text-muted-foreground">
										{epic.completed}/{epic.total}
									</span>
								</div>
								{epic.tasks && epic.tasks.length > 0 && (
									<div className="flex flex-wrap gap-1">
										{epic.tasks.map((t) => (
											<Badge
												key={t.slug}
												variant="outline"
												className="text-[10px]"
												style={{
													borderColor:
														t.status === "completed"
															? "rgba(45,139,122,0.3)"
															: "rgba(107,114,128,0.3)",
													color: t.status === "completed" ? "#2d8b7a" : "#6b7280",
												}}
											>
												{t.slug}
											</Badge>
										))}
									</div>
								)}
							</CardContent>
						</Card>
					);
				})}
			</div>
		</div>
	);
}

function formatTimestamp(ts: string): string {
	try {
		const d = new Date(ts);
		return d.toLocaleString("ja-JP", {
			month: "2-digit",
			day: "2-digit",
			hour: "2-digit",
			minute: "2-digit",
			second: "2-digit",
		});
	} catch {
		return ts;
	}
}
