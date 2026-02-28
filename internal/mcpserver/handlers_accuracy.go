package mcpserver

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/hir4ta/claude-buddy/internal/store"
)

// accuracyHandler returns accuracy metrics for suggestion and signal delivery.
func accuracyHandler(st *store.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		days := req.GetInt("days", 30)
		if days < 1 {
			days = 30
		}

		metrics, err := st.ComputeAccuracyMetrics(days)
		if err != nil {
			return mcp.NewToolResultError("accuracy metrics: " + err.Error()), nil
		}

		data, err := json.MarshalIndent(metrics, "", "  ")
		if err != nil {
			return mcp.NewToolResultError("marshal accuracy: " + err.Error()), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
