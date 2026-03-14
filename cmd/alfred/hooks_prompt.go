package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hir4ta/claude-alfred/internal/store"
)

// openStore is the function used to obtain a store connection.
// Overridable in tests.
var openStore = func() (*store.Store, error) {
	return store.OpenDefaultCached()
}

// Default scoring thresholds for knowledge injection.
// Overridden by: .alfred/config.json (project) > environment variable > default.
const (
	defaultRelevanceThreshold      = 0.40
	defaultHighConfidenceThreshold = 0.65
	defaultSingleKeywordDampen     = 0.80
)

// Scoring weights for keyword-aware relevance computation.
const (
	kwPathWeight    = 0.40 // weight for keyword hits in doc section path
	kwContentWeight = 0.20 // weight for keyword hits in doc content
	coverageWeight  = 0.30 // weight for prompt token coverage in doc
	earlyBonus      = 0.05 // bonus per early (first-line) hit
)

// scored pairs a document with its relevance score.
type scored struct {
	doc   store.DocRow
	score float64
}

// envFloat returns the environment variable as float64 or the default value.
func envFloat(key string, defaultVal float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		debugf("envFloat: invalid %s=%q, using default %v", key, v, defaultVal)
		return defaultVal
	}
	if f < 0 || f > 1 {
		debugf("envFloat: %s=%v out of range [0,1], clamping", key, f)
		return max(0, min(1, f))
	}
	return f
}

// ---------------------------------------------------------------------------
// Architecture note: Hook vs MCP knowledge injection
//
// UserPromptSubmit hook (this file + hooks_semantic.go):
//   - Passive/proactive: fires automatically on every prompt
//   - When VOYAGE_API_KEY is set: semantic search (embed + hybrid RRF)
//   - When unavailable: fallback to FTS5 keyword pipeline
//   - Scope: injects up to 2 short snippets (300 chars each)
//   - Purpose: surface relevant context BEFORE Claude starts working
//
// MCP "knowledge" tool (mcpserver/handlers_search.go):
//   - Active: called explicitly by Claude or user
//   - Full pipeline: hybrid vector + FTS5 + Voyage rerank
//   - Scope: returns full search results with scores
//   - Purpose: deep research when Claude needs detailed information
//
// The hook provides semantic priming; the MCP tool provides deep answers.
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// UserPromptSubmit: Claude Code config keyword detection + knowledge injection
// ---------------------------------------------------------------------------

// shouldRemindPrompt reports whether the user's prompt mentions Claude Code
// configuration paths (.claude, CLAUDE.md, MEMORY.md, .mcp.json).
func shouldRemindPrompt(prompt string) bool {
	lower := strings.ToLower(prompt)
	for _, term := range []string{".claude", "claude.md", "memory.md", ".mcp.json"} {
		if strings.Contains(lower, term) {
			return true
		}
	}
	return false
}

// scoreRelevance computes a relevance score (0.0-1.0) for injecting a document.
//
// Two-signal design:
//   - Primary signal: matched Claude Code keywords in doc path/content (high weight)
//   - Secondary signal: prompt content token coverage in doc (bonus)
//
// Single-keyword matches are dampened to require content coverage for injection.
// This prevents generic docs from being injected when the prompt merely mentions a keyword
// without actually asking about that topic.
//
// matchedKeywords are the Claude Code keywords detected in the prompt by Gate 1.
// promptLower is the full prompt (lowercased) for secondary coverage scoring.
// dampen is the single-keyword dampening factor.
func scoreRelevance(matchedKeywords []string, promptLower string, doc store.DocRow, dampen float64) float64 {
	pathLower := strings.ToLower(doc.SectionPath)
	contentLower := strings.ToLower(doc.Content)
	firstLine := contentLower
	if idx := strings.IndexByte(firstLine, '\n'); idx > 0 {
		firstLine = firstLine[:idx]
	}

	// Primary signal: matched Claude Code keywords in doc.
	// For katakana keywords, also check their English equivalents against the English KB.
	kwPathHits := 0
	kwContentHits := 0
	for _, kw := range matchedKeywords {
		kwCheck := kw
		if en, ok := store.TranslateTerm(kw); ok {
			kwCheck = en
		}
		if strings.Contains(pathLower, kwCheck) {
			kwPathHits++
		}
		if strings.Contains(contentLower, kwCheck) {
			kwContentHits++
		}
	}
	nkw := max(len(matchedKeywords), 1)
	keywordScore := float64(kwPathHits)*kwPathWeight/float64(nkw) + float64(kwContentHits)*kwContentWeight/float64(nkw)

	// Dampen single-keyword confidence: one keyword alone is weak signal.
	if len(matchedKeywords) == 1 {
		keywordScore *= dampen
	}

	// Secondary signal: content token coverage in doc.
	// Uses POS-filtered tokens with base forms for better cross-lingual matching.
	meaningful := contentTokensForScoring(promptLower)

	contentHits := 0
	earlyHits := 0
	for _, w := range meaningful {
		if strings.Contains(contentLower, w) {
			contentHits++
			if strings.Contains(firstLine, w) {
				earlyHits++
			}
		}
	}

	coverageScore := 0.0
	if len(meaningful) > 0 {
		coverageScore = float64(contentHits) / float64(len(meaningful)) * coverageWeight
	}
	earlyScore := float64(earlyHits) * earlyBonus

	return min(keywordScore+coverageScore+earlyScore, 1.0)
}

// handleUserPromptSubmit emits config reminders and proactively injects
// relevant knowledge using Voyage semantic search (embed + hybrid RRF).
//
// Semantic-first design (v5):
// 1. Embed prompt via Voyage API → hybrid search (vector + FTS via RRF)
// 2. Apply feedback boost + spec/session context boost
// 3. Inject 1-2 results based on confidence
//
// If VOYAGE_API_KEY is not set, knowledge injection is skipped entirely.
// Workflow hints, remember hints, and instincts still work without Voyage.
func handleUserPromptSubmit(ctx context.Context, ev *hookEvent) {
	cfg := loadProjectConfig(ev.ProjectPath)
	var quietPtr *bool
	if cfg != nil {
		quietPtr = cfg.Quiet
	}

	if resolveBool(quietPtr, "ALFRED_QUIET") {
		debugf("UserPromptSubmit: quiet mode, skipping")
		return
	}

	if shouldRemindPrompt(ev.Prompt) {
		emitAdditionalContext("UserPromptSubmit", configReminder)
		return
	}

	prompt := strings.TrimSpace(ev.Prompt)
	if len([]rune(prompt)) < 10 {
		return
	}

	// Detect workflow opportunities and suggest skills proactively.
	workflowHint := detectWorkflowOpportunity(prompt, ev.ProjectPath)
	if ctxHint := estimateContextPressure(ev); ctxHint != "" {
		if workflowHint != "" {
			workflowHint += "\n\n" + ctxHint
		} else {
			workflowHint = ctxHint
		}
	}

	// Detect "remember this" intent for recall tool suggestion.
	rememberHint := ""
	if detectRememberIntent(prompt) {
		rememberHint = "User wants to save information. Use the recall tool with action=save to persist this as permanent memory. " +
			"Parameters: content (what to save), label (short description), project (optional context)."
	}

	// Load spec/session context for proactive knowledge push.
	var specCtx *specContext
	var ctxBoostDisable *bool
	if cfg != nil {
		ctxBoostDisable = cfg.ContextBoostDisable
	}
	if !resolveBool(ctxBoostDisable, "ALFRED_CONTEXT_BOOST_DISABLE") {
		specCtx = loadSpecContext(ev.ProjectPath)
	}

	// Semantic search: embed prompt → hybrid search → boost → inject.
	// If Voyage is unavailable, only workflow/remember hints are emitted.
	if handleSemanticSearch(ctx, ev, cfg, prompt, specCtx, workflowHint, rememberHint) {
		return
	}

	// Voyage unavailable — emit non-search hints only.
	var buf strings.Builder
	if workflowHint != "" {
		buf.WriteString(workflowHint)
	}
	if rememberHint != "" {
		if buf.Len() > 0 {
			buf.WriteString("\n\n")
		}
		buf.WriteString(rememberHint)
	}
	if buf.Len() > 0 {
		emitAdditionalContext("UserPromptSubmit", buf.String())
	}
}

// rememberKeywords are phrases indicating the user wants to save information.
var rememberKeywords = []string{
	"覚えて", "覚えておいて", "記憶して", "記憶しておいて",
	"メモして", "メモしておいて",
	"remember this", "remember that", "save this", "save that",
	"don't forget",
}

// detectRememberIntent returns true if the prompt contains a "remember this" keyword.
func detectRememberIntent(prompt string) bool {
	lower := strings.ToLower(prompt)
	for _, kw := range rememberKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// evaluateInjectionFeedback checks if docs injected in the previous prompt
// are referenced in the current prompt (implicit positive signal).
// Uses a 10-minute window to capture multi-turn conversations.
func evaluateInjectionFeedback(ctx context.Context, prompt string, st *store.Store) {
	recentIDs, err := st.GetRecentInjections(ctx, 10*time.Minute)
	if err != nil || len(recentIDs) == 0 {
		return
	}

	// Load the injected docs and check if their topics appear in the current prompt.
	docs, err := st.GetDocsByIDs(ctx, recentIDs)
	if err != nil || len(docs) == 0 {
		return
	}

	promptLower := strings.ToLower(prompt)
	// Skip negative feedback for very short prompts (e.g., "ok", "continue", "はい").
	// Short prompts can't plausibly reference injected topics, so absence of
	// overlap is not a meaningful negative signal.
	promptWords := strings.Fields(promptLower)
	shortPrompt := len(promptWords) < 5

	for _, doc := range docs {
		// Extract significant words from the doc's section path.
		pathWords := strings.Fields(strings.ToLower(doc.SectionPath))
		hits := 0
		meaningful := 0
		for _, w := range pathWords {
			w = strings.Trim(w, ">|")
			if len(w) >= 3 {
				meaningful++
				if strings.Contains(promptLower, w) {
					hits++
				}
			}
		}
		if meaningful == 0 {
			continue
		}
		positive := float64(hits)/float64(meaningful) >= 0.3
		if positive {
			if err := st.RecordFeedback(ctx, doc.ID, true); err != nil {
				debugf("evaluateInjectionFeedback: positive feedback error: %v", err)
			}
		} else if !shortPrompt {
			// Only record negative feedback for substantive prompts.
			if err := st.RecordFeedback(ctx, doc.ID, false); err != nil {
				debugf("evaluateInjectionFeedback: negative feedback error: %v", err)
			}
		}
	}
}

// searchMemoryForPrompt searches memory docs for the user's prompt.
// Returns formatted snippet lines (max 2) or nil if no relevant memories found.
func searchMemoryForPrompt(ctx context.Context, prompt string, st *store.Store) []string {
	keywords := extractSearchKeywords(prompt, 6)
	if keywords == "" {
		return nil
	}

	docs, err := st.SearchDocsFTS(ctx, keywords, store.SourceMemory, 2)
	if err != nil || len(docs) == 0 {
		return nil
	}

	var results []string
	for _, d := range docs {
		snippet := safeSnippet(d.Content, 200)
		results = append(results, fmt.Sprintf("- [%s] %s\n", d.SectionPath, snippet))
	}
	debugf("UserPromptSubmit: memory search found %d results for keywords=%s", len(results), keywords)
	return results
}
