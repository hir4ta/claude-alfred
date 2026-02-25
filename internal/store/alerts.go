package store

import "fmt"

// AlertRow represents a row in the alerts table.
type AlertRow struct {
	ID           int64
	SessionID    string
	PatternType  string
	Level        string
	Situation    string
	Observation  string
	Suggestion   string
	EventCount   int
	FirstEventID int64
	LastEventID  int64
	Timestamp    string
}

// InsertAlert inserts an alert row and returns its ID.
func (s *Store) InsertAlert(sessionID string, a AlertRow) (int64, error) {
	res, err := s.db.Exec(`
		INSERT INTO alerts (
			session_id, pattern_type, level,
			situation, observation, suggestion,
			event_count, first_event_id, last_event_id, timestamp
		) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		sessionID, a.PatternType, a.Level,
		a.Situation, a.Observation, a.Suggestion,
		a.EventCount, a.FirstEventID, a.LastEventID, a.Timestamp,
	)
	if err != nil {
		return 0, fmt.Errorf("store: insert alert: %w", err)
	}
	return res.LastInsertId()
}

// GetAlerts returns recent alerts for a session.
func (s *Store) GetAlerts(sessionID string, limit int) ([]AlertRow, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(`
		SELECT id, session_id, pattern_type, level,
			COALESCE(situation,''), COALESCE(observation,''), COALESCE(suggestion,''),
			event_count, COALESCE(first_event_id,0), COALESCE(last_event_id,0), timestamp
		FROM alerts
		WHERE session_id = ?
		ORDER BY id DESC
		LIMIT ?`, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("store: get alerts: %w", err)
	}
	defer rows.Close()

	var result []AlertRow
	for rows.Next() {
		var a AlertRow
		if err := rows.Scan(
			&a.ID, &a.SessionID, &a.PatternType, &a.Level,
			&a.Situation, &a.Observation, &a.Suggestion,
			&a.EventCount, &a.FirstEventID, &a.LastEventID, &a.Timestamp,
		); err != nil {
			continue
		}
		result = append(result, a)
	}
	return result, nil
}

// PatternFrequency represents how often an anti-pattern occurs in a project.
type PatternFrequency struct {
	PatternType string
	Count       int
	LastSeen    string
}

// GetAlertPatternFrequency returns anti-pattern frequency aggregated by project.
func (s *Store) GetAlertPatternFrequency(projectPath string) ([]PatternFrequency, error) {
	rows, err := s.db.Query(`
		SELECT a.pattern_type, COUNT(*) AS cnt, MAX(a.timestamp) AS last_seen
		FROM alerts a
		JOIN sessions s ON a.session_id = s.id
		WHERE s.project_path = ?
		GROUP BY a.pattern_type
		ORDER BY cnt DESC`, projectPath)
	if err != nil {
		return nil, fmt.Errorf("store: get alert pattern frequency: %w", err)
	}
	defer rows.Close()

	var result []PatternFrequency
	for rows.Next() {
		var pf PatternFrequency
		if err := rows.Scan(&pf.PatternType, &pf.Count, &pf.LastSeen); err != nil {
			continue
		}
		result = append(result, pf)
	}
	return result, nil
}

// InsertAlertEvents links alert IDs to event IDs via a junction table.
func (s *Store) InsertAlertEvents(alertID int64, eventIDs []int64) error {
	for _, eid := range eventIDs {
		if _, err := s.db.Exec(`
			INSERT INTO alert_events (alert_id, event_id) VALUES (?,?)`,
			alertID, eid,
		); err != nil {
			return fmt.Errorf("store: insert alert_event: %w", err)
		}
	}
	return nil
}
