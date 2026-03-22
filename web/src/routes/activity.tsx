import { Link, createFileRoute } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { activityQueryOptions, analyticsQueryOptions, heatmapQueryOptions } from "@/lib/api";
import { SummaryCards, ReworkChart, CycleTimeChart } from "@/components/activity/metrics-section";
import { ActivityHeatmap } from "@/components/activity/activity-heatmap";
import { DetailDrawer } from "@/components/detail-drawer";
import { useI18n } from "@/lib/i18n";

export const Route = createFileRoute("/activity")({
	component: ActivityPage,
});

function ActivityPage() {
	const { t } = useI18n();
	const { data: analytics } = useQuery(analyticsQueryOptions());
	const { data: heatmap } = useQuery(heatmapQueryOptions());
	const [page, setPage] = useState(0);
	const { data: activity } = useQuery(activityQueryOptions(page));
	const [selectedEntry, setSelectedEntry] = useState<AuditEntry | null>(null);

	const hasMetrics = analytics && (
		(analytics.reworkRates?.length ?? 0) > 0 ||
		(analytics.cycleTimeBreakdown?.length ?? 0) > 0
	);

	return (
		<div className="flex flex-col gap-6 h-[calc(100vh-8rem)]">
			<h1
				className="text-2xl font-bold tracking-tight shrink-0"
				style={{ fontFamily: "var(--font-display)" }}
			>
				{t("activity.title")}
			</h1>

			{hasMetrics ? (
				<div className="shrink-0 space-y-6">
					<SummaryCards analytics={analytics!} />
					<div className="grid gap-6 lg:grid-cols-2">
						<ReworkChart analytics={analytics!} />
						<CycleTimeChart analytics={analytics!} />
					</div>
				</div>
			) : (
				<div className="flex flex-col items-center justify-center py-12 text-center shrink-0">
					<p className="text-lg font-medium text-muted-foreground" style={{ fontFamily: "var(--font-display)" }}>
						{t("activity.empty.title")}
					</p>
					<p className="mt-2 text-sm text-muted-foreground/70">
						{t("activity.empty.description")}
					</p>
				</div>
			)}

			{/* Heatmap */}
			<div className="shrink-0">
				<ActivityHeatmap data={heatmap?.data ?? []} weeks={heatmap?.weeks ?? 16} />
			</div>

			<AuditLogTable
				entries={activity?.entries ?? []}
				total={activity?.total ?? 0}
				page={page}
				onPageChange={setPage}
				onSelect={setSelectedEntry}
			/>

			{/* Activity detail drawer */}
			<DetailDrawer
				open={!!selectedEntry}
				onClose={() => setSelectedEntry(null)}
				title={selectedEntry ? (selectedEntry.target ? `${selectedEntry.action} — ${selectedEntry.target}` : selectedEntry.action) : ""}
			>
				{selectedEntry && (
					<div className="space-y-3 text-sm">
						<div className="flex items-center gap-2">
							<EventBadge event={selectedEntry.action} />
						</div>
						<div className="space-y-2">
							<div>
								<p className="text-[11px] text-muted-foreground uppercase tracking-wide">{t("activity.log.slug")}</p>
								{selectedEntry.target ? (
									<Link to="/tasks/$slug" params={{ slug: selectedEntry.target }} className="font-mono text-sm hover:underline" style={{ color: "#40513b" }}>
										{selectedEntry.target}
									</Link>
								) : (
									<p className="text-muted-foreground">—</p>
								)}
							</div>
							<div>
								<p className="text-[11px] text-muted-foreground uppercase tracking-wide">{t("activity.log.actor")}</p>
								<p>{selectedEntry.actor || "—"}</p>
							</div>
							<div>
								<p className="text-[11px] text-muted-foreground uppercase tracking-wide">{t("activity.log.time")}</p>
								<p className="font-mono text-[11px]">{new Date(selectedEntry.timestamp).toLocaleString()}</p>
							</div>
							<div>
								<p className="text-[11px] text-muted-foreground uppercase tracking-wide">{t("activity.log.detail")}</p>
								<p className="whitespace-pre-wrap break-all text-[12px] leading-relaxed">{selectedEntry.detail || "—"}</p>
							</div>
						</div>
					</div>
				)}
			</DetailDrawer>
		</div>
	);
}

// --- Audit Log Table (sticky header, scrollable body) ---

interface AuditEntry {
	timestamp: string;
	action: string;
	target: string;
	detail: string;
	actor: string;
	project_name?: string;
}

function AuditLogTable({
	entries,
	total,
	page,
	onPageChange,
	onSelect,
}: {
	entries: AuditEntry[];
	total: number;
	page: number;
	onPageChange: (p: number) => void;
	onSelect: (entry: AuditEntry) => void;
}) {
	const { t } = useI18n();
	const totalPages = Math.ceil(total / 50);

	return (
		<div className="rounded-organic border border-border/60 bg-card flex flex-col min-h-0 flex-1">
			<div className="flex items-center justify-between py-3 px-4 border-b border-border/30 shrink-0">
				<h3 className="text-sm font-semibold">{t("activity.log.title")}</h3>
				{totalPages > 1 && (
					<div className="flex items-center gap-2 text-[11px] text-muted-foreground">
						<span>{total} {t("activity.log.entries")}</span>
						<button
							type="button"
							disabled={page === 0}
							onClick={() => onPageChange(page - 1)}
							className="px-2 py-0.5 rounded border border-border/40 disabled:opacity-30"
						>
							{t("activity.log.prev")}
						</button>
						<span>{page + 1}/{totalPages}</span>
						<button
							type="button"
							disabled={page >= totalPages - 1}
							onClick={() => onPageChange(page + 1)}
							className="px-2 py-0.5 rounded border border-border/40 disabled:opacity-30"
						>
							{t("activity.log.next")}
						</button>
					</div>
				)}
			</div>
			{entries.length === 0 ? (
				<p className="text-sm text-muted-foreground py-8 text-center">{t("activity.noMetrics")}</p>
			) : (
				<div className="overflow-auto flex-1 min-h-0">
					<table className="w-full text-sm">
						<thead className="sticky top-0 bg-card z-10">
							<tr className="border-b border-border/40 text-left text-[11px] text-muted-foreground uppercase tracking-wider">
								<th className="py-2 px-4 whitespace-nowrap">{t("activity.log.time")}</th>
								<th className="py-2 pr-3 whitespace-nowrap">{t("activity.log.event")}</th>
								<th className="py-2 pr-3 whitespace-nowrap">{t("activity.log.slug")}</th>
								<th className="py-2 pr-3 whitespace-nowrap">{t("activity.log.actor")}</th>
								<th className="py-2 pr-4">{t("activity.log.detail")}</th>
							</tr>
						</thead>
						<tbody>
							{entries.map((e, i) => (
								<AuditRow key={i} entry={e} onSelect={() => onSelect(e)} />
							))}
						</tbody>
					</table>
				</div>
			)}
		</div>
	);
}

// --- Event Badge ---

const EVENT_COLORS: Record<string, { bg: string; text: string }> = {
	"spec.init": { bg: "#40513b20", text: "#40513b" },
	"spec.complete": { bg: "#62814120", text: "#628141" },
	"review.submit": { bg: "#2d8b7a20", text: "#2d8b7a" },
	"gate.set": { bg: "#e67e2220", text: "#e67e22" },
	"gate.clear": { bg: "#62814120", text: "#628141" },
	"gate.fix": { bg: "#e67e2220", text: "#e67e22" },
	"first_commit": { bg: "#7b6b8d20", text: "#7b6b8d" },
	"task.status_change": { bg: "#44403c15", text: "#44403c" },
	"living-spec.update": { bg: "#2d8b7a15", text: "#2d8b7a" },
	"rework.checked": { bg: "#c0392b20", text: "#c0392b" },
};

function EventBadge({ event }: { event: string }) {
	const colors = EVENT_COLORS[event] ?? { bg: "#44403c10", text: "#44403c" };
	return (
		<span
			className="inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium font-mono whitespace-nowrap"
			style={{ backgroundColor: colors.bg, color: colors.text }}
		>
			{event}
		</span>
	);
}

function AuditRow({ entry: e, onSelect }: { entry: AuditEntry; onSelect: () => void }) {
	return (
		<>
			<tr
				className="border-b border-border/10 last:border-0 hover:bg-muted/20 cursor-pointer"
				onClick={onSelect}
			>
				<td className="py-1.5 px-4 text-[11px] text-muted-foreground font-mono whitespace-nowrap">
					{formatTimestamp(e.timestamp)}
				</td>
				<td className="py-1.5 pr-3">
					<EventBadge event={e.action} />
				</td>
				<td className="py-1.5 pr-3 font-mono text-[11px]">{e.target}</td>
				<td className="py-1.5 pr-3 text-[11px] text-muted-foreground">{e.actor}</td>
				<td className="py-1.5 pr-4 text-[11px] text-muted-foreground">
					<span className="truncate max-w-[300px] inline-block align-bottom">{e.detail}</span>
				</td>
			</tr>
		</>
	);
}

function formatTimestamp(ts: string): string {
	const d = new Date(ts);
	const month = String(d.getMonth() + 1).padStart(2, "0");
	const day = String(d.getDate()).padStart(2, "0");
	const hours = String(d.getHours()).padStart(2, "0");
	const mins = String(d.getMinutes()).padStart(2, "0");
	return `${month}/${day} ${hours}:${mins}`;
}
