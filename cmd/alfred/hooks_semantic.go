package main

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hir4ta/claude-alfred/internal/embedder"
	"github.com/hir4ta/claude-alfred/internal/store"
)

// semanticRelevanceThreshold is the minimum RRF score for semantic search results.
// Lower than the keyword threshold (0.40) because RRF scores are on a different scale.
const semanticRelevanceThreshold = 0.01

// semanticOverRetrieve is the over-retrieval factor for hybrid search.
// Retrieve more candidates than needed, then trim after feedback/context boost.
const semanticOverRetrieve = 20

// newEmbedder creates a Voyage embedder, returning nil if unavailable.
// Overridable in tests.
var newEmbedder = func() *embedder.Embedder {
	emb, err := embedder.NewEmbedder()
	if err != nil {
		debugf("semantic: embedder unavailable: %v", err)
		return nil
	}
	return emb
}

// handleSemanticSearch performs semantic knowledge injection using Voyage embeddings.
// Returns true if it handled the injection (caller should skip keyword pipeline).
// Returns false if Voyage is unavailable or search failed (caller should fallback).
func handleSemanticSearch(ctx context.Context, ev *hookEvent, _ *ProjectConfig,
	prompt string, specCtx *specContext, workflowHint, rememberHint string) bool {

	emb := newEmbedder()
	if emb == nil {
		return false
	}

	st, err := openStore()
	if err != nil {
		debugf("semantic: store open failed: %v", err)
		return false
	}
	st.ExpectedDims = emb.Dims()
	st.ExpectedModel = emb.Model()

	// Embed the user's prompt for semantic search.
	queryVec, err := emb.EmbedForSearch(ctx, prompt)
	if err != nil {
		debugf("semantic: embed failed: %v", err)
		return false // no injection; Voyage is required
	}

	// Build FTS query from prompt keywords for hybrid search.
	// Use JoinFTS5Terms to ensure proper sanitization (extractSearchKeywords
	// may contain bare OR operators from the CJK path).
	ftsTerms := strings.Fields(extractSearchKeywords(prompt, 8))
	ftsQuery := store.JoinFTS5Terms(ftsTerms)

	// Hybrid search: vector similarity + FTS, combined via RRF.
	matches, err := st.HybridSearch(ctx, queryVec, ftsQuery, store.SourceDocs, semanticOverRetrieve, semanticOverRetrieve)
	if err != nil {
		debugf("semantic: hybrid search failed: %v", err)
		return false
	}
	if len(matches) == 0 {
		debugf("semantic: no matches found")
		notifyUser("semantic search: no matches (embeddings may not be indexed yet)")
		return false
	}

	// Fetch full doc rows.
	ids := make([]int64, len(matches))
	rrfScores := make(map[int64]float64, len(matches))
	for i, m := range matches {
		ids[i] = m.DocID
		rrfScores[m.DocID] = m.RRFScore
	}
	docs, err := st.GetDocsByIDs(ctx, ids)
	if err != nil {
		debugf("semantic: GetDocsByIDs failed: %v", err)
		return false
	}

	// Build scored candidates preserving RRF order.
	var candidates []scored
	for _, doc := range docs {
		s := rrfScores[doc.ID]
		if s >= semanticRelevanceThreshold {
			candidates = append(candidates, scored{doc, s})
		}
	}
	if len(candidates) == 0 {
		debugf("semantic: no candidates above threshold")
		return false
	}

	// Sort by RRF score descending.
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	// Implicit feedback tracking.
	if ctx.Err() == nil {
		evaluateInjectionFeedback(ctx, prompt, st)
	}

	// Apply feedback boost.
	boostIDs := make([]int64, len(candidates))
	for i := range candidates {
		boostIDs[i] = candidates[i].doc.ID
	}
	boosts := st.FeedbackBoostBatch(ctx, boostIDs)
	for i := range candidates {
		if b, ok := boosts[candidates[i].doc.ID]; ok {
			candidates[i].score += b - 1.0
		}
	}

	// Apply spec/session context boost.
	ctxBoostedIDs := applyContextBoost(candidates, specCtx)
	if len(ctxBoostedIDs) > 0 {
		debugf("semantic: context boost applied to %d candidates", len(ctxBoostedIDs))
	}

	// Re-sort after boosts.
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	// Confidence-based injection: 1 by default, 2 if top score is significantly
	// above the second. RRF scores are typically 0.01-0.05, so we use a relative
	// ratio rather than an absolute threshold.
	maxResults := 1
	if len(candidates) > 1 && candidates[0].score >= candidates[1].score*1.5 {
		maxResults = 2
	}
	if len(candidates) > maxResults {
		candidates = candidates[:maxResults]
	}

	// Build output.
	var buf strings.Builder
	if workflowHint != "" {
		buf.WriteString(workflowHint + "\n\n")
	}
	if rememberHint != "" {
		buf.WriteString(rememberHint + "\n\n")
	}

	// Separate context-boosted results from semantic results.
	var regular, contextAware []scored
	for _, c := range candidates {
		if ctxBoostedIDs[c.doc.ID] {
			contextAware = append(contextAware, c)
		} else {
			regular = append(regular, c)
		}
	}
	if len(regular) > 0 {
		buf.WriteString("Relevant best practices from alfred knowledge base:\n")
		for _, c := range regular {
			snippet := safeSnippet(c.doc.Content, 300)
			fmt.Fprintf(&buf, "- [%s] %s\n", c.doc.SectionPath, snippet)
		}
	}
	if len(contextAware) > 0 {
		if len(regular) > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString("Context-aware suggestions (based on current task):\n")
		for _, c := range contextAware {
			snippet := safeSnippet(c.doc.Content, 300)
			fmt.Fprintf(&buf, "- [%s] %s\n", c.doc.SectionPath, snippet)
		}
	}

	// Also search memories semantically.
	memSnippets := searchMemorySemantic(ctx, queryVec, st)
	if len(memSnippets) > 0 {
		buf.WriteString("\nRelated past experience:\n")
		for _, m := range memSnippets {
			buf.WriteString(m)
		}
	}

	// Instinct injection (same as keyword pipeline — FTS-based, fast).
	instinctSnippets := searchRelevantInstincts(ctx, prompt, ev.ProjectPath, st)
	if len(instinctSnippets) > 0 {
		buf.WriteString("\nLearned patterns from past sessions:\n")
		for _, s := range instinctSnippets {
			buf.WriteString(s)
		}
	}

	// Record injected docs for feedback tracking.
	var injectedIDs []int64
	for _, c := range candidates {
		injectedIDs = append(injectedIDs, c.doc.ID)
	}
	if err := st.RecordInjection(ctx, injectedIDs); err != nil {
		debugf("semantic: record injection error: %v", err)
	}

	emitAdditionalContext("UserPromptSubmit", buf.String())
	notifyUser("injected %d snippet(s) via semantic search (top: %.4f)", len(candidates), candidates[0].score)
	debugf("semantic: injected %d snippets (top: %.4f), %d memory hints",
		len(candidates), candidates[0].score, len(memSnippets))
	return true
}

// searchMemorySemantic searches memory docs using vector similarity.
// Returns formatted snippet lines (max 2) or nil.
func searchMemorySemantic(ctx context.Context, queryVec []float32, st *store.Store) []string {
	if queryVec == nil {
		return nil
	}

	matches, err := st.VectorSearch(ctx, queryVec, "docs", 4, store.SourceMemory)
	if err != nil || len(matches) == 0 {
		return nil
	}

	// Fetch top 2 by similarity.
	limit := min(2, len(matches))
	ids := make([]int64, limit)
	for i := 0; i < limit; i++ {
		ids[i] = matches[i].SourceID
	}
	docs, err := st.GetDocsByIDs(ctx, ids)
	if err != nil || len(docs) == 0 {
		return nil
	}

	var results []string
	for _, d := range docs {
		snippet := safeSnippet(d.Content, 200)
		results = append(results, fmt.Sprintf("- [%s] %s\n", d.SectionPath, snippet))
	}
	debugf("semantic: memory search found %d results", len(results))
	return results
}
