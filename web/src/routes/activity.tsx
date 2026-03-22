import { createFileRoute } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import {
	BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, Cell, Legend,
} from "recharts";
import { activityQueryOptions, analyticsQueryOptions } from "@/lib/api";
import type { AnalyticsResponse } from "@/lib/api";
import { useI18n } from "@/lib/i18n";

export const Route = createFileRoute("/activity")({
	component: ActivityPage,
});

function ActivityPage() {
	const { t } = useI18n();
	const { data: analytics } = useQuery(analyticsQueryOptions());
	const [page, setPage] = useState(0);
	const { data: activity } = useQuery(activityQueryOptions(page));

	const hasMetrics = analytics && (
		(analytics.reworkRates?.length ?? 0) > 0 ||
		(analytics.cycleTimeBreakdown?.length ?? 0) > 0
	);

	return (
		<div className="space-y-6">
			<h1
				className="text-2xl font-bold tracking-tight"
				style={{ fontFamily: "var(--font-display)" }}
			>
				{t("activity.title")}
			</h1>

			{hasMetrics ? (
				<>
					<SummaryCards analytics={analytics!} />
					<div className="grid gap-6 lg:grid-cols-2">
						<ReworkChart analytics={analytics!} />
						<CycleTimeChart analytics={analytics!} />
					</div>
				</>
			) : (
				<div className="flex flex-col items-center justify-center py-16 text-center">
					<p className="text-lg font-medium text-muted-foreground" style={{ fontFamily: "var(--font-display)" }}>
						{t("activity.empty.title")}
					</p>
					<p className="mt-2 text-sm text-muted-foreground/70">
						{t("activity.empty.description")}
					</p>
				</div>
			)}

			<AuditLogTable
				entries={activity?.entries ?? []}
				total={activity?.total ?? 0}
				page={page}
				onPageChange={setPage}
			/>
		</div>
	);
}

// --- Summary Cards ---

function SummaryCards({ analytics }: { analytics: AnalyticsResponse }) {
	const { t } = useI18n();

	const confirmedRework = analytics.reworkRates.filter((r) => !r.pending);
	const avgRework = confirmedRework.length > 0
		? confirmedRework.reduce((s, r) => s + r.reworkRate, 0) / confirmedRework.length
		: 0;

	const avgCycle = analytics.cycleTimeBreakdown.length > 0
		? analytics.cycleTimeBreakdown.reduce((s, r) => s + r.phases.total, 0) / analytics.cycleTimeBreakdown.length
		: 0;

	const totalSpecs = analytics.cycleTimeBreakdown.length;

	const cards = [
		{ label: t("activity.avgCycleTime"), value: `${avgCycle.toFixed(1)}`, unit: t("activity.days"), color: "#628141" },
		{ label: t("activity.avgReworkRate"), value: `${(avgRework * 100).toFixed(0)}%`, unit: "", color: avgRework > 0.15 ? "#c0392b" : "#2d8b7a" },
		{ label: t("activity.totalSpecs"), value: String(totalSpecs), unit: "", color: "#40513b" },
	];

	return (
		<div className="grid gap-4 sm:grid-cols-3">
			{cards.map((card) => (
				<div
					key={card.label}
					className="rounded-organic border border-border/60 bg-card py-4 px-4 hover:-translate-y-0.5 transition-transform"
				>
					<p className="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">{card.label}</p>
					<p className="mt-1 text-2xl font-bold" style={{ fontFamily: "var(--font-display)", color: card.color }}>
						{card.value}
						{card.unit && <span className="ml-1 text-sm font-normal text-muted-foreground">{card.unit}</span>}
					</p>
				</div>
			))}
		</div>
	);
}

// --- Rework Rate Chart ---

function ReworkChart({ analytics }: { analytics: AnalyticsResponse }) {
	const { t } = useI18n();
	if (analytics.reworkRates.length === 0) return null;

	const data = analytics.reworkRates.map((r) => ({
		slug: r.slug.length > 16 ? `${r.slug.slice(0, 14)}..` : r.slug,
		rate: Math.round(r.reworkRate * 100),
		pending: r.pending,
	}));

	return (
		<div className="rounded-organic border border-border/60 bg-card py-4 px-4">
			<h3 className="text-sm font-semibold mb-3">{t("activity.rework.title")}</h3>
			<ResponsiveContainer width="100%" height={220}>
				<BarChart data={data} margin={{ top: 4, right: 8, bottom: 4, left: 0 }}>
					<XAxis dataKey="slug" tick={{ fontSize: 10 }} />
					<YAxis tick={{ fontSize: 10 }} unit="%" />
					<Tooltip formatter={(v: number) => `${v}%`} />
					<Bar dataKey="rate" name={t("activity.rework.rate")} radius={[4, 4, 0, 0]}>
						{data.map((entry, i) => (
							<Cell
								key={i}
								fill={entry.pending ? "#e67e22" : "#2d8b7a"}
								fillOpacity={entry.pending ? 0.5 : 1}
							/>
						))}
					</Bar>
				</BarChart>
			</ResponsiveContainer>
			{data.some((d) => d.pending) && (
				<p className="text-[10px] text-muted-foreground mt-1">{t("activity.rework.pending")}</p>
			)}
		</div>
	);
}

// --- Cycle Time Chart ---

const PHASE_COLORS = {
	planning: "#628141",
	approvalWait: "#e67e22",
	implementation: "#2d8b7a",
};

function CycleTimeChart({ analytics }: { analytics: AnalyticsResponse }) {
	const { t } = useI18n();
	if (analytics.cycleTimeBreakdown.length === 0) return null;

	const data = analytics.cycleTimeBreakdown.map((r) => ({
		slug: r.slug.length > 16 ? `${r.slug.slice(0, 14)}..` : r.slug,
		planning: r.phases.planning ?? 0,
		approval: r.phases.approvalWait ?? 0,
		implementation: r.phases.implementation ?? 0,
	}));

	return (
		<div className="rounded-organic border border-border/60 bg-card py-4 px-4">
			<h3 className="text-sm font-semibold mb-3">{t("activity.cycleTime.title")}</h3>
			<ResponsiveContainer width="100%" height={220}>
				<BarChart data={data} margin={{ top: 4, right: 8, bottom: 4, left: 0 }}>
					<XAxis dataKey="slug" tick={{ fontSize: 10 }} />
					<YAxis tick={{ fontSize: 10 }} unit="d" />
					<Tooltip />
					<Legend wrapperStyle={{ fontSize: 11 }} />
					<Bar dataKey="planning" name={t("activity.cycleTime.planning")} stackId="a" fill={PHASE_COLORS.planning} radius={[0, 0, 0, 0]} />
					<Bar dataKey="approval" name={t("activity.cycleTime.approval")} stackId="a" fill={PHASE_COLORS.approvalWait} />
					<Bar dataKey="implementation" name={t("activity.cycleTime.implementation")} stackId="a" fill={PHASE_COLORS.implementation} radius={[4, 4, 0, 0]} />
				</BarChart>
			</ResponsiveContainer>
		</div>
	);
}

// --- Audit Log Table ---

function AuditLogTable({
	entries,
	total,
	page,
	onPageChange,
}: {
	entries: Array<{ timestamp: string; event: string; slug: string; actor: string; detail: string }>;
	total: number;
	page: number;
	onPageChange: (p: number) => void;
}) {
	const { t } = useI18n();
	const totalPages = Math.ceil(total / 50);

	return (
		<div className="rounded-organic border border-border/60 bg-card py-4 px-4">
			<h3 className="text-sm font-semibold mb-3">{t("activity.log.title")}</h3>
			{entries.length === 0 ? (
				<p className="text-sm text-muted-foreground py-4 text-center">{t("activity.noMetrics")}</p>
			) : (
				<>
					<div className="overflow-x-auto">
						<table className="w-full text-sm">
							<thead>
								<tr className="border-b border-border/40 text-left text-[11px] text-muted-foreground uppercase tracking-wider">
									<th className="py-2 pr-3">{t("activity.log.time")}</th>
									<th className="py-2 pr-3">{t("activity.log.event")}</th>
									<th className="py-2 pr-3">{t("activity.log.slug")}</th>
									<th className="py-2 pr-3">{t("activity.log.actor")}</th>
									<th className="py-2">{t("activity.log.detail")}</th>
								</tr>
							</thead>
							<tbody>
								{entries.map((e, i) => (
									<tr key={i} className="border-b border-border/20 last:border-0">
										<td className="py-1.5 pr-3 text-[11px] text-muted-foreground font-mono whitespace-nowrap">
											{new Date(e.timestamp).toLocaleString()}
										</td>
										<td className="py-1.5 pr-3 font-mono text-[11px]">{e.event}</td>
										<td className="py-1.5 pr-3 font-mono text-[11px]">{e.slug}</td>
										<td className="py-1.5 pr-3 text-[11px]">{e.actor}</td>
										<td className="py-1.5 text-[11px] text-muted-foreground max-w-[200px] truncate">{e.detail}</td>
									</tr>
								))}
							</tbody>
						</table>
					</div>
					{totalPages > 1 && (
						<div className="flex items-center justify-between mt-3 text-[11px] text-muted-foreground">
							<span>{total} {t("activity.log.entries")}</span>
							<div className="flex gap-2">
								<button
									type="button"
									disabled={page === 0}
									onClick={() => onPageChange(page - 1)}
									className="px-2 py-0.5 rounded border border-border/40 disabled:opacity-30"
								>
									{t("activity.log.prev")}
								</button>
								<span>{page + 1} / {totalPages}</span>
								<button
									type="button"
									disabled={page >= totalPages - 1}
									onClick={() => onPageChange(page + 1)}
									className="px-2 py-0.5 rounded border border-border/40 disabled:opacity-30"
								>
									{t("activity.log.next")}
								</button>
							</div>
						</div>
					)}
				</>
			)}
		</div>
	);
}
