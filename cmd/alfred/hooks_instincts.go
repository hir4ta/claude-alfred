package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/hir4ta/claude-alfred/internal/spec"
	"github.com/hir4ta/claude-alfred/internal/store"
)

// ---------------------------------------------------------------------------
// Instinct extraction from session.md (SessionEnd)
// ---------------------------------------------------------------------------

// extractAndSaveInstincts analyzes session.md content to detect behavioral
// patterns and saves them as instincts. Called from handleSessionEnd after
// session summary persistence.
//
// Pattern types:
//   - Decisions with rationale → workflow/code-style instincts
//   - Correction patterns ("not X, instead Y") → high-confidence preferences
//   - Recurring tool/framework choices → code-style instincts
func extractAndSaveInstincts(ctx context.Context, projectPath, taskSlug, session string) {
	if ctx.Err() != nil {
		return
	}

	st, err := store.OpenDefaultCached()
	if err != nil {
		debugf("extractInstincts: DB error: %v", err)
		return
	}

	projectHash := projectHashFromPath(projectPath)
	patterns := classifyInstinctPatterns(session)
	if len(patterns) == 0 {
		debugf("extractInstincts: no patterns found in session")
		return
	}

	saved := 0
	for _, p := range patterns {
		// Check for duplicates before inserting.
		existing, _ := st.FindDuplicateInstinct(ctx, p.trigger, p.action, projectHash)
		if existing != nil {
			// Reinforce existing instinct.
			if err := st.UpdateInstinctConfidence(ctx, existing.ID, 0.05); err != nil {
				debugf("extractInstincts: reinforce error: %v", err)
			}
			debugf("extractInstincts: reinforced existing instinct %d (+0.05)", existing.ID)
			continue
		}

		if ctx.Err() != nil {
			debugf("extractInstincts: context expired after %d saves", saved)
			return
		}

		inst := &store.Instinct{
			Trigger:       p.trigger,
			Action:        p.action,
			Confidence:    p.confidence,
			Domain:        p.domain,
			Scope:         store.ScopeProject,
			ProjectHash:   projectHash,
			SourceSession: taskSlug,
			Evidence:      p.evidence,
		}
		if _, err := st.InsertInstinct(ctx, inst); err != nil {
			debugf("extractInstincts: insert error: %v", err)
			continue
		}
		saved++
	}

	if saved > 0 {
		debugf("extractInstincts: saved %d new instincts for %s", saved, taskSlug)
	}
}

// rawInstinct is an intermediate representation before DB persistence.
type rawInstinct struct {
	trigger    string
	action     string
	confidence float64
	domain     string
	evidence   string
}

// classifyInstinctPatterns extracts behavioral patterns from session.md content.
func classifyInstinctPatterns(session string) []rawInstinct {
	cleaned := stripCompactMarkers(session)
	var patterns []rawInstinct

	// Extract from "Recent Decisions" section.
	decisions := extractSectionFallback(cleaned, "## Recent Decisions", "## Recent Decisions (last 3)")
	if decisions != "" {
		patterns = append(patterns, extractFromDecisions(decisions)...)
	}

	// Extract correction patterns from "Currently Working On" + general content.
	workingOn := extractSectionFallback(cleaned, "## Currently Working On", "## Current Position")
	if workingOn != "" {
		patterns = append(patterns, extractCorrectionPatterns(workingOn)...)
	}

	// Cap at 5 instincts per session to avoid noise.
	if len(patterns) > 5 {
		patterns = patterns[:5]
	}

	return patterns
}

// extractFromDecisions converts decision lines into instinct patterns.
// Decisions with rationale markers are high-value instincts.
func extractFromDecisions(decisions string) []rawInstinct {
	var patterns []rawInstinct
	lines := strings.Split(decisions, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Strip list markers.
		line = strings.TrimLeft(line, "0123456789.-) ")
		if len(line) < 15 {
			continue
		}

		trigger, action := splitDecisionToInstinct(line)
		if trigger == "" || action == "" {
			continue
		}

		confidence := 0.5
		domain := classifyDomain(trigger, action)

		// Boost confidence if rationale is present.
		lower := strings.ToLower(line)
		for _, marker := range rationaleMarkersForInstinct {
			if strings.Contains(lower, marker) {
				confidence = 0.65
				break
			}
		}

		patterns = append(patterns, rawInstinct{
			trigger:    trigger,
			action:     action,
			confidence: confidence,
			domain:     domain,
			evidence:   truncateStr(line, 200),
		})
	}

	return patterns
}

// rationaleMarkersForInstinct are phrases indicating a decision has reasoning.
var rationaleMarkersForInstinct = []string{
	"because", "since", "due to", "in order to",
	"ため", "なので", "から", "ので", "として",
	"rather than", "instead of", "not",
	"ではなく", "代わりに",
}

// splitDecisionToInstinct attempts to split a decision sentence into trigger + action.
// Examples:
//   - "Use table-driven tests for Go (because ...)" → trigger:"Go テスト作成時", action:"table-driven tests を使う"
//   - "Recency floor=0.5 (0.75では...)" → trigger:"recency 設定", action:"floor=0.5"
func splitDecisionToInstinct(decision string) (trigger, action string) {
	lower := strings.ToLower(decision)

	// Pattern 1: "X は Y で/に" (Japanese decision pattern)
	jpSeps := []string{" は ", " を ", " で "}
	if idx, sep := indexAnyWithLen(lower, jpSeps); idx > 0 {
		trigger = strings.TrimSpace(decision[:idx])
		action = strings.TrimSpace(decision[idx+sep:])
		// Clean up parenthetical rationale from action.
		if pIdx := strings.Index(action, "（"); pIdx > 0 {
			action = strings.TrimSpace(action[:pIdx])
		}
		if pIdx := strings.Index(action, "("); pIdx > 0 {
			action = strings.TrimSpace(action[:pIdx])
		}
		if len([]rune(trigger)) >= 2 && len([]rune(action)) >= 3 {
			return trigger, action
		}
	}

	// Pattern 2: "Use X for Y" / "Prefer X over Y"
	for _, prefix := range []string{"use ", "prefer ", "always ", "choose "} {
		if strings.HasPrefix(lower, prefix) {
			action = decision
			trigger = "関連する作業時"
			return trigger, action
		}
	}

	// Pattern 3: Contains " → " or " -> " separator
	for _, sep := range []string{" → ", " -> ", " => "} {
		if idx := strings.Index(decision, sep); idx > 0 {
			trigger = strings.TrimSpace(decision[:idx])
			action = strings.TrimSpace(decision[idx+len(sep):])
			if len([]rune(trigger)) >= 2 && len([]rune(action)) >= 2 {
				return trigger, action
			}
		}
	}

	// Pattern 4: Colon separator "X: Y"
	if idx := strings.Index(decision, ": "); idx > 3 && idx < len(decision)-3 {
		trigger = strings.TrimSpace(decision[:idx])
		action = strings.TrimSpace(decision[idx+2:])
		if len([]rune(trigger)) >= 2 && len([]rune(action)) >= 3 {
			return trigger, action
		}
	}

	return "", ""
}

// extractCorrectionPatterns detects correction/preference signals.
// Key signals: "ではなく", "instead of", "not X, Y", "X より Y"
func extractCorrectionPatterns(content string) []rawInstinct {
	var patterns []rawInstinct
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)

		for _, marker := range correctionMarkers {
			if !strings.Contains(lower, marker) {
				continue
			}
			// Use strings.Index on the original line (not lower) to get
			// correct byte offset for multi-byte characters. The marker
			// itself is always lowercase/ASCII or exact Japanese match.
			idx := strings.Index(strings.ToLower(line), marker)
			if idx < 0 {
				continue
			}
			before := strings.TrimSpace(line[:idx])
			after := strings.TrimSpace(line[idx+len(marker):])
			if len([]rune(before)) < 2 || len([]rune(after)) < 2 {
				continue
			}

			patterns = append(patterns, rawInstinct{
				trigger:    before + " の場面で",
				action:     after,
				confidence: 0.7, // Corrections are high-confidence signals.
				domain:     store.DomainPreferences,
				evidence:   truncateStr(line, 200),
			})
			break // One pattern per line.
		}
	}

	return patterns
}

var correctionMarkers = []string{
	"ではなく", "代わりに", "より",
	"instead of", "rather than", "not ",
}

// classifyDomain determines the domain of an instinct from its trigger+action text.
func classifyDomain(trigger, action string) string {
	combined := strings.ToLower(trigger + " " + action)

	domainKeywords := map[string][]string{
		store.DomainTesting:     {"test", "テスト", "assert", "mock", "coverage", "カバレッジ"},
		store.DomainGit:         {"commit", "コミット", "branch", "ブランチ", "merge", "マージ", "push", "rebase", "pull request"},
		store.DomainCodeStyle:   {"style", "format", "naming", "命名", "convention", "lint", "indent", "import"},
		store.DomainDebugging:   {"debug", "デバッグ", "log", "ログ", "error", "エラー", "trace", "breakpoint"},
		store.DomainWorkflow:    {"workflow", "ワークフロー", "deploy", "pipeline", "ci/cd", "cicd"},
		store.DomainPreferences: {"prefer", "always", "never", "好む", "常に", "絶対"},
	}

	for domain, keywords := range domainKeywords {
		for _, kw := range keywords {
			if strings.Contains(combined, kw) {
				return domain
			}
		}
	}

	return store.DomainGeneral
}

// projectHashFromPath computes a project identifier from the git remote URL.
// Returns first 12 hex chars of SHA-256(remote URL), or "local-<dir-hash>" if no remote.
// Uses a 1-second timeout to prevent git hangs from blocking the hook.
func projectHashFromPath(projectPath string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", projectPath, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err == nil {
		remote := strings.TrimSpace(string(out))
		if remote != "" {
			h := sha256.Sum256([]byte(remote))
			return fmt.Sprintf("%x", h[:6]) // 12 hex chars
		}
	}

	// Fallback: hash the project path itself.
	h := sha256.Sum256([]byte(projectPath))
	return fmt.Sprintf("local-%x", h[:6])
}

// ---------------------------------------------------------------------------
// Cross-project promotion (SessionStart)
// ---------------------------------------------------------------------------

// checkInstinctPromotion auto-promotes project-scoped instincts to global
// when they appear in 2+ projects with avg confidence >= 0.8.
func checkInstinctPromotion(ctx context.Context, st *store.Store, projectPath string) {
	projectHash := projectHashFromPath(projectPath)
	instincts, err := st.SearchInstincts(ctx, projectHash, "", 50)
	if err != nil || len(instincts) == 0 {
		return
	}

	promoted := 0
	for _, inst := range instincts {
		if inst.Scope == store.ScopeGlobal || inst.Confidence < 0.8 {
			continue
		}
		if ctx.Err() != nil {
			return
		}

		crossMatches, err := st.FindCrossProjectInstincts(ctx, inst.Trigger, projectHash, 0.6)
		if err != nil || len(crossMatches) == 0 {
			continue
		}

		// Check avg confidence across all matches (including current).
		totalConf := inst.Confidence
		for _, m := range crossMatches {
			totalConf += m.Confidence
		}
		avgConf := totalConf / float64(len(crossMatches)+1)

		if avgConf >= 0.8 {
			if err := st.PromoteInstinct(ctx, inst.ID); err == nil {
				promoted++
				debugf("instinct promotion: %q → global (avg confidence: %.2f, %d projects)",
					inst.Trigger, avgConf, len(crossMatches)+1)
			}
		}
	}
	if promoted > 0 {
		debugf("checkInstinctPromotion: promoted %d instincts to global", promoted)
	}
}

// ---------------------------------------------------------------------------
// Instinct injection (UserPromptSubmit)
// ---------------------------------------------------------------------------

// searchRelevantInstincts finds instincts relevant to the current prompt.
// Returns formatted snippet lines (max 2) for injection into additionalContext.
func searchRelevantInstincts(ctx context.Context, prompt, projectPath string, st *store.Store) []string {
	keywords := extractSearchKeywords(prompt, 6)
	if keywords == "" {
		return nil
	}

	projectHash := projectHashFromPath(projectPath)
	instincts, err := st.SearchInstinctsFTS(ctx, keywords, projectHash, 5)
	if err != nil || len(instincts) == 0 {
		return nil
	}

	// Filter by confidence threshold and limit to 2.
	var results []string
	for _, inst := range instincts {
		if inst.Confidence < 0.6 {
			continue
		}
		results = append(results, fmt.Sprintf("- [%s] When: %s → %s (confidence: %.0f%%)\n",
			inst.Domain, inst.Trigger, inst.Action, inst.Confidence*100))

		// Track injection for feedback.
		_ = st.IncrementApplied(ctx, inst.ID)

		if len(results) >= 2 {
			break
		}
	}

	if len(results) > 0 {
		debugf("UserPromptSubmit: injected %d instincts (keywords: %s)", len(results), keywords)
	}
	return results
}

// ---------------------------------------------------------------------------
// Workflow opportunity detection (UserPromptSubmit)
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Context usage monitoring (UserPromptSubmit)
// ---------------------------------------------------------------------------

// estimateContextPressure checks transcript file size as a proxy for context usage.
// Returns a hint if the context appears to be filling up.
func estimateContextPressure(ev *hookEvent) string {
	if ev.TranscriptPath == "" {
		return ""
	}
	fi, err := os.Stat(ev.TranscriptPath)
	if err != nil {
		return ""
	}
	sizeMB := float64(fi.Size()) / (1024 * 1024)

	// Heuristic: transcript > 5MB ≈ context getting full.
	// Opus 4.6 1M context ≈ ~4M chars ≈ ~8MB of JSONL transcript.
	if sizeMB > 5.0 {
		return fmt.Sprintf("Context pressure: transcript is %.1fMB. Consider /compact to free space or /clear between tasks. Performance may degrade above 70%% context usage.", sizeMB)
	}
	return ""
}

// largeTaskKeywords indicate a prompt that would benefit from /alfred:plan.
var largeTaskKeywords = []string{
	// English
	"implement", "build", "create", "develop", "design", "architect",
	"refactor", "rewrite", "migrate", "overhaul",
	"add feature", "new feature", "major change",
	// Japanese
	"実装して", "実装する", "実装したい", "作って", "作りたい", "作成して",
	"開発して", "開発する", "設計して", "設計する",
	"リファクタ", "リファクタリング", "書き直し", "移行して", "移行する",
	"機能追加", "新機能", "大きな変更",
	"全部やろう", "全部やって", "徹底的に",
}

// reviewKeywords indicate a prompt where review should be suggested.
var reviewKeywords = []string{
	"レビューして", "レビューする", "レビューお願い",
	"セルフレビュー", "見直して", "確認して",
	"review", "check my code", "look over",
	"コミットして", "commit",
}

// detectWorkflowOpportunity checks if the prompt suggests a workflow action
// and returns an additionalContext hint, or "" if none.
//
// Multi-signal detection:
//   - Keywords (large task / review / ingest intent)
//   - Prompt length (>500 chars = likely complex)
//   - Multiple component mentions (DB + API + UI = multi-file)
//   - File reference count (many @files = context-heavy)
//   - Active spec presence
func detectWorkflowOpportunity(prompt, projectPath string) string {
	lower := strings.ToLower(prompt)
	promptRunes := len([]rune(prompt))

	// Check for review intent.
	for _, kw := range reviewKeywords {
		if strings.Contains(lower, kw) {
			return "Workflow suggestion: Consider using /alfred:review for a thorough multi-agent code review (security, logic, design)."
		}
	}

	// Check for ingest intent (many files or reference materials).
	if detectIngestOpportunity(lower) {
		return "Workflow suggestion: Looks like you're providing reference materials. Consider /alfred:ingest to structure and persist this context (survives compaction and session boundaries)."
	}

	// Multi-signal complexity score (0-10).
	complexity := 0

	// Signal 1: Keywords.
	for _, kw := range largeTaskKeywords {
		if strings.Contains(lower, kw) {
			complexity += 3
			break
		}
	}

	// Signal 2: Prompt length.
	if promptRunes > 500 {
		complexity += 2
	} else if promptRunes > 200 {
		complexity += 1
	}

	// Signal 3: Multiple component mentions.
	components := 0
	for _, comp := range componentKeywords {
		if strings.Contains(lower, comp) {
			components++
		}
	}
	if components >= 3 {
		complexity += 3
	} else if components >= 2 {
		complexity += 2
	}

	// Signal 4: File references.
	atMentions := strings.Count(prompt, "@")
	if atMentions >= 3 {
		complexity += 2
	}

	if complexity < 4 {
		return ""
	}

	hasSpec := false
	if projectPath != "" {
		if _, err := spec.ReadActive(projectPath); err == nil {
			hasSpec = true
		}
	}

	if hasSpec {
		return "Workflow note: Active spec detected. Use the spec MCP tool (action=status) to check current state before starting. After completion, consider /alfred:review."
	}
	return "Workflow suggestion: This looks like a substantial task (complexity signals detected). Consider using /alfred:plan to create a structured spec — or if the task is clear enough, ask the user if they want to dive straight in."
}

// componentKeywords indicate multi-component tasks.
var componentKeywords = []string{
	"database", "db", "api", "ui", "frontend", "backend", "server", "client",
	"auth", "test", "migration", "schema", "endpoint", "component", "page",
	"データベース", "画面", "サーバー", "クライアント", "認証", "テスト",
	"マイグレーション", "スキーマ", "エンドポイント", "コンポーネント",
}

// ingestKeywords indicate reference material processing opportunity.
var ingestKeywords = []string{
	".csv", ".txt", ".pdf", ".xlsx", ".json",
	"資料", "ドキュメント", "読んで", "読み込んで", "キャッチアップ",
	"reference", "document", "read these", "review this", "catch up",
}

// detectIngestOpportunity checks if the prompt indicates reference material processing.
// Uses multiple signals: file extensions, @ mentions, keywords, prompt patterns.
func detectIngestOpportunity(lower string) bool {
	signals := 0

	// Signal 1: Multiple file references.
	fileExts := []string{".csv", ".txt", ".pdf", ".xlsx", ".json", ".md", ".doc", ".yaml", ".yml", ".xml", ".log", ".sql"}
	fileCount := 0
	for _, ext := range fileExts {
		fileCount += strings.Count(lower, ext)
	}
	if fileCount >= 2 {
		signals += 2
	} else if fileCount >= 1 {
		signals++
	}

	// Signal 2: Multiple @ file references.
	if strings.Count(lower, "@") >= 3 {
		signals += 2
	}

	// Signal 3: Ingest-related keywords.
	for _, kw := range ingestKeywords {
		if strings.Contains(lower, kw) {
			signals++
			break
		}
	}

	// Signal 4: Long paste-like content (>1000 chars with lots of newlines suggests pasted data).
	if len(lower) > 1000 && strings.Count(lower, "\n") > 10 {
		signals++
	}

	return signals >= 2
}

// indexAnyWithLen returns the index and matched length of the first occurrence
// of any of the substrings, or (-1, 0) if none found.
func indexAnyWithLen(s string, subs []string) (int, int) {
	bestIdx := -1
	bestLen := 0
	for _, sub := range subs {
		if idx := strings.Index(s, sub); idx >= 0 && (bestIdx < 0 || idx < bestIdx) {
			bestIdx = idx
			bestLen = len(sub)
		}
	}
	return bestIdx, bestLen
}
