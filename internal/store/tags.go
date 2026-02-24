package store

import "fmt"

// GetOrCreateTag returns the tag ID for the given name, creating it if necessary.
func (s *Store) GetOrCreateTag(name string) (int64, error) {
	var id int64
	err := s.db.QueryRow(`SELECT id FROM tags WHERE name = ?`, name).Scan(&id)
	if err == nil {
		return id, nil
	}
	res, err := s.db.Exec(`INSERT INTO tags (name) VALUES (?)`, name)
	if err != nil {
		return 0, fmt.Errorf("store: insert tag: %w", err)
	}
	return res.LastInsertId()
}

// LinkPatternTag creates a pattern_tags association.
func (s *Store) LinkPatternTag(patternID, tagID int64) error {
	_, err := s.db.Exec(`INSERT OR IGNORE INTO pattern_tags (pattern_id, tag_id) VALUES (?, ?)`, patternID, tagID)
	return err
}

// LinkPatternFile creates a pattern_files association.
func (s *Store) LinkPatternFile(patternID int64, filePath, role string) error {
	_, err := s.db.Exec(`INSERT OR IGNORE INTO pattern_files (pattern_id, file_path, role) VALUES (?, ?, ?)`, patternID, filePath, role)
	return err
}
