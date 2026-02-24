package analyzer

import (
	"testing"
	"time"

	"github.com/hir4ta/claude-buddy/internal/parser"
)

func makeToolEvent(name, input string, ts time.Time) parser.SessionEvent {
	return parser.SessionEvent{
		Type:      parser.EventToolUse,
		ToolName:  name,
		ToolInput: input,
		Timestamp: ts,
	}
}

func makeUserEvent(text string, ts time.Time) parser.SessionEvent {
	return parser.SessionEvent{
		Type:      parser.EventUserMessage,
		UserText:  text,
		Timestamp: ts,
	}
}

func makeAssistantEvent(text string, ts time.Time) parser.SessionEvent {
	return parser.SessionEvent{
		Type:          parser.EventAssistantText,
		AssistantText: text,
		Timestamp:     ts,
	}
}

func makeCompactEvent(ts time.Time) parser.SessionEvent {
	return parser.SessionEvent{
		Type:          parser.EventCompactBoundary,
		AssistantText: "Summary of previous context",
		Timestamp:     ts,
	}
}

func TestDetectRetryLoop(t *testing.T) {
	d := NewDetector()
	now := time.Now()

	// Start with a user message
	d.Update(makeUserEvent("do something", now))

	// Feed 5 identical Bash events
	var alerts []Alert
	for i := range 5 {
		ts := now.Add(time.Duration(i+1) * time.Second)
		result := d.Update(makeToolEvent("Bash", "ls -la", ts))
		alerts = append(alerts, result...)
	}

	// Should get at least a Warning (at 3 consecutive)
	found := false
	for _, a := range alerts {
		if a.Pattern == PatternRetryLoop {
			found = true
			if a.Level < LevelWarning {
				t.Errorf("expected at least Warning level, got %d", a.Level)
			}
		}
	}
	if !found {
		t.Error("expected retry-loop alert, got none")
	}
}

func TestDetectRetryLoopFalsePositive(t *testing.T) {
	d := NewDetector()
	now := time.Now()

	d.Update(makeUserEvent("do something", now))

	// Feed 2 identical reads, then an Edit, then 2 more reads
	d.Update(makeToolEvent("Read", "/foo/bar.go", now.Add(1*time.Second)))
	d.Update(makeToolEvent("Read", "/foo/bar.go", now.Add(2*time.Second)))
	d.Update(makeToolEvent("Edit", "/foo/bar.go", now.Add(3*time.Second)))
	alerts3 := d.Update(makeToolEvent("Read", "/foo/bar.go", now.Add(4*time.Second)))
	alerts4 := d.Update(makeToolEvent("Read", "/foo/bar.go", now.Add(5*time.Second)))

	// Should NOT trigger retry loop (the Edit breaks the consecutive chain)
	for _, a := range append(alerts3, alerts4...) {
		if a.Pattern == PatternRetryLoop {
			t.Error("expected no retry-loop alert when Edit breaks the chain")
		}
	}
}

func TestDetectDestructiveCmd(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantHit bool
	}{
		{"rm -rf", "rm -rf /tmp/foo", true},
		{"rm -fr", "rm -fr /tmp", true},
		{"rm -r -f", "rm -rf somedir", true},
		{"git push --force", "git push --force origin main", true},
		{"git push -f", "git push -f origin main", true},
		{"git reset --hard", "git reset --hard HEAD~1", true},
		{"git checkout -- .", "git checkout -- .", true},
		{"git restore .", "git restore .", true},
		{"git clean -f", "git clean -f", true},
		{"git clean -fd", "git clean -fd", true},
		{"git branch -D", "git branch -D feature", true},
		{"chmod 777", "chmod 777 /tmp/file", true},
		{"safe rm", "rm file.txt", false},
		{"git push normal", "git push origin main", false},
		{"force-with-lease", "git push --force-with-lease origin main", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDetector()
			now := time.Now()
			d.Update(makeUserEvent("test", now))

			alerts := d.Update(makeToolEvent("Bash", tt.input, now.Add(time.Second)))
			gotHit := false
			for _, a := range alerts {
				if a.Pattern == PatternDestructiveCmd {
					gotHit = true
					if a.Level != LevelAction {
						t.Errorf("expected Action level, got %d", a.Level)
					}
				}
			}
			if gotHit != tt.wantHit {
				t.Errorf("input=%q: gotHit=%v, wantHit=%v", tt.input, gotHit, tt.wantHit)
			}
		})
	}
}

func TestDetectDestructiveCmdSafe(t *testing.T) {
	d := NewDetector()
	now := time.Now()
	d.Update(makeUserEvent("test", now))

	alerts := d.Update(makeToolEvent("Bash", "rm file.txt", now.Add(time.Second)))
	for _, a := range alerts {
		if a.Pattern == PatternDestructiveCmd {
			t.Error("expected no destructive-cmd alert for safe rm")
		}
	}
}

func TestDetectExcessiveTools(t *testing.T) {
	d := NewDetector()
	now := time.Now()
	d.Update(makeUserEvent("do something big", now))

	var alerts []Alert
	for i := range 30 {
		ts := now.Add(time.Duration(i+1) * time.Second)
		// Use different inputs to avoid triggering retry-loop
		result := d.Update(makeToolEvent("Read", "/file"+itoa(i)+".go", ts))
		alerts = append(alerts, result...)
	}

	found := false
	for _, a := range alerts {
		if a.Pattern == PatternExcessiveTools && a.Level >= LevelWarning {
			found = true
		}
	}
	if !found {
		t.Error("expected excessive-tools warning after 25+ tool calls")
	}
}

func TestDetectFileReadLoop(t *testing.T) {
	d := NewDetector()
	now := time.Now()
	d.Update(makeUserEvent("help me", now))

	var alerts []Alert
	for i := range 6 {
		ts := now.Add(time.Duration(i+1) * time.Second)
		result := d.Update(makeToolEvent("Read", "/same/file.go", ts))
		alerts = append(alerts, result...)
	}

	found := false
	for _, a := range alerts {
		if a.Pattern == PatternFileReadLoop && a.Level >= LevelWarning {
			found = true
		}
	}
	if !found {
		t.Error("expected file-read-loop warning after 5+ reads of same file")
	}
}

func TestDetectContextThrashing(t *testing.T) {
	d := NewDetector()
	now := time.Now()
	d.Update(makeUserEvent("start", now))

	// 3 compacts within 10 minutes
	var alerts []Alert
	for i := range 3 {
		ts := now.Add(time.Duration(i*4) * time.Minute)
		result := d.Update(makeCompactEvent(ts))
		alerts = append(alerts, result...)
	}

	foundWarning := false
	foundAction := false
	for _, a := range alerts {
		if a.Pattern == PatternContextThrashing {
			if a.Level == LevelWarning {
				foundWarning = true
			}
			if a.Level == LevelAction {
				foundAction = true
			}
		}
	}
	if !foundWarning {
		t.Error("expected context-thrashing warning after 2 compacts")
	}
	if !foundAction {
		t.Error("expected context-thrashing action after 3 compacts")
	}
}

func TestDetectExploreLoop(t *testing.T) {
	d := NewDetector()
	now := time.Now()
	d.Update(makeUserEvent("explore codebase", now))

	var alerts []Alert
	// 15 Read/Grep events over 6 minutes without Write
	for i := range 15 {
		ts := now.Add(time.Duration(i*24) * time.Second) // spread over 6 minutes
		toolName := "Read"
		if i%3 == 0 {
			toolName = "Grep"
		}
		result := d.Update(makeToolEvent(toolName, "/file"+itoa(i)+".go", ts))
		alerts = append(alerts, result...)
	}

	found := false
	for _, a := range alerts {
		if a.Pattern == PatternExploreLoop && a.Level >= LevelWarning {
			found = true
		}
	}
	if !found {
		t.Error("expected explore-loop warning after 5+ min of reads without writes")
	}
}

func TestDetectCompactAmnesia(t *testing.T) {
	d := NewDetector()
	now := time.Now()
	d.Update(makeUserEvent("start", now))

	// Read files A, B, C before compact
	d.Update(makeToolEvent("Read", "/a.go", now.Add(1*time.Second)))
	d.Update(makeToolEvent("Read", "/b.go", now.Add(2*time.Second)))
	d.Update(makeToolEvent("Read", "/c.go", now.Add(3*time.Second)))

	// Compact boundary
	d.Update(makeCompactEvent(now.Add(4 * time.Second)))

	// Re-read same files A, B, C after compact + additional events to reach 30 threshold
	var alerts []Alert
	d.Update(makeToolEvent("Read", "/a.go", now.Add(5*time.Second)))
	d.Update(makeToolEvent("Read", "/b.go", now.Add(6*time.Second)))
	d.Update(makeToolEvent("Read", "/c.go", now.Add(7*time.Second)))

	// Fill up to 30 post-compact events
	for i := range 27 {
		ts := now.Add(time.Duration(8+i) * time.Second)
		result := d.Update(makeToolEvent("Bash", "echo "+itoa(i), ts))
		alerts = append(alerts, result...)
	}

	found := false
	for _, a := range alerts {
		if a.Pattern == PatternCompactAmnesia {
			found = true
		}
	}
	if !found {
		t.Error("expected compact-amnesia alert when re-reading same files after compact")
	}
}

func TestCooldown(t *testing.T) {
	d := NewDetector()
	now := time.Now()
	d.Update(makeUserEvent("test", now))

	// Trigger destructive cmd alert
	alerts1 := d.Update(makeToolEvent("Bash", "rm -rf /tmp", now.Add(1*time.Second)))
	if len(alerts1) == 0 {
		t.Fatal("expected alert on first destructive cmd")
	}

	// Immediately trigger same pattern — should be suppressed by cooldown
	alerts2 := d.Update(makeToolEvent("Bash", "rm -rf /other", now.Add(2*time.Second)))
	for _, a := range alerts2 {
		if a.Pattern == PatternDestructiveCmd {
			t.Error("expected destructive-cmd alert to be suppressed by cooldown")
		}
	}
}

func TestSessionHealth(t *testing.T) {
	d := NewDetector()
	now := time.Now()
	d.Update(makeUserEvent("test", now))

	// Before any alerts
	if h := d.SessionHealth(); h != 1.0 {
		t.Errorf("expected health 1.0 before alerts, got %f", h)
	}

	// Trigger a Warning-level alert
	d.Update(makeToolEvent("Bash", "rm -rf /tmp", now.Add(1*time.Second)))

	// Health should decrease
	h := d.SessionHealth()
	if h >= 1.0 {
		t.Errorf("expected health < 1.0 after alert, got %f", h)
	}
	if h < 0 {
		t.Errorf("expected health >= 0, got %f", h)
	}
}

func TestPatternName(t *testing.T) {
	tests := []struct {
		pattern PatternType
		want    string
	}{
		{PatternRetryLoop, "retry-loop"},
		{PatternCompactAmnesia, "compact-amnesia"},
		{PatternExcessiveTools, "excessive-tools"},
		{PatternDestructiveCmd, "destructive-cmd"},
		{PatternFileReadLoop, "file-read-loop"},
		{PatternContextThrashing, "context-thrashing"},
		{PatternTestFailCycle, "test-fail-cycle"},
		{PatternApologizeRetry, "apologize-retry"},
		{PatternExploreLoop, "explore-loop"},
		{PatternRateLimitStuck, "rate-limit-stuck"},
	}
	for _, tt := range tests {
		got := PatternName(tt.pattern)
		if got != tt.want {
			t.Errorf("PatternName(%d) = %q, want %q", tt.pattern, got, tt.want)
		}
	}
}

func TestNewDetectorInitialization(t *testing.T) {
	d := NewDetector()
	if d == nil {
		t.Fatal("NewDetector returned nil")
	}
	if len(d.window) != windowCapacity {
		t.Errorf("window size = %d, want %d", len(d.window), windowCapacity)
	}
	if d.cooldowns == nil {
		t.Error("cooldowns map not initialized")
	}
	if d.burst.fileReads == nil {
		t.Error("burst.fileReads map not initialized")
	}
}

func TestDetectApologizeRetry(t *testing.T) {
	d := NewDetector()
	now := time.Now()
	d.Update(makeUserEvent("fix this bug", now))

	var alerts []Alert
	apologyTexts := []string{
		"I apologize for the confusion. Let me try again.",
		"Sorry about that, I made an error. Let me fix this.",
		"My mistake, I should have done it differently. Let me fix this.",
	}
	for i, text := range apologyTexts {
		ts := now.Add(time.Duration(i+1) * time.Second)
		result := d.Update(makeAssistantEvent(text, ts))
		alerts = append(alerts, result...)
	}

	found := false
	for _, a := range alerts {
		if a.Pattern == PatternApologizeRetry {
			found = true
		}
	}
	if !found {
		t.Error("expected apologize-retry alert after 3 apologies")
	}
}

func TestDetectTestFailCycle(t *testing.T) {
	d := NewDetector()
	now := time.Now()
	d.Update(makeUserEvent("fix test", now))

	var alerts []Alert
	for i := range 3 {
		offset := time.Duration(i*3) * time.Second
		d.Update(makeToolEvent("Edit", "/test_file.go", now.Add(offset+1*time.Second)))
		result := d.Update(makeToolEvent("Bash", "go test ./...", now.Add(offset+2*time.Second)))
		alerts = append(alerts, result...)
	}

	found := false
	for _, a := range alerts {
		if a.Pattern == PatternTestFailCycle {
			found = true
		}
	}
	if !found {
		t.Error("expected test-fail-cycle alert after 3 edit-test cycles")
	}
}
