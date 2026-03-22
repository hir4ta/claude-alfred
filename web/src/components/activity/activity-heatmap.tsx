import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useI18n } from "@/lib/i18n";

interface HeatmapData {
	date: string;
	count: number;
}

const CELL_SIZE = 12;
const CELL_GAP = 2;
const CELL_STEP = CELL_SIZE + CELL_GAP;
const LABEL_WIDTH = 28;
const HEADER_HEIGHT = 16;
const ROWS = 7;

const COLOR_LEVELS = [
	"var(--color-card)",
	"#c8c4b8",
	"#a8c4a0",
	"#628141",
	"#40513b",
];

function getColorLevel(count: number): string {
	if (count === 0) return COLOR_LEVELS[0]!;
	if (count <= 2) return COLOR_LEVELS[1]!;
	if (count <= 5) return COLOR_LEVELS[2]!;
	if (count <= 10) return COLOR_LEVELS[3]!;
	return COLOR_LEVELS[4]!;
}

function getMonday(d: Date): Date {
	const day = d.getDay();
	const diff = d.getDate() - day + (day === 0 ? -6 : 1);
	return new Date(d.getFullYear(), d.getMonth(), diff);
}

export function ActivityHeatmap({
	data,
	weeks = 16,
}: {
	data: HeatmapData[];
	weeks?: number;
}) {
	const { t } = useI18n();

	const countMap = new Map<string, number>();
	for (const d of data) {
		countMap.set(d.date, d.count);
	}

	// Build grid: weeks columns × 7 rows (Mon=0 to Sun=6)
	const today = new Date();
	const startMonday = getMonday(new Date(today.getFullYear(), today.getMonth(), today.getDate() - (weeks - 1) * 7));

	const cells: Array<{ date: string; count: number; col: number; row: number }> = [];
	const monthLabels: Array<{ label: string; col: number }> = [];
	let lastMonth = -1;

	for (let week = 0; week < weeks; week++) {
		for (let day = 0; day < ROWS; day++) {
			const d = new Date(startMonday);
			d.setDate(d.getDate() + week * 7 + day);
			if (d > today) continue;

			const dateStr = d.toISOString().slice(0, 10);
			const count = countMap.get(dateStr) ?? 0;
			cells.push({ date: dateStr, count, col: week, row: day });

			// Month labels (first occurrence of each month)
			if (day === 0 && d.getMonth() !== lastMonth) {
				lastMonth = d.getMonth();
				monthLabels.push({
					label: d.toLocaleDateString("en", { month: "short" }),
					col: week,
				});
			}
		}
	}

	const svgWidth = LABEL_WIDTH + weeks * CELL_STEP;
	const svgHeight = HEADER_HEIGHT + ROWS * CELL_STEP;

	const dayLabels = ["Mon", "", "Wed", "", "Fri", "", ""];

	return (
		<div className="rounded-organic border border-border/60 bg-card py-4 px-4">
			<h3 className="text-sm font-semibold mb-3">{t("heatmap.title")}</h3>
			<div className="overflow-x-auto">
				<svg width={svgWidth} height={svgHeight} className="block">
					{/* Month labels */}
					{monthLabels.map((m) => (
						<text
							key={`month-${m.col}`}
							x={LABEL_WIDTH + m.col * CELL_STEP}
							y={HEADER_HEIGHT - 4}
							className="fill-muted-foreground"
							fontSize={10}
						>
							{m.label}
						</text>
					))}

					{/* Day labels */}
					{dayLabels.map((label, i) =>
						label ? (
							<text
								key={`day-${i}`}
								x={0}
								y={HEADER_HEIGHT + i * CELL_STEP + CELL_SIZE - 2}
								className="fill-muted-foreground"
								fontSize={10}
							>
								{label}
							</text>
						) : null,
					)}

					{/* Cells */}
					{cells.map((cell) => (
						<Tooltip key={cell.date}>
							<TooltipTrigger asChild>
								<rect
									x={LABEL_WIDTH + cell.col * CELL_STEP}
									y={HEADER_HEIGHT + cell.row * CELL_STEP}
									width={CELL_SIZE}
									height={CELL_SIZE}
									rx={2}
									fill={getColorLevel(cell.count)}
									className="outline-1 outline-border/20"
									aria-label={t("heatmap.tooltip", { date: cell.date, count: cell.count })}
								/>
							</TooltipTrigger>
							<TooltipContent>
								{t("heatmap.tooltip", { date: cell.date, count: cell.count })}
							</TooltipContent>
						</Tooltip>
					))}
				</svg>
			</div>

			{/* Legend */}
			<div className="flex items-center gap-1.5 mt-2 justify-end">
				<span className="text-[10px] text-muted-foreground">{t("heatmap.less")}</span>
				{COLOR_LEVELS.map((color, i) => (
					<span
						key={i}
						className="inline-block size-3 rounded-sm border border-border/20"
						style={{ backgroundColor: color }}
					/>
				))}
				<span className="text-[10px] text-muted-foreground">{t("heatmap.more")}</span>
			</div>
		</div>
	);
}
