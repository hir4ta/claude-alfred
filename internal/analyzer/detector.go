package analyzer

import (
	"hash/fnv"
	"regexp"
	"strings"
	"time"

	"github.com/hir4ta/claude-buddy/internal/parser"
)

// PatternType represents an anti-pattern detected in a Claude Code session.
type PatternType int

const (
	PatternRetryLoop        PatternType = iota // same tool+input 3+ times consecutively
	PatternCompactAmnesia                      // re-reading same files after compact
	PatternExcessiveTools                      // 25+ tool calls without user turn
	PatternDestructiveCmd                      // rm -rf, git push --force, etc.
	PatternFileReadLoop                        // same file read 5+ times
	PatternContextThrashing                    // 2+ compacts in 15 minutes
	PatternTestFailCycle                       // test->edit->test fail 3+ cycles
	PatternApologizeRetry                      // apologize + same approach 3+ times
	PatternExploreLoop                         // 5+ min of Read/Grep only, no Write/Edit
	PatternRateLimitStuck                      // rate limit text + no progress for 5 min
)

// Alert represents a detected anti-pattern.
type Alert struct {
	Pattern     PatternType
	Level       FeedbackLevel
	Situation   string
	Observation string
	Suggestion  string
	Timestamp   time.Time
	EventCount  int
}

// EventFingerprint is a lightweight hash of a tool event.
type EventFingerprint struct {
	ToolName  string
	InputHash uint64
	FilePath  string
	Timestamp time.Time
	IsUser    bool
	IsCompact bool
	IsWrite   bool
}

// BurstTracker tracks events between user messages.
type BurstTracker struct {
	toolCount    int
	fileReads    map[string]int
	uniqueFiles  map[string]bool
	hasWrite     bool
	startTime    time.Time
	lastToolTime time.Time
}

// CompactionTracker is a state machine for compact amnesia detection.
type CompactionTracker struct {
	preCompactReads  map[string]bool
	postCompactReads map[string]bool
	inPostCompact    bool
	postCompactCount int
	compactTimes     []time.Time
}

// Detector detects anti-patterns in Claude Code sessions.
type Detector struct {
	window     []EventFingerprint
	windowSize int
	pos        int
	count      int

	burst      BurstTracker
	compaction CompactionTracker
	cooldowns  map[PatternType]time.Time
	alerts     []Alert

	// For apologize-retry detection
	recentApologies   int
	lastApologyTime   time.Time
	assistantTurnsSinceReset int

	// For test-fail cycle detection
	testCycleCount int
	lastTestFile   string
	lastEditSeen   bool
}

const windowCapacity = 50

// Cooldown durations per feedback level.
const (
	cooldownInfo    = 3 * time.Minute
	cooldownWarning = 5 * time.Minute
	cooldownAction  = 10 * time.Minute
)

// NewDetector creates a Detector with initialized state.
func NewDetector() *Detector {
	return &Detector{
		window:     make([]EventFingerprint, windowCapacity),
		windowSize: windowCapacity,
		cooldowns:  make(map[PatternType]time.Time),
		burst: BurstTracker{
			fileReads:   make(map[string]int),
			uniqueFiles: make(map[string]bool),
		},
		compaction: CompactionTracker{
			preCompactReads:  make(map[string]bool),
			postCompactReads: make(map[string]bool),
		},
	}
}

// Update processes a new event and returns any newly detected alerts.
func (d *Detector) Update(ev parser.SessionEvent) []Alert {
	fp := d.fingerprint(ev)
	d.addToWindow(fp)

	// Reset burst tracker on user message
	if fp.IsUser {
		d.resetBurst(fp.Timestamp)
	}

	// Track compaction state
	if fp.IsCompact {
		d.handleCompact(fp.Timestamp)
	}

	// Update burst tracker for tool events
	if ev.Type == parser.EventToolUse {
		d.burst.toolCount++
		d.burst.lastToolTime = fp.Timestamp
		if fp.FilePath != "" {
			d.burst.fileReads[fp.FilePath]++
			d.burst.uniqueFiles[fp.FilePath] = true
		}
		if fp.IsWrite {
			d.burst.hasWrite = true
			// Reset file read counter for written file
			delete(d.burst.fileReads, fp.FilePath)
		}
	}

	// Track post-compact reads
	if d.compaction.inPostCompact && ev.Type == parser.EventToolUse {
		d.compaction.postCompactCount++
		if fp.FilePath != "" && isReadTool(ev.ToolName) {
			d.compaction.postCompactReads[fp.FilePath] = true
		}
	}

	// Track pre-compact reads (always, until compact boundary resets)
	if !d.compaction.inPostCompact && ev.Type == parser.EventToolUse && fp.FilePath != "" && isReadTool(ev.ToolName) {
		d.compaction.preCompactReads[fp.FilePath] = true
	}

	// Run all detectors
	var newAlerts []Alert

	if a := d.detectRetryLoop(); a != nil {
		newAlerts = append(newAlerts, *a)
	}
	if a := d.detectCompactAmnesia(); a != nil {
		newAlerts = append(newAlerts, *a)
	}
	if a := d.detectExcessiveTools(); a != nil {
		newAlerts = append(newAlerts, *a)
	}
	if a := d.detectDestructiveCmd(ev); a != nil {
		newAlerts = append(newAlerts, *a)
	}
	if a := d.detectFileReadLoop(); a != nil {
		newAlerts = append(newAlerts, *a)
	}
	if a := d.detectContextThrashing(); a != nil {
		newAlerts = append(newAlerts, *a)
	}
	if a := d.detectTestFailCycle(ev); a != nil {
		newAlerts = append(newAlerts, *a)
	}
	if a := d.detectApologizeRetry(ev); a != nil {
		newAlerts = append(newAlerts, *a)
	}
	if a := d.detectExploreLoop(); a != nil {
		newAlerts = append(newAlerts, *a)
	}
	if a := d.detectRateLimitStuck(ev); a != nil {
		newAlerts = append(newAlerts, *a)
	}

	// Apply cooldown filtering (allow level escalation)
	var filtered []Alert
	now := ev.Timestamp
	if now.IsZero() {
		now = time.Now()
	}
	for _, a := range newAlerts {
		if d.isOnCooldown(a.Pattern, now) {
			// Allow escalation: if new alert is higher severity, bypass cooldown
			if a.Level <= d.lastAlertLevel(a.Pattern) {
				continue
			}
		}
		d.setCooldown(a.Pattern, a.Level, now)
		a.Timestamp = now
		filtered = append(filtered, a)
		d.alerts = append(d.alerts, a)
	}

	return filtered
}

// ActiveAlerts returns alerts filtered by cooldown.
func (d *Detector) ActiveAlerts() []Alert {
	now := time.Now()
	var active []Alert
	for _, a := range d.alerts {
		cd := cooldownForLevel(a.Level)
		if now.Sub(a.Timestamp) < cd {
			active = append(active, a)
		}
	}
	return active
}

// SessionHealth returns a score from 0.0 to 1.0 based on active alerts.
func (d *Detector) SessionHealth() float64 {
	active := d.ActiveAlerts()
	health := 1.0
	for _, a := range active {
		switch a.Level {
		case LevelWarning:
			health -= 0.1
		case LevelAction:
			health -= 0.2
		}
	}
	if health < 0 {
		health = 0
	}
	return health
}

// PatternName returns a human-readable name for a pattern type.
func PatternName(p PatternType) string {
	switch p {
	case PatternRetryLoop:
		return "retry-loop"
	case PatternCompactAmnesia:
		return "compact-amnesia"
	case PatternExcessiveTools:
		return "excessive-tools"
	case PatternDestructiveCmd:
		return "destructive-cmd"
	case PatternFileReadLoop:
		return "file-read-loop"
	case PatternContextThrashing:
		return "context-thrashing"
	case PatternTestFailCycle:
		return "test-fail-cycle"
	case PatternApologizeRetry:
		return "apologize-retry"
	case PatternExploreLoop:
		return "explore-loop"
	case PatternRateLimitStuck:
		return "rate-limit-stuck"
	default:
		return "unknown"
	}
}

// --- Internal methods ---

func (d *Detector) fingerprint(ev parser.SessionEvent) EventFingerprint {
	fp := EventFingerprint{
		Timestamp: ev.Timestamp,
	}

	switch ev.Type {
	case parser.EventUserMessage:
		fp.IsUser = true
	case parser.EventCompactBoundary:
		fp.IsCompact = true
	case parser.EventToolUse:
		fp.ToolName = ev.ToolName
		fp.InputHash = hashString(ev.ToolInput)
		fp.FilePath = extractFilePath(ev)
		fp.IsWrite = isWriteTool(ev.ToolName)
	}

	return fp
}

func (d *Detector) addToWindow(fp EventFingerprint) {
	d.window[d.pos] = fp
	d.pos = (d.pos + 1) % d.windowSize
	if d.count < d.windowSize {
		d.count++
	}
}

func (d *Detector) resetBurst(ts time.Time) {
	d.burst = BurstTracker{
		fileReads:   make(map[string]int),
		uniqueFiles: make(map[string]bool),
		startTime:   ts,
	}
	d.recentApologies = 0
	d.assistantTurnsSinceReset = 0
	d.lastEditSeen = false
	d.testCycleCount = 0
}

func (d *Detector) handleCompact(ts time.Time) {
	// Save pre-compact reads, start post-compact tracking
	d.compaction.inPostCompact = true
	d.compaction.postCompactReads = make(map[string]bool)
	d.compaction.postCompactCount = 0
	d.compaction.compactTimes = append(d.compaction.compactTimes, ts)

	// Keep only recent compact times (last 10)
	if len(d.compaction.compactTimes) > 10 {
		d.compaction.compactTimes = d.compaction.compactTimes[len(d.compaction.compactTimes)-10:]
	}
}

// getRecentFingerprints returns the last n fingerprints from the ring buffer (newest first).
func (d *Detector) getRecentFingerprints(n int) []EventFingerprint {
	if n > d.count {
		n = d.count
	}
	result := make([]EventFingerprint, n)
	for i := range n {
		idx := (d.pos - 1 - i + d.windowSize) % d.windowSize
		result[i] = d.window[idx]
	}
	return result
}

// detectRetryLoop scans last 10 events for 3+ consecutive identical tool calls.
func (d *Detector) detectRetryLoop() *Alert {
	recent := d.getRecentFingerprints(10)
	if len(recent) < 3 {
		return nil
	}

	// Count consecutive identical tool calls (from newest)
	consecutiveCount := 1
	for i := 1; i < len(recent); i++ {
		cur := recent[i-1]
		prev := recent[i]
		if cur.ToolName == "" || prev.ToolName == "" {
			break
		}
		if cur.ToolName == prev.ToolName && cur.InputHash == prev.InputHash {
			consecutiveCount++
		} else {
			break
		}
	}

	if consecutiveCount >= 5 {
		return &Alert{
			Pattern:     PatternRetryLoop,
			Level:       LevelAction,
			Situation:   "Claude is repeating the same tool call",
			Observation: "Same tool+input called " + itoa(consecutiveCount) + " times consecutively",
			Suggestion:  "Interrupt and provide a different approach or clarify the goal",
			EventCount:  consecutiveCount,
		}
	}
	if consecutiveCount >= 3 {
		return &Alert{
			Pattern:     PatternRetryLoop,
			Level:       LevelWarning,
			Situation:   "Claude is repeating the same tool call",
			Observation: "Same tool+input called " + itoa(consecutiveCount) + " times consecutively",
			Suggestion:  "Consider interrupting if the retries don't seem productive",
			EventCount:  consecutiveCount,
		}
	}
	return nil
}

// detectCompactAmnesia checks if files are being re-read after compact.
func (d *Detector) detectCompactAmnesia() *Alert {
	if !d.compaction.inPostCompact {
		return nil
	}
	if d.compaction.postCompactCount < 30 {
		return nil
	}
	if len(d.compaction.preCompactReads) == 0 {
		return nil
	}

	overlap := 0
	for f := range d.compaction.postCompactReads {
		if d.compaction.preCompactReads[f] {
			overlap++
		}
	}

	if len(d.compaction.postCompactReads) == 0 {
		return nil
	}

	ratio := float64(overlap) / float64(len(d.compaction.postCompactReads))
	if ratio > 0.6 {
		d.compaction.inPostCompact = false // only alert once
		return &Alert{
			Pattern:     PatternCompactAmnesia,
			Level:       LevelWarning,
			Situation:   "Context was compacted recently",
			Observation: "Claude is re-reading files it already read before compaction (" + itoa(overlap) + " overlapping)",
			Suggestion:  "Use buddy_recall to recover lost context instead of re-reading files",
			EventCount:  overlap,
		}
	}
	return nil
}

// detectExcessiveTools checks for too many tool calls without user input.
func (d *Detector) detectExcessiveTools() *Alert {
	if d.burst.toolCount >= 40 {
		return &Alert{
			Pattern:     PatternExcessiveTools,
			Level:       LevelAction,
			Situation:   "Long autonomous tool burst",
			Observation: itoa(d.burst.toolCount) + " tool calls without user input",
			Suggestion:  "Interrupt to check progress — Claude may be going in circles",
			EventCount:  d.burst.toolCount,
		}
	}
	if d.burst.toolCount >= 25 {
		return &Alert{
			Pattern:     PatternExcessiveTools,
			Level:       LevelWarning,
			Situation:   "Extended autonomous tool burst",
			Observation: itoa(d.burst.toolCount) + " tool calls without user input",
			Suggestion:  "Check that Claude is making progress toward the goal",
			EventCount:  d.burst.toolCount,
		}
	}
	return nil
}

// Regex patterns for destructive command detection.
var (
	rmRFPattern        = regexp.MustCompile(`\brm\s+(-[a-zA-Z]*r[a-zA-Z]*f[a-zA-Z]*|-[a-zA-Z]*f[a-zA-Z]*r[a-zA-Z]*)\b`)
	gitPushForcePattern = regexp.MustCompile(`\bgit\s+push\s+(-f\b|--force\b)`)
	gitResetHardPattern = regexp.MustCompile(`\bgit\s+reset\s+--hard\b`)
	gitCheckoutDot      = regexp.MustCompile(`\bgit\s+checkout\s+--\s*\.`)
	gitRestoreDot       = regexp.MustCompile(`\bgit\s+restore\s+\.`)
	gitCleanF           = regexp.MustCompile(`\bgit\s+clean\s+-f`)
	gitBranchD          = regexp.MustCompile(`\bgit\s+branch\s+-D\b`)
	chmod777            = regexp.MustCompile(`\bchmod\s+777\b`)
)

// detectDestructiveCmd checks for dangerous shell commands.
func (d *Detector) detectDestructiveCmd(ev parser.SessionEvent) *Alert {
	if ev.Type != parser.EventToolUse || ev.ToolName != "Bash" {
		return nil
	}

	input := ev.ToolInput
	if input == "" {
		return nil
	}

	var observation string
	switch {
	case rmRFPattern.MatchString(input):
		observation = "Detected rm -rf command"
	case gitPushForcePattern.MatchString(input) && !strings.Contains(input, "--force-with-lease"):
		observation = "Detected git push --force"
	case gitResetHardPattern.MatchString(input):
		observation = "Detected git reset --hard"
	case gitCheckoutDot.MatchString(input):
		observation = "Detected git checkout -- . (discards all changes)"
	case gitRestoreDot.MatchString(input):
		observation = "Detected git restore . (discards all changes)"
	case gitCleanF.MatchString(input):
		observation = "Detected git clean -f (removes untracked files)"
	case gitBranchD.MatchString(input):
		observation = "Detected git branch -D (force delete branch)"
	case chmod777.MatchString(input):
		observation = "Detected chmod 777 (world-writable permissions)"
	default:
		return nil
	}

	return &Alert{
		Pattern:     PatternDestructiveCmd,
		Level:       LevelAction,
		Situation:   "Potentially destructive command executed",
		Observation: observation,
		Suggestion:  "Verify this was intentional — these commands can cause data loss",
		EventCount:  1,
	}
}

// detectFileReadLoop checks for the same file being read repeatedly.
func (d *Detector) detectFileReadLoop() *Alert {
	maxCount := 0
	maxFile := ""
	for f, c := range d.burst.fileReads {
		if c > maxCount {
			maxCount = c
			maxFile = f
		}
	}

	if maxCount >= 8 {
		return &Alert{
			Pattern:     PatternFileReadLoop,
			Level:       LevelAction,
			Situation:   "Repeated file reads",
			Observation: maxFile + " read " + itoa(maxCount) + " times without editing",
			Suggestion:  "Claude may be stuck — provide specific guidance about this file",
			EventCount:  maxCount,
		}
	}
	if maxCount >= 5 {
		return &Alert{
			Pattern:     PatternFileReadLoop,
			Level:       LevelWarning,
			Situation:   "Repeated file reads",
			Observation: maxFile + " read " + itoa(maxCount) + " times without editing",
			Suggestion:  "Check if Claude is stuck in a read loop",
			EventCount:  maxCount,
		}
	}
	return nil
}

// detectContextThrashing checks for frequent context compactions.
func (d *Detector) detectContextThrashing() *Alert {
	if len(d.compaction.compactTimes) < 2 {
		return nil
	}

	window := 15 * time.Minute
	latest := d.compaction.compactTimes[len(d.compaction.compactTimes)-1]
	compactsInWindow := 0
	for _, ct := range d.compaction.compactTimes {
		if latest.Sub(ct) <= window {
			compactsInWindow++
		}
	}

	if compactsInWindow >= 3 {
		return &Alert{
			Pattern:     PatternContextThrashing,
			Level:       LevelAction,
			Situation:   "Frequent context compactions",
			Observation: itoa(compactsInWindow) + " compactions in 15 minutes",
			Suggestion:  "Context is churning — break the task into smaller steps or start fresh",
			EventCount:  compactsInWindow,
		}
	}
	if compactsInWindow >= 2 {
		return &Alert{
			Pattern:     PatternContextThrashing,
			Level:       LevelWarning,
			Situation:   "Multiple context compactions",
			Observation: itoa(compactsInWindow) + " compactions in 15 minutes",
			Suggestion:  "Session context is filling fast — consider summarizing or narrowing scope",
			EventCount:  compactsInWindow,
		}
	}
	return nil
}

// Test command patterns for test-fail cycle detection.
var testCmdPattern = regexp.MustCompile(`\b(go\s+test|npm\s+test|npx\s+jest|pytest|jest|cargo\s+test|make\s+test)\b`)

// detectTestFailCycle detects test->edit->test fail cycles.
func (d *Detector) detectTestFailCycle(ev parser.SessionEvent) *Alert {
	if ev.Type != parser.EventToolUse {
		return nil
	}

	if ev.ToolName == "Edit" || ev.ToolName == "Write" {
		d.lastEditSeen = true
		return nil
	}

	if ev.ToolName == "Bash" && testCmdPattern.MatchString(ev.ToolInput) {
		if d.lastEditSeen {
			d.testCycleCount++
			d.lastEditSeen = false
		}
	}

	if d.testCycleCount >= 3 {
		return &Alert{
			Pattern:     PatternTestFailCycle,
			Level:       LevelWarning,
			Situation:   "Test-edit-test cycle detected",
			Observation: itoa(d.testCycleCount) + " test-edit-retest cycles without passing",
			Suggestion:  "Claude may be fixing symptoms, not root cause — describe the expected behavior",
			EventCount:  d.testCycleCount,
		}
	}
	return nil
}

// Apology keywords for apologize-retry detection.
var apologyKeywords = []string{
	"i apologize",
	"sorry about that",
	"let me fix",
	"my mistake",
	"i'm sorry",
	"my apologies",
	"\u7533\u3057\u8a33",    // 申し訳
	"\u3059\u307f\u307e\u305b\u3093", // すみません
}

// detectApologizeRetry detects repeated apologies in assistant text.
func (d *Detector) detectApologizeRetry(ev parser.SessionEvent) *Alert {
	if ev.Type != parser.EventAssistantText {
		return nil
	}

	d.assistantTurnsSinceReset++
	lower := strings.ToLower(ev.AssistantText)
	isApology := false
	for _, kw := range apologyKeywords {
		if strings.Contains(lower, kw) {
			isApology = true
			break
		}
	}

	if isApology {
		d.recentApologies++
		d.lastApologyTime = ev.Timestamp
	}

	if d.recentApologies >= 3 && d.assistantTurnsSinceReset <= 10 {
		return &Alert{
			Pattern:     PatternApologizeRetry,
			Level:       LevelWarning,
			Situation:   "Repeated apologies detected",
			Observation: itoa(d.recentApologies) + " apologies in " + itoa(d.assistantTurnsSinceReset) + " assistant turns",
			Suggestion:  "Claude keeps failing and apologizing — try a different approach or rephrase the task",
			EventCount:  d.recentApologies,
		}
	}
	return nil
}

// detectExploreLoop detects prolonged read-only exploration without writes.
func (d *Detector) detectExploreLoop() *Alert {
	if d.burst.hasWrite {
		return nil
	}
	if d.burst.toolCount < 10 {
		return nil
	}
	if d.burst.startTime.IsZero() || d.burst.lastToolTime.IsZero() {
		return nil
	}

	elapsed := d.burst.lastToolTime.Sub(d.burst.startTime)

	if elapsed > 7*time.Minute {
		return &Alert{
			Pattern:     PatternExploreLoop,
			Level:       LevelAction,
			Situation:   "Extended read-only exploration",
			Observation: "Over 7 minutes of Read/Grep without any Write/Edit",
			Suggestion:  "Nudge Claude to start making changes or ask what's blocking it",
			EventCount:  d.burst.toolCount,
		}
	}
	if elapsed > 5*time.Minute {
		return &Alert{
			Pattern:     PatternExploreLoop,
			Level:       LevelWarning,
			Situation:   "Prolonged exploration phase",
			Observation: "Over 5 minutes of Read/Grep without any Write/Edit",
			Suggestion:  "Check if Claude needs guidance to start making changes",
			EventCount:  d.burst.toolCount,
		}
	}
	return nil
}

// Rate-limit keywords.
var rateLimitKeywords = []string{"rate limit", "overloaded", "429", "529"}

// detectRateLimitStuck detects being stuck on rate limits.
func (d *Detector) detectRateLimitStuck(ev parser.SessionEvent) *Alert {
	if ev.Type != parser.EventAssistantText {
		return nil
	}

	lower := strings.ToLower(ev.AssistantText)
	hasRateLimit := false
	for _, kw := range rateLimitKeywords {
		if strings.Contains(lower, kw) {
			hasRateLimit = true
			break
		}
	}

	if !hasRateLimit {
		return nil
	}

	// Check if no meaningful progress in last few events
	recent := d.getRecentFingerprints(10)
	hasProgress := false
	for _, fp := range recent {
		if fp.IsUser || fp.IsWrite {
			hasProgress = true
			break
		}
	}

	if !hasProgress && d.burst.startTime.After(time.Time{}) {
		elapsed := ev.Timestamp.Sub(d.burst.startTime)
		if elapsed > 5*time.Minute {
			return &Alert{
				Pattern:     PatternRateLimitStuck,
				Level:       LevelAction,
				Situation:   "Rate limit detected with no progress",
				Observation: "Rate limit/overload messages and no productive output for over 5 minutes",
				Suggestion:  "Wait a few minutes or try again later — continued retries won't help",
				EventCount:  1,
			}
		}
	}
	return nil
}

// --- Helpers ---

func (d *Detector) isOnCooldown(p PatternType, now time.Time) bool {
	expiry, ok := d.cooldowns[p]
	if !ok {
		return false
	}
	return now.Before(expiry)
}

// lastAlertLevel returns the level of the most recent alert for a pattern, or -1 if none.
func (d *Detector) lastAlertLevel(p PatternType) FeedbackLevel {
	for i := len(d.alerts) - 1; i >= 0; i-- {
		if d.alerts[i].Pattern == p {
			return d.alerts[i].Level
		}
	}
	return FeedbackLevel(-1)
}

func (d *Detector) setCooldown(p PatternType, level FeedbackLevel, now time.Time) {
	d.cooldowns[p] = now.Add(cooldownForLevel(level))
}

func cooldownForLevel(level FeedbackLevel) time.Duration {
	switch level {
	case LevelAction:
		return cooldownAction
	case LevelWarning:
		return cooldownWarning
	default:
		return cooldownInfo
	}
}

func hashString(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

func extractFilePath(ev parser.SessionEvent) string {
	// ToolInput for Read/Write/Edit is typically the file path summary
	switch ev.ToolName {
	case "Read", "Write", "Edit":
		return ev.ToolInput
	}
	return ""
}

func isReadTool(name string) bool {
	return name == "Read" || name == "Grep" || name == "Glob"
}

func isWriteTool(name string) bool {
	return name == "Write" || name == "Edit"
}

func itoa(n int) string {
	if n < 0 {
		return "-" + itoa(-n)
	}
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}
