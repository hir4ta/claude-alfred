package store

import "fmt"

// UserProfileMetric represents a tracked user behavior metric.
type UserProfileMetric struct {
	MetricName  string
	EWMAValue   float64
	SampleCount int
}

// GetUserProfile returns the EWMA value for a metric.
func (s *Store) GetUserProfile(metricName string) (float64, int, error) {
	var val float64
	var count int
	err := s.db.QueryRow(
		`SELECT ewma_value, sample_count FROM user_profile WHERE metric_name = ?`,
		metricName,
	).Scan(&val, &count)
	if err != nil {
		return 0, 0, fmt.Errorf("store: get user profile: %w", err)
	}
	return val, count, nil
}

// UpdateUserProfile updates a user profile metric using EWMA (alpha=0.3).
func (s *Store) UpdateUserProfile(metricName string, value float64) error {
	const alpha = 0.3
	_, err := s.db.Exec(
		`INSERT INTO user_profile (metric_name, ewma_value, sample_count, updated_at)
		 VALUES (?, ?, 1, datetime('now'))
		 ON CONFLICT(metric_name) DO UPDATE SET
		     ewma_value = ? * ? + (1 - ?) * ewma_value,
		     sample_count = sample_count + 1,
		     updated_at = datetime('now')`,
		metricName, value, alpha, value, alpha,
	)
	if err != nil {
		return fmt.Errorf("store: update user profile: %w", err)
	}
	return nil
}

// UserCluster classifies the user's coding style based on profile metrics.
// Returns one of: "conservative", "balanced", "aggressive".
// - conservative: high read_write_ratio (>3), high test_frequency (>0.7)
// - aggressive: low read_write_ratio (<1.5), low test_frequency (<0.3)
// - balanced: everything else
func (s *Store) UserCluster() string {
	rw, rwCount, _ := s.GetUserProfile("read_write_ratio")
	tf, tfCount, _ := s.GetUserProfile("test_frequency")

	// Need minimum data to classify.
	if rwCount < 3 && tfCount < 3 {
		return "balanced"
	}

	conservative := 0
	aggressive := 0

	if rwCount >= 3 {
		if rw > 3.0 {
			conservative++
		} else if rw < 1.5 {
			aggressive++
		}
	}
	if tfCount >= 3 {
		if tf > 0.7 {
			conservative++
		} else if tf < 0.3 {
			aggressive++
		}
	}

	if conservative >= 2 {
		return "conservative"
	}
	if aggressive >= 2 {
		return "aggressive"
	}
	return "balanced"
}

// AllUserProfile returns all user profile metrics.
func (s *Store) AllUserProfile() ([]UserProfileMetric, error) {
	rows, err := s.db.Query(
		`SELECT metric_name, ewma_value, sample_count FROM user_profile ORDER BY metric_name`,
	)
	if err != nil {
		return nil, fmt.Errorf("store: all user profile: %w", err)
	}
	defer rows.Close()

	var metrics []UserProfileMetric
	for rows.Next() {
		var m UserProfileMetric
		if err := rows.Scan(&m.MetricName, &m.EWMAValue, &m.SampleCount); err != nil {
			continue
		}
		metrics = append(metrics, m)
	}
	return metrics, rows.Err()
}
