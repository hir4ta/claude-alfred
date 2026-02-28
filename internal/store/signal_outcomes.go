package store

import (
	"crypto/sha256"
	"fmt"
)

// PriorityStats holds accuracy metrics for a single priority level.
type PriorityStats struct {
	Delivered int
	ActedOn   int
	Rate      float64
}

// InsertSignalOutcome records a briefing signal delivery for accuracy tracking.
// Returns the row ID for later resolution marking.
func (s *Store) InsertSignalOutcome(sessionID string, priority int, kind, detail string) (int64, error) {
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(detail)))[:16]
	res, err := s.db.Exec(
		`INSERT INTO signal_outcomes (session_id, priority, kind, detail_hash) VALUES (?, ?, ?, ?)`,
		sessionID, priority, kind, hash,
	)
	if err != nil {
		return 0, fmt.Errorf("store: insert signal outcome: %w", err)
	}
	id, _ := res.LastInsertId()
	return id, nil
}

// ResolveSignalOutcome marks a signal outcome as acted upon.
func (s *Store) ResolveSignalOutcome(id int64) error {
	_, err := s.db.Exec(
		`UPDATE signal_outcomes SET acted_on = 1 WHERE id = ? AND acted_on = 0`, id,
	)
	if err != nil {
		return fmt.Errorf("store: resolve signal outcome: %w", err)
	}
	return nil
}

// PriorityAccuracy computes per-priority and overall accuracy over the last N days.
// Accuracy = acted_on / delivered for each priority level.
func (s *Store) PriorityAccuracy(days int) (map[int]PriorityStats, float64, error) {
	rows, err := s.db.Query(
		`SELECT priority, COUNT(*) AS total, COALESCE(SUM(acted_on), 0) AS acted
		 FROM signal_outcomes
		 WHERE delivered_at > datetime('now', ? || ' days')
		 GROUP BY priority
		 ORDER BY priority`,
		fmt.Sprintf("-%d", days),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("store: priority accuracy: %w", err)
	}
	defer rows.Close()

	stats := make(map[int]PriorityStats)
	totalAll, actedAll := 0, 0
	for rows.Next() {
		var priority, total, acted int
		if err := rows.Scan(&priority, &total, &acted); err != nil {
			continue
		}
		rate := 0.0
		if total > 0 {
			rate = float64(acted) / float64(total)
		}
		stats[priority] = PriorityStats{
			Delivered: total,
			ActedOn:   acted,
			Rate:      rate,
		}
		totalAll += total
		actedAll += acted
	}

	overall := 0.0
	if totalAll > 0 {
		overall = float64(actedAll) / float64(totalAll)
	}
	return stats, overall, rows.Err()
}
