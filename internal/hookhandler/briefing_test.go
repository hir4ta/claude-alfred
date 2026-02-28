package hookhandler

import (
	"strings"
	"testing"

	"github.com/hir4ta/claude-buddy/internal/sessiondb"
)

func openTestSDB(t *testing.T) *sessiondb.SessionDB {
	t.Helper()
	id := "test-" + strings.ReplaceAll(t.Name(), "/", "-")
	sdb, err := sessiondb.Open(id)
	if err != nil {
		t.Fatalf("sessiondb.Open(%q) = %v", id, err)
	}
	t.Cleanup(func() { _ = sdb.Destroy() })
	return sdb
}

func TestSelectTopSignal_Silent(t *testing.T) {
	t.Parallel()
	sdb := openTestSDB(t)

	sig := selectTopSignal(newPromptContext(sdb),"", "")
	if sig != nil {
		t.Errorf("selectTopSignal(empty) = %+v, want nil", sig)
	}
}

func TestSelectTopSignal_P0_Alert(t *testing.T) {
	t.Parallel()
	sdb := openTestSDB(t)

	// Insert an action-level detection.
	err := sdb.InsertDetection("retry_loop", "action", "Same tool called 3 times")
	if err != nil {
		t.Fatalf("InsertDetection = %v", err)
	}

	sig := selectTopSignal(newPromptContext(sdb),"", "")
	if sig == nil {
		t.Fatal("selectTopSignal = nil, want P0 alert")
	}
	if sig.Priority != 0 {
		t.Errorf("Priority = %d, want 0", sig.Priority)
	}
	if sig.Kind != "alert" {
		t.Errorf("Kind = %q, want %q", sig.Kind, "alert")
	}
	if sig.Detail != "Same tool called 3 times" {
		t.Errorf("Detail = %q, want %q", sig.Detail, "Same tool called 3 times")
	}
}

func TestSelectTopSignal_P0_Cooldown(t *testing.T) {
	t.Parallel()
	sdb := openTestSDB(t)

	err := sdb.InsertDetection("retry_loop", "action", "Same tool called 3 times")
	if err != nil {
		t.Fatalf("InsertDetection = %v", err)
	}

	// First call should return the alert.
	sig := selectTopSignal(newPromptContext(sdb),"", "")
	if sig == nil || sig.Priority != 0 {
		t.Fatal("first call should return P0 alert")
	}

	// Second call should be silent (cooldown active).
	sig = selectTopSignal(newPromptContext(sdb),"", "")
	if sig != nil {
		t.Errorf("second call = %+v, want nil (cooldown)", sig)
	}
}

func TestSelectTopSignal_P6_HighErrorRate(t *testing.T) {
	t.Parallel()
	sdb := openTestSDB(t)

	// Simulate high error rate.
	_ = sdb.SetContext("ewma_error_rate", "0.5")

	sig := selectTopSignal(newPromptContext(sdb),"", "")
	if sig == nil {
		t.Fatal("selectTopSignal = nil, want P6 health signal")
	}
	if sig.Priority != 6 {
		t.Errorf("Priority = %d, want 6", sig.Priority)
	}
	if sig.Kind != "health" {
		t.Errorf("Kind = %q, want %q", sig.Kind, "health")
	}
}

func TestSelectTopSignal_P0_Beats_P6(t *testing.T) {
	t.Parallel()
	sdb := openTestSDB(t)

	// Both P0 and P6 conditions present.
	_ = sdb.InsertDetection("retry_loop", "action", "Retry loop detected")
	_ = sdb.SetContext("ewma_error_rate", "0.5")

	sig := selectTopSignal(newPromptContext(sdb),"", "")
	if sig == nil {
		t.Fatal("selectTopSignal = nil, want signal")
	}
	// P0 should win over P6.
	if sig.Priority != 0 {
		t.Errorf("Priority = %d, want 0 (P0 beats P6)", sig.Priority)
	}
}

func TestFormatBriefing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		sig  *Signal
		want string
	}{
		{"nil", nil, ""},
		{"alert", &Signal{Priority: 0, Kind: "alert", Detail: "Retry detected"}, "[buddy:briefing] Retry detected"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatBriefing(tt.sig)
			if got != tt.want {
				t.Errorf("formatBriefing() = %q, want %q", got, tt.want)
			}
		})
	}
}
