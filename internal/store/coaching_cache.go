package store

import "time"

// GetCachedCoaching returns cached coaching text for the given context.
// Returns "", false if no cache entry exists or it's older than maxAge.
func (s *Store) GetCachedCoaching(project, taskType, domain string, maxAge time.Duration) (string, bool) {
	if domain == "" {
		domain = "general"
	}
	var text string
	var createdAt string
	err := s.db.QueryRow(`
		SELECT coaching_text, created_at FROM coaching_cache
		WHERE project = ? AND task_type = ? AND domain = ?`,
		project, taskType, domain,
	).Scan(&text, &createdAt)
	if err != nil {
		return "", false
	}

	// Check age.
	t, err := time.Parse("2006-01-02 15:04:05", createdAt)
	if err == nil && time.Since(t) > maxAge {
		return "", false
	}
	return text, true
}

// SetCachedCoaching stores coaching text for the given context.
func (s *Store) SetCachedCoaching(project, taskType, domain, text, contextSummary string) error {
	if domain == "" {
		domain = "general"
	}
	_, err := s.db.Exec(`
		INSERT INTO coaching_cache (project, task_type, domain, coaching_text, context_summary)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (project, task_type, domain)
		DO UPDATE SET coaching_text = excluded.coaching_text,
		              context_summary = excluded.context_summary,
		              created_at = CURRENT_TIMESTAMP`,
		project, taskType, domain, text, contextSummary,
	)
	return err
}
