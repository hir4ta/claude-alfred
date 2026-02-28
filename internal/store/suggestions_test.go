package store

import (
	"path/filepath"
	"testing"
)

func TestPatternSavings(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	pattern := "code-quality"

	// Insert resolved outcomes with low tools_after (user acted quickly).
	for range 3 {
		id, err := s.InsertSuggestionOutcome("s1", pattern, "fix lint issue")
		if err != nil {
			t.Fatalf("InsertSuggestionOutcome: %v", err)
		}
		if err := s.ResolveSuggestion(id); err != nil {
			t.Fatalf("ResolveSuggestion: %v", err)
		}
		if err := s.UpdateToolsAfter(id, 2); err != nil {
			t.Fatalf("UpdateToolsAfter: %v", err)
		}
	}

	// Insert unresolved outcomes with high tools_after (user ignored).
	for range 3 {
		id, err := s.InsertSuggestionOutcome("s1", pattern, "fix lint issue")
		if err != nil {
			t.Fatalf("InsertSuggestionOutcome: %v", err)
		}
		if err := s.UpdateToolsAfter(id, 8); err != nil {
			t.Fatalf("UpdateToolsAfter: %v", err)
		}
	}

	saved, instances, err := s.PatternSavings(pattern)
	if err != nil {
		t.Fatalf("PatternSavings(%q) error: %v", pattern, err)
	}
	if instances < 2 {
		t.Errorf("PatternSavings(%q) instances = %d, want >= 2", pattern, instances)
	}
	// avgUnresolved=8, avgResolved=2 → saved=6
	if saved != 6 {
		t.Errorf("PatternSavings(%q) saved = %d, want 6", pattern, saved)
	}
}

func TestPatternSavings_InsufficientData(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	// No data at all.
	saved, _, err := s.PatternSavings("nonexistent")
	if err != nil {
		t.Fatalf("PatternSavings error: %v", err)
	}
	if saved != 0 {
		t.Errorf("PatternSavings(nonexistent) saved = %d, want 0", saved)
	}
}

func TestUpdateToolsAfter(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	id, err := s.InsertSuggestionOutcome("s1", "test-pattern", "suggestion")
	if err != nil {
		t.Fatalf("InsertSuggestionOutcome: %v", err)
	}

	if err := s.UpdateToolsAfter(id, 5); err != nil {
		t.Fatalf("UpdateToolsAfter: %v", err)
	}

	// Verify the value was stored.
	var toolsAfter int
	err = s.db.QueryRow(
		`SELECT tools_after FROM suggestion_outcomes WHERE id = ?`, id,
	).Scan(&toolsAfter)
	if err != nil {
		t.Fatalf("query tools_after: %v", err)
	}
	if toolsAfter != 5 {
		t.Errorf("tools_after = %d, want 5", toolsAfter)
	}
}
