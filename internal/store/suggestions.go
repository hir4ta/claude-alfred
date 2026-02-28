package store

import (
	"fmt"
	"math"
	"strings"
	"time"
)

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

// UpdateToolsAfter records the number of tool calls between delivery and outcome.
func (s *Store) UpdateToolsAfter(id int64, toolsAfter int) error {
	_, err := s.db.Exec(
		`UPDATE suggestion_outcomes SET tools_after = ? WHERE id = ?`,
		toolsAfter, id,
	)
	if err != nil {
		return fmt.Errorf("store: update tools_after: %w", err)
	}
	return nil
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
// with a consistently very low resolution rate, as measured by decayed effectiveness.
// Uses a 5% threshold as a safety net; Thompson Sampling handles the 5-50% range.
func (s *Store) ShouldSuppressPattern(pattern string) bool {
	delivered, resolved, err := s.DecayedPatternEffectiveness(pattern)
	if err != nil || delivered < 15 {
		return false
	}
	rate := resolved / delivered
	return rate < 0.05
}

// PatternSavings estimates the tool count saved by acting on a pattern.
// Looks at sessions where the pattern was resolved vs ignored and compares
// subsequent tool counts within the same burst.
func (s *Store) PatternSavings(pattern string) (savedTools int, instances int, err error) {
	// Compare resolved vs unresolved outcomes: average tools-after for each group.
	row := s.db.QueryRow(`
		SELECT
			COALESCE(AVG(CASE WHEN resolved = 0 THEN tools_after END), 0),
			COALESCE(AVG(CASE WHEN resolved = 1 THEN tools_after END), 0),
			COUNT(CASE WHEN resolved = 1 THEN 1 END)
		FROM suggestion_outcomes
		WHERE pattern = ? AND tools_after > 0`, pattern)

	var avgUnresolved, avgResolved float64
	var resolvedCount int
	if err := row.Scan(&avgUnresolved, &avgResolved, &resolvedCount); err != nil {
		return 0, 0, fmt.Errorf("store: pattern savings: %w", err)
	}

	if resolvedCount < 2 || avgUnresolved <= avgResolved {
		return 0, resolvedCount, nil
	}

	return int(avgUnresolved - avgResolved), resolvedCount, nil
}

// ComputeSNR computes the signal-to-noise ratio from suggestion outcomes
// over the last N days. SNR = resolved_count / total_count.
// Returns the ratio, total sample size, and any error.
// Returns 0.0 with total=0 when no data is available.
func (s *Store) ComputeSNR(days int) (float64, int, error) {
	var total, resolved int
	err := s.db.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(resolved), 0)
		 FROM suggestion_outcomes
		 WHERE delivered_at > datetime('now', ? || ' days')`,
		fmt.Sprintf("-%d", days),
	).Scan(&total, &resolved)
	if err != nil {
		return 0, 0, fmt.Errorf("store: compute snr: %w", err)
	}
	if total == 0 {
		return 0, 0, nil
	}
	return float64(resolved) / float64(total), total, nil
}

// LowestPerformingPatterns returns patterns with the worst resolution rates
// over the last N days, requiring at least minDeliveries to be included.
// Results are ordered by resolution rate ascending (worst first).
func (s *Store) LowestPerformingPatterns(days, minDeliveries, limit int) ([]PatternSNR, error) {
	rows, err := s.db.Query(
		`SELECT pattern, COUNT(*) AS total, COALESCE(SUM(resolved), 0) AS res
		 FROM suggestion_outcomes
		 WHERE delivered_at > datetime('now', ? || ' days')
		 GROUP BY pattern
		 HAVING total >= ?
		 ORDER BY (CAST(res AS REAL) / total) ASC
		 LIMIT ?`,
		fmt.Sprintf("-%d", days), minDeliveries, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("store: lowest performing patterns: %w", err)
	}
	defer rows.Close()

	var results []PatternSNR
	for rows.Next() {
		var ps PatternSNR
		if err := rows.Scan(&ps.Pattern, &ps.Total, &ps.Resolved); err != nil {
			continue
		}
		if ps.Total > 0 {
			ps.Rate = float64(ps.Resolved) / float64(ps.Total)
		}
		results = append(results, ps)
	}
	return results, rows.Err()
}

// PatternSNR holds per-pattern signal-to-noise data.
type PatternSNR struct {
	Pattern  string
	Total    int
	Resolved int
	Rate     float64
}

// SNREntry holds a single SNR history data point.
type SNREntry struct {
	SessionID  string
	SNR        float64
	SampleSize int
	Eliminated string
	RecordedAt string
}

// InsertSNRHistory records a SNR measurement at session end.
func (s *Store) InsertSNRHistory(sessionID string, snr float64, sampleSize int, eliminated string) error {
	_, err := s.db.Exec(
		`INSERT INTO snr_history (session_id, snr_value, sample_size, eliminated) VALUES (?, ?, ?, ?)`,
		sessionID, snr, sampleSize, eliminated,
	)
	if err != nil {
		return fmt.Errorf("store: insert snr history: %w", err)
	}
	return nil
}

// RecentSNRTrend returns the most recent N SNR history entries, ordered newest first.
func (s *Store) RecentSNRTrend(limit int) ([]SNREntry, error) {
	rows, err := s.db.Query(
		`SELECT session_id, snr_value, sample_size, eliminated, recorded_at
		 FROM snr_history ORDER BY recorded_at DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("store: recent snr trend: %w", err)
	}
	defer rows.Close()

	var entries []SNREntry
	for rows.Next() {
		var e SNREntry
		if err := rows.Scan(&e.SessionID, &e.SNR, &e.SampleSize, &e.Eliminated, &e.RecordedAt); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// AutoEliminateNoisePatterns identifies patterns that have been the lowest-performing
// for minSessions consecutive sessions and suppresses them. Returns the list of
// newly suppressed pattern names.
func (s *Store) AutoEliminateNoisePatterns(targetSNR float64, minSessions int) ([]string, error) {
	snr, total, err := s.ComputeSNR(30)
	if err != nil || total < 10 || snr >= targetSNR {
		return nil, err
	}

	// Find patterns with consistently terrible performance (< 10% resolution).
	worst, err := s.LowestPerformingPatterns(30, 3, 10)
	if err != nil {
		return nil, err
	}

	// Check that these patterns have been bad for minSessions consecutive entries.
	trend, err := s.RecentSNRTrend(minSessions)
	if err != nil || len(trend) < minSessions {
		return nil, err
	}

	// Build set of previously eliminated patterns.
	eliminated := make(map[string]int)
	for _, entry := range trend {
		if entry.Eliminated != "" {
			for _, p := range splitCSV(entry.Eliminated) {
				eliminated[p]++
			}
		}
	}

	var suppressed []string
	for _, w := range worst {
		if w.Rate >= 0.10 {
			break // sorted by rate ascending; stop once above threshold
		}
		// Already counted as eliminated in recent sessions → candidate for suppression.
		if eliminated[w.Pattern] >= minSessions-1 || w.Rate < 0.05 {
			suppressed = append(suppressed, w.Pattern)
		}
	}
	return suppressed, nil
}

// splitCSV splits a comma-separated string into trimmed parts.
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	var parts []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

// DecayedPatternEffectiveness returns time-weighted delivery and resolution counts.
// Recent outcomes (last 30 days) count fully; older outcomes are exponentially
// decayed with a half-life of 30 days. Returns float64 counts to preserve decay precision.
func (s *Store) DecayedPatternEffectiveness(pattern string, halfLifeOverride ...time.Duration) (delivered, resolved float64, err error) {
	rows, err := s.db.Query(
		`SELECT resolved, delivered_at FROM suggestion_outcomes WHERE pattern = ?`,
		pattern,
	)
	if err != nil {
		return 0, 0, fmt.Errorf("store: decayed pattern effectiveness: %w", err)
	}
	defer rows.Close()

	now := time.Now()
	halfLife := 30 * 24 * time.Hour
	if len(halfLifeOverride) > 0 && halfLifeOverride[0] > 0 {
		halfLife = halfLifeOverride[0]
	}
	lambda := math.Ln2 / halfLife.Seconds()

	for rows.Next() {
		var resolvedInt int
		var deliveredAt string
		if err := rows.Scan(&resolvedInt, &deliveredAt); err != nil {
			continue
		}
		ts, err := time.Parse("2006-01-02 15:04:05", deliveredAt)
		if err != nil {
			continue
		}
		age := now.Sub(ts).Seconds()
		if age < 0 {
			age = 0
		}
		weight := math.Exp(-lambda * age)
		delivered += weight
		if resolvedInt == 1 {
			resolved += weight
		}
	}
	return delivered, resolved, rows.Err()
}

// PatternSaving holds per-pattern savings data for TotalSavings.
type PatternSaving struct {
	Pattern   string
	Saved     int
	Instances int
}

// TotalSavings aggregates cross-session cumulative tool savings across all patterns
// over the last N days. Only includes patterns with >= 3 resolved instances and
// a positive savings delta. Returns total saved tools, total instances, and top patterns.
func (s *Store) TotalSavings(days int) (totalSaved, totalInstances int, topPatterns []PatternSaving, err error) {
	rows, err := s.db.Query(`
		SELECT pattern,
			COALESCE(AVG(CASE WHEN resolved = 0 THEN tools_after END), 0) AS avg_unresolved,
			COALESCE(AVG(CASE WHEN resolved = 1 THEN tools_after END), 0) AS avg_resolved,
			COUNT(CASE WHEN resolved = 1 THEN 1 END) AS resolved_count
		FROM suggestion_outcomes
		WHERE tools_after > 0 AND delivered_at > datetime('now', ? || ' days')
		GROUP BY pattern
		HAVING resolved_count >= 3`,
		fmt.Sprintf("-%d", days),
	)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("store: total savings: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var pattern string
		var avgUnresolved, avgResolved float64
		var resolvedCount int
		if err := rows.Scan(&pattern, &avgUnresolved, &avgResolved, &resolvedCount); err != nil {
			continue
		}
		saved := int(avgUnresolved - avgResolved)
		if saved <= 0 {
			continue
		}
		topPatterns = append(topPatterns, PatternSaving{
			Pattern:   pattern,
			Saved:     saved,
			Instances: resolvedCount,
		})
		totalSaved += saved * resolvedCount
		totalInstances += resolvedCount
	}
	return totalSaved, totalInstances, topPatterns, rows.Err()
}
