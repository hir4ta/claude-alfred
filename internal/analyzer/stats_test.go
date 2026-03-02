package analyzer

import (
	"testing"
	"time"

	"github.com/hir4ta/claude-alfred/internal/parser"
)

func TestNewStats(t *testing.T) {
	t.Parallel()
	s := NewStats()
	if s.TurnCount != 0 {
		t.Errorf("TurnCount = %d, want 0", s.TurnCount)
	}
	if s.ToolUseCount != 0 {
		t.Errorf("ToolUseCount = %d, want 0", s.ToolUseCount)
	}
	if s.ToolFreq == nil {
		t.Fatal("ToolFreq is nil, want initialized map")
	}
	if len(s.ToolFreq) != 0 {
		t.Errorf("ToolFreq len = %d, want 0", len(s.ToolFreq))
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		events []parser.SessionEvent
		check  func(t *testing.T, s *Stats)
	}{
		{
			name: "single user message",
			events: []parser.SessionEvent{
				{Type: parser.EventUserMessage, UserText: "hello world", Timestamp: now},
			},
			check: func(t *testing.T, s *Stats) {
				if s.TurnCount != 1 {
					t.Errorf("TurnCount = %d, want 1", s.TurnCount)
				}
				if s.InputChars != 11 {
					t.Errorf("InputChars = %d, want 11", s.InputChars)
				}
			},
		},
		{
			name: "CJK user message counts runes",
			events: []parser.SessionEvent{
				{Type: parser.EventUserMessage, UserText: "こんにちは世界", Timestamp: now},
			},
			check: func(t *testing.T, s *Stats) {
				if s.InputChars != 7 {
					t.Errorf("InputChars = %d, want 7 (rune count)", s.InputChars)
				}
			},
		},
		{
			name: "tool use tracking",
			events: []parser.SessionEvent{
				{Type: parser.EventToolUse, ToolName: "Read", Timestamp: now},
				{Type: parser.EventToolUse, ToolName: "Read", Timestamp: now.Add(time.Second)},
				{Type: parser.EventToolUse, ToolName: "Edit", Timestamp: now.Add(2 * time.Second)},
			},
			check: func(t *testing.T, s *Stats) {
				if s.ToolUseCount != 3 {
					t.Errorf("ToolUseCount = %d, want 3", s.ToolUseCount)
				}
				if s.ToolFreq["Read"] != 2 {
					t.Errorf("ToolFreq[Read] = %d, want 2", s.ToolFreq["Read"])
				}
				if s.ToolFreq["Edit"] != 1 {
					t.Errorf("ToolFreq[Edit] = %d, want 1", s.ToolFreq["Edit"])
				}
			},
		},
		{
			name: "assistant text tracking",
			events: []parser.SessionEvent{
				{Type: parser.EventAssistantText, AssistantText: "I'll help you", Timestamp: now},
			},
			check: func(t *testing.T, s *Stats) {
				if s.AssistantMsgCount != 1 {
					t.Errorf("AssistantMsgCount = %d, want 1", s.AssistantMsgCount)
				}
				if s.OutputChars != 13 {
					t.Errorf("OutputChars = %d, want 13", s.OutputChars)
				}
			},
		},
		{
			name: "timestamp tracking",
			events: []parser.SessionEvent{
				{Type: parser.EventUserMessage, UserText: "a", Timestamp: now},
				{Type: parser.EventAssistantText, AssistantText: "b", Timestamp: now.Add(5 * time.Minute)},
			},
			check: func(t *testing.T, s *Stats) {
				if !s.StartTime.Equal(now) {
					t.Errorf("StartTime = %v, want %v", s.StartTime, now)
				}
				if !s.LastActivity.Equal(now.Add(5 * time.Minute)) {
					t.Errorf("LastActivity = %v, want %v", s.LastActivity, now.Add(5*time.Minute))
				}
			},
		},
		{
			name: "empty user text still increments turn",
			events: []parser.SessionEvent{
				{Type: parser.EventUserMessage, UserText: ""},
			},
			check: func(t *testing.T, s *Stats) {
				if s.TurnCount != 1 {
					t.Errorf("TurnCount = %d, want 1", s.TurnCount)
				}
				if s.InputChars != 0 {
					t.Errorf("InputChars = %d, want 0", s.InputChars)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := NewStats()
			for _, ev := range tt.events {
				s.Update(ev)
			}
			tt.check(t, &s)
		})
	}
}

func TestLongestPause(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	s := NewStats()
	s.Update(parser.SessionEvent{Type: parser.EventUserMessage, UserText: "a", Timestamp: now})
	s.Update(parser.SessionEvent{Type: parser.EventAssistantText, AssistantText: "b", Timestamp: now.Add(2 * time.Second)})
	s.Update(parser.SessionEvent{Type: parser.EventUserMessage, UserText: "c", Timestamp: now.Add(12 * time.Second)}) // 10s gap
	s.Update(parser.SessionEvent{Type: parser.EventAssistantText, AssistantText: "d", Timestamp: now.Add(15 * time.Second)}) // 3s gap

	if s.LongestPause != 10*time.Second {
		t.Errorf("LongestPause = %v, want 10s", s.LongestPause)
	}
}

func TestToolsPerTurn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		turns int
		tools int
		want  float64
	}{
		{"zero turns", 0, 5, 0},
		{"3 tools 1 turn", 1, 3, 3.0},
		{"5 tools 2 turns", 2, 5, 2.5},
		{"0 tools 3 turns", 3, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := Stats{TurnCount: tt.turns, ToolUseCount: tt.tools}
			got := s.ToolsPerTurn()
			if got != tt.want {
				t.Errorf("ToolsPerTurn() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestElapsed(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		s    Stats
		want time.Duration
	}{
		{
			name: "zero start time",
			s:    Stats{},
			want: 0,
		},
		{
			name: "start and last activity set",
			s:    Stats{StartTime: now, LastActivity: now.Add(5 * time.Minute)},
			want: 5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.s.Elapsed()
			if got != tt.want {
				t.Errorf("Elapsed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTopTools(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		freq map[string]int
		n    int
		want []ToolCount
	}{
		{
			name: "empty",
			freq: map[string]int{},
			n:    3,
			want: nil,
		},
		{
			name: "single tool",
			freq: map[string]int{"Read": 5},
			n:    3,
			want: []ToolCount{{"Read", 5}},
		},
		{
			name: "top 2 of 3",
			freq: map[string]int{"Read": 10, "Edit": 5, "Bash": 3},
			n:    2,
			want: []ToolCount{{"Read", 10}, {"Edit", 5}},
		},
		{
			name: "tied counts sorted alphabetically",
			freq: map[string]int{"Bash": 5, "Edit": 5, "Read": 5},
			n:    3,
			want: []ToolCount{{"Bash", 5}, {"Edit", 5}, {"Read", 5}},
		},
		{
			name: "n exceeds len",
			freq: map[string]int{"Read": 1},
			n:    5,
			want: []ToolCount{{"Read", 1}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := Stats{ToolFreq: tt.freq}
			got := s.TopTools(tt.n)

			if tt.want == nil {
				if got != nil {
					t.Errorf("TopTools(%d) = %v, want nil", tt.n, got)
				}
				return
			}

			if len(got) != len(tt.want) {
				t.Fatalf("TopTools(%d) returned %d items, want %d", tt.n, len(got), len(tt.want))
			}
			for i, w := range tt.want {
				if got[i].Name != w.Name || got[i].Count != w.Count {
					t.Errorf("TopTools[%d] = {%s, %d}, want {%s, %d}", i, got[i].Name, got[i].Count, w.Name, w.Count)
				}
			}
		})
	}
}

func TestEstimatedTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		in, out       int
		wantIn, wantO int
	}{
		{"zero", 0, 0, 0, 0},
		{"normal", 400, 800, 100, 200},
		{"rounding", 3, 7, 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := Stats{InputChars: tt.in, OutputChars: tt.out}
			gotIn, gotOut := s.EstimatedTokens()
			if gotIn != tt.wantIn || gotOut != tt.wantO {
				t.Errorf("EstimatedTokens() = (%d, %d), want (%d, %d)", gotIn, gotOut, tt.wantIn, tt.wantO)
			}
		})
	}
}
