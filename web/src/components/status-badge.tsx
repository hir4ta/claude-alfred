import { TASK_STATUS_COLORS } from "../lib/types";

export function StatusBadge({ status }: { status: string }) {
	const config = TASK_STATUS_COLORS[status] ?? TASK_STATUS_COLORS["pending"]!;
	return (
		<span
			style={{ backgroundColor: config.bg, color: config.text }}
			className="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium"
		>
			{config.label}
		</span>
	);
}
