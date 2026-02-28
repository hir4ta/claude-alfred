package hookhandler

import (
	"fmt"
	"strings"
	"time"

	"github.com/hir4ta/claude-buddy/internal/sessiondb"
)

// AnomalyType classifies detected behavioral anomalies.
type AnomalyType string

const (
	AnomalyExploreSpiral AnomalyType = "explore_spiral"
	AnomalyDebugSpiral   AnomalyType = "debug_spiral"
	AnomalyNone          AnomalyType = ""
)

// phaseWindow is the number of recent phases to analyze.
const phaseWindow = 10

// detectAnomaly analyzes recent phase distribution to detect behavioral patterns.
// Returns AnomalyNone if the session is healthy.
func detectAnomaly(sdb *sessiondb.SessionDB) (AnomalyType, string) {
	// Use raw (non-collapsed) phase sequence for distribution analysis.
	phases, err := sdb.GetRawPhaseSequence(phaseWindow * 2)
	if err != nil || len(phases) < phaseWindow {
		return AnomalyNone, ""
	}

	// Analyze the most recent window.
	recent := phases
	if len(recent) > phaseWindow {
		recent = recent[len(recent)-phaseWindow:]
	}

	dist := phaseDist(recent)

	// Explore spiral: read phases exceed adaptive threshold.
	// Prefer EWMV-based control limit when sufficient data is available.
	readThreshold := adaptiveReadThreshold(sdb)
	if readPct := dist["read"]; readPct > readThreshold {
		return AnomalyExploreSpiral, fmt.Sprintf(
			"%.0f%% of recent actions are reads without edits — consider narrowing the search scope or starting implementation.",
			readPct*100)
	}

	// Debug spiral: high error rate + Edit→Bash loop.
	// Use EWMV-based error threshold when available.
	errThreshold := adaptiveErrorThreshold(sdb)
	errRate := getFloat(sdb, "ewma_error_rate")
	if errRate > errThreshold {
		editBashCount := countTransitions(recent, "write", "compile")
		editBashCount += countTransitions(recent, "write", "test")
		cycleThreshold := int(adaptiveThreshold("debug_edit_cycles", 2.0, 3.0))
		if editBashCount >= cycleThreshold {
			return AnomalyDebugSpiral, fmt.Sprintf(
				"Error rate %.0f%% with %d edit→run cycles — consider stepping back: read the error carefully, check a different approach, or ask for help.",
				errRate*100, editBashCount)
		}
	}

	return AnomalyNone, ""
}

// phaseDist returns the fraction of each phase in the slice.
func phaseDist(phases []string) map[string]float64 {
	counts := make(map[string]int)
	for _, p := range phases {
		counts[p]++
	}
	dist := make(map[string]float64)
	total := float64(len(phases))
	for k, v := range counts {
		dist[k] = float64(v) / total
	}
	return dist
}

// countTransitions counts A→B transitions in a phase sequence.
func countTransitions(phases []string, from, to string) int {
	count := 0
	for i := 1; i < len(phases); i++ {
		if phases[i-1] == from && phases[i] == to {
			count++
		}
	}
	return count
}

// adaptiveReadThreshold returns the explore-spiral read percentage threshold.
// Uses EWMV-based control limits when sufficient data exists, falling back
// to the Welford-based adaptive threshold.
func adaptiveReadThreshold(sdb *sessiondb.SessionDB) float64 {
	const fallback = 0.7
	var threshold float64
	if FlowEventCount(sdb) < minEWMVSamples {
		threshold = adaptiveThreshold("explore_read_pct", 2.0, fallback)
	} else {
		// Use the current read fraction's running mean as baseline.
		// A phase sequence with >70% reads is unusual; EWMV helps detect when
		// the session has naturally high read ratios (e.g., research tasks).
		threshold = adaptiveThreshold("explore_read_pct", 2.0, fallback)
	}
	// Clamp: threshold > 0.95 makes spiral detection impossible (dist maxes at 1.0).
	if threshold > 0.95 {
		threshold = 0.95
	}
	return threshold
}

// adaptiveErrorThreshold returns the debug-spiral error rate threshold.
// Prefers EWMV-based upper control limit (mean + k*sigma) when data is available.
func adaptiveErrorThreshold(sdb *sessiondb.SessionDB) float64 {
	const fallback = 0.3
	if FlowEventCount(sdb) < minEWMVSamples {
		return adaptiveThreshold("debug_error_rate", 2.0, fallback)
	}
	errMean := getFloat(sdb, "ewma_error_rate")
	errSigma := ErrorRateSigma(sdb)
	if errSigma < 0.01 {
		return adaptiveThreshold("debug_error_rate", 2.0, fallback)
	}
	// Upper control limit: mean + k*sigma. Clamp to [0.15, 0.6].
	ucl := errMean + ewmvK*errSigma
	if ucl < 0.15 {
		ucl = 0.15
	}
	if ucl > 0.6 {
		ucl = 0.6
	}
	return ucl
}

// checkAnomaly runs anomaly detection and delivers a suggestion if needed.
// Called from checkPeriodicHealth.
func checkAnomaly(sdb *sessiondb.SessionDB) string {
	anomaly, msg := detectAnomaly(sdb)
	if anomaly == AnomalyNone {
		return ""
	}

	key := "anomaly:" + string(anomaly)
	on, _ := sdb.IsOnCooldown(key)
	if on {
		return ""
	}
	_ = sdb.SetCooldown(key, 10*time.Minute)

	var b strings.Builder
	switch anomaly {
	case AnomalyExploreSpiral:
		fmt.Fprintf(&b, "[buddy] Explore spiral detected: %s", msg)
	case AnomalyDebugSpiral:
		fmt.Fprintf(&b, "[buddy] Debug spiral detected: %s", msg)
	}

	return b.String()
}
