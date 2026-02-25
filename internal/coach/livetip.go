package coach

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/hir4ta/claude-buddy/internal/analyzer"
	"github.com/hir4ta/claude-buddy/internal/locale"
	"github.com/hir4ta/claude-buddy/internal/parser"
)

// GenerateFeedback calls claude -p with recent activity to get usage coaching feedback.
// activeAlerts and sessionHealth provide anti-pattern context from the Detector.
func GenerateFeedback(ctx context.Context, events []parser.SessionEvent, stats analyzer.Stats, activeAlerts []analyzer.Alert, sessionHealth float64, lang locale.Lang, prevFeedbacks []analyzer.Feedback) (analyzer.Feedback, error) {
	if _, err := exec.LookPath("claude"); err != nil {
		return analyzer.Feedback{}, fmt.Errorf("claude CLI not found: %w", err)
	}

	summary := buildSummary(events, stats, activeAlerts, sessionHealth, prevFeedbacks)

	// Fetch latest best practices (cached for 1 hour)
	bestPractices, _ := FetchBestPractices()

	prompt := buildFeedbackPrompt(summary, lang, bestPractices)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "claude", "-p", prompt)
	cmd.Dir = os.TempDir() // Avoid creating session files in the watched project
	out, err := cmd.Output()
	if err != nil {
		return analyzer.Feedback{}, err
	}

	return parseFeedbackOutput(string(out)), nil
}

func buildFeedbackPrompt(summary string, lang locale.Lang, bestPractices string) string {
	bpSection := ""
	if bestPractices != "" {
		bpSection = fmt.Sprintf(`
## Reference: Official Best Practices (from code.claude.com)
%s
`, bestPractices)
	}

	if lang.Code == "ja" {
		return fmt.Sprintf(`あなたはClaude Codeの「使い方コーチ」です。
コードの品質ではなく、ユーザーがClaude Codeをどれだけ効果的に活用しているかを評価してください。

## セッション状況
%s
%s
## 評価軸（優先順に）
1. 指示の具体性: ユーザーのメッセージにファイルパスや期待する動作が含まれているか？曖昧な丸投げ指示になっていないか？
2. Plan Mode活用: 複数ファイルの変更にPlan Modeを使っているか？大規模タスクの前に方針確認しているか？
3. コンテキスト管理: auto-compactの頻度は高すぎないか？compact後にファイルを再読込していないか？セッション分割は適切か？
4. ツール効率: リトライループ（同じ操作の繰り返し）はないか？ユーザー入力なしの長いツール連鎖はないか？
5. 機能活用: CLAUDE.md、スラッシュコマンド、hooks、サブエージェント(Task)を知っていて使っているか？

## 出力ルール
- 具体的なターン番号を参照すること（例: "Turn 5で8ファイルの変更をplan modeなしに指示しています"）
- コードの品質には一切言及しないこと（それはClaude Codeの仕事）
- Claude Codeの使い方・活用法に関する提案のみ行うこと
- 提案はユーザーが今すぐ実行できるアクションに限定
- 「あなた（ユーザー）がこうすべき」の形式で書く
- ラベル(SITUATION:/OBSERVATION:/SUGGESTION:/LEVEL:)は必ず英語のまま出力。内容は日本語で書く
- LEVELは low(表示不要), info(一般), insight(非自明な発見), warning(潜在的問題), action(即時対応推奨) から選択

以下のフォーマットで正確に4行のみ出力:

SITUATION: (ユーザーが今やろうとしていること)
OBSERVATION: (Claude Codeの使い方に関する具体的な事実・パターン)
SUGGESTION: (Claude Codeの使い方を改善する具体的アクション1つ)
LEVEL: low|info|insight|warning|action`, summary, bpSection)
	}

	return fmt.Sprintf(`You are a "Claude Code usage coach".
Evaluate how effectively the user is leveraging Claude Code — NOT the quality of their code.

## Session Status
%s
%s
## Evaluation Criteria (in priority order)
1. Instruction clarity: Do user messages include file paths and expected behavior? Are they vague one-liners?
2. Plan Mode usage: Is Plan Mode used before multi-file changes? Is direction confirmed before large tasks?
3. Context management: Is auto-compact frequency too high? Are files re-read after compact? Are sessions split appropriately?
4. Tool efficiency: Are there retry loops (same operation repeated)? Long tool chains without user input?
5. Feature awareness: Is the user leveraging CLAUDE.md, slash commands, hooks, and subagents (Task)?

## Output Rules
- Reference specific turn numbers (e.g., "At Turn 5, you instructed changes to 8 files without Plan Mode")
- NEVER comment on code quality (that's Claude Code's job)
- Only suggest improvements to Claude Code USAGE and workflow
- Suggestions must be actions the user can take RIGHT NOW
- Write as "You should..." (addressing the user)
- Labels (SITUATION:/OBSERVATION:/SUGGESTION:/LEVEL:) must be in ASCII English
- LEVEL: low (nothing notable), info (general), insight (non-obvious finding), warning (potential issue), action (immediate action needed)

Output exactly 4 lines in this format:

SITUATION: (What the user is currently trying to do)
OBSERVATION: (A concrete fact about their Claude Code usage pattern)
SUGGESTION: (One specific action to improve their Claude Code usage)
LEVEL: low|info|insight|warning|action`, summary, bpSection)
}

func buildSummary(events []parser.SessionEvent, stats analyzer.Stats, activeAlerts []analyzer.Alert, sessionHealth float64, prevFeedbacks []analyzer.Feedback) string {
	var sb strings.Builder

	// Basic stats
	sb.WriteString(fmt.Sprintf("Turns: %d, Tools: %d (%.1f/turn), Elapsed: %dmin\n",
		stats.TurnCount, stats.ToolUseCount, stats.ToolsPerTurn(), int(stats.Elapsed().Minutes())))
	sb.WriteString(fmt.Sprintf("Session Health: %.0f%%\n", sessionHealth*100))
	if stats.LongestPause > 0 {
		sb.WriteString(fmt.Sprintf("Longest pause: %dmin\n", int(stats.LongestPause.Minutes())))
	}

	if len(stats.TopTools(5)) > 0 {
		var parts []string
		for _, t := range stats.TopTools(5) {
			parts = append(parts, fmt.Sprintf("%s:%d", t.Name, t.Count))
		}
		sb.WriteString("Top tools: " + strings.Join(parts, ", ") + "\n")
	}

	// Usage signals
	claudeMDUsed := false
	skillsUsed := false
	planModeUsed := false
	subagentUsed := false
	for _, ev := range events {
		if ev.Type == parser.EventToolUse {
			switch ev.ToolName {
			case "Read":
				if strings.Contains(ev.ToolInput, "CLAUDE.md") {
					claudeMDUsed = true
				}
				if strings.Contains(ev.ToolInput, ".claude/") {
					skillsUsed = true
				}
			case "Skill":
				skillsUsed = true
			case "EnterPlanMode":
				planModeUsed = true
			case "Task":
				subagentUsed = true
			}
		}
	}
	sb.WriteString(fmt.Sprintf("CLAUDE.md referenced: %v\n", claudeMDUsed))
	sb.WriteString(fmt.Sprintf(".claude/ skills/agents used: %v\n", skillsUsed))
	sb.WriteString(fmt.Sprintf("Plan Mode used: %v\n", planModeUsed))
	sb.WriteString(fmt.Sprintf("Subagent (Task) used: %v\n", subagentUsed))

	// Usage analysis hints
	hints := computeUsageHints(events, stats)
	if hints != "" {
		sb.WriteString("\n## Usage Analysis\n")
		sb.WriteString(hints)
	}

	// Active anti-pattern alerts
	if len(activeAlerts) > 0 {
		sb.WriteString("\n## Active Anti-Pattern Alerts\n")
		for _, a := range activeAlerts {
			sb.WriteString(fmt.Sprintf("- [%s] %s: %s\n",
				analyzer.PatternName(a.Pattern),
				levelLabel(a.Level),
				a.Observation))
		}
	}

	// Previous feedback (avoid repetition)
	if len(prevFeedbacks) > 0 {
		sb.WriteString("\nPrevious feedback (DO NOT repeat):\n")
		for _, fb := range prevFeedbacks {
			sb.WriteString(fmt.Sprintf("- %s | %s\n", fb.Observation, fb.Suggestion))
		}
	}

	// Turn-numbered event stream
	sb.WriteString("\n## Event Stream (turn-numbered)\n")
	turnNum := 0
	start := 0
	if len(events) > 30 {
		start = len(events) - 30
		// Count turns before the window to get accurate turn numbers
		for _, ev := range events[:start] {
			if ev.Type == parser.EventUserMessage {
				turnNum++
			}
		}
	}
	for _, ev := range events[start:] {
		switch ev.Type {
		case parser.EventUserMessage:
			turnNum++
			sb.WriteString(fmt.Sprintf("T%d U: %s\n", turnNum, parser.Truncate(ev.UserText, 100)))
		case parser.EventToolUse:
			sb.WriteString(fmt.Sprintf("   T: %s(%s)\n", ev.ToolName, parser.Truncate(ev.ToolInput, 60)))
		case parser.EventAssistantText:
			sb.WriteString(fmt.Sprintf("   A: %s\n", parser.Truncate(ev.AssistantText, 80)))
		case parser.EventCompactBoundary:
			sb.WriteString("   --- auto-compact ---\n")
		}
	}
	return sb.String()
}

// computeUsageHints analyzes events for usage quality signals.
func computeUsageHints(events []parser.SessionEvent, stats analyzer.Stats) string {
	var hints []string

	// Instruction quality: check user message lengths
	shortMsgCount := 0
	totalUserMsgs := 0
	for _, ev := range events {
		if ev.Type == parser.EventUserMessage {
			totalUserMsgs++
			if len([]rune(ev.UserText)) < 20 {
				shortMsgCount++
			}
		}
	}
	if totalUserMsgs > 2 && shortMsgCount > totalUserMsgs/2 {
		hints = append(hints, fmt.Sprintf("- %d/%d user messages are under 20 chars (vague instructions)",
			shortMsgCount, totalUserMsgs))
	}

	// Multi-file changes without plan mode: count files per burst
	planModeActive := false
	maxFilesInBurst := 0
	burstFiles := make(map[string]bool)
	worstTurn := 0
	turnNum := 0
	for _, ev := range events {
		switch ev.Type {
		case parser.EventUserMessage:
			turnNum++
			if len(burstFiles) > maxFilesInBurst {
				maxFilesInBurst = len(burstFiles)
				if !planModeActive && maxFilesInBurst >= 5 {
					worstTurn = turnNum - 1
				}
			}
			burstFiles = make(map[string]bool)
		case parser.EventToolUse:
			switch ev.ToolName {
			case "Write", "Edit":
				burstFiles[ev.ToolInput] = true
			case "EnterPlanMode":
				planModeActive = true
			case "ExitPlanMode":
				planModeActive = false
			}
		}
	}
	if len(burstFiles) > maxFilesInBurst {
		maxFilesInBurst = len(burstFiles)
	}
	if maxFilesInBurst >= 5 && !planModeActive && worstTurn > 0 {
		hints = append(hints, fmt.Sprintf("- Turn %d: %d files modified without Plan Mode",
			worstTurn, maxFilesInBurst))
	}

	// Tool burst size: max consecutive tools between user messages
	maxBurst := 0
	currentBurst := 0
	for _, ev := range events {
		if ev.Type == parser.EventUserMessage {
			if currentBurst > maxBurst {
				maxBurst = currentBurst
			}
			currentBurst = 0
		} else if ev.Type == parser.EventToolUse {
			currentBurst++
		}
	}
	if currentBurst > maxBurst {
		maxBurst = currentBurst
	}
	if maxBurst > 20 {
		hints = append(hints, fmt.Sprintf("- Max consecutive tools without user input: %d", maxBurst))
	}

	// Compact count
	compactCount := 0
	for _, ev := range events {
		if ev.Type == parser.EventCompactBoundary {
			compactCount++
		}
	}
	if compactCount >= 2 {
		hints = append(hints, fmt.Sprintf("- Session has been compacted %d times (context loss risk)", compactCount))
	}

	if len(hints) == 0 {
		return ""
	}
	return strings.Join(hints, "\n") + "\n"
}

func levelLabel(level analyzer.FeedbackLevel) string {
	switch level {
	case analyzer.LevelWarning:
		return "WARNING"
	case analyzer.LevelAction:
		return "ACTION"
	case analyzer.LevelInsight:
		return "INSIGHT"
	case analyzer.LevelInfo:
		return "INFO"
	default:
		return "INFO"
	}
}

func parseFeedbackOutput(raw string) analyzer.Feedback {
	var fb analyzer.Feedback
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "SITUATION:"); ok {
			fb.Situation = strings.TrimSpace(after)
		} else if after, ok := strings.CutPrefix(line, "OBSERVATION:"); ok {
			fb.Observation = strings.TrimSpace(after)
		} else if after, ok := strings.CutPrefix(line, "SUGGESTION:"); ok {
			fb.Suggestion = strings.TrimSpace(after)
		} else if after, ok := strings.CutPrefix(line, "LEVEL:"); ok {
			fb.Level = analyzer.ParseLevel(after)
		}
	}
	if fb.Situation == "" {
		fb.Situation = "Analyzing session..."
	}
	if fb.Observation == "" {
		fb.Observation = "Gathering session data"
	}
	if fb.Suggestion == "" {
		fb.Suggestion = "Include specific file paths in your instructions for better accuracy"
	}
	return fb
}
