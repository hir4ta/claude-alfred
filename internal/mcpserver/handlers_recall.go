package mcpserver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/hir4ta/claude-alfred/internal/embedder"
	"github.com/hir4ta/claude-alfred/internal/store"
)

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
			return recallSave(ctx, st, req)
		default:
			return mcp.NewToolResultError(fmt.Sprintf("unknown action %q: use 'search' or 'save'", action)), nil
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
	limit := req.GetInt("limit", 10)
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	var docs []store.DocRow
	searchMethod := "fts5_only"
	var warnings []string

	// Try hybrid search if embedder is available.
	if emb != nil {
		queryVec, embedErr := emb.EmbedForSearch(ctx, query)
		if embedErr != nil {
			warnings = append(warnings, fmt.Sprintf("vector embedding failed, using FTS-only: %v", embedErr))
		} else if queryVec != nil {
			overRetrieve := limit * 4
			if overRetrieve < 20 {
				overRetrieve = 20
			}
			matches, hybridErr := st.HybridSearch(ctx, queryVec, query, store.SourceMemory, overRetrieve, overRetrieve)
			if hybridErr != nil {
				warnings = append(warnings, fmt.Sprintf("hybrid search degraded: %v", hybridErr))
			} else if len(matches) > 0 {
				ids := make([]int64, len(matches))
				for i, m := range matches {
					ids[i] = m.DocID
				}
				fetchedDocs, fetchErr := st.GetDocsByIDs(ctx, ids)
				if fetchErr != nil {
					warnings = append(warnings, fmt.Sprintf("doc fetch failed, using FTS-only: %v", fetchErr))
				} else {
					// Preserve RRF ordering.
					docMap := make(map[int64]store.DocRow, len(fetchedDocs))
					for _, d := range fetchedDocs {
						docMap[d.ID] = d
					}
					ordered := make([]store.DocRow, 0, len(ids))
					for _, id := range ids {
						if d, ok := docMap[id]; ok {
							ordered = append(ordered, d)
						}
					}
					docs = ordered
					searchMethod = "hybrid_rrf"

					// Rerank if we have enough candidates.
					if len(docs) > limit {
						contents := make([]string, len(docs))
						for i, d := range docs {
							contents[i] = d.SectionPath + "\n" + d.Content
						}
						reranked, rerankErr := emb.Rerank(ctx, query, contents, limit)
						if rerankErr != nil {
							warnings = append(warnings, fmt.Sprintf("rerank failed, using RRF order: %v", rerankErr))
						} else if len(reranked) > 0 {
							reorderedDocs := make([]store.DocRow, 0, len(reranked))
							for _, r := range reranked {
								if r.Index >= 0 && r.Index < len(docs) {
									reorderedDocs = append(reorderedDocs, docs[r.Index])
								}
							}
							docs = reorderedDocs
							searchMethod = "hybrid_rrf+rerank"
						}
					}
				}
			}
		}
	}

	// Fallback to FTS-only.
	if len(docs) == 0 {
		searchMethod = "fts5_only"
		var err error
		docs, err = st.SearchDocsFTS(ctx, query, store.SourceMemory, limit)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("memory search failed: %v", err)), nil
		}
	}

	// Trim to limit.
	if len(docs) > limit {
		docs = docs[:limit]
	}

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
	if len(warnings) > 0 {
		result["warning"] = strings.Join(warnings, "; ")
	}
	return marshalResult(result)
}

// recallSave saves a new memory entry to the knowledge base.
func recallSave(ctx context.Context, st *store.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	content := req.GetString("content", "")
	if content == "" {
		return mcp.NewToolResultError("content parameter is required for save"), nil
	}
	label := req.GetString("label", "")
	if label == "" {
		return mcp.NewToolResultError("label parameter is required for save (short description)"), nil
	}
	project := req.GetString("project", "general")

	date := time.Now().Format("2006-01-02")
	url := fmt.Sprintf("memory://user/%s/manual/%s", project, date)
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

	return marshalResult(map[string]any{
		"status":       status,
		"id":           id,
		"section_path": sectionPath,
		"url":          url,
	})
}
