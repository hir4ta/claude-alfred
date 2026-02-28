package store

import "fmt"

// AccuracyMetrics holds computed accuracy statistics over a time window.
type AccuracyMetrics struct {
	// Suggestion-level metrics
	SNR       float64 `json:"snr"`       // (resolved + helpful) / total_delivered
	Precision float64 `json:"precision"` // resolved / (resolved + not_helpful)
	Recall    float64 `json:"recall"`    // resolved / (resolved + timeout)

	TotalDelivered int `json:"total_delivered"`
	TotalResolved  int `json:"total_resolved"`
	TotalTimeout   int `json:"total_timeout"`
	TotalNegative  int `json:"total_negative"` // explicit not_helpful

	// Signal-level metrics (JARVIS briefing)
	SignalAccuracy float64            `json:"signal_accuracy"` // overall acted_on rate
	ByPriority     map[int]float64    `json:"by_priority"`     // per-priority acted_on rate
	TopPatterns    []PatternAccuracy  `json:"top_patterns"`    // best-performing patterns
	WorstPatterns  []PatternAccuracy  `json:"worst_patterns"`  // worst-performing patterns
}

// PatternAccuracy holds accuracy data for a single pattern.
type PatternAccuracy struct {
	Pattern  string  `json:"pattern"`
	Delivery int     `json:"delivery"`
	Resolved int     `json:"resolved"`
	Rate     float64 `json:"rate"`
}

// ComputeAccuracyMetrics computes comprehensive accuracy statistics over the last N days.
func (s *Store) ComputeAccuracyMetrics(days int) (*AccuracyMetrics, error) {
	m := &AccuracyMetrics{
		ByPriority: make(map[int]float64),
	}

	dayStr := fmt.Sprintf("-%d", days)

	// Suggestion-level: total, resolved, feedback-based negatives
	err := s.db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(resolved), 0)
		FROM suggestion_outcomes
		WHERE delivered_at > datetime('now', ? || ' days')`, dayStr,
	).Scan(&m.TotalDelivered, &m.TotalResolved)
	if err != nil {
		return nil, fmt.Errorf("store: accuracy suggestion stats: %w", err)
	}

	// Timeout count: unresolved with tools_after >= 4
	_ = s.db.QueryRow(`
		SELECT COUNT(*)
		FROM suggestion_outcomes
		WHERE delivered_at > datetime('now', ? || ' days')
		  AND resolved = 0 AND tools_after >= 4`, dayStr,
	).Scan(&m.TotalTimeout)

	// Explicit not_helpful from feedbacks
	_ = s.db.QueryRow(`
		SELECT COUNT(*)
		FROM feedbacks
		WHERE created_at > datetime('now', ? || ' days')
		  AND rating = 'not_helpful' AND source = 'explicit'`, dayStr,
	).Scan(&m.TotalNegative)

	// Compute ratios
	if m.TotalDelivered > 0 {
		m.SNR = float64(m.TotalResolved) / float64(m.TotalDelivered)
	}
	denomPrecision := m.TotalResolved + m.TotalNegative
	if denomPrecision > 0 {
		m.Precision = float64(m.TotalResolved) / float64(denomPrecision)
	}
	denomRecall := m.TotalResolved + m.TotalTimeout
	if denomRecall > 0 {
		m.Recall = float64(m.TotalResolved) / float64(denomRecall)
	}

	// Signal-level accuracy
	priorityStats, overall, err := s.PriorityAccuracy(days)
	if err == nil {
		m.SignalAccuracy = overall
		for p, ps := range priorityStats {
			m.ByPriority[p] = ps.Rate
		}
	}

	// Top and worst patterns
	m.TopPatterns = s.topPatternsByRate(dayStr, 5, true)
	m.WorstPatterns = s.topPatternsByRate(dayStr, 5, false)

	return m, nil
}

// topPatternsByRate returns the N best or worst patterns by resolution rate.
func (s *Store) topPatternsByRate(dayStr string, limit int, best bool) []PatternAccuracy {
	order := "DESC"
	if !best {
		order = "ASC"
	}
	rows, err := s.db.Query(fmt.Sprintf(`
		SELECT pattern, COUNT(*) AS total, COALESCE(SUM(resolved), 0) AS res
		FROM suggestion_outcomes
		WHERE delivered_at > datetime('now', ? || ' days')
		GROUP BY pattern
		HAVING total >= 3
		ORDER BY (CAST(res AS REAL) / total) %s
		LIMIT ?`, order),
		dayStr, limit,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var results []PatternAccuracy
	for rows.Next() {
		var pa PatternAccuracy
		if err := rows.Scan(&pa.Pattern, &pa.Delivery, &pa.Resolved); err != nil {
			continue
		}
		if pa.Delivery > 0 {
			pa.Rate = float64(pa.Resolved) / float64(pa.Delivery)
		}
		results = append(results, pa)
	}
	return results
}

// UpdateUserPatternEffectiveness updates per-project individual pattern tracking.
func (s *Store) UpdateUserPatternEffectiveness(projectPath, pattern, taskType string, resolved bool) error {
	col := "not_resolved"
	if resolved {
		col = "resolved"
	}
	_, err := s.db.Exec(fmt.Sprintf(`
		INSERT INTO user_pattern_effectiveness (project_path, pattern, task_type, %s, last_updated)
		VALUES (?, ?, ?, 1, datetime('now'))
		ON CONFLICT (project_path, pattern, task_type) DO UPDATE SET
			%s = %s + 1,
			last_updated = datetime('now')`, col, col, col),
		projectPath, pattern, taskType,
	)
	if err != nil {
		return fmt.Errorf("store: update user pattern effectiveness: %w", err)
	}
	return nil
}

// UpdateUserPatternExplicitFeedback records explicit helpful/not_helpful feedback.
func (s *Store) UpdateUserPatternExplicitFeedback(projectPath, pattern, taskType string, helpful bool) error {
	col := "explicit_not_helpful"
	if helpful {
		col = "explicit_helpful"
	}
	_, err := s.db.Exec(fmt.Sprintf(`
		INSERT INTO user_pattern_effectiveness (project_path, pattern, task_type, %s, last_updated)
		VALUES (?, ?, ?, 1, datetime('now'))
		ON CONFLICT (project_path, pattern, task_type) DO UPDATE SET
			%s = %s + 1,
			last_updated = datetime('now')`, col, col, col),
		projectPath, pattern, taskType,
	)
	if err != nil {
		return fmt.Errorf("store: update user pattern explicit feedback: %w", err)
	}
	return nil
}

// UserPatternStats holds per-user pattern effectiveness data.
type UserPatternStats struct {
	Resolved     int
	NotResolved  int
	Helpful      int
	NotHelpful   int
	SampleSize   int
	Rate         float64
}

// GetUserPatternEffectiveness returns effectiveness stats for a pattern at a specific project.
func (s *Store) GetUserPatternEffectiveness(projectPath, pattern, taskType string) (*UserPatternStats, error) {
	var stats UserPatternStats
	err := s.db.QueryRow(`
		SELECT resolved, not_resolved, explicit_helpful, explicit_not_helpful
		FROM user_pattern_effectiveness
		WHERE project_path = ? AND pattern = ? AND task_type = ?`,
		projectPath, pattern, taskType,
	).Scan(&stats.Resolved, &stats.NotResolved, &stats.Helpful, &stats.NotHelpful)
	if err != nil {
		return nil, err
	}
	stats.SampleSize = stats.Resolved + stats.NotResolved
	if stats.SampleSize > 0 {
		stats.Rate = float64(stats.Resolved) / float64(stats.SampleSize)
	}
	return &stats, nil
}
