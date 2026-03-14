package mcpserver

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/hir4ta/claude-alfred/internal/embedder"
	"github.com/hir4ta/claude-alfred/internal/spec"
	"github.com/hir4ta/claude-alfred/internal/store"
)

// bgEmbedWG tracks in-flight background embedding goroutines.
var bgEmbedWG sync.WaitGroup

// WaitBackground blocks until all background embedding goroutines complete.
// Call during graceful shutdown to ensure all embeddings are persisted.
func WaitBackground() { bgEmbedWG.Wait() }

// recallHandler provides memory-specific search and save operations.
// Unlike the general "knowledge" tool, recall focuses on user memories:
// past sessions, decisions, and explicitly saved notes.
func recallHandler(st *store.Store, emb *embedder.Embedder) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		action := req.GetString("action", "search")

		switch action {
		case "search":
			return recallSearch(ctx, st, emb, req)
		case "save":
			return recallSave(ctx, st, emb, req)
		case "instincts":
			return recallInstincts(ctx, st, req)
		case "instinct-feedback":
			return recallInstinctFeedback(ctx, st, req)
		default:
			return mcp.NewToolResultError(fmt.Sprintf("unknown action %q: use 'search', 'save', 'instincts', or 'instinct-feedback'", action)), nil
		}
	}
}

// recallSearch searches memory entries (source_type=store.SourceMemory) using hybrid
// or FTS-only search depending on embedder availability.
func recallSearch(ctx context.Context, st *store.Store, emb *embedder.Embedder, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := req.GetString("query", "")
	if query == "" {
		return mcp.NewToolResultError("query parameter is required for search"), nil
	}
	if len([]rune(query)) > 10000 {
		return mcp.NewToolResultError("query too long (max 10000 characters)"), nil
	}
	limit := req.GetInt("limit", 10)
	if limit < 1 {
		limit = 10
	}
	limitCapped := false
	if limit > 100 {
		limit = 100
		limitCapped = true
	}

	overRetrieve := limit * 4
	if overRetrieve < 20 {
		overRetrieve = 20
	}

	sr := hybridSearchPipeline(ctx, st, emb, query, store.SourceMemory, limit, overRetrieve)
	docs := sr.Docs
	searchMethod := sr.SearchMethod
	warnings := sr.Warnings

	results := make([]map[string]any, 0, len(docs))
	for _, d := range docs {
		dm := map[string]any{
			"section_path": d.SectionPath,
			"content":      d.Content,
			"url":          d.URL,
		}
		if d.CrawledAt != "" {
			dm["saved_at"] = d.CrawledAt
		}
		results = append(results, dm)
	}

	result := map[string]any{
		"query":         query,
		"results":       results,
		"count":         len(results),
		"search_method": searchMethod,
	}
	if limitCapped {
		warnings = append(warnings, "limit capped to 100 (maximum allowed)")
	}
	if len(warnings) > 0 {
		result["warning"] = strings.Join(warnings, "; ")
	}
	return marshalResult(result)
}

// maxContentBytes limits content size for MCP write operations (256KB).
// Shared by recall save and spec update to prevent oversized payloads
// from bloating the DB and embedding pipeline.
const maxContentBytes = 256 * 1024

// recallSave saves a new memory entry to the knowledge base.
// If an embedder is available, it asynchronously generates an embedding for semantic search.
func recallSave(ctx context.Context, st *store.Store, emb *embedder.Embedder, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	content := req.GetString("content", "")
	if content == "" {
		return mcp.NewToolResultError("content parameter is required for save"), nil
	}
	if len(content) > maxContentBytes {
		return mcp.NewToolResultError(fmt.Sprintf("content too large: %d bytes (max %d bytes / 256KB)", len(content), maxContentBytes)), nil
	}
	label := req.GetString("label", "")
	if label == "" {
		return mcp.NewToolResultError("label parameter is required for save (short description)"), nil
	}
	project := req.GetString("project", "general")

	// Validate project name to prevent path traversal and section_path parsing issues.
	if !spec.ValidSlug.MatchString(project) {
		return mcp.NewToolResultError("invalid project name: use lowercase letters, digits, and hyphens only (max 64 chars)"), nil
	}

	ts := time.Now().Format("2006-01-02T150405")
	url := fmt.Sprintf("memory://user/%s/manual/%s", project, ts)
	sectionPath := fmt.Sprintf("%s > manual > %s", project, truncate(label, 60))

	id, changed, err := st.UpsertDoc(ctx, &store.DocRow{
		URL:         url,
		SectionPath: sectionPath,
		Content:     strings.TrimSpace(content),
		SourceType:  store.SourceMemory,
		TTLDays:     0, // permanent
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("save failed: %v", err)), nil
	}

	status := "saved"
	if !changed {
		status = "unchanged (duplicate)"
	}

	// Async embedding: generate vector for semantic recall search.
	// Tracked by bgEmbedWG for graceful shutdown; embedding failure only degrades vector search.
	if emb != nil && changed {
		bgEmbedWG.Add(1)
		go func() {
			defer bgEmbedWG.Done()
			embCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			vec, err := emb.EmbedForStorage(embCtx, strings.TrimSpace(content))
			if err != nil {
				if store.DebugLog != nil {
					store.DebugLog("recall save: async embed failed: %v", err)
				}
				return
			}
			if err := st.InsertEmbedding("docs", id, emb.Model(), vec); err != nil {
				if store.DebugLog != nil {
					store.DebugLog("recall save: insert embedding failed: %v", err)
				}
			}
		}()
	}

	embeddingStatus := "none"
	if emb != nil && changed {
		embeddingStatus = "pending"
	}

	return marshalResult(map[string]any{
		"status":           status,
		"id":               id,
		"section_path":     sectionPath,
		"url":              url,
		"embedding_status": embeddingStatus,
	})
}

// recallInstincts lists or searches learned behavioral patterns.
func recallInstincts(ctx context.Context, st *store.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := req.GetString("query", "")
	project := req.GetString("project", "")
	limit := req.GetInt("limit", 20)
	if limit > 100 {
		limit = 100
	}

	var instincts []store.Instinct
	var err error
	if query != "" {
		instincts, err = st.SearchInstinctsFTS(ctx, query, project, limit)
	} else {
		instincts, err = st.SearchInstincts(ctx, project, "", limit)
	}
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("instinct search failed: %v", err)), nil
	}

	results := make([]map[string]any, 0, len(instincts))
	for _, inst := range instincts {
		results = append(results, map[string]any{
			"id":            inst.ID,
			"trigger":       inst.Trigger,
			"action":        inst.Action,
			"confidence":    inst.Confidence,
			"domain":        inst.Domain,
			"scope":         inst.Scope,
			"times_applied": inst.TimesApplied,
			"updated_at":    inst.UpdatedAt,
		})
	}

	return marshalResult(map[string]any{
		"action":  "instincts",
		"results": results,
		"count":   len(results),
	})
}

// recallInstinctFeedback allows manual confidence adjustment of an instinct.
func recallInstinctFeedback(ctx context.Context, st *store.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	instinctID := req.GetInt("instinct_id", 0)
	if instinctID <= 0 {
		return mcp.NewToolResultError("instinct_id is required (positive integer)"), nil
	}

	adjustment := req.GetFloat("adjustment", 0)
	if adjustment == 0 {
		return mcp.NewToolResultError("adjustment is required (e.g., 0.1 or -0.2)"), nil
	}
	if adjustment < -1 || adjustment > 1 {
		return mcp.NewToolResultError("adjustment must be between -1.0 and 1.0"), nil
	}

	if err := st.UpdateInstinctConfidence(ctx, int64(instinctID), adjustment); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("update failed: %v", err)), nil
	}

	// Auto-prune if confidence dropped below 0.2.
	if adjustment < 0 {
		if _, err := st.PruneInstincts(ctx, 0.2); err != nil {
			// Best-effort; don't fail the request.
			_ = err
		}
	}

	return marshalResult(map[string]any{
		"action":      "instinct-feedback",
		"instinct_id": instinctID,
		"adjustment":  adjustment,
		"status":      "updated",
	})
}
