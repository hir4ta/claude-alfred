import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
	knowledgeQueryOptions,
	knowledgeSearchQueryOptions,
	knowledgeStatsQueryOptions,
	useToggleEnabledMutation,
} from "@/lib/api";
import { contentPreview, formatDate, formatLabel } from "@/lib/format";
import type { KnowledgeEntry, KnowledgeStats } from "@/lib/types";
import { SUB_TYPE_COLORS } from "@/lib/types";
import { cn } from "@/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { Eye, EyeOff, Search } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";

export const Route = createFileRoute("/knowledge")({
	component: KnowledgePage,
});

function KnowledgePage() {
	const [searchInput, setSearchInput] = useState("");
	const [debouncedSearch, setDebouncedSearch] = useState("");
	const [localFilter, setLocalFilter] = useState("");
	const [selected, setSelected] = useState<number | null>(null);
	const timerRef = useRef<ReturnType<typeof setTimeout>>(null);
	const isSearching = debouncedSearch.length > 0;

	const handleSearchChange = useCallback((value: string) => {
		setSearchInput(value);
		if (timerRef.current) clearTimeout(timerRef.current);
		timerRef.current = setTimeout(() => setDebouncedSearch(value), 300);
	}, []);

	useEffect(() => {
		return () => {
			if (timerRef.current) clearTimeout(timerRef.current);
		};
	}, []);

	const { data: browseData, isLoading: browseLoading } = useQuery(knowledgeQueryOptions());
	const { data: searchData, isLoading: searchLoading } = useQuery(
		knowledgeSearchQueryOptions(debouncedSearch),
	);
	const { data: statsData } = useQuery(knowledgeStatsQueryOptions());

	const entries = isSearching ? (searchData?.entries ?? []) : (browseData?.entries ?? []);
	const filtered = localFilter
		? entries.filter(
				(e) =>
					e.label.toLowerCase().includes(localFilter.toLowerCase()) ||
					e.content.toLowerCase().includes(localFilter.toLowerCase()),
			)
		: entries;
	const isLoading = isSearching ? searchLoading : browseLoading;
	const selectedEntry = selected ? filtered.find((e) => e.id === selected) : null;

	return (
		<div className="flex gap-6 h-[calc(100vh-8rem)]">
			{/* Left: list */}
			<div className="flex w-full flex-col gap-4 lg:w-[420px] lg:shrink-0">
				{/* Search bar */}
				<div className="flex gap-2">
					<div className="relative flex-1">
						<Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
						<Input
							placeholder="Semantic search..."
							value={searchInput}
							onChange={(e) => handleSearchChange(e.target.value)}
							className="pl-9"
						/>
					</div>
					<Input
						placeholder="Filter..."
						value={localFilter}
						onChange={(e) => setLocalFilter(e.target.value)}
						className="w-32"
					/>
				</div>

				{/* Stats */}
				{statsData && (
					<StatsBar stats={statsData} isSearching={isSearching} searchData={searchData} />
				)}

				{/* List */}
				<ScrollArea className="flex-1">
					{isLoading ? (
						<div className="space-y-2 pr-3">
							{Array.from({ length: 8 }).map((_, i) => (
								<Skeleton key={i} className="h-16 w-full rounded-lg" />
							))}
						</div>
					) : (
						<div className="space-y-1.5 pr-3">
							{filtered.map((entry) => (
								<KnowledgeCard
									key={entry.id}
									entry={entry}
									isSelected={selected === entry.id}
									onSelect={() => setSelected(selected === entry.id ? null : entry.id)}
								/>
							))}
							{filtered.length === 0 && (
								<p className="py-8 text-center text-sm text-muted-foreground">
									{isSearching ? "No results found." : "No memories yet."}
								</p>
							)}
						</div>
					)}
				</ScrollArea>
			</div>

			{/* Right: detail */}
			<div className="hidden flex-1 lg:block">
				{selectedEntry ? (
					<KnowledgeDetail entry={selectedEntry} />
				) : (
					<div className="flex h-full items-center justify-center">
						<p className="text-sm text-muted-foreground">Select a memory to view details.</p>
					</div>
				)}
			</div>
		</div>
	);
}

function StatsBar({
	stats,
	isSearching,
	searchData,
}: {
	stats: KnowledgeStats;
	isSearching: boolean;
	searchData?: { entries: KnowledgeEntry[]; method: string; partial: boolean };
}) {
	return (
		<div className="flex items-center gap-3 text-xs text-muted-foreground">
			{isSearching && searchData ? (
				<>
					<span>
						{searchData.entries.length} results via {searchData.method}
					</span>
					{searchData.partial && <span style={{ color: "#e67e22" }}>(partial — timeout)</span>}
				</>
			) : (
				<>
					<span>{stats.total} memories</span>
					<Separator orientation="vertical" className="h-3" />
					<StatBadge label="decision" count={stats.decision} color={SUB_TYPE_COLORS.decision!} />
					<StatBadge label="pattern" count={stats.pattern} color={SUB_TYPE_COLORS.pattern!} />
					<StatBadge label="rule" count={stats.rule} color={SUB_TYPE_COLORS.rule!} />
					<StatBadge label="general" count={stats.general} color={SUB_TYPE_COLORS.general!} />
				</>
			)}
		</div>
	);
}

function StatBadge({ label, count, color }: { label: string; count: number; color: string }) {
	return (
		<span className="flex items-center gap-1">
			<span className="size-1.5 rounded-full" style={{ backgroundColor: color }} />
			{count}
		</span>
	);
}

function KnowledgeCard({
	entry,
	isSelected,
	onSelect,
}: {
	entry: KnowledgeEntry;
	isSelected: boolean;
	onSelect: () => void;
}) {
	const { title, source } = formatLabel(entry.label);
	const color = SUB_TYPE_COLORS[entry.sub_type] ?? SUB_TYPE_COLORS.general!;
	const toggleMutation = useToggleEnabledMutation();

	return (
		<Card
			className={cn(
				"cursor-pointer border-stone-200 transition-all hover:border-stone-300 hover:shadow-sm dark:border-stone-700",
				isSelected && "ring-1 ring-brand-pattern/30 border-brand-pattern/20",
				!entry.enabled && "opacity-40",
			)}
			onClick={onSelect}
		>
			<CardContent className="p-3 space-y-1.5">
				<div className="flex items-start justify-between gap-2">
					<p className="text-sm font-medium leading-snug line-clamp-2">{title}</p>
					<div className="flex shrink-0 items-center gap-1.5">
						{entry.score ? (
							<span className="text-[10px] tabular-nums text-muted-foreground">
								{entry.score.toFixed(2)}
							</span>
						) : null}
						<Badge
							variant="outline"
							className="rounded-full text-[10px] px-1.5 py-0"
							style={{ borderColor: `${color}50`, color }}
						>
							{entry.sub_type}
						</Badge>
					</div>
				</div>
				<p className="text-xs text-muted-foreground line-clamp-1">
					{contentPreview(entry.content, 80)}
				</p>
				<div className="flex items-center justify-between">
					<div className="flex items-center gap-2 text-[10px] text-muted-foreground">
						{source && <span>{source}</span>}
						<span>{formatDate(entry.saved_at ?? "")}</span>
						{entry.hit_count > 0 && <span>{entry.hit_count} hits</span>}
					</div>
					<Tooltip>
						<TooltipTrigger asChild>
							<button
								type="button"
								onClick={(e) => {
									e.stopPropagation();
									toggleMutation.mutate({ id: entry.id, enabled: !entry.enabled });
								}}
								className="text-muted-foreground hover:text-foreground transition-colors"
							>
								{entry.enabled ? <Eye className="size-3.5" /> : <EyeOff className="size-3.5" />}
							</button>
						</TooltipTrigger>
						<TooltipContent>{entry.enabled ? "Disable" : "Enable"}</TooltipContent>
					</Tooltip>
				</div>
			</CardContent>
		</Card>
	);
}

function KnowledgeDetail({ entry }: { entry: KnowledgeEntry }) {
	const { title, source } = formatLabel(entry.label);
	const color = SUB_TYPE_COLORS[entry.sub_type] ?? SUB_TYPE_COLORS.general!;

	// Parse structured decision content
	const fields = parseDecisionFields(entry.content);

	return (
		<Card className="h-full border-stone-200 dark:border-stone-700">
			<div className="p-6 space-y-4">
				{/* Header */}
				<div className="space-y-2">
					<div className="flex items-center gap-2">
						<Badge
							variant="outline"
							className="rounded-full text-xs"
							style={{ borderColor: `${color}50`, color }}
						>
							{entry.sub_type}
						</Badge>
						{source && <span className="text-xs text-muted-foreground">{source}</span>}
					</div>
					<h2 className="text-lg font-semibold" style={{ fontFamily: "var(--font-display)" }}>
						{title}
					</h2>
					<div className="flex gap-4 text-xs text-muted-foreground">
						<span>Saved {formatDate(entry.saved_at ?? "")}</span>
						<span>{entry.hit_count} hits</span>
						<span>{entry.enabled ? "Active" : "Disabled"}</span>
					</div>
				</div>

				<Separator />

				{/* Structured fields for decisions */}
				{fields.length > 0 ? (
					<div className="space-y-3">
						{fields.map((f) => (
							<div key={f.key}>
								<p className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-1">
									{f.key}
								</p>
								<p className="text-sm leading-relaxed">{f.value}</p>
							</div>
						))}
					</div>
				) : (
					<ScrollArea className="h-[calc(100vh-20rem)]">
						<div className="prose prose-sm prose-stone max-w-none">
							<pre className="whitespace-pre-wrap break-words text-sm leading-relaxed font-sans">
								{cleanContent(entry.content)}
							</pre>
						</div>
					</ScrollArea>
				)}
			</div>
		</Card>
	);
}

// Parse "- **Key:** value" patterns from decision content.
function parseDecisionFields(content: string): { key: string; value: string }[] {
	const fields: { key: string; value: string }[] = [];
	for (const line of content.split("\n")) {
		const match = line.match(/^-\s*\*\*([^*]+)\*\*:?\s*(.+)/);
		if (match?.[1] && match[2]) {
			fields.push({ key: match[1], value: match[2] });
		}
	}
	return fields;
}

// Strip markdown headers, confidence annotations, and status lines.
function cleanContent(content: string): string {
	return content
		.split("\n")
		.filter(
			(l) =>
				!l.startsWith("# ") &&
				!l.startsWith("## ") &&
				!l.startsWith("<!-- confidence") &&
				!l.match(/^-\s*\*\*Status\*\*/),
		)
		.join("\n")
		.trim();
}
