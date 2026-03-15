package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hir4ta/claude-alfred/internal/spec"
)

// handlePostToolUse fires after a tool executes.
// Two responsibilities:
//  1. On Bash failure: search memory for similar errors and inject solutions.
//  2. On Bash success: check if the command completes a Next Steps item in session.md.
func handlePostToolUse(ctx context.Context, ev *hookEvent) {
	if ev.ToolName != "Bash" {
		return
	}

	// Parse tool_input to get the command.
	var input struct {
		Command string `json:"command"`
	}
	_ = json.Unmarshal(ev.ToolInput, &input) // best-effort

	// Parse tool_response to check for errors.
	var resp struct {
		Stdout   string `json:"stdout"`
		Stderr   string `json:"stderr"`
		ExitCode int    `json:"exitCode"`
	}
	if err := json.Unmarshal(ev.ToolResponse, &resp); err != nil {
		return
	}

	// On success: try to auto-check Next Steps.
	if resp.ExitCode == 0 {
		tryAutoCheckNextSteps(ctx, ev.ProjectPath, input.Command, resp.Stdout)
		return
	}

	// Extract error keywords from stderr (or stdout if stderr is empty).
	errorText := resp.Stderr
	if errorText == "" {
		errorText = resp.Stdout
	}
	if len(errorText) > 2000 {
		errorText = errorText[:2000]
	}

	keywords := extractErrorKeywords(errorText)
	if len(keywords) == 0 {
		return
	}

	// Search memory for related past errors.
	query := strings.Join(keywords, " ")
	st, err := openStore()
	if err != nil {
		return
	}

	docs, err := st.SearchMemoriesFTS(ctx, query, 2)
	if err != nil || len(docs) == 0 {
		return
	}

	var buf strings.Builder
	buf.WriteString("Related past experience for this error:\n")
	for _, d := range docs {
		snippet := safeSnippet(d.Content, 300)
		buf.WriteString(fmt.Sprintf("- [%s] %s\n", d.SectionPath, snippet))
	}

	emitAdditionalContext("PostToolUse", buf.String())
}

// extractErrorKeywords pulls meaningful terms from error output.
// Looks for common error patterns: package names, function names, error types.
func extractErrorKeywords(text string) []string {
	// Take first 5 lines of error (most relevant).
	lines := strings.Split(text, "\n")
	if len(lines) > 5 {
		lines = lines[:5]
	}

	seen := make(map[string]bool)
	var keywords []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Extract words that look meaningful (4+ chars, not common noise).
		for _, word := range strings.Fields(line) {
			// Clean punctuation.
			word = strings.Trim(word, ".:;,()[]{}\"'`")
			lower := strings.ToLower(word)
			if len(lower) < 4 || isNoiseWord(lower) || seen[lower] {
				continue
			}
			seen[lower] = true
			keywords = append(keywords, lower)
			if len(keywords) >= 8 {
				return keywords
			}
		}
	}
	return keywords
}

// isNoiseWord returns true for common words that don't help search.
func isNoiseWord(w string) bool {
	noise := map[string]bool{
		"error": true, "fatal": true, "failed": true, "cannot": true,
		"could": true, "would": true, "should": true, "that": true,
		"this": true, "with": true, "from": true, "have": true,
		"line": true, "file": true, "exit": true, "code": true,
		"status": true, "expected": true, "unexpected": true,
	}
	return noise[w]
}

// actionSignals maps command patterns to completion signal words.
// When a Bash command matches a pattern, its signal words are used
// to match against Next Steps items.
var actionSignals = []struct {
	cmdContains string   // substring to match in the command
	signals     []string // words that indicate what was accomplished
}{
	{"git commit", []string{"commit", "コミット"}},
	{"git push", []string{"push", "プッシュ"}},
	{"go test", []string{"test", "テスト"}},
	{"go vet", []string{"vet", "lint", "静的解析"}},
	{"go install", []string{"install", "build", "ビルド", "インストール"}},
	{"go build", []string{"build", "ビルド"}},
	{"npm test", []string{"test", "テスト"}},
	{"npm run build", []string{"build", "ビルド"}},
	{"gh pr create", []string{"pr", "pull request", "プルリクエスト"}},
}

// tryAutoCheckNextSteps checks if a successful Bash command completes
// any Next Steps item in session.md and marks it as done.
func tryAutoCheckNextSteps(ctx context.Context, projectPath, command, stdout string) {
	if projectPath == "" || command == "" {
		return
	}

	// Read active task's session.md.
	taskSlug, err := spec.ReadActive(projectPath)
	if err != nil {
		return
	}
	sd := &spec.SpecDir{ProjectPath: projectPath, TaskSlug: taskSlug}
	session, err := sd.ReadFile(spec.FileSession)
	if err != nil {
		return
	}

	nextSteps := extractSection(session, "## Next Steps")
	if nextSteps == "" || !strings.Contains(nextSteps, "- [ ] ") {
		return
	}

	// Build context text from command + stdout + action signals.
	cmdLower := strings.ToLower(command)
	var contextBuf strings.Builder
	contextBuf.WriteString(cmdLower)
	contextBuf.WriteByte(' ')

	// Add action-specific signal words.
	for _, sig := range actionSignals {
		if strings.Contains(cmdLower, sig.cmdContains) {
			for _, s := range sig.signals {
				contextBuf.WriteString(s)
				contextBuf.WriteByte(' ')
			}
		}
	}

	// Add first 500 chars of stdout (may contain useful context like commit messages).
	if len(stdout) > 500 {
		stdout = stdout[:500]
	}
	contextBuf.WriteString(strings.ToLower(stdout))
	contextText := contextBuf.String()

	// Check each unchecked item against the context.
	lines := strings.Split(nextSteps, "\n")
	updated := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "- [ ] ") {
			continue
		}
		itemText := strings.TrimPrefix(trimmed, "- [ ] ")
		if isStepMatchedByAction(itemText, contextText) {
			lines[i] = strings.Replace(line, "- [ ] ", "- [x] ", 1)
			updated = true
		}
	}

	if !updated {
		return
	}

	// Write updated session.md.
	updatedNextSteps := strings.Join(lines, "\n")
	updatedSession := replaceSection(session, "## Next Steps", updatedNextSteps)
	if err := sd.WriteFile(ctx, spec.FileSession, updatedSession); err != nil {
		return
	}

	// Check if all steps are now done → auto-complete task.
	if allNextStepsCompleted(updatedNextSteps) {
		autoCompleteTask(projectPath, taskSlug, updatedSession)
	}
}

// stepTokenDelimiters splits on whitespace and common delimiters
// (including full-width Japanese punctuation) for better tokenization.
var stepTokenDelimiters = strings.NewReplacer(
	"（", " ", "）", " ", "＆", " ", "、", " ",
	"(", " ", ")", " ", "&", " ", "/", " ",
	"—", " ", ":", " ", "：", " ",
)

// isStepMatchedByAction checks if a Next Steps item text matches
// the action context (command + signals + stdout).
// Splits on whitespace + delimiters, requires 50%+ of 3+ rune tokens to match.
func isStepMatchedByAction(itemText, contextText string) bool {
	normalized := stepTokenDelimiters.Replace(strings.ToLower(itemText))
	var tokens []string
	for _, w := range strings.Fields(normalized) {
		if len([]rune(w)) >= 3 {
			tokens = append(tokens, w)
		}
	}
	if len(tokens) == 0 {
		return false
	}
	hits := 0
	for _, w := range tokens {
		if strings.Contains(contextText, w) {
			hits++
		}
	}
	return float64(hits)/float64(len(tokens)) >= 0.5
}

// replaceSection replaces the content under a ## heading with new content.
func replaceSection(content, heading, newBody string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inSection := false
	replaced := false
	for _, line := range lines {
		if line == heading || strings.HasPrefix(line, heading+" ") {
			inSection = true
			result = append(result, line)
			result = append(result, newBody)
			replaced = true
			continue
		}
		if inSection {
			if strings.HasPrefix(line, "## ") {
				inSection = false
				result = append(result, line)
			}
			// Skip old content in section.
			continue
		}
		result = append(result, line)
	}
	if !replaced {
		return content
	}
	return strings.Join(result, "\n")
}

