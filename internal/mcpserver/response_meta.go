package mcpserver

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/hir4ta/claude-buddy/internal/store"
)

// ResponseMeta provides context about the quality and source of a response.
type ResponseMeta struct {
	Confidence   float64 `json:"confidence,omitempty"`    // 0-1
	Source       string  `json:"source,omitempty"`        // "session" | "project" | "global" | "seed"
	DataMaturity string  `json:"data_maturity,omitempty"` // "learning" | "growing" | "mature"
	SessionCount int     `json:"session_count,omitempty"`
}

// buildResponseMeta creates metadata based on current data maturity.
func buildResponseMeta(st *store.Store, source string) *ResponseMeta {
	meta := &ResponseMeta{
		Source: source,
	}

	if st == nil {
		meta.DataMaturity = "learning"
		return meta
	}

	sessionCount := 0
	if stats, err := st.GetProjectSessionStats(""); err == nil && stats != nil {
		sessionCount = stats.TotalSessions
	}
	meta.SessionCount = sessionCount

	patternCount, _ := st.CountPatterns()

	switch {
	case sessionCount < 3:
		meta.DataMaturity = "learning"
	case patternCount < 10:
		meta.DataMaturity = "growing"
	default:
		meta.DataMaturity = "mature"
	}

	return meta
}

// withMetaHandler wraps a handler to inject _meta into JSON responses.
func withMetaHandler(h server.ToolHandlerFunc, st *store.Store, source string) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, err := h(ctx, req)
		if err != nil || result == nil || result.IsError {
			return result, err
		}
		if len(result.Content) == 0 {
			return result, nil
		}

		tc, ok := result.Content[0].(mcp.TextContent)
		if !ok || tc.Text == "" {
			return result, nil
		}

		// Try to parse as JSON object.
		var obj map[string]any
		if jsonErr := json.Unmarshal([]byte(tc.Text), &obj); jsonErr != nil {
			return result, nil
		}

		obj["_meta"] = buildResponseMeta(st, source)

		data, jsonErr := json.MarshalIndent(obj, "", "  ")
		if jsonErr != nil {
			return result, nil
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}
