package store

import (
	"context"
	"testing"
)

func TestInstinctCRUD(t *testing.T) {
	t.Parallel()
	st := openTestStore(t)
	ctx := context.Background()

	// Insert.
	inst := &Instinct{
		Trigger:     "when writing tests",
		Action:      "use table-driven tests with t.Parallel()",
		Confidence:  0.5,
		Domain:      DomainTesting,
		Scope:       ScopeProject,
		ProjectHash: "abc123",
	}
	id, err := st.InsertInstinct(ctx, inst)
	if err != nil {
		t.Fatalf("InsertInstinct: %v", err)
	}
	if id <= 0 {
		t.Fatalf("InsertInstinct returned id=%d, want >0", id)
	}

	// Search by project.
	results, err := st.SearchInstincts(ctx, "abc123", "", 10)
	if err != nil {
		t.Fatalf("SearchInstincts: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("SearchInstincts: got %d results, want 1", len(results))
	}
	if results[0].Trigger != "when writing tests" {
		t.Errorf("trigger = %q, want %q", results[0].Trigger, "when writing tests")
	}

	// Search with domain filter.
	results, err = st.SearchInstincts(ctx, "abc123", DomainCodeStyle, 10)
	if err != nil {
		t.Fatalf("SearchInstincts with domain: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("SearchInstincts with wrong domain: got %d, want 0", len(results))
	}

	// Search by different project should return empty.
	results, err = st.SearchInstincts(ctx, "other", "", 10)
	if err != nil {
		t.Fatalf("SearchInstincts other project: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("SearchInstincts other project: got %d, want 0", len(results))
	}
}

func TestInstinctConfidence(t *testing.T) {
	t.Parallel()
	st := openTestStore(t)
	ctx := context.Background()

	id, err := st.InsertInstinct(ctx, &Instinct{
		Trigger:     "test trigger",
		Action:      "test action",
		Confidence:  0.5,
		ProjectHash: "proj1",
	})
	if err != nil {
		t.Fatalf("InsertInstinct: %v", err)
	}

	// Positive adjustment.
	if err := st.UpdateInstinctConfidence(ctx, id, 0.2); err != nil {
		t.Fatalf("UpdateInstinctConfidence +0.2: %v", err)
	}
	results, _ := st.SearchInstincts(ctx, "proj1", "", 10)
	if len(results) != 1 || results[0].Confidence < 0.69 || results[0].Confidence > 0.71 {
		t.Errorf("confidence after +0.2: got %.2f, want ~0.70", results[0].Confidence)
	}

	// Clamp at 1.0.
	if err := st.UpdateInstinctConfidence(ctx, id, 0.5); err != nil {
		t.Fatalf("UpdateInstinctConfidence +0.5: %v", err)
	}
	results, _ = st.SearchInstincts(ctx, "proj1", "", 10)
	if results[0].Confidence != 1.0 {
		t.Errorf("confidence should clamp at 1.0, got %.2f", results[0].Confidence)
	}

	// Negative adjustment and clamp at 0.0.
	if err := st.UpdateInstinctConfidence(ctx, id, -1.5); err != nil {
		t.Fatalf("UpdateInstinctConfidence -1.5: %v", err)
	}
	results, _ = st.SearchInstincts(ctx, "proj1", "", 10)
	if results[0].Confidence != 0.0 {
		t.Errorf("confidence should clamp at 0.0, got %.2f", results[0].Confidence)
	}
}

func TestInstinctInsertClampsConfidence(t *testing.T) {
	t.Parallel()
	st := openTestStore(t)
	ctx := context.Background()

	// Confidence > 1 should be clamped.
	_, err := st.InsertInstinct(ctx, &Instinct{
		Trigger:     "high conf",
		Action:      "action",
		Confidence:  1.5,
		ProjectHash: "p1",
	})
	if err != nil {
		t.Fatalf("InsertInstinct: %v", err)
	}
	results, _ := st.SearchInstincts(ctx, "p1", "", 10)
	if results[0].Confidence != 1.0 {
		t.Errorf("confidence should be clamped to 1.0, got %.2f", results[0].Confidence)
	}
}

func TestInstinctPromotion(t *testing.T) {
	t.Parallel()
	st := openTestStore(t)
	ctx := context.Background()

	id, _ := st.InsertInstinct(ctx, &Instinct{
		Trigger:     "when committing",
		Action:      "run tests first",
		Confidence:  0.8,
		Scope:       ScopeProject,
		ProjectHash: "proj1",
	})

	if err := st.PromoteInstinct(ctx, id); err != nil {
		t.Fatalf("PromoteInstinct: %v", err)
	}

	// Should now be visible from other projects.
	results, _ := st.SearchInstincts(ctx, "other-project", "", 10)
	if len(results) != 1 {
		t.Fatalf("promoted instinct not visible from other project: got %d", len(results))
	}
	if results[0].Scope != ScopeGlobal {
		t.Errorf("scope = %q, want %q", results[0].Scope, ScopeGlobal)
	}
}

func TestInstinctPrune(t *testing.T) {
	t.Parallel()
	st := openTestStore(t)
	ctx := context.Background()

	st.InsertInstinct(ctx, &Instinct{Trigger: "low", Action: "a", Confidence: 0.1, ProjectHash: "p"})
	st.InsertInstinct(ctx, &Instinct{Trigger: "mid", Action: "b", Confidence: 0.5, ProjectHash: "p"})
	st.InsertInstinct(ctx, &Instinct{Trigger: "high", Action: "c", Confidence: 0.9, ProjectHash: "p"})

	deleted, err := st.PruneInstincts(ctx, 0.2)
	if err != nil {
		t.Fatalf("PruneInstincts: %v", err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}

	remaining, _ := st.SearchInstincts(ctx, "p", "", 10)
	if len(remaining) != 2 {
		t.Errorf("remaining = %d, want 2", len(remaining))
	}
}

func TestInstinctFTSSearch(t *testing.T) {
	t.Parallel()
	st := openTestStore(t)
	ctx := context.Background()

	st.InsertInstinct(ctx, &Instinct{
		Trigger:     "when writing Go tests",
		Action:      "use table-driven pattern",
		Confidence:  0.7,
		ProjectHash: "p1",
	})
	st.InsertInstinct(ctx, &Instinct{
		Trigger:     "when debugging",
		Action:      "check logs first",
		Confidence:  0.6,
		ProjectHash: "p1",
	})

	results, err := st.SearchInstinctsFTS(ctx, "tests table", "p1", 5)
	if err != nil {
		t.Fatalf("SearchInstinctsFTS: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("SearchInstinctsFTS: got %d, want 1", len(results))
	}
	if results[0].Trigger != "when writing Go tests" {
		t.Errorf("wrong instinct matched: %q", results[0].Trigger)
	}
}

func TestInstinctCrossProject(t *testing.T) {
	t.Parallel()
	st := openTestStore(t)
	ctx := context.Background()

	st.InsertInstinct(ctx, &Instinct{
		Trigger:     "when committing",
		Action:      "run tests",
		Confidence:  0.8,
		Scope:       ScopeProject,
		ProjectHash: "proj-a",
	})
	st.InsertInstinct(ctx, &Instinct{
		Trigger:     "when committing code",
		Action:      "run tests first",
		Confidence:  0.9,
		Scope:       ScopeProject,
		ProjectHash: "proj-b",
	})

	// Should find cross-project match from proj-a's perspective.
	matches, err := st.FindCrossProjectInstincts(ctx, "committing", "proj-a", 0.6)
	if err != nil {
		t.Fatalf("FindCrossProjectInstincts: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("cross-project: got %d matches, want 1", len(matches))
	}
	if matches[0].ProjectHash != "proj-b" {
		t.Errorf("cross-project match from wrong project: %q", matches[0].ProjectHash)
	}
}

func TestInstinctCount(t *testing.T) {
	t.Parallel()
	st := openTestStore(t)
	ctx := context.Background()

	n, _ := st.CountInstincts(ctx)
	if n != 0 {
		t.Errorf("initial count = %d, want 0", n)
	}

	st.InsertInstinct(ctx, &Instinct{Trigger: "t1", Action: "a1", ProjectHash: "p"})
	st.InsertInstinct(ctx, &Instinct{Trigger: "t2", Action: "a2", ProjectHash: "p"})

	n, _ = st.CountInstincts(ctx)
	if n != 2 {
		t.Errorf("count = %d, want 2", n)
	}
}
