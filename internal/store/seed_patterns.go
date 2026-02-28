package store

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"time"
)

//go:embed seed_data.json
var seedDataJSON []byte

type seedEntry struct {
	Type    string `json:"type"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

// SeedPatterns returns the built-in seed patterns for cold-start bootstrapping.
func SeedPatterns() []PatternRow {
	var entries []seedEntry
	if err := json.Unmarshal(seedDataJSON, &entries); err != nil {
		return nil
	}

	now := time.Now().UTC().Format(time.RFC3339)
	patterns := make([]PatternRow, 0, len(entries))
	for _, e := range entries {
		p := PatternRow{
			SessionID:   "seed-v1",
			PatternType: e.Type,
			Title:       e.Title,
			Content:     e.Content,
			EmbedText:   e.Title + " " + e.Content,
			Language:    "en",
			Scope:       "seed",
			Timestamp:   now,
			Tags:        []string{e.Type, "seed"},
		}
		patterns = append(patterns, p)
	}
	return patterns
}

// SeedIfEmpty inserts seed patterns when the patterns table is empty.
// This provides useful defaults for first-time users with no prior sessions.
func SeedIfEmpty(s *Store) error {
	var count int
	if err := s.db.QueryRow(`SELECT count(*) FROM patterns`).Scan(&count); err != nil {
		return fmt.Errorf("store: count patterns for seed check: %w", err)
	}
	if count > 0 {
		return nil
	}

	seeds := SeedPatterns()
	for i := range seeds {
		if _, err := s.InsertPattern(&seeds[i]); err != nil {
			return fmt.Errorf("store: insert seed pattern %q: %w", seeds[i].Title, err)
		}
	}
	return nil
}
