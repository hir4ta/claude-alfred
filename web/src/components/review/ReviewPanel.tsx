import { useQuery } from "@tanstack/react-query";
import { CheckSquare, History, MessageSquare, Square } from "@animated-color-icons/lucide-react";
import { useCallback, useMemo, useState } from "react";
import CodeMirror from "@uiw/react-codemirror";
import { markdown } from "@codemirror/lang-markdown";
import { EditorView, Decoration, type DecorationSet, ViewPlugin, type ViewUpdate, gutter, GutterMarker } from "@codemirror/view";
import { StateField, StateEffect, RangeSetBuilder } from "@codemirror/state";
import { SpecHistory } from "./SpecHistory";
import { Card } from "@/components/ui/card";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";
import { reviewHistoryQueryOptions } from "@/lib/api";
import type { Review, ReviewComment } from "@/lib/types";
import { useI18n } from "@/lib/i18n";
import { cn } from "@/lib/utils";

interface ReviewPanelProps {
	slug: string;
	reviewStatus: string;
	specContent: string;
	currentFile: string;
	comments?: Array<{ file: string; line: number; body: string }>;
	onAddComment?: (line: number, body: string) => void;
	onRemoveComment?: (line: number, body: string) => void;
}

export function ReviewPanel({
	slug,
	reviewStatus,
	specContent,
	currentFile,
	comments = [],
	onAddComment,
	onRemoveComment,
}: ReviewPanelProps) {
	const { t } = useI18n();
	const { data: historyData } = useQuery(reviewHistoryQueryOptions(slug));
	const [newComment, setNewComment] = useState("");
	const [selectedLine, setSelectedLine] = useState<number | null>(null);
	const [activeTab, setActiveTab] = useState<"review" | "history">("review");
	const [resolvedOverrides, setResolvedOverrides] = useState<Map<string, boolean>>(new Map());

	const reviews = historyData?.reviews ?? [];
	const latestReview = reviews[reviews.length - 1];
	const unresolvedFromPrevious =
		latestReview?.comments?.filter((c) => !c.resolved && c.file === currentFile) ?? [];

	const togglePreviousResolved = (comment: ReviewComment) => {
		const key = `${comment.file}:${comment.line}:${comment.body}`;
		setResolvedOverrides((prev) => {
			const next = new Map(prev);
			const current = next.get(key) ?? comment.resolved ?? false;
			next.set(key, !current);
			return next;
		});
	};

	const isResolved = (comment: ReviewComment) => {
		const key = `${comment.file}:${comment.line}:${comment.body}`;
		return resolvedOverrides.get(key) ?? comment.resolved ?? false;
	};

	const addComment = useCallback(() => {
		if (!newComment.trim() || selectedLine === null || !onAddComment) return;
		onAddComment(selectedLine, newComment.trim());
		setNewComment("");
		setSelectedLine(null);
	}, [newComment, selectedLine, onAddComment]);

	// Compute highlighted lines (lines with comments)
	const commentedLines = useMemo(() => {
		const lines = new Set<number>();
		for (const c of comments) lines.add(c.line);
		for (const c of unresolvedFromPrevious) lines.add(c.line);
		return lines;
	}, [comments, unresolvedFromPrevious]);

	const handleLineClick = useCallback((line: number) => {
		if (reviewStatus !== "pending") return;
		setSelectedLine((prev) => (prev === line ? null : line));
	}, [reviewStatus]);

	// CM6 extensions
	const extensions = useMemo(() => [
		markdown(),
		EditorView.editable.of(false),
		EditorView.lineWrapping,
		EditorView.theme({
			"&": { fontSize: "12px", fontFamily: "var(--font-mono, monospace)" },
			".cm-content": { padding: "8px 0" },
			".cm-line": { padding: "1px 8px" },
			".cm-gutters": { borderRight: "1px solid rgba(0,0,0,0.08)", backgroundColor: "transparent", cursor: "pointer" },
			".cm-lineNumbers .cm-gutterElement": { padding: "0 8px 0 4px", minWidth: "2.5em", color: "rgba(0,0,0,0.3)" },
			".cm-activeLine": { backgroundColor: "rgba(98,129,65,0.08)" },
			".cm-comment-line": { backgroundColor: "rgba(230,126,34,0.06)" },
			".cm-selected-line": { backgroundColor: "rgba(45,139,122,0.12)" },
		}),
	], []);

	return (
		<div className="space-y-3">
			<div className="flex items-center gap-2">
				<button type="button" onClick={() => setActiveTab("review")}
					className={cn("text-sm font-medium px-2 py-0.5 rounded-lg transition-colors", activeTab === "review" ? "bg-accent text-foreground" : "text-muted-foreground hover:text-foreground")}
				>Review</button>
				<button type="button" onClick={() => setActiveTab("history")}
					className={cn("text-sm font-medium px-2 py-0.5 rounded-lg transition-colors flex items-center gap-1", activeTab === "history" ? "bg-accent text-foreground" : "text-muted-foreground hover:text-foreground")}
				><History className="size-3.5" />{t("review.history")}</button>
			</div>

			{activeTab === "history" && <SpecHistory slug={slug} file={currentFile} />}

			{activeTab === "review" && (
				<>
					{/* CodeMirror viewer with line click */}
					<div className="rounded-lg border border-border/60 overflow-hidden">
						<LineClickCodeMirror
							content={specContent}
							extensions={extensions}
							selectedLine={selectedLine}
							commentedLines={commentedLines}
							onLineClick={handleLineClick}
						/>
					</div>

					{/* Inline comments display */}
					{(comments.length > 0 || unresolvedFromPrevious.length > 0) && (
						<div className="space-y-1.5">
							{unresolvedFromPrevious.map((c, i) => (
								<InlineComment
									key={`prev-${i}`}
									file={currentFile}
									comment={c}
									isPrevious
									resolved={isResolved(c)}
									onToggleResolved={() => togglePreviousResolved(c)}
								/>
							))}
							{comments.map((c, i) => (
								<InlineComment
									key={`new-${i}`}
									file={currentFile}
									comment={{ ...c, resolved: false }}
									onRemove={() => onRemoveComment?.(c.line, c.body)}
								/>
							))}
						</div>
					)}

					{/* Add comment */}
					{selectedLine !== null && reviewStatus === "pending" && (
						<div className="flex gap-2 items-end">
							<div className="flex-1 space-y-1">
								<p className="text-xs text-muted-foreground">
									{t("review.commentOn")} {currentFile}:{selectedLine}
								</p>
								<Textarea
									value={newComment}
									onChange={(e) => setNewComment(e.target.value)}
									placeholder={t("review.addComment")}
									className="min-h-[60px] text-sm"
									autoFocus
									onKeyDown={(e) => {
										if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
											e.preventDefault();
											addComment();
										}
									}}
								/>
							</div>
							<Button size="sm" onClick={addComment} disabled={!newComment.trim()}>
								<MessageSquare className="h-3.5 w-3.5 mr-1" />
								{t("review.add")}
							</Button>
						</div>
					)}

					{reviews.length > 0 && <ReviewHistory reviews={reviews} />}
				</>
			)}
		</div>
	);
}

// --- CodeMirror with line click handling ---

function LineClickCodeMirror({
	content,
	extensions,
	selectedLine,
	commentedLines,
	onLineClick,
}: {
	content: string;
	extensions: ReturnType<typeof markdown>[];
	selectedLine: number | null;
	commentedLines: Set<number>;
	onLineClick: (line: number) => void;
}) {
	const handleClick = useCallback(
		(e: React.MouseEvent<HTMLDivElement>) => {
			const target = e.target as HTMLElement;
			// Find the CM line from the click target
			const lineEl = target.closest(".cm-line") ?? target.closest(".cm-gutterElement");
			if (!lineEl) return;

			const cmContent = (e.currentTarget as HTMLElement).querySelector(".cm-content");
			if (!cmContent) return;

			// For gutter clicks, extract line number from text
			if (target.closest(".cm-gutterElement")) {
				const num = parseInt(target.textContent ?? "", 10);
				if (!Number.isNaN(num)) onLineClick(num);
				return;
			}

			// For content clicks, find line index
			const lines = cmContent.querySelectorAll(".cm-line");
			for (let i = 0; i < lines.length; i++) {
				if (lines[i] === lineEl || lines[i]?.contains(lineEl)) {
					onLineClick(i + 1);
					return;
				}
			}
		},
		[onLineClick],
	);

	// Build className overrides for highlighted lines
	const lineStyles = useMemo(() => {
		const styles: string[] = [];
		if (selectedLine) {
			styles.push(`&.cm-editor .cm-content > .cm-line:nth-child(${selectedLine}) { background-color: rgba(45,139,122,0.12); }`);
		}
		for (const line of commentedLines) {
			if (line !== selectedLine) {
				styles.push(`&.cm-editor .cm-content > .cm-line:nth-child(${line}) { background-color: rgba(230,126,34,0.06); }`);
			}
		}
		return EditorView.theme(Object.fromEntries(styles.map((s, i) => [`&_hl${i}`, {}])));
	}, [selectedLine, commentedLines]);

	// Use CSS injection for line highlighting (CM6 decoration API requires editor state)
	const highlightCSS = useMemo(() => {
		const rules: string[] = [];
		if (selectedLine) {
			rules.push(`.cm-content > .cm-line:nth-child(${selectedLine}) { background-color: rgba(45,139,122,0.12) !important; }`);
		}
		for (const line of commentedLines) {
			if (line !== selectedLine) {
				rules.push(`.cm-content > .cm-line:nth-child(${line}) { background-color: rgba(230,126,34,0.06) !important; }`);
			}
		}
		return rules.join("\n");
	}, [selectedLine, commentedLines]);

	return (
		<div onClick={handleClick} className="relative">
			{highlightCSS && <style>{highlightCSS}</style>}
			<CodeMirror
				value={content}
				extensions={extensions}
				editable={false}
				basicSetup={{
					lineNumbers: true,
					foldGutter: false,
					highlightActiveLine: false,
					highlightSelectionMatches: false,
					drawSelection: false,
				}}
				maxHeight="500px"
			/>
		</div>
	);
}

// --- InlineComment ---

function InlineComment({
	file,
	comment,
	isPrevious,
	resolved,
	onRemove,
	onToggleResolved,
}: {
	file: string;
	comment: ReviewComment;
	isPrevious?: boolean;
	resolved?: boolean;
	onRemove?: () => void;
	onToggleResolved?: () => void;
}) {
	const isR = resolved ?? comment.resolved ?? false;
	return (
		<div
			className={cn(
				"rounded-lg px-3 py-2 text-xs border",
				isPrevious ? "border-brand-rule/20 bg-brand-rule/[0.04]" : "border-brand-decision/20 bg-brand-decision/[0.04]",
				isR && "opacity-50",
			)}
		>
			<div className="flex items-start justify-between gap-2">
				<div className="space-y-0.5">
					<span className="font-mono text-[10px] text-muted-foreground">{file}:{comment.line}</span>
					<p className="whitespace-pre-wrap">{comment.body}</p>
				</div>
				<div className="flex items-center gap-1 shrink-0">
					{isPrevious && onToggleResolved && (
						<button type="button" onClick={onToggleResolved}
							className="text-muted-foreground hover:text-foreground transition-colors"
							title={isR ? "Mark unresolved" : "Mark resolved"}
						>
							{isR ? <CheckSquare className="size-3.5" /> : <Square className="size-3.5" />}
						</button>
					)}
					{onRemove && (
						<button type="button" onClick={onRemove}
							className="text-muted-foreground hover:text-foreground text-[10px]"
						>✕</button>
					)}
				</div>
			</div>
		</div>
	);
}

// --- ReviewHistory ---

function ReviewHistory({ reviews }: { reviews: Review[] }) {
	const { t, locale } = useI18n();
	return (
		<Card className="p-3">
			<p className="text-xs font-medium text-muted-foreground mb-2">
				{t("review.history")} ({reviews.length})
			</p>
			<div className="space-y-2">
				{reviews.map((review) => (
					<div key={review.timestamp} className="flex items-center gap-2 text-xs">
						<ReviewStatusBadge status={review.status} />
						<span className="text-muted-foreground">
							{new Date(review.timestamp).toLocaleString(locale === "ja" ? "ja-JP" : "en-US", {
								month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit",
							})}
						</span>
						{review.comments && review.comments.length > 0 && (
							<span className="text-muted-foreground/60">
								{review.comments.length} {t("review.comments")}
							</span>
						)}
					</div>
				))}
			</div>
		</Card>
	);
}

function ReviewStatusBadge({ status }: { status: string }) {
	const colors: Record<string, { color: string; bg: string }> = {
		pending: { color: "#6b7280", bg: "rgba(107,114,128,0.15)" },
		approved: { color: "#2d8b7a", bg: "rgba(45,139,122,0.15)" },
		changes_requested: { color: "#e67e22", bg: "rgba(230,126,34,0.15)" },
	};
	const s = colors[status] ?? colors.pending!;
	return (
		<span className="inline-flex items-center rounded px-1.5 py-0.5 text-xs font-medium"
			style={{ backgroundColor: s.bg, color: s.color }}
		>{status}</span>
	);
}
