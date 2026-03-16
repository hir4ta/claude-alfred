package mcpserver

import (
	"math"
	"testing"
	"time"

	"github.com/hir4ta/claude-alfred/internal/store"
)

func TestRecencyFactor(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		crawledAt  string
		sourceType string
		subType    string
		wantApprox float64
		tolerance  float64
	}{
		{
			name:       "memory general today",
			crawledAt:  "2026-03-10T12:00:00Z",
			sourceType: store.SourceMemory,
			subType:    store.SubTypeGeneral,
			wantApprox: 1.0,
			tolerance:  0.01,
		},
		{
			name:       "memory general 60 days ago (one half-life)",
			crawledAt:  "2026-01-09T12:00:00Z",
			sourceType: store.SourceMemory,
			subType:    store.SubTypeGeneral,
			wantApprox: 0.5, // exp(-ln2 * 60/60) = 0.5, exactly at floor
			tolerance:  0.02,
		},
		{
			name:       "memory general 120 days ago (two half-lives, clamped)",
			crawledAt:  "2025-11-10T12:00:00Z",
			sourceType: store.SourceMemory,
			subType:    store.SubTypeGeneral,
			wantApprox: 0.5, // exp(-ln2 * 120/60) = 0.25, clamped to floor 0.5
			tolerance:  0.01,
		},
		{
			name:       "memory rule 60 days ago (half of rule half-life=120)",
			crawledAt:  "2026-01-09T12:00:00Z",
			sourceType: store.SourceMemory,
			subType:    store.SubTypeRule,
			wantApprox: 0.707, // exp(-ln2 * 60/120) ≈ 0.707
			tolerance:  0.02,
		},
		{
			name:       "memory rule 120 days ago (one half-life)",
			crawledAt:  "2025-11-10T12:00:00Z",
			sourceType: store.SourceMemory,
			subType:    store.SubTypeRule,
			wantApprox: 0.5, // exp(-ln2 * 120/120) = 0.5
			tolerance:  0.02,
		},
		{
			name:       "memory assumption 30 days ago (one half-life)",
			crawledAt:  "2026-02-08T12:00:00Z",
			sourceType: store.SourceMemory,
			subType:    "assumption",
			wantApprox: 0.5,
			tolerance:  0.02,
		},
		{
			name:       "memory decision 90 days ago (one half-life)",
			crawledAt:  "2025-12-10T12:00:00Z",
			sourceType: store.SourceMemory,
			subType:    store.SubTypeDecision,
			wantApprox: 0.5,
			tolerance:  0.02,
		},
		{
			name:       "memory general 30 days ago (half of half-life)",
			crawledAt:  "2026-02-08T12:00:00Z",
			sourceType: store.SourceMemory,
			subType:    store.SubTypeGeneral,
			wantApprox: 0.707, // exp(-ln2 * 30/60) ≈ 0.707
			tolerance:  0.02,
		},
		{
			name:       "docs no decay",
			crawledAt:  "2024-01-01T00:00:00Z",
			sourceType: "records",
			subType:    "",
			wantApprox: 1.0,
			tolerance:  0.001,
		},
		{
			name:       "spec no decay",
			crawledAt:  "2024-06-15T00:00:00Z",
			sourceType: store.SourceSpec,
			subType:    "",
			wantApprox: 1.0,
			tolerance:  0.001,
		},
		{
			name:       "empty crawledAt",
			crawledAt:  "",
			sourceType: store.SourceMemory,
			subType:    store.SubTypeGeneral,
			wantApprox: 1.0,
			tolerance:  0.001,
		},
		{
			name:       "sqlite datetime format",
			crawledAt:  "2026-03-10 12:00:00",
			sourceType: store.SourceMemory,
			subType:    store.SubTypeGeneral,
			wantApprox: 1.0,
			tolerance:  0.01,
		},
		{
			name:       "future crawledAt",
			crawledAt:  "2026-03-11T12:00:00Z",
			sourceType: store.SourceMemory,
			subType:    store.SubTypeGeneral,
			wantApprox: 1.0,
			tolerance:  0.001,
		},
		{
			name:       "unknown sub_type falls back to general 60d",
			crawledAt:  "2026-01-09T12:00:00Z",
			sourceType: store.SourceMemory,
			subType:    "unknown",
			wantApprox: 0.5, // 60 days / 60-day half-life
			tolerance:  0.02,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := recencyFactor(tt.crawledAt, tt.sourceType, tt.subType, now)
			if math.Abs(got-tt.wantApprox) > tt.tolerance {
				t.Errorf("recencyFactor(%q, %q, %q) = %f, want ~%f (±%f)",
					tt.crawledAt, tt.sourceType, tt.subType, got, tt.wantApprox, tt.tolerance)
			}
		})
	}
}

func TestApplyRecencySignal_NoDecay(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	docs := []store.DocRow{
		{ID: 1, SourceType: "project", CrawledAt: "2024-01-01T00:00:00Z"},
		{ID: 2, SourceType: "project", CrawledAt: "2026-03-10T00:00:00Z"},
	}

	result := applyRecencySignal(docs, now)

	// No source_type needs recency → order preserved.
	if result[0].ID != 1 || result[1].ID != 2 {
		t.Errorf("expected order preserved for docs-only, got IDs %d, %d", result[0].ID, result[1].ID)
	}
}

func TestApplyRecencySignal_MemoryBoost(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)

	// 3 docs: #1 very old, #2 medium, #3 brand new.
	docs3 := []store.DocRow{
		{ID: 1, SourceType: store.SourceMemory, SubType: store.SubTypeGeneral, CrawledAt: "2025-01-01T00:00:00Z"}, // ~434 days, pos 1
		{ID: 2, SourceType: store.SourceMemory, SubType: store.SubTypeGeneral, CrawledAt: "2025-06-01T00:00:00Z"}, // ~282 days, pos 2
		{ID: 3, SourceType: store.SourceMemory, SubType: store.SubTypeGeneral, CrawledAt: "2026-03-10T11:00:00Z"}, // <1 day, pos 3
	}

	result3 := applyRecencySignal(docs3, now)

	// Doc 1: posScore=1.0, recency=0.5 (floored) → 0.50
	// Doc 2: posScore=0.5, recency=0.5 (floored) → 0.25
	// Doc 3: posScore=0.333, recency≈1.0 → 0.333
	// Order should be: 1, 3, 2 (fresh doc 3 overtakes old doc 2).
	if result3[0].ID != 1 {
		t.Errorf("expected doc 1 to stay first (position dominates), got ID %d", result3[0].ID)
	}
}

func TestApplyRecencySignal_SubTypeAware(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)

	// Rule with 60-day age: half-life=120d, so decay = exp(-ln2*60/120) ≈ 0.707
	// General with 60-day age: half-life=60d, so decay = 0.5 (at floor)
	docs := []store.DocRow{
		{ID: 1, SourceType: store.SourceMemory, SubType: store.SubTypeGeneral, CrawledAt: "2026-01-09T12:00:00Z"},
		{ID: 2, SourceType: store.SourceMemory, SubType: store.SubTypeRule, CrawledAt: "2026-01-09T12:00:00Z"},
	}

	result := applyRecencySignal(docs, now)

	// Doc 1: pos=1.0, recency=0.5, boost=1.0 → 0.5
	// Doc 2: pos=0.5, recency≈0.707, boost=2.0 → 0.707
	// Rule should overtake general due to slower decay + higher boost.
	if result[0].ID != 2 {
		t.Errorf("expected rule (ID=2) to rank first due to slower decay + boost, got ID %d", result[0].ID)
	}
}

func TestApplyRecencySignal_MixedSourceTypes(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	docs := []store.DocRow{
		{ID: 1, SourceType: "project", CrawledAt: "2024-01-01T00:00:00Z"},                                         // docs: no decay
		{ID: 2, SourceType: store.SourceMemory, SubType: store.SubTypeGeneral, CrawledAt: "2026-03-10T11:00:00Z"}, // memory: fresh
	}

	result := applyRecencySignal(docs, now)

	// Doc 1: posScore=1.0, recency=1.0 (docs) → 1.0
	// Doc 2: posScore=0.5, recency≈1.0 (fresh memory) → ≈0.5
	// Order: 1, 2
	if result[0].ID != 1 {
		t.Error("expected docs (no decay) to stay first")
	}
	if result[1].ID != 2 {
		t.Error("expected fresh memory at position 2")
	}
}

func TestApplyRecencySignal_SingleDoc(t *testing.T) {
	docs := []store.DocRow{
		{ID: 1, SourceType: store.SourceMemory, SubType: store.SubTypeGeneral, CrawledAt: "2025-01-01T00:00:00Z"},
	}
	result := applyRecencySignal(docs, time.Now())
	if len(result) != 1 || result[0].ID != 1 {
		t.Error("single doc should be returned unchanged")
	}
}

func TestApplyRecencySignal_Empty(t *testing.T) {
	result := applyRecencySignal(nil, time.Now())
	if result != nil {
		t.Error("nil input should return nil")
	}
}
