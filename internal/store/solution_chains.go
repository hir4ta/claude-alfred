package store

import (
	"fmt"
	"time"
)

// SolutionChain represents a multi-step failure→resolution playbook.
type SolutionChain struct {
	ID           int
	SessionID    string
	FailureSig   string
	ToolSequence string // JSON array of tool names
	Outcome      string
	StepCount    int
	TimesReplayed int
	Timestamp    time.Time
}

// InsertSolutionChain records a complete failure→resolution tool sequence.
func (s *Store) InsertSolutionChain(sessionID, failureSig, toolSequence string, stepCount int) error {
	_, err := s.db.Exec(
		`INSERT INTO solution_chains (session_id, failure_sig, tool_sequence, step_count)
		 VALUES (?, ?, ?, ?)`,
		sessionID, failureSig, toolSequence, stepCount,
	)
	if err != nil {
		return fmt.Errorf("store: insert solution chain: %w", err)
	}
	return nil
}

// SearchSolutionChains finds resolution playbooks for a given failure signature.
// Ordered by times_replayed (most proven first), then recency.
func (s *Store) SearchSolutionChains(failureSig string, limit int) ([]SolutionChain, error) {
	rows, err := s.db.Query(
		`SELECT id, session_id, failure_sig, tool_sequence, outcome, step_count, times_replayed, timestamp
		 FROM solution_chains
		 WHERE failure_sig = ?
		 ORDER BY times_replayed DESC, timestamp DESC
		 LIMIT ?`,
		failureSig, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("store: search solution chains: %w", err)
	}
	defer rows.Close()

	var results []SolutionChain
	for rows.Next() {
		var sc SolutionChain
		var ts string
		if err := rows.Scan(&sc.ID, &sc.SessionID, &sc.FailureSig, &sc.ToolSequence,
			&sc.Outcome, &sc.StepCount, &sc.TimesReplayed, &ts); err != nil {
			continue
		}
		sc.Timestamp, _ = time.Parse("2006-01-02 15:04:05", ts)
		results = append(results, sc)
	}
	return results, rows.Err()
}

// IncrementChainReplayed increments the replay counter for a solution chain.
func (s *Store) IncrementChainReplayed(chainID int) error {
	_, err := s.db.Exec(
		`UPDATE solution_chains SET times_replayed = times_replayed + 1 WHERE id = ?`,
		chainID,
	)
	return err
}
