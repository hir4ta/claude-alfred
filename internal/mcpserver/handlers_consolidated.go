package mcpserver

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/hir4ta/claude-buddy/internal/embedder"
	"github.com/hir4ta/claude-buddy/internal/store"
)

// stateConsolidatedHandler consolidates stats + current_state + session_outlook.
// Routes based on the "detail" parameter: brief, standard (default), outlook.
func stateConsolidatedHandler(claudeHome string, st *store.Store) server.ToolHandlerFunc {
	statsFn := withMetaHandler(statsHandler(claudeHome), st, "session")
	currentStateFn := withMetaHandler(currentStateHandler(claudeHome), st, "session")
	outlookFn := withMetaHandler(sessionOutlookHandler(claudeHome, st), st, "session")

	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		detail := req.GetString("detail", "standard")
		switch detail {
		case "brief":
			return statsFn(ctx, req)
		case "outlook":
			return outlookFn(ctx, req)
		default:
			return currentStateFn(ctx, req)
		}
	}
}

// knowledgeConsolidatedHandler consolidates patterns + decisions + cross_project.
// Routes based on "scope" (project/global) and "type" parameters.
func knowledgeConsolidatedHandler(st *store.Store, emb *embedder.Embedder) server.ToolHandlerFunc {
	patternsFn := withMetaHandler(patternsHandler(st, emb), st, "project")
	decisionsFn := withMetaHandler(decisionsHandler(st), st, "project")
	crossProjectFn := withMetaHandler(crossProjectHandler(), st, "global")

	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		scope := req.GetString("scope", "project")
		patternType := req.GetString("type", "all")

		if scope == "global" {
			return crossProjectFn(ctx, req)
		}
		if patternType == "decision" {
			return decisionsFn(ctx, req)
		}
		return patternsFn(ctx, req)
	}
}

// guidanceConsolidatedHandler consolidates suggest + alerts + next_step + pending_nudges.
// Routes based on "focus" parameter: all (default), alerts, recommendations, next_steps, pending.
func guidanceConsolidatedHandler(claudeHome string, st *store.Store) server.ToolHandlerFunc {
	suggestFn := withMetaHandler(suggestHandler(claudeHome), st, "session")
	alertsFn := withMetaHandler(alertsHandler(claudeHome), st, "session")
	nextStepFn := withMetaHandler(nextStepHandler(claudeHome), st, "session")
	pendingFn := withMetaHandler(pendingNudgesHandler(st), st, "session")

	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		focus := req.GetString("focus", "all")
		switch focus {
		case "alerts":
			return alertsFn(ctx, req)
		case "next_steps":
			return nextStepFn(ctx, req)
		case "pending":
			return pendingFn(ctx, req)
		case "recommendations":
			return suggestFn(ctx, req)
		default:
			return suggestFn(ctx, req)
		}
	}
}

// planConsolidatedHandler consolidates estimate + task_progress + strategic_plan.
// Routes based on "mode" parameter: estimate, progress, strategy, all.
func planConsolidatedHandler(st *store.Store) server.ToolHandlerFunc {
	estimateFn := withMetaHandler(estimateHandler(st), st, "project")
	progressFn := withMetaHandler(taskProgressHandler(st), st, "project")
	strategyFn := withMetaHandler(strategicPlanHandler(st), st, "project")

	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		mode := req.GetString("mode", "estimate")
		switch mode {
		case "progress":
			return progressFn(ctx, req)
		case "strategy":
			return strategyFn(ctx, req)
		default:
			return estimateFn(ctx, req)
		}
	}
}

// diagnoseConsolidatedHandler consolidates diagnose + fix.
// If error_output is provided → diagnosis mode.
// If file_path + finding_rule/message → fix mode.
func diagnoseConsolidatedHandler(st *store.Store) server.ToolHandlerFunc {
	diagnoseFn := withMetaHandler(diagnoseHandler(st), st, "session")
	fixFn := withMetaHandler(fixHandler(), st, "project")

	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		errorOutput := req.GetString("error_output", "")
		findingRule := req.GetString("finding_rule", "")
		message := req.GetString("message", "")
		filePath := req.GetString("file_path", "")

		if errorOutput == "" && filePath != "" && (findingRule != "" || message != "") {
			return fixFn(ctx, req)
		}
		if errorOutput == "" {
			return mcp.NewToolResultError("error_output or (file_path + finding_rule) is required"), nil
		}
		return diagnoseFn(ctx, req)
	}
}
