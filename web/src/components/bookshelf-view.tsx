import { useCallback, useEffect, useRef, useState } from "react";
import { ButlerEmpty } from "@/components/butler-empty";
import { VerificationBadge } from "@/components/verification-badge";
import type { KnowledgeEntry } from "@/lib/types";
import { SUB_TYPE_COLORS } from "@/lib/types";
import { formatLabel } from "@/lib/format";

const SPINE_WIDTH_BASE = 56; // w-14
const SPINE_GAP = 4;
const SHELF_PADDING = 16;

// Vary height by sub_type to create visual rhythm
const SPINE_HEIGHTS: Record<string, number> = {
	rule: 280,
	decision: 320,
	pattern: 260,
	snapshot: 240,
};

// Darker shade for spine left-edge shadow
const SPINE_DARK: Record<string, string> = {
	rule: "#b8620f",
	decision: "#4a6530",
	pattern: "#1f6b5e",
	snapshot: "#6b5f4f",
};

function splitIntoShelves(entries: KnowledgeEntry[], perShelf: number): KnowledgeEntry[][] {
	if (entries.length === 0) return [];
	const shelves: KnowledgeEntry[][] = [];
	for (let i = 0; i < entries.length; i += perShelf) {
		shelves.push(entries.slice(i, i + perShelf));
	}
	return shelves;
}

function BookSpine({
	entry,
	onClick,
}: {
	entry: KnowledgeEntry;
	onClick: () => void;
}) {
	const color = SUB_TYPE_COLORS[entry.sub_type as keyof typeof SUB_TYPE_COLORS] ?? "#44403c";
	const darkColor = SPINE_DARK[entry.sub_type] ?? "#333";
	const height = SPINE_HEIGHTS[entry.sub_type] ?? 260;
	const { title } = formatLabel(entry.label);
	// Vary width slightly based on content length
	const width = entry.content.length > 200 ? 64 : entry.content.length > 80 ? 56 : 48;
	const truncTitle = title.length > 18 ? `${title.slice(0, 18)}…` : title;

	return (
		<button
			type="button"
			onClick={onClick}
			className="relative flex flex-col items-center justify-between rounded-sm cursor-pointer shrink-0 transition-[transform,box-shadow] duration-300 ease-out hover:-translate-y-2 hover:scale-[1.02] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-current"
			style={{
				width: `${width}px`,
				height: `${height}px`,
				backgroundColor: color,
				boxShadow: `4px 0 10px rgba(0,0,0,0.1), inset 1px 0 0 rgba(255,255,255,0.1)`,
			}}
			title={title}
			aria-label={title}
		>
			{/* Left edge shadow (spine depth illusion) */}
			<div
				className="absolute inset-y-0 left-0 w-1"
				style={{ backgroundColor: `${darkColor}80` }}
			/>

			{/* Top decoration */}
			<div className="pt-4 px-1.5 shrink-0">
				{entry.verification_due ? (
					<VerificationBadge entry={entry} />
				) : (
					<div className="w-6 h-[2px] bg-white/20 rounded-full" />
				)}
			</div>

			{/* Title — vertical text */}
			<div
				className="flex-1 flex items-center justify-center px-1 overflow-hidden min-h-0"
				style={{ writingMode: "vertical-rl" }}
			>
				<span className="text-[11px] font-semibold text-white/90 leading-tight"
					style={{ fontFamily: "var(--font-display)" }}
				>
					{truncTitle}
				</span>
			</div>

			{/* Bottom: hit count or sub_type initial */}
			<div className="pb-3 px-1.5 shrink-0">
				{(entry.hit_count ?? 0) > 0 ? (
					<span className="text-[9px] text-white/50 font-mono tabular-nums">
						{entry.hit_count}
					</span>
				) : (
					<div className="w-5 h-5 rounded-full border border-white/20 flex items-center justify-center">
						<span className="text-[8px] text-white/60 font-bold uppercase">
							{entry.sub_type.charAt(0)}
						</span>
					</div>
				)}
			</div>
		</button>
	);
}

function Shelf({
	entries,
	onSelect,
}: {
	entries: KnowledgeEntry[];
	onSelect: (entry: KnowledgeEntry) => void;
}) {
	return (
		<div className="relative">
			{/* Books on shelf — aligned to bottom */}
			<div className="flex items-end gap-1 overflow-x-auto pb-0 px-4 scroll-smooth"
				style={{ scrollbarWidth: "none" }}
			>
				{entries.map((entry) => (
					<BookSpine
						key={entry.id}
						entry={entry}
						onClick={() => onSelect(entry)}
					/>
				))}
			</div>

			{/* Shelf plank */}
			<div
				className="h-4 rounded-sm relative z-10"
				style={{
					background: "linear-gradient(to bottom, #40513b 0%, #2a3627 100%)",
					boxShadow: "0 4px 12px rgba(0,0,0,0.15)",
				}}
			/>
			{/* Shelf shadow on wall */}
			<div className="h-3 bg-gradient-to-b from-black/5 to-transparent" />

			{/* Mobile scroll affordance */}
			<div className="absolute top-0 right-0 bottom-7 w-8 bg-gradient-to-l from-background to-transparent pointer-events-none sm:hidden" />
		</div>
	);
}

export function BookshelfView({
	entries,
	onSelect,
}: {
	entries: KnowledgeEntry[];
	onSelect: (entry: KnowledgeEntry) => void;
}) {
	const containerRef = useRef<HTMLDivElement>(null);
	const [perShelf, setPerShelf] = useState(8);

	const updatePerShelf = useCallback(() => {
		if (!containerRef.current) return;
		const width = containerRef.current.clientWidth;
		const count = Math.max(3, Math.floor((width - SHELF_PADDING * 2) / (SPINE_WIDTH_BASE + SPINE_GAP)));
		setPerShelf(count);
	}, []);

	useEffect(() => {
		updatePerShelf();
		const observer = new ResizeObserver(updatePerShelf);
		if (containerRef.current) observer.observe(containerRef.current);
		return () => observer.disconnect();
	}, [updatePerShelf]);

	if (entries.length === 0) {
		return <ButlerEmpty scene="bookshelf" messageKey="empty.noMemories" />;
	}

	const shelves = splitIntoShelves(entries, perShelf);

	return (
		<div ref={containerRef} className="space-y-6">
			{shelves.map((shelf, i) => (
				<Shelf key={i} entries={shelf} onSelect={onSelect} />
			))}
		</div>
	);
}
