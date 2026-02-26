package hookhandler

import (
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"strconv"
	"time"

	"github.com/hir4ta/claude-buddy/internal/sessiondb"
	"github.com/hir4ta/claude-buddy/internal/store"
)

// SuggestionPriority determines the delivery channel for a suggestion.
type SuggestionPriority int

const (
	// PriorityCritical: immediate delivery via additionalContext or deny.
	PriorityCritical SuggestionPriority = iota
	// PriorityHigh: immediate additionalContext (capped at 3 per burst).
	PriorityHigh
	// PriorityMedium: queued to nudge_outbox for next UserPromptSubmit.
	PriorityMedium
	// PriorityLow: only surfaced via MCP tool on explicit request.
	PriorityLow
	// PrioritySuppressed: do not deliver at all.
	PrioritySuppressed
)

// DeliveryChannel describes how a suggestion should be delivered.
type DeliveryChannel int

const (
	ChannelImmediate DeliveryChannel = iota // return in current hook response
	ChannelNudge                            // enqueue to nudge_outbox
	ChannelDefer                            // store for MCP tool only
	ChannelSuppress                         // do not deliver
)

// DeliveryDecision holds the routing decision for a suggestion.
type DeliveryDecision struct {
	Channel  DeliveryChannel
	Priority SuggestionPriority
}

// RouteDelivery decides how to deliver a suggestion based on:
// 1. User's historical response rate for this pattern (effectiveness_score).
// 2. Number of suggestions already delivered in this burst.
// 3. Standard suppression check.
// 4. Workflow boundary boost (phase transitions, commits, task switches).
func RouteDelivery(sdb *sessiondb.SessionDB, pattern string, priority SuggestionPriority) DeliveryDecision {
	// Suppress non-critical suggestions during productive flow or suggestion fatigue.
	if priority > PriorityCritical && (isInFlow(sdb) || suggestionFatigue(sdb)) {
		return DeliveryDecision{Channel: ChannelDefer, Priority: priority}
	}

	// Workflow boundary boost: promote Medium → High at phase transitions,
	// commits, and task switches (52% engagement vs 31% mid-task).
	if priority == PriorityMedium && isAtWorkflowBoundary(sdb) {
		priority = PriorityHigh
	}

	// Critical priority bypasses Thompson Sampling — always deliver immediately.
	if priority == PriorityCritical {
		return DeliveryDecision{Channel: ChannelImmediate, Priority: priority}
	}

	// Apply adaptive priority adjustment using Thompson Sampling.
	rng := getSessionRNG(sdb)
	adjusted := adjustPriority(rng, pattern, priority)
	if adjusted >= PrioritySuppressed {
		return DeliveryDecision{Channel: ChannelSuppress, Priority: adjusted}
	}

	// Check burst suggestion count to prevent fatigue.
	burstCount := getBurstSuggestionCount(sdb)
	if adjusted <= PriorityHigh && burstCount >= 3 {
		// Too many suggestions this burst — downgrade to nudge.
		if adjusted == PriorityHigh {
			adjusted = PriorityMedium
		}
	}

	switch adjusted {
	case PriorityCritical:
		return DeliveryDecision{Channel: ChannelImmediate, Priority: adjusted}
	case PriorityHigh:
		incrementBurstSuggestionCount(sdb)
		return DeliveryDecision{Channel: ChannelImmediate, Priority: adjusted}
	case PriorityMedium:
		return DeliveryDecision{Channel: ChannelNudge, Priority: adjusted}
	default:
		return DeliveryDecision{Channel: ChannelDefer, Priority: adjusted}
	}
}

// isAtWorkflowBoundary checks and consumes the at_workflow_boundary flag.
// The flag is set by recordPhase on phase transitions, git commits, and task switches.
// It is consumed (cleared) after reading to ensure single-use per boundary event.
func isAtWorkflowBoundary(sdb *sessiondb.SessionDB) bool {
	val, _ := sdb.GetContext("at_workflow_boundary")
	if val != "true" {
		return false
	}
	_ = sdb.SetContext("at_workflow_boundary", "")
	return true
}

// adjustPriority uses contextual Thompson Sampling to adaptively adjust suggestion priority.
// It builds a contextual key from (pattern, task_type, velocity_state) and uses that
// for finer-grained adaptation. Falls back to the base pattern key when context data is sparse.
// For patterns with UserPref data, it uses the weighted effectiveness score (deterministic).
// For patterns with only delivery/resolution counts, it draws from a Beta distribution
// to naturally balance exploration (new patterns) and exploitation (proven patterns).
// Returns PrioritySuppressed if the pattern should not be delivered at all.
func adjustPriority(rng *rand.Rand, pattern string, base SuggestionPriority) SuggestionPriority {
	st, err := store.OpenDefault()
	if err != nil {
		return base
	}
	defer st.Close()

	// Build contextual key for finer-grained Thompson Sampling.
	ctxKey := contextualPatternKey(pattern)

	// Check contextual UserPref first, then fall back to base pattern.
	pref, err := st.UserPreference(ctxKey)
	if err == nil && pref != nil {
		return adjustFromUserPref(pref, base)
	}
	pref, err = st.UserPreference(pattern)
	if err == nil && pref != nil {
		return adjustFromUserPref(pref, base)
	}

	// Hard suppression safety net for truly dead patterns.
	if st.ShouldSuppressPattern(pattern) {
		return PrioritySuppressed
	}

	// Thompson Sampling: try contextual key first, fall back to base pattern.
	delivered, resolved, err := st.DecayedPatternEffectiveness(ctxKey)
	if err != nil || delivered < 3.0 {
		// Insufficient contextual data — fall back to base pattern.
		delivered, resolved, err = st.DecayedPatternEffectiveness(pattern)
	}
	if err != nil || delivered < 0.5 {
		// No data at all — uniform prior Beta(1,1), sample for exploration.
		sample := betaSample(rng, 1, 1)
		return adjustFromEstimate(sample, base)
	}

	// Draw from Beta posterior for exploration-exploitation balance.
	alpha := resolved + 1
	beta := delivered - resolved + 1
	sample := betaSample(rng, alpha, beta)
	return adjustFromEstimate(sample, base)
}

// adjustFromUserPref uses the weighted effectiveness score from UserPref.
func adjustFromUserPref(pref *store.UserPref, base SuggestionPriority) SuggestionPriority {
	return adjustFromEstimate(pref.EffectivenessScore, base)
}

// adjustFromEstimate maps an effectiveness estimate [0,1] to a priority adjustment.
func adjustFromEstimate(estimate float64, base SuggestionPriority) SuggestionPriority {
	switch {
	case estimate > 0.5:
		return base // likely effective, deliver as-is
	case estimate > 0.25:
		if base < PriorityLow {
			return base + 1 // downgrade by 1 level
		}
		return base
	case estimate > 0.10:
		if base+2 < PrioritySuppressed {
			return base + 2 // downgrade by 2 levels
		}
		return PriorityLow
	default:
		return PrioritySuppressed
	}
}

// betaExpectation returns the mean of a Beta(alpha, beta) distribution.
// This is the deterministic analog of Thompson Sampling — it produces the same
// priority ordering as random sampling in expectation, without adding randomness
// to hook output (which should be deterministic for reproducibility).
func betaExpectation(alpha, beta float64) float64 {
	return alpha / (alpha + beta)
}

// Deliver routes a suggestion through the appropriate channel.
// For ChannelImmediate, the caller should include the returned string in the hook output.
// For ChannelNudge, the suggestion is enqueued to nudge_outbox.
// For ChannelDefer and ChannelSuppress, nothing is delivered.
func Deliver(sdb *sessiondb.SessionDB, pattern, level, observation, suggestion string, priority SuggestionPriority) (immediate string) {
	// Phase-aware gating: suppress suggestions inappropriate for current phase.
	if shouldGateForPhase(sdb, pattern) {
		return ""
	}

	decision := RouteDelivery(sdb, pattern, priority)

	switch decision.Channel {
	case ChannelImmediate:
		return fmt.Sprintf("[buddy] %s (%s): %s\n→ %s", pattern, level, observation, suggestion)
	case ChannelNudge:
		_ = sdb.EnqueueNudge(pattern, level, observation, suggestion)
		return ""
	case ChannelDefer, ChannelSuppress:
		return ""
	}
	return ""
}

// contextualPatternKey builds a contextual key from (pattern, task_type, velocity_state).
// This allows Thompson Sampling to learn that e.g. "workflow" suggestions are effective
// during bugfix+slow but not during feature+fast. Returns "pattern:task_type:velocity_state".
func contextualPatternKey(pattern string) string {
	taskType := currentTaskType()
	velState := currentVelocityState()
	if taskType == "" && velState == "" {
		return pattern
	}
	if taskType == "" {
		taskType = "unknown"
	}
	if velState == "" {
		velState = "normal"
	}
	return pattern + ":" + taskType + ":" + velState
}

// currentTaskType reads the task_type from the current sessiondb.
// Returns empty string if unavailable (called from short-lived hook process).
func currentTaskType() string {
	// Read from process-level cache set by the hook handler.
	return ctxTaskType
}

// currentVelocityState classifies current velocity into fast/normal/slow.
func currentVelocityState() string {
	return ctxVelocityState
}

// SetDeliveryContext caches task_type and velocity for contextual Thompson Sampling.
// Called once per hook invocation before any Deliver calls.
func SetDeliveryContext(sdb *sessiondb.SessionDB) {
	ctxTaskType, _ = sdb.GetContext("task_type")
	vel := getFloat(sdb, "ewma_tool_velocity")
	switch {
	case vel > 8.0:
		ctxVelocityState = "fast"
	case vel < 2.0:
		ctxVelocityState = "slow"
	default:
		ctxVelocityState = "normal"
	}
}

// Process-level cache for contextual delivery (set once per hook invocation).
var (
	ctxTaskType      string
	ctxVelocityState string
)

func getBurstSuggestionCount(sdb *sessiondb.SessionDB) int {
	val, _ := sdb.GetContext("suggestions_this_burst")
	if val == "" {
		return 0
	}
	n, _ := strconv.Atoi(val)
	return n
}

func incrementBurstSuggestionCount(sdb *sessiondb.SessionDB) {
	count := getBurstSuggestionCount(sdb) + 1
	if err := sdb.SetContext("suggestions_this_burst", strconv.Itoa(count)); err != nil {
		fmt.Fprintf(os.Stderr, "[buddy] increment burst suggestion count: %v\n", err)
	}
}

// getSessionRNG returns a per-session RNG seeded from a stored seed.
// Each hook invocation within a session gets a fresh RNG from the same seed,
// providing cross-session exploration diversity while being debuggable.
func getSessionRNG(sdb *sessiondb.SessionDB) *rand.Rand {
	seedStr, _ := sdb.GetContext("thompson_seed")
	if seedStr == "" {
		seed := uint64(time.Now().UnixNano())
		seedStr = strconv.FormatUint(seed, 10)
		_ = sdb.SetContext("thompson_seed", seedStr)
	}
	seedVal, _ := strconv.ParseUint(seedStr, 10, 64)
	return rand.New(rand.NewPCG(seedVal, seedVal>>32))
}

// betaSample draws a random sample from a Beta(alpha, beta) distribution
// using the Gamma variate method: Beta(a,b) = X/(X+Y) where X~Gamma(a,1), Y~Gamma(b,1).
func betaSample(rng *rand.Rand, alpha, beta float64) float64 {
	if alpha <= 0 {
		alpha = 1
	}
	if beta <= 0 {
		beta = 1
	}
	x := gammaSample(rng, alpha)
	y := gammaSample(rng, beta)
	if x+y == 0 {
		return 0.5
	}
	return x / (x + y)
}

// gammaSample draws from Gamma(shape, 1) using Marsaglia and Tsang's method.
// For shape < 1, uses the boost: Gamma(a) = Gamma(a+1) * U^(1/a).
func gammaSample(rng *rand.Rand, shape float64) float64 {
	if shape < 1 {
		return gammaSample(rng, shape+1) * math.Pow(rng.Float64(), 1.0/shape)
	}
	d := shape - 1.0/3.0
	c := 1.0 / math.Sqrt(9.0*d)
	for {
		var x, v float64
		for {
			x = rng.NormFloat64()
			v = 1.0 + c*x
			if v > 0 {
				break
			}
		}
		v = v * v * v
		u := rng.Float64()
		if u < 1.0-0.0331*(x*x)*(x*x) {
			return d * v
		}
		if math.Log(u) < 0.5*x*x+d*(1.0-v+math.Log(v)) {
			return d * v
		}
	}
}
