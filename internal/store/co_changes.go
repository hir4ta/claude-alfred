package store

import "fmt"

// CoChange represents a pair of files frequently changed together.
type CoChange struct {
	FileA        string
	FileB        string
	SessionCount int
}

// RecordCoChanges records all pairwise co-changes from a set of files.
// Each pair is stored with file_a < file_b to avoid duplicates.
func (s *Store) RecordCoChanges(files []string) error {
	if len(files) < 2 {
		return nil
	}

	for i := 0; i < len(files); i++ {
		for j := i + 1; j < len(files); j++ {
			a, b := files[i], files[j]
			if a > b {
				a, b = b, a
			}
			_, err := s.db.Exec(
				`INSERT INTO file_co_changes (file_a, file_b, session_count, last_seen)
				 VALUES (?, ?, 1, datetime('now'))
				 ON CONFLICT(file_a, file_b) DO UPDATE SET
				     session_count = session_count + 1,
				     last_seen = datetime('now')`,
				a, b,
			)
			if err != nil {
				return fmt.Errorf("store: record co-change: %w", err)
			}
		}
	}
	return nil
}

// CoChangedFiles returns files most frequently changed together with the given file.
// Ordered by session_count descending.
func (s *Store) CoChangedFiles(filePath string, limit int) ([]CoChange, error) {
	rows, err := s.db.Query(
		`SELECT file_a, file_b, session_count FROM file_co_changes
		 WHERE file_a = ? OR file_b = ?
		 ORDER BY session_count DESC
		 LIMIT ?`,
		filePath, filePath, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("store: co-changed files: %w", err)
	}
	defer rows.Close()

	var results []CoChange
	for rows.Next() {
		var cc CoChange
		if err := rows.Scan(&cc.FileA, &cc.FileB, &cc.SessionCount); err != nil {
			continue
		}
		results = append(results, cc)
	}
	return results, rows.Err()
}

// TopCoChangePairs returns the most frequently co-changed file pairs globally.
// Ordered by session_count descending.
func (s *Store) TopCoChangePairs(limit int) ([]CoChange, error) {
	if limit <= 0 {
		limit = 3
	}
	rows, err := s.db.Query(
		`SELECT file_a, file_b, session_count FROM file_co_changes
		 ORDER BY session_count DESC
		 LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("store: top co-change pairs: %w", err)
	}
	defer rows.Close()

	var results []CoChange
	for rows.Next() {
		var cc CoChange
		if err := rows.Scan(&cc.FileA, &cc.FileB, &cc.SessionCount); err != nil {
			continue
		}
		results = append(results, cc)
	}
	return results, rows.Err()
}
