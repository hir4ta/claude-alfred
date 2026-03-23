import { BookOpen, Gavel, Lightbulb, Shield } from "@animated-color-icons/lucide-react";
import { motion } from "motion/react";
import { VerificationBadge } from "@/components/verification-badge";
import { useI18n } from "@/lib/i18n";
import { formatDate, formatLabel } from "@/lib/format";
import { butlerSpring } from "@/lib/motion";
import type { KnowledgeEntry } from "@/lib/types";
import { SUB_TYPE_COLORS } from "@/lib/types";
import { cn } from "@/lib/utils";

const SUB_TYPE_ICONS: Record<string, React.ReactNode> = {
	rule: <Shield className="size-3.5" />,
	decision: <Gavel className="size-3.5" />,
	pattern: <Lightbulb className="size-3.5" />,
	snapshot: <BookOpen className="size-3.5" />,
};

function contentPreview(content: string): string {
	const line = content.replace(/^#+\s.*/m, "").replace(/\n/g, " ").trim();
	return line.length > 120 ? `${line.slice(0, 120)}…` : line;
}

export function KnowledgeListView({
	entries,
	onSelect,
}: {
	entries: KnowledgeEntry[];
	onSelect: (entry: KnowledgeEntry) => void;
}) {
	const { locale } = useI18n();

	return (
		<div className="space-y-1">
			{entries.map((entry) => {
				const { title } = formatLabel(entry.label);
				const color = SUB_TYPE_COLORS[entry.sub_type] ?? SUB_TYPE_COLORS.snapshot!;
				const icon = SUB_TYPE_ICONS[entry.sub_type] ?? SUB_TYPE_ICONS.snapshot;
				const preview = contentPreview(entry.content);

				return (
					<motion.button
						key={entry.id}
						type="button"
						onClick={() => onSelect(entry)}
						className={cn(
							"al-icon-wrapper flex w-full gap-2 rounded-organic border border-transparent text-left cursor-pointer py-2 px-3",
							!entry.enabled && "opacity-60",
						)}
						whileHover={{
							x: 4,
							backgroundColor: `${color}0a`,
							transition: { type: "spring", ...butlerSpring },
						}}
					>
						<motion.div
							className="flex size-5 items-center justify-center rounded shrink-0 mt-0.5"
							style={{ backgroundColor: `${color}18`, color }}
							whileHover={{ scale: 1.2, transition: { type: "spring", ...butlerSpring } }}
						>
							{icon}
						</motion.div>
						<div className="flex-1 min-w-0">
							<div className="flex items-center gap-2">
								<span className="text-sm font-medium truncate min-w-0 flex-1">{title}</span>
								{entry.verification_due && (
									<VerificationBadge entry={entry} />
								)}
								<span className="text-[10px] tabular-nums text-muted-foreground shrink-0">
									{(entry.hit_count ?? 0) > 0 && `${entry.hit_count} hits`}
								</span>
								<span className="text-[10px] text-muted-foreground shrink-0">
									{formatDate(entry.saved_at ?? "", locale)}
								</span>
							</div>
							{preview && (
								<p className="text-xs text-muted-foreground/70 truncate mt-0.5">
									{preview}
								</p>
							)}
						</div>
					</motion.button>
				);
			})}
		</div>
	);
}
