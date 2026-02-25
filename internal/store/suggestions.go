package store

import "fmt"

// InsertSuggestionOutcome records a nudge delivery for effectiveness tracking.
func (s *Store) InsertSuggestionOutcome(sessionID, pattern, suggestion string) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO suggestion_outcomes (session_id, pattern, suggestion) VALUES (?, ?, ?)`,
		sessionID, pattern, suggestion,
	)
	if err != nil {
		return 0, fmt.Errorf("store: insert suggestion outcome: %w", err)
	}
	id, _ := res.LastInsertId()
	return id, nil
}

// ResolveSuggestion marks a suggestion outcome as resolved (acted upon).
func (s *Store) ResolveSuggestion(id int64) error {
	_, err := s.db.Exec(
		`UPDATE suggestion_outcomes SET resolved = 1 WHERE id = ? AND resolved = 0`, id,
	)
	if err != nil {
		return fmt.Errorf("store: resolve suggestion: %w", err)
	}
	return nil
}

// ResolveLastSuggestion marks the most recent unresolved outcome for a session+pattern as resolved.
func (s *Store) ResolveLastSuggestion(sessionID, pattern string) error {
	_, err := s.db.Exec(
		`UPDATE suggestion_outcomes SET resolved = 1
		 WHERE id = (
		     SELECT id FROM suggestion_outcomes
		     WHERE session_id = ? AND pattern = ? AND resolved = 0
		     ORDER BY id DESC LIMIT 1
		 )`,
		sessionID, pattern,
	)
	if err != nil {
		return fmt.Errorf("store: resolve last suggestion: %w", err)
	}
	return nil
}

// PatternEffectiveness returns delivery and resolution counts for a nudge pattern.
func (s *Store) PatternEffectiveness(pattern string) (delivered, resolved int, err error) {
	err = s.db.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(resolved), 0)
		 FROM suggestion_outcomes WHERE pattern = ?`, pattern,
	).Scan(&delivered, &resolved)
	if err != nil {
		return 0, 0, fmt.Errorf("store: pattern effectiveness: %w", err)
	}
	return delivered, resolved, nil
}

// ShouldSuppressPattern returns true if a pattern has been delivered enough times
// (20+) with a low resolution rate (<10%), indicating it's not useful.
func (s *Store) ShouldSuppressPattern(pattern string) bool {
	delivered, resolved, err := s.PatternEffectiveness(pattern)
	if err != nil || delivered < 20 {
		return false
	}
	rate := float64(resolved) / float64(delivered)
	return rate < 0.10
}
