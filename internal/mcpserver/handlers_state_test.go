package mcpserver

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hir4ta/claude-alfred/internal/store"
)

func TestAssessRisk(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		health     float64
		alertCount int
		toolCount  int
		want       string
	}{
		{"healthy", 0.9, 0, 10, "low"},
		{"low health", 0.3, 0, 10, "high"},
		{"many alerts", 0.8, 3, 10, "high"},
		{"one alert", 0.8, 1, 10, "medium"},
		{"medium health", 0.6, 0, 10, "medium"},
		{"many tools", 0.9, 0, 60, "medium"},
		{"borderline low", 0.7, 0, 50, "low"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := assessRisk(tt.health, tt.alertCount, tt.toolCount)
			if got != tt.want {
				t.Errorf("assessRisk(%v, %d, %d) = %q, want %q",
					tt.health, tt.alertCount, tt.toolCount, got, tt.want)
			}
		})
	}
}

func TestDefaultWorkflow(t *testing.T) {
	t.Parallel()
	tests := []struct {
		taskType string
		wantLen  int
		first    string
	}{
		{"bugfix", 4, "read"},
		{"feature", 4, "read"},
		{"refactor", 4, "read"},
		{"test", 3, "read"},
		{"explore", 3, "read"},
		{"unknown", 3, "read"},
	}
	for _, tt := range tests {
		t.Run(tt.taskType, func(t *testing.T) {
			t.Parallel()
			got := defaultWorkflow(tt.taskType)
			if len(got) != tt.wantLen {
				t.Errorf("defaultWorkflow(%q) len = %d, want %d", tt.taskType, len(got), tt.wantLen)
			}
			if got[0] != tt.first {
				t.Errorf("defaultWorkflow(%q)[0] = %q, want %q", tt.taskType, got[0], tt.first)
			}
		})
	}
}

func TestClusterAdvice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		cluster  string
		taskType string
		wantSub  string
	}{
		{"conservative", "bugfix", "methodical"},
		{"aggressive", "bugfix", "Read phase"},
		{"aggressive", "refactor", "test suite"},
		{"aggressive", "feature", "exploration"},
		{"balanced", "feature", "balanced"},
	}
	for _, tt := range tests {
		t.Run(tt.cluster+"_"+tt.taskType, func(t *testing.T) {
			t.Parallel()
			got := clusterAdvice(tt.cluster, tt.taskType)
			if got == "" {
				t.Error("clusterAdvice() returned empty string")
			}
			if !strings.Contains(got, tt.wantSub) {
				t.Errorf("clusterAdvice(%q, %q) = %q, want substring %q",
					tt.cluster, tt.taskType, got, tt.wantSub)
			}
		})
	}
}

func TestShortID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"abcdefghij", "abcdefgh"},
		{"short", "short"},
		{"12345678", "12345678"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			if got := shortID(tt.input); got != tt.want {
				t.Errorf("shortID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTruncateRunes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		max   int
		want  string
	}{
		{"short", "hello", 10, "hello"},
		{"exact", "hello", 5, "hello"},
		{"truncated", "hello world", 5, "hello..."},
		{"empty", "", 5, ""},
		{"cjk", "日本語テスト", 3, "日本語..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := truncateRunes(tt.input, tt.max); got != tt.want {
				t.Errorf("truncateRunes(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.want)
			}
		})
	}
}

func TestEstimateTask_NilStore(t *testing.T) {
	t.Parallel()
	est, err := EstimateTask(nil, "", "bugfix")
	if err != nil {
		t.Fatalf("EstimateTask(nil) error = %v", err)
	}
	if est.TaskType != "bugfix" {
		t.Errorf("TaskType = %q, want %q", est.TaskType, "bugfix")
	}
	if est.SessionCount != 0 {
		t.Errorf("SessionCount = %d, want 0", est.SessionCount)
	}
}

func TestEstimateTask_WithWorkflows(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("store.Open() = %v", err)
	}
	defer st.Close()

	// Insert session rows first (FK constraint).
	for i := range 5 {
		sid := "sess-" + string(rune('a'+i))
		_ = st.UpsertSession(&store.SessionRow{ID: sid})
	}

	// Insert workflow data.
	for i := range 5 {
		_ = st.InsertWorkflowSequence(
			"sess-"+string(rune('a'+i)),
			"feature",
			[]string{"read", "write", "test"},
			true,
			20+i*5,
			300,
		)
	}

	est, err := EstimateTask(st, "", "feature")
	if err != nil {
		t.Fatalf("EstimateTask() error = %v", err)
	}
	if est.SessionCount == 0 {
		t.Error("SessionCount = 0, want > 0")
	}
	if est.MedianTools == 0 {
		t.Error("MedianTools = 0, want > 0")
	}
	if est.SuccessRate == 0 {
		t.Error("SuccessRate = 0, want > 0")
	}
}

func TestEstimateTask_NoMatchingType(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("store.Open() = %v", err)
	}
	defer st.Close()

	est, err := EstimateTask(st, "", "nonexistent")
	if err != nil {
		t.Fatalf("EstimateTask() error = %v", err)
	}
	if est.SessionCount != 0 {
		t.Errorf("SessionCount = %d, want 0 for nonexistent type", est.SessionCount)
	}
}

