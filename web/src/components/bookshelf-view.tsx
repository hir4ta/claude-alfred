import { useCallback, useEffect, useRef, useState } from "react";
import { ButlerEmpty } from "@/components/butler-empty";
import { VerificationBadge } from "@/components/verification-badge";
import type { KnowledgeEntry } from "@/lib/types";
import { SUB_TYPE_COLORS } from "@/lib/types";
import { formatLabel } from "@/lib/format";

const SPINE_W = 56;
const GAP = 4;
const PAD = 16;

// Height variation by type — creates visual rhythm on the shelf
const HEIGHTS: Record<string, number> = { rule: 240, decision: 270, pattern: 220, snapshot: 200 };

// Slightly darker shades for the spine edge
const EDGE_COLORS: Record<string, string> = {
	rule: "#c06a18",
	decision: "#4d6b33",
	pattern: "#227060",
	snapshot: "#5d5244",
};

function splitShelves(entries: KnowledgeEntry[], perShelf: number): KnowledgeEntry[][] {
	const out: KnowledgeEntry[][] = [];
	for (let i = 0; i < entries.length; i += perShelf) out.push(entries.slice(i, i + perShelf));
	return out;
}

function BookSpine({ entry, onClick }: { entry: KnowledgeEntry; onClick: () => void }) {
	const color = SUB_TYPE_COLORS[entry.sub_type as keyof typeof SUB_TYPE_COLORS] ?? "#44403c";
	const edge = EDGE_COLORS[entry.sub_type] ?? "#333";
	const h = HEIGHTS[entry.sub_type] ?? 210;
	const { title } = formatLabel(entry.label);
	// Width varies slightly with content length for organic feel
	const w = entry.content.length > 300 ? 64 : entry.content.length > 100 ? 56 : 48;
	const label = title.length > 16 ? `${title.slice(0, 16)}…` : title;

	return (
		<button
			type="button"
			onClick={onClick}
			className="relative flex flex-col items-center justify-between rounded-[2px] cursor-pointer shrink-0 group focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-current"
			style={{
				width: w,
				height: h,
				backgroundColor: color,
				boxShadow: "3px 1px 8px rgba(0,0,0,0.12), inset 1px 0 0 rgba(255,255,255,0.08)",
				transition: "transform 0.25s cubic-bezier(0.22,1,0.36,1), box-shadow 0.25s ease",
			}}
			title={title}
			aria-label={title}
			onMouseEnter={(e) => {
				e.currentTarget.style.transform = "translateY(-6px)";
				e.currentTarget.style.boxShadow = "6px 8px 20px rgba(0,0,0,0.18), inset 1px 0 0 rgba(255,255,255,0.08)";
			}}
			onMouseLeave={(e) => {
				e.currentTarget.style.transform = "translateY(0)";
				e.currentTarget.style.boxShadow = "3px 1px 8px rgba(0,0,0,0.12), inset 1px 0 0 rgba(255,255,255,0.08)";
			}}
		>
			{/* Spine edge — depth illusion */}
			<div className="absolute inset-y-0 left-0 w-[3px] rounded-l-[2px]" style={{ backgroundColor: edge }} />

			{/* Top ornament */}
			<div className="pt-3 px-1.5 shrink-0">
				{entry.verification_due ? (
					<VerificationBadge entry={entry} />
				) : (
					<div className="w-5 h-[2px] bg-white/15 rounded-full mx-auto" />
				)}
			</div>

			{/* Title — vertical */}
			<div
				className="flex-1 flex items-center justify-center px-0.5 overflow-hidden min-h-0"
				style={{ writingMode: "vertical-rl" }}
			>
				<span
					className="text-[11px] font-semibold text-white/85 leading-tight tracking-wide"
					style={{ fontFamily: "var(--font-display)" }}
				>
					{label}
				</span>
			</div>

			{/* Bottom — type initial or hit count */}
			<div className="pb-2.5 px-1.5 shrink-0">
				{(entry.hit_count ?? 0) > 0 ? (
					<span className="text-[9px] text-white/40 font-mono tabular-nums">{entry.hit_count}</span>
				) : (
					<div className="w-4 h-4 rounded-full border border-white/15 flex items-center justify-center">
						<span className="text-[7px] text-white/50 font-bold uppercase">{entry.sub_type.charAt(0)}</span>
					</div>
				)}
			</div>
		</button>
	);
}

function Shelf({ entries, onSelect }: { entries: KnowledgeEntry[]; onSelect: (e: KnowledgeEntry) => void }) {
	return (
		<div className="relative">
			{/* Books — bottom-aligned */}
			<div
				className="flex items-end gap-[3px] overflow-x-auto px-3 pb-0"
				style={{ scrollbarWidth: "none" }}
			>
				{entries.map((e) => (
					<BookSpine key={e.id} entry={e} onClick={() => onSelect(e)} />
				))}
			</div>

			{/* Shelf plank — wood-like gradient */}
			<div
				className="h-3 relative z-10"
				style={{
					background: "linear-gradient(to bottom, #3d4a37 0%, #2a3425 60%, #232d1f 100%)",
					boxShadow: "0 3px 10px rgba(0,0,0,0.12), inset 0 1px 0 rgba(255,255,255,0.05)",
				}}
			/>
			{/* Shelf underside shadow */}
			<div className="h-2 bg-gradient-to-b from-black/[0.04] to-transparent" />

			{/* Mobile scroll fade */}
			<div className="absolute top-0 right-0 bottom-5 w-6 bg-gradient-to-l from-background to-transparent pointer-events-none sm:hidden" />
		</div>
	);
}

export function BookshelfView({ entries, onSelect }: { entries: KnowledgeEntry[]; onSelect: (e: KnowledgeEntry) => void }) {
	const ref = useRef<HTMLDivElement>(null);
	const [perShelf, setPerShelf] = useState(8);

	const calc = useCallback(() => {
		if (!ref.current) return;
		setPerShelf(Math.max(3, Math.floor((ref.current.clientWidth - PAD * 2) / (SPINE_W + GAP))));
	}, []);

	useEffect(() => {
		calc();
		const obs = new ResizeObserver(calc);
		if (ref.current) obs.observe(ref.current);
		return () => obs.disconnect();
	}, [calc]);

	if (entries.length === 0) return <ButlerEmpty scene="bookshelf" messageKey="empty.noMemories" />;

	return (
		<div ref={ref} className="space-y-4">
			{splitShelves(entries, perShelf).map((shelf, i) => (
				<Shelf key={i} entries={shelf} onSelect={onSelect} />
			))}
		</div>
	);
}
