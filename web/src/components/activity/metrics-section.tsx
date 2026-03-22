import type { AnalyticsResponse } from "@/lib/api";
import { useI18n } from "@/lib/i18n";

export function SummaryCards({ analytics }: { analytics: AnalyticsResponse }) {
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
					className="rounded-organic border border-border/60 bg-card py-4 px-4"
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

export function ReworkChart({ analytics }: { analytics: AnalyticsResponse }) {
	const { t } = useI18n();
	if (analytics.reworkRates.length === 0) return null;

	const maxRate = Math.max(...analytics.reworkRates.map((r) => r.reworkRate), 0.01);

	return (
		<div className="rounded-organic border border-border/60 bg-card py-4 px-4">
			<h3 className="text-sm font-semibold mb-3">{t("activity.rework.title")}</h3>
			<div className="space-y-3">
				{analytics.reworkRates.map((r) => {
					const pct = Math.round(r.reworkRate * 100);
					const width = Math.max((r.reworkRate / maxRate) * 100, 2);
					return (
						<div key={r.slug}>
							<div className="flex items-baseline justify-between mb-1">
								<span className="text-[11px] font-mono text-foreground/80">{r.slug}</span>
								<span className="text-[11px] font-mono text-muted-foreground">{pct}%</span>
							</div>
							<div className="h-4 bg-muted/30 rounded overflow-hidden">
								<div
									className="h-full rounded"
									style={{
										width: `${width}%`,
										backgroundColor: r.pending ? "#e67e22" : "#2d8b7a",
										opacity: r.pending ? 0.5 : 1,
									}}
								/>
							</div>
						</div>
					);
				})}
			</div>
			{analytics.reworkRates.some((r) => r.pending) && (
				<p className="text-[10px] text-muted-foreground mt-2">{t("activity.rework.pending")}</p>
			)}
		</div>
	);
}

const PHASE_COLORS = {
	planning: "#628141",
	approval: "#e67e22",
	implementation: "#2d8b7a",
};

export function CycleTimeChart({ analytics }: { analytics: AnalyticsResponse }) {
	const { t } = useI18n();
	if (analytics.cycleTimeBreakdown.length === 0) return null;

	const maxTotal = Math.max(...analytics.cycleTimeBreakdown.map((r) => r.phases.total), 0.1);

	return (
		<div className="rounded-organic border border-border/60 bg-card py-4 px-4">
			<h3 className="text-sm font-semibold mb-3">{t("activity.cycleTime.title")}</h3>
			<div className="space-y-3">
				{analytics.cycleTimeBreakdown.map((r) => {
					const p = r.phases;
					const planW = ((p.planning ?? 0) / maxTotal) * 100;
					const apprW = ((p.approvalWait ?? 0) / maxTotal) * 100;
					const implW = ((p.implementation ?? 0) / maxTotal) * 100;
					return (
						<div key={r.slug}>
							<div className="flex items-baseline justify-between mb-1">
								<span className="text-[11px] font-mono text-foreground/80">{r.slug}</span>
								<span className="text-[11px] font-mono text-muted-foreground">{p.total.toFixed(1)}d</span>
							</div>
							<div className="h-4 bg-muted/30 rounded overflow-hidden flex">
								{planW > 0 && (
									<div className="h-full" style={{ width: `${planW}%`, backgroundColor: PHASE_COLORS.planning }} title={`${t("activity.cycleTime.planning")}: ${p.planning?.toFixed(1)}d`} />
								)}
								{apprW > 0 && (
									<div className="h-full" style={{ width: `${apprW}%`, backgroundColor: PHASE_COLORS.approval }} title={`${t("activity.cycleTime.approval")}: ${p.approvalWait?.toFixed(1)}d`} />
								)}
								{implW > 0 && (
									<div className="h-full" style={{ width: `${implW}%`, backgroundColor: PHASE_COLORS.implementation }} title={`${t("activity.cycleTime.implementation")}: ${p.implementation?.toFixed(1)}d`} />
								)}
							</div>
						</div>
					);
				})}
			</div>
			<div className="flex gap-4 mt-3">
				{(["planning", "approval", "implementation"] as const).map((key) => (
					<span key={key} className="flex items-center gap-1 text-[10px] text-muted-foreground">
						<span className="size-2 rounded-sm" style={{ backgroundColor: PHASE_COLORS[key] }} />
						{t(`activity.cycleTime.${key}` as "activity.cycleTime.planning")}
					</span>
				))}
			</div>
		</div>
	);
}
