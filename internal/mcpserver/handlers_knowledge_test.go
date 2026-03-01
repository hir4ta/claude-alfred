package mcpserver

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/hir4ta/claude-alfred/internal/store"
)

func openKnowledgeTestStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("store.Open() = %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestDecisionsHandler_Empty(t *testing.T) {
	t.Parallel()
	st := openKnowledgeTestStore(t)
	handler := decisionsHandler(st)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"query": "test",
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("decisionsHandler() error = %v", err)
	}
	if result.IsError {
		t.Error("decisionsHandler() returned error for valid empty query")
	}
}

func TestDecisionsHandler_WithData(t *testing.T) {
	t.Parallel()
	st := openKnowledgeTestStore(t)

	// Insert test decisions.
	for i := 0; i < 3; i++ {
		_ = st.InsertDecision(&store.DecisionRow{
			SessionID:    "test-sess",
			Timestamp:    time.Now().Format(time.RFC3339),
			Topic:        "architecture",
			DecisionText: "Use SQLite for storage",
			Reasoning:    "Simple, embedded, fast",
			FilePaths:    `["store.go"]`,
		})
	}

	handler := decisionsHandler(st)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"project": "",
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("decisionsHandler() error = %v", err)
	}
	if result.IsError {
		t.Errorf("decisionsHandler() returned error: %v", result.Content)
	}
}

func TestDecisionsHandler_FTSSearch(t *testing.T) {
	t.Parallel()
	st := openKnowledgeTestStore(t)

	_ = st.InsertDecision(&store.DecisionRow{
		SessionID:    "test-sess",
		Timestamp:    time.Now().Format(time.RFC3339),
		Topic:        "database",
		DecisionText: "Use SQLite for vector storage",
	})
	_ = st.InsertDecision(&store.DecisionRow{
		SessionID:    "test-sess",
		Timestamp:    time.Now().Format(time.RFC3339),
		Topic:        "api",
		DecisionText: "REST API design pattern",
	})

	handler := decisionsHandler(st)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"query": "SQLite vector",
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("decisionsHandler() error = %v", err)
	}
	if result.IsError {
		t.Errorf("decisionsHandler() returned error for FTS query")
	}
}

func TestDecisionsHandler_NilStore(t *testing.T) {
	t.Parallel()
	handler := decisionsHandler(nil)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("decisionsHandler(nil) error = %v", err)
	}
	if !result.IsError {
		t.Error("decisionsHandler(nil) should return error result")
	}
}

func TestKnowledgeConsolidatedHandler_Routing(t *testing.T) {
	t.Parallel()
	st := openKnowledgeTestStore(t)

	// Test routing to decisions handler (scope=project, type=decision).
	_ = st.InsertDecision(&store.DecisionRow{
		SessionID:    "test-routing",
		Timestamp:    time.Now().Format(time.RFC3339),
		Topic:        "routing test",
		DecisionText: "test decision",
	})

	handler := knowledgeConsolidatedHandler(st, nil)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"query": "routing",
		"type":  "decision",
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("knowledgeConsolidatedHandler(decision) error = %v", err)
	}
	if result.IsError {
		t.Error("knowledgeConsolidatedHandler(decision) returned error")
	}
}

func TestRecallHandler_NilStore(t *testing.T) {
	t.Parallel()
	handler := recallHandler(nil)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"query": "test",
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("recallHandler(nil) error = %v", err)
	}
	if !result.IsError {
		t.Error("recallHandler(nil) should return error")
	}
}

func TestRecallHandler_MissingQuery(t *testing.T) {
	t.Parallel()
	st := openKnowledgeTestStore(t)
	handler := recallHandler(st)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("recallHandler() error = %v", err)
	}
	if !result.IsError {
		t.Error("recallHandler() without query should return error")
	}
}
