package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Instinct domain constants.
const (
	DomainGeneral     = "general"
	DomainCodeStyle   = "code-style"
	DomainTesting     = "testing"
	DomainGit         = "git"
	DomainDebugging   = "debugging"
	DomainWorkflow    = "workflow"
	DomainPreferences = "preferences"
)

// Instinct scope constants.
const (
	ScopeProject = "project"
	ScopeGlobal  = "global"
)

// Instinct represents a learned behavioral pattern.
type Instinct struct {
	ID            int64
	Trigger       string  // when this pattern applies
	Action        string  // what to do
	Confidence    float64 // 0.0-1.0
	Domain        string
	Scope         string // "project" or "global"
	ProjectHash   string
	SourceSession string
	Evidence      string
	TimesApplied  int
	CreatedAt     string
	UpdatedAt     string
}

// InsertInstinct adds a new instinct. Returns the row ID.
func (s *Store) InsertInstinct(ctx context.Context, inst *Instinct) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	if inst.Confidence < 0 {
		inst.Confidence = 0
	} else if inst.Confidence > 1 {
		inst.Confidence = 1
	}
	if inst.Domain == "" {
		inst.Domain = DomainGeneral
	}
	if inst.Scope == "" {
		inst.Scope = ScopeProject
	}

	result, err := s.db.ExecContext(ctx,
		`INSERT INTO instincts (trigger, action, confidence, domain, scope, project_hash,
		 source_session, evidence, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		inst.Trigger, inst.Action, inst.Confidence, inst.Domain, inst.Scope,
		inst.ProjectHash, inst.SourceSession, inst.Evidence, now, now,
	)
	if err != nil {
		return 0, fmt.Errorf("store: insert instinct: %w", err)
	}
	return result.LastInsertId()
}

// UpdateInstinctConfidence adjusts confidence by delta, clamped to [0, 1].
func (s *Store) UpdateInstinctConfidence(ctx context.Context, id int64, delta float64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE instincts SET
		 confidence = MAX(0.0, MIN(1.0, confidence + ?)),
		 updated_at = datetime('now')
		 WHERE id = ?`,
		delta, id,
	)
	if err != nil {
		return fmt.Errorf("store: update instinct confidence: %w", err)
	}
	return nil
}

// IncrementApplied bumps times_applied for an instinct.
func (s *Store) IncrementApplied(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE instincts SET times_applied = times_applied + 1, updated_at = datetime('now') WHERE id = ?`,
		id,
	)
	return err
}

// SearchInstincts returns instincts matching the given project (+ global), optionally filtered by domain.
func (s *Store) SearchInstincts(ctx context.Context, projectHash string, domain string, limit int) ([]Instinct, error) {
	if limit <= 0 {
		limit = 10
	}
	var args []any
	query := `SELECT id, trigger, action, confidence, domain, scope, project_hash,
	          source_session, evidence, times_applied, created_at, updated_at
	          FROM instincts WHERE (scope = 'global' OR project_hash = ?)`
	args = append(args, projectHash)

	if domain != "" {
		query += " AND domain = ?"
		args = append(args, domain)
	}
	query += " ORDER BY confidence DESC, times_applied DESC LIMIT ?"
	args = append(args, limit)

	return s.queryInstincts(ctx, query, args...)
}

// SearchInstinctsFTS searches instincts by keyword match in trigger+action text.
// Uses LIKE for simplicity (instincts table is small).
func (s *Store) SearchInstinctsFTS(ctx context.Context, keywords string, projectHash string, limit int) ([]Instinct, error) {
	if limit <= 0 {
		limit = 5
	}
	words := strings.Fields(strings.ToLower(keywords))
	if len(words) == 0 {
		return nil, nil
	}

	// Build OR conditions for each keyword against trigger and action.
	var conditions []string
	var args []any
	for _, w := range words {
		escaped := escapeLIKEContains(w)
		conditions = append(conditions, "(LOWER(trigger) LIKE ? ESCAPE '\\' OR LOWER(action) LIKE ? ESCAPE '\\')")
		args = append(args, escaped, escaped)
	}

	query := fmt.Sprintf(
		`SELECT id, trigger, action, confidence, domain, scope, project_hash,
		 source_session, evidence, times_applied, created_at, updated_at
		 FROM instincts
		 WHERE (scope = 'global' OR project_hash = ?)
		 AND confidence >= 0.3
		 AND (%s)
		 ORDER BY confidence DESC, times_applied DESC LIMIT ?`,
		strings.Join(conditions, " OR "),
	)
	args = append([]any{projectHash}, args...)
	args = append(args, limit)

	return s.queryInstincts(ctx, query, args...)
}

// FindDuplicateInstinct checks for an existing instinct with similar trigger+action.
func (s *Store) FindDuplicateInstinct(ctx context.Context, trigger, action, projectHash string) (*Instinct, error) {
	rows, err := s.queryInstincts(ctx,
		`SELECT id, trigger, action, confidence, domain, scope, project_hash,
		 source_session, evidence, times_applied, created_at, updated_at
		 FROM instincts
		 WHERE (project_hash = ? OR scope = 'global')
		 AND LOWER(trigger) LIKE ? ESCAPE '\'
		 AND LOWER(action) LIKE ? ESCAPE '\'
		 LIMIT 1`,
		projectHash,
		escapeLIKEContains(strings.ToLower(trigger)),
		escapeLIKEContains(strings.ToLower(action)),
	)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return &rows[0], nil
}

// FindCrossProjectInstincts finds instincts with similar triggers in other projects.
func (s *Store) FindCrossProjectInstincts(ctx context.Context, trigger string, excludeProject string, minConfidence float64) ([]Instinct, error) {
	return s.queryInstincts(ctx,
		`SELECT id, trigger, action, confidence, domain, scope, project_hash,
		 source_session, evidence, times_applied, created_at, updated_at
		 FROM instincts
		 WHERE scope = 'project' AND project_hash != ?
		 AND confidence >= ?
		 AND LOWER(trigger) LIKE ? ESCAPE '\'
		 LIMIT 10`,
		excludeProject, minConfidence, escapeLIKEContains(strings.ToLower(trigger)),
	)
}

// PromoteInstinct changes scope from "project" to "global".
func (s *Store) PromoteInstinct(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE instincts SET scope = 'global', updated_at = datetime('now') WHERE id = ?`,
		id,
	)
	return err
}

// PruneInstincts deletes instincts below the given confidence threshold.
// Returns the number of deleted rows.
func (s *Store) PruneInstincts(ctx context.Context, minConfidence float64) (int64, error) {
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM instincts WHERE confidence < ?`,
		minConfidence,
	)
	if err != nil {
		return 0, fmt.Errorf("store: prune instincts: %w", err)
	}
	return result.RowsAffected()
}

// ListInstinctsByProject returns all instincts for a project (+ global).
func (s *Store) ListInstinctsByProject(ctx context.Context, projectHash string) ([]Instinct, error) {
	return s.queryInstincts(ctx,
		`SELECT id, trigger, action, confidence, domain, scope, project_hash,
		 source_session, evidence, times_applied, created_at, updated_at
		 FROM instincts
		 WHERE scope = 'global' OR project_hash = ?
		 ORDER BY confidence DESC, updated_at DESC`,
		projectHash,
	)
}

// CountInstincts returns the total number of instincts.
func (s *Store) CountInstincts(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM instincts`).Scan(&n)
	return n, err
}

// queryInstincts is a shared scanner for instinct queries.
func (s *Store) queryInstincts(ctx context.Context, query string, args ...any) ([]Instinct, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query instincts: %w", err)
	}
	defer rows.Close()

	var result []Instinct
	for rows.Next() {
		var inst Instinct
		var sourceSession, evidence sql.NullString
		if err := rows.Scan(
			&inst.ID, &inst.Trigger, &inst.Action, &inst.Confidence,
			&inst.Domain, &inst.Scope, &inst.ProjectHash,
			&sourceSession, &evidence,
			&inst.TimesApplied, &inst.CreatedAt, &inst.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("store: scan instinct: %w", err)
		}
		inst.SourceSession = sourceSession.String
		inst.Evidence = evidence.String
		result = append(result, inst)
	}
	return result, rows.Err()
}
