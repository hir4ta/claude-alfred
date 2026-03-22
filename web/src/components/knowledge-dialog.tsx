import {
	Dialog,
	DialogContent,
	DialogHeader,
	DialogTitle,
	DialogDescription,
} from "@/components/ui/dialog";
import { KnowledgeDrawerContent } from "@/components/knowledge-detail";
import { formatLabel } from "@/lib/format";
import { SUB_TYPE_COLORS } from "@/lib/types";
import type { KnowledgeEntry } from "@/lib/types";

export function KnowledgeDialog({
	entry,
	onClose,
}: {
	entry: KnowledgeEntry | null;
	onClose: () => void;
}) {
	if (!entry) return null;

	const { title, source } = formatLabel(entry.label);
	const color = SUB_TYPE_COLORS[entry.sub_type] ?? "#44403c";

	return (
		<Dialog open={!!entry} onOpenChange={(open) => { if (!open) onClose(); }}>
			<DialogContent className="sm:max-w-2xl max-h-[85vh] overflow-y-auto rounded-organic p-0">
				{/* Book cover header — colored bar */}
				<div
					className="h-1.5 rounded-t-organic shrink-0"
					style={{ backgroundColor: color }}
				/>

				<div className="px-6 pb-6 pt-3 space-y-4">
					<DialogHeader>
						<DialogTitle
							className="text-xl font-bold tracking-tight"
							style={{ fontFamily: "var(--font-display)" }}
						>
							{title}
						</DialogTitle>
						{source && (
							<DialogDescription className="text-xs">
								{source}
							</DialogDescription>
						)}
					</DialogHeader>

					<KnowledgeDrawerContent entry={entry} onClose={onClose} />
				</div>
			</DialogContent>
		</Dialog>
	);
}
