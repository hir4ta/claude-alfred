package analyzer

import (
	"testing"
	"time"
)

func TestScoreInitial(t *testing.T) {
	t.Parallel()
	sc := NewScoreCalculator()
	score := sc.Score()
	if score.Total != 100 {
		t.Errorf("initial score = %d, want 100", score.Total)
	}
	if score.Label != "Good" {
		t.Errorf("initial label = %q, want %q", score.Label, "Good")
	}
}

func TestScoreAlertPenalty(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		alerts  []Alert
		wantPen int
	}{
		{
			name:    "single warning",
			alerts:  []Alert{{Level: LevelWarning}},
			wantPen: -5,
		},
		{
			name:    "single action",
			alerts:  []Alert{{Level: LevelAction}},
			wantPen: -15,
		},
		{
			name:    "warning + action",
			alerts:  []Alert{{Level: LevelWarning}, {Level: LevelAction}},
			wantPen: -20,
		},
		{
			name:    "info ignored",
			alerts:  []Alert{{Level: LevelInfo}},
			wantPen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sc := NewScoreCalculator()
			now := time.Now()
			sc.Update(makeUserEvent("test", now), tt.alerts)
			score := sc.Score()
			if score.Components.AlertPenalty != tt.wantPen {
				t.Errorf("alert penalty = %d, want %d", score.Components.AlertPenalty, tt.wantPen)
			}
		})
	}
}

func TestScoreAlertPenaltyCap(t *testing.T) {
	t.Parallel()
	sc := NewScoreCalculator()
	now := time.Now()
	for i := 0; i < 10; i++ {
		sc.Update(makeToolEvent("Bash", "cmd", now), []Alert{{Level: LevelAction}})
	}
	score := sc.Score()
	if score.Components.AlertPenalty != -50 {
		t.Errorf("alert penalty = %d, want -50 (capped)", score.Components.AlertPenalty)
	}
}

func TestScorePlanModeBonus(t *testing.T) {
	t.Parallel()
	sc := NewScoreCalculator()
	now := time.Now()
	sc.Update(makeToolEvent("EnterPlanMode", "", now), nil)
	score := sc.Score()
	if score.Components.PlanMode != 5 {
		t.Errorf("plan mode = %d, want 5", score.Components.PlanMode)
	}
}

func TestScorePlanModePenalty(t *testing.T) {
	t.Parallel()
	sc := NewScoreCalculator()
	now := time.Now()
	sc.Update(makeUserEvent("do everything", now), nil)
	for i := 0; i < 5; i++ {
		sc.Update(makeToolEvent("Edit", "/path/file"+string(rune('a'+i))+".go", now), nil)
	}
	score := sc.Score()
	if score.Components.PlanMode != -10 {
		t.Errorf("plan mode = %d, want -10 (no plan mode + 5 files)", score.Components.PlanMode)
	}
}

func TestScoreCLAUDEMDBonus(t *testing.T) {
	t.Parallel()
	sc := NewScoreCalculator()
	now := time.Now()
	sc.Update(makeToolEvent("Read", "/project/CLAUDE.md", now), nil)
	score := sc.Score()
	if score.Components.CLAUDEMD != 3 {
		t.Errorf("CLAUDEMD = %d, want 3", score.Components.CLAUDEMD)
	}
}

func TestScoreSubagentBonus(t *testing.T) {
	t.Parallel()
	sc := NewScoreCalculator()
	now := time.Now()
	sc.Update(makeToolEvent("Task", "explore codebase", now), nil)
	score := sc.Score()
	if score.Components.Subagent != 3 {
		t.Errorf("subagent = %d, want 3", score.Components.Subagent)
	}
}

func TestScoreContextPenalty(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		compacts int
		want     int
	}{
		{"no compacts", 0, 0},
		{"one compact", 1, 0},
		{"two compacts", 2, -5},
		{"three compacts", 3, -15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sc := NewScoreCalculator()
			now := time.Now()
			for i := 0; i < tt.compacts; i++ {
				sc.Update(makeCompactEvent(now), nil)
			}
			score := sc.Score()
			if score.Components.ContextMgmt != tt.want {
				t.Errorf("context mgmt = %d, want %d", score.Components.ContextMgmt, tt.want)
			}
		})
	}
}

func TestScoreInstructionQuality(t *testing.T) {
	t.Parallel()
	sc := NewScoreCalculator()
	now := time.Now()
	// 4 long messages (>50 runes), 1 short → 80% > 70%
	for i := 0; i < 4; i++ {
		sc.Update(makeUserEvent("Please implement the authentication module with JWT tokens and proper error handling", now), nil)
	}
	sc.Update(makeUserEvent("fix", now), nil)
	score := sc.Score()
	if score.Components.InstructionQual != 5 {
		t.Errorf("instruction quality = %d, want 5", score.Components.InstructionQual)
	}
}

func TestScoreInstructionQualityLow(t *testing.T) {
	t.Parallel()
	sc := NewScoreCalculator()
	now := time.Now()
	// 4 short messages, 1 long → 20% < 70%
	for i := 0; i < 4; i++ {
		sc.Update(makeUserEvent("fix this", now), nil)
	}
	sc.Update(makeUserEvent("Please implement the authentication module with JWT tokens and proper error handling", now), nil)
	score := sc.Score()
	if score.Components.InstructionQual != 0 {
		t.Errorf("instruction quality = %d, want 0", score.Components.InstructionQual)
	}
}

func TestScoreToolEfficiency(t *testing.T) {
	t.Parallel()
	sc := NewScoreCalculator()
	now := time.Now()
	sc.Update(makeUserEvent("work", now), nil)
	for i := 0; i < 26; i++ {
		sc.Update(makeToolEvent("Read", "/some/file.go", now), nil)
	}
	score := sc.Score()
	if score.Components.ToolEfficiency != -10 {
		t.Errorf("tool efficiency = %d, want -10", score.Components.ToolEfficiency)
	}
}

func TestScoreBurstResetsOnUserMessage(t *testing.T) {
	t.Parallel()
	sc := NewScoreCalculator()
	now := time.Now()
	sc.Update(makeUserEvent("work", now), nil)
	for i := 0; i < 20; i++ {
		sc.Update(makeToolEvent("Read", "/some/file.go", now), nil)
	}
	// New user message resets burst
	sc.Update(makeUserEvent("next task", now), nil)
	score := sc.Score()
	if score.Components.ToolEfficiency != 0 {
		t.Errorf("tool efficiency after reset = %d, want 0", score.Components.ToolEfficiency)
	}
}

func TestScoreFloorAndCeiling(t *testing.T) {
	t.Parallel()
	// Floor: 0
	sc := NewScoreCalculator()
	now := time.Now()
	for i := 0; i < 10; i++ {
		sc.Update(makeToolEvent("Bash", "cmd", now), []Alert{{Level: LevelAction}})
	}
	score := sc.Score()
	if score.Total < 0 {
		t.Errorf("score below 0: %d", score.Total)
	}

	// Ceiling: 100
	sc2 := NewScoreCalculator()
	sc2.Update(makeToolEvent("EnterPlanMode", "", now), nil)
	sc2.Update(makeToolEvent("Read", "CLAUDE.md", now), nil)
	sc2.Update(makeToolEvent("Task", "explore", now), nil)
	score2 := sc2.Score()
	if score2.Total > 100 {
		t.Errorf("score above 100: %d", score2.Total)
	}
}

func TestScoreLabelThresholds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		score int
		want  string
	}{
		{100, "Good"},
		{80, "Good"},
		{79, "Fair"},
		{60, "Fair"},
		{59, "Needs Work"},
		{0, "Needs Work"},
	}

	for _, tt := range tests {
		got := scoreLabel(tt.score)
		if got != tt.want {
			t.Errorf("scoreLabel(%d) = %q, want %q", tt.score, got, tt.want)
		}
	}
}
