import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
	knowledgeQueryOptions,
	knowledgeSearchQueryOptions,
	knowledgeStatsQueryOptions,
	useToggleEnabledMutation,
} from "@/lib/api";
import type { KnowledgeEntry, KnowledgeStats } from "@/lib/types";
import { SUB_TYPE_COLORS } from "@/lib/types";
import { cn } from "@/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { Search } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";

export const Route = createFileRoute("/knowledge")({
	component: KnowledgePage,
});

function KnowledgePage() {
	const [searchInput, setSearchInput] = useState("");
	const [debouncedSearch, setDebouncedSearch] = useState("");
	const [localFilter, setLocalFilter] = useState("");
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

	return (
		<div className="space-y-4">
			<div className="flex items-center gap-4">
				<div className="relative flex-1 max-w-md">
					<Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
					<Input
						placeholder="Semantic search (Voyage AI)..."
						value={searchInput}
						onChange={(e) => handleSearchChange(e.target.value)}
						className="pl-9"
					/>
				</div>
				<div className="relative max-w-xs">
					<Input
						placeholder="Local filter..."
						value={localFilter}
						onChange={(e) => setLocalFilter(e.target.value)}
					/>
				</div>
				{statsData && <StatsDisplay stats={statsData} />}
			</div>

			{isSearching && searchData && (
				<p className="text-xs text-muted-foreground">
					{searchData.entries.length} results via {searchData.method}
					{searchData.partial && " (partial — timeout)"}
				</p>
			)}

			{isLoading ? (
				<div className="space-y-2">
					{Array.from({ length: 5 }).map((_, i) => (
						<Skeleton key={`skel-${i}`} className="h-12 w-full" />
					))}
				</div>
			) : (
				<KnowledgeTable entries={filtered} />
			)}
		</div>
	);
}

function StatsDisplay({ stats }: { stats: KnowledgeStats }) {
	return (
		<div className="flex gap-2 text-xs text-muted-foreground">
			<span>{stats.total} total</span>
			<Separator orientation="vertical" className="h-4" />
			<span>{stats.decision} dec</span>
			<span>{stats.pattern} pat</span>
			<span>{stats.rule} rule</span>
			<span>{stats.general} gen</span>
		</div>
	);
}

function KnowledgeTable({ entries }: { entries: KnowledgeEntry[] }) {
	const toggleMutation = useToggleEnabledMutation();
	const [expanded, setExpanded] = useState<number | null>(null);
	const hasScores = entries.some((e) => e.score);

	return (
		<Table>
			<TableHeader>
				<TableRow>
					<TableHead className="w-8" />
					<TableHead>Label</TableHead>
					<TableHead className="w-24">Type</TableHead>
					<TableHead className="w-16 text-right">Hits</TableHead>
					{hasScores && <TableHead className="w-20 text-right">Score</TableHead>}
				</TableRow>
			</TableHeader>
			<TableBody>
				{entries.map((entry) => (
					<KnowledgeRow
						key={entry.id}
						entry={entry}
						isExpanded={expanded === entry.id}
						showScore={hasScores}
						onToggleExpand={() => setExpanded(expanded === entry.id ? null : entry.id)}
						onToggleEnabled={() => toggleMutation.mutate({ id: entry.id, enabled: !entry.enabled })}
					/>
				))}
				{entries.length === 0 && (
					<TableRow>
						<TableCell colSpan={5} className="text-center text-sm text-muted-foreground">
							No entries found.
						</TableCell>
					</TableRow>
				)}
			</TableBody>
		</Table>
	);
}

function KnowledgeRow({
	entry,
	isExpanded,
	showScore,
	onToggleExpand,
	onToggleEnabled,
}: {
	entry: KnowledgeEntry;
	isExpanded: boolean;
	showScore: boolean;
	onToggleExpand: () => void;
	onToggleEnabled: () => void;
}) {
	const color = SUB_TYPE_COLORS[entry.sub_type] ?? SUB_TYPE_COLORS.general;

	return (
		<>
			<TableRow
				className={cn("cursor-pointer", !entry.enabled && "opacity-50")}
				onClick={onToggleExpand}
			>
				<TableCell>
					<Tooltip>
						<TooltipTrigger asChild>
							<button
								type="button"
								onClick={(e) => {
									e.stopPropagation();
									onToggleEnabled();
								}}
								className="text-xs"
							>
								{entry.enabled ? "[x]" : "[ ]"}
							</button>
						</TooltipTrigger>
						<TooltipContent>{entry.enabled ? "Disable" : "Enable"} memory</TooltipContent>
					</Tooltip>
				</TableCell>
				<TableCell className="font-medium text-sm">{entry.label}</TableCell>
				<TableCell>
					<Badge variant="outline" className="text-xs" style={{ borderColor: `${color}40`, color }}>
						{entry.sub_type}
					</Badge>
				</TableCell>
				<TableCell className="text-right text-xs text-muted-foreground">
					{entry.hit_count}
				</TableCell>
				{showScore && (
					<TableCell className="text-right text-xs text-muted-foreground">
						{entry.score ? entry.score.toFixed(3) : ""}
					</TableCell>
				)}
			</TableRow>
			{isExpanded && (
				<TableRow>
					<TableCell colSpan={5} className="bg-muted/30">
						<div className="space-y-1 py-2">
							<p className="text-xs text-muted-foreground">
								Source: {entry.source} | Saved: {entry.saved_at}
							</p>
							<pre className="text-xs whitespace-pre-wrap break-words font-mono max-h-64 overflow-auto">
								{entry.content}
							</pre>
						</div>
					</TableCell>
				</TableRow>
			)}
		</>
	);
}
