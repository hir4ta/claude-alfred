package hookhandler

import (
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hir4ta/claude-alfred/internal/sessiondb"
	"github.com/hir4ta/claude-alfred/internal/store"
)

func newTestRNG() *rand.Rand {
	return rand.New(rand.NewPCG(42, 0))
}

func TestBetaSample_StatisticalProperties(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		alpha   float64
		beta    float64
		wantMin float64
		wantMax float64
	}{
		{"high alpha", 10, 1, 0.8, 1.0},
		{"high beta", 1, 10, 0.0, 0.2},
		{"uniform prior", 1, 1, 0.35, 0.65},
		{"balanced", 5, 5, 0.35, 0.65},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rng := newTestRNG()
			var sum float64
			n := 1000
			for range n {
				sum += betaSample(rng, tt.alpha, tt.beta)
			}
			mean := sum / float64(n)
			if mean < tt.wantMin || mean > tt.wantMax {
				t.Errorf("betaSample(%v, %v) mean over %d draws = %v, want [%v, %v]",
					tt.alpha, tt.beta, n, mean, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestBetaSample_Exploration(t *testing.T) {
	t.Parallel()
	// A pattern with ~20% resolution rate should still sometimes produce
	// samples above 0.5 (exploration) but not too often.
	rng := newTestRNG()
	alpha := 5.0  // 4 resolved + 1 prior
	beta := 17.0  // 16 not resolved + 1 prior
	aboveHalf := 0
	n := 1000
	for range n {
		if betaSample(rng, alpha, beta) > 0.5 {
			aboveHalf++
		}
	}
	if aboveHalf == 0 {
		t.Error("Beta(5,17) never exceeded 0.5 in 1000 draws — no exploration")
	}
	if aboveHalf > 200 {
		t.Errorf("Beta(5,17) exceeded 0.5 in %d/1000 draws — too much exploration", aboveHalf)
	}
}

func TestBetaSample_EdgeCases(t *testing.T) {
	t.Parallel()
	rng := newTestRNG()

	// Zero/negative parameters should clamp to 1.
	s := betaSample(rng, 0, 0)
	if s < 0 || s > 1 {
		t.Errorf("betaSample(0, 0) = %v, want [0, 1]", s)
	}

	// Very small shape parameters (fractional, <1).
	s = betaSample(rng, 0.1, 0.1)
	if s < 0 || s > 1 {
		t.Errorf("betaSample(0.1, 0.1) = %v, want [0, 1]", s)
	}
}

func TestGammaSample_Positive(t *testing.T) {
	t.Parallel()
	rng := newTestRNG()
	for range 100 {
		g := gammaSample(rng, 2.0)
		if g < 0 {
			t.Fatalf("gammaSample returned negative: %v", g)
		}
	}
}

func TestSetDeliveryContext(t *testing.T) {
	// Not parallel — modifies package globals.
	sdb := openDeliveryTestDB(t)

	// Set task_type and velocity.
	_ = sdb.SetContext("task_type", "refactor")
	_ = sdb.SetContext("ewma_tool_velocity", "12.0")
	SetDeliveryContext(sdb)

	if ctxTaskType != "refactor" {
		t.Errorf("ctxTaskType = %q, want %q", ctxTaskType, "refactor")
	}
	if ctxVelocityState != "fast" {
		t.Errorf("ctxVelocityState = %q, want %q", ctxVelocityState, "fast")
	}

	// Low velocity → slow.
	_ = sdb.SetContext("ewma_tool_velocity", "1.0")
	SetDeliveryContext(sdb)
	if ctxVelocityState != "slow" {
		t.Errorf("ctxVelocityState = %q, want %q", ctxVelocityState, "slow")
	}

	// Normal velocity.
	_ = sdb.SetContext("ewma_tool_velocity", "5.0")
	SetDeliveryContext(sdb)
	if ctxVelocityState != "normal" {
		t.Errorf("ctxVelocityState = %q, want %q", ctxVelocityState, "normal")
	}
}

func openDeliveryTestDB(t *testing.T) *sessiondb.SessionDB {
	t.Helper()
	id := "test-delivery-" + strings.ReplaceAll(t.Name(), "/", "-")
	sdb, err := sessiondb.Open(id)
	if err != nil {
		t.Fatalf("sessiondb.Open(%q) = %v", id, err)
	}
	t.Cleanup(func() { _ = sdb.Destroy() })
	return sdb
}

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	os.Setenv("CLAUDE_ALFRED_DB", dbPath)
	t.Cleanup(func() { os.Unsetenv("CLAUDE_ALFRED_DB") })
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("store.Open() = %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestAdjustPriority_NoData(t *testing.T) {
	ctxTaskType = ""
	ctxUserCluster = ""
	st := openTestStore(t)
	rng := newTestRNG()
	result := adjustPriorityWithConfidence(rng, st, "unknown", PriorityMedium)
	if result.Priority != PriorityMedium {
		t.Errorf("priority = %d, want %d (unchanged)", result.Priority, PriorityMedium)
	}
	if result.Confidence != 0 {
		t.Errorf("confidence = %v, want 0 (no data)", result.Confidence)
	}
}

func TestAdjustPriority_NilStore(t *testing.T) {
	t.Parallel()
	rng := newTestRNG()
	result := adjustPriorityWithConfidence(rng, nil, "pattern", PriorityHigh)
	if result.Priority != PriorityHigh {
		t.Errorf("priority = %d, want %d (unchanged)", result.Priority, PriorityHigh)
	}
}

func TestAdjustPriority_HighResolution(t *testing.T) {
	ctxTaskType = ""
	ctxUserCluster = ""
	st := openTestStore(t)
	for range 18 {
		_ = st.UpsertUserPreference("great-pattern", true, 1.0)
	}
	for range 2 {
		_ = st.UpsertUserPreference("great-pattern", false, 0)
	}
	boosted := 0
	n := 100
	for i := range n {
		rng := rand.New(rand.NewPCG(uint64(i+1), 0))
		result := adjustPriorityWithConfidence(rng, st, "great-pattern", PriorityMedium)
		if result.Priority < PriorityMedium {
			boosted++
		}
	}
	if boosted < 50 {
		t.Errorf("boosted %d/%d times, want majority", boosted, n)
	}
}

func TestAdjustPriority_LowResolution(t *testing.T) {
	ctxTaskType = ""
	ctxUserCluster = ""
	st := openTestStore(t)
	for range 2 {
		_ = st.UpsertUserPreference("poor-pattern", true, 1.0)
	}
	for range 18 {
		_ = st.UpsertUserPreference("poor-pattern", false, 0)
	}
	demoted := 0
	n := 100
	for i := range n {
		rng := rand.New(rand.NewPCG(uint64(i+1), 0))
		result := adjustPriorityWithConfidence(rng, st, "poor-pattern", PriorityMedium)
		if result.Priority > PriorityMedium {
			demoted++
		}
	}
	if demoted < 50 {
		t.Errorf("demoted %d/%d times, want majority", demoted, n)
	}
}

func TestAdjustPriority_Confidence(t *testing.T) {
	ctxTaskType = ""
	ctxUserCluster = ""
	st := openTestStore(t)
	rng := newTestRNG()
	for range 3 {
		_ = st.UpsertUserPreference("few-obs", true, 1.0)
	}
	fewResult := adjustPriorityWithConfidence(rng, st, "few-obs", PriorityMedium)

	for range 50 {
		_ = st.UpsertUserPreference("many-obs", true, 1.0)
	}
	manyResult := adjustPriorityWithConfidence(rng, st, "many-obs", PriorityMedium)

	if manyResult.Confidence <= fewResult.Confidence {
		t.Errorf("many observations confidence (%v) should exceed few observations (%v)",
			manyResult.Confidence, fewResult.Confidence)
	}
}
