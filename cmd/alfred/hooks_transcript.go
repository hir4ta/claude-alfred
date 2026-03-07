package main

import (
	"encoding/json"
	"sort"
	"strings"
)

// ---------------------------------------------------------------------------
// Decision extraction from transcript
// ---------------------------------------------------------------------------

// trivialVerbs are verbs that follow decision keywords but indicate
// routine actions rather than real design decisions.
var trivialVerbs = []string{
	"read ", "check ", "look ", "run ", "open ", "try ", "start ",
	"continue ", "proceed ", "skip ", "move ", "fix ", "update ",
	"install ", "build ", "test ", "debug ", "print ", "log ",
	"add ", "remove ", "delete ", "rename ", "import ", "copy ",
	"format ", "lint ", "commit ", "push ", "pull ", "merge ",
	"revert ", "rebase ",
}

// rationaleMarkers indicate the sentence contains a reason/justification,
// which strongly suggests a real design decision.
var rationaleMarkers = []string{
	"because ", "since ", "due to ", "given that ", "in order to ",
	"so that ", "for better ", "to ensure ", "to avoid ", "to reduce ",
	"to improve ", "to support ", "for the sake of ",
}

// alternativeMarkers indicate the sentence compares options,
// which is a strong signal for a design decision.
var alternativeMarkers = []string{
	" over ", " instead of ", " rather than ", " vs ", " versus ",
	" compared to ", " as opposed to ",
}

// architectureTerms boost confidence when the sentence mentions design concepts.
var architectureTerms = []string{
	"architecture", "pattern", "approach", "strategy", "trade-off",
	"tradeoff", "schema", "interface", "protocol", "abstraction",
	"design", "api ", "migration", "infrastructure",
}

// scoreDecisionConfidence returns a confidence score (0.0-1.0) for whether
// a sentence represents a real design decision vs an implementation action.
func scoreDecisionConfidence(sentence string) float64 {
	lower := strings.ToLower(sentence)
	score := 0.4 // base score for having a decision keyword

	// Rationale clause: strong positive signal.
	for _, marker := range rationaleMarkers {
		if strings.Contains(lower, marker) {
			score += 0.25
			break
		}
	}

	// Alternative comparison: strong positive signal.
	for _, marker := range alternativeMarkers {
		if strings.Contains(lower, marker) {
			score += 0.3
			break
		}
	}

	// Architecture vocabulary: moderate positive signal.
	for _, term := range architectureTerms {
		if strings.Contains(lower, term) {
			score += 0.15
			break
		}
	}

	// Code artifact penalty: backticks, file paths, camelCase.
	if strings.Contains(sentence, "`") {
		score -= 0.15
	}
	if strings.Contains(sentence, "/") && strings.Contains(sentence, ".") {
		// Likely a file path like "src/main.go".
		score -= 0.1
	}

	// Hedging words penalty: "just", "simply", "quickly".
	for _, hedge := range []string{"just ", "simply ", "quickly ", "also "} {
		if strings.Contains(lower, hedge) {
			score -= 0.1
			break
		}
	}

	return min(max(score, 0), 1.0)
}

// isTrivialDecision returns true if the sentence describes a routine action
// rather than a meaningful design/architecture decision.
func isTrivialDecision(sentence string) bool {
	lower := strings.ToLower(sentence)
	for _, v := range trivialVerbs {
		// Check if a trivial verb follows a decision keyword.
		for _, kw := range []string{"decided to ", "chose to ", "going to "} {
			if strings.Contains(lower, kw+v) {
				return true
			}
		}
	}
	// Too short to be a real decision.
	if len(sentence) < 30 {
		return true
	}
	return false
}

// extractDecisionsFromTranscript scans the transcript for meaningful design decisions
// from the assistant. Uses keyword matching + structured pattern detection + trivial filtering.
func extractDecisionsFromTranscript(transcriptPath string) []string {
	data, err := readFileTail(transcriptPath, 64*1024)
	if err != nil {
		return nil
	}

	// Keyword patterns that indicate design decisions (not routine actions).
	decisionKeywords := []string{
		"decided to ", "chose ", "going with ", "selected ",
		"decision: ", "we'll use ", "opting for ",
		"settled on ", "choosing ", "picked ",
	}

	// Structured patterns from spec format or explicit decision markers.
	structuredPrefixes := []string{
		"**chosen:**", "**decision:**", "**selected:**",
		"- chosen: ", "- decision: ", "- selected: ",
	}

	type scoredDecision struct {
		text       string
		confidence float64
	}
	var decisions []scoredDecision
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line[0] != '{' {
			continue
		}
		var entry transcriptEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		role := entry.Role
		if role == "" {
			role = entry.Message.Role
		}
		if role != "assistant" && entry.Type != "assistant" {
			continue
		}

		text := extractTextContent(entry)
		if text == "" {
			continue
		}
		textLower := strings.ToLower(text)

		// Strategy 1: Structured patterns (high confidence = 0.9).
		for _, prefix := range structuredPrefixes {
			idx := strings.Index(textLower, prefix)
			if idx < 0 {
				continue
			}
			rest := strings.TrimSpace(text[idx+len(prefix):])
			end := strings.IndexAny(rest, "\n")
			if end < 0 {
				end = min(len(rest), 200)
			}
			value := strings.TrimSpace(rest[:end])
			if len(value) > 5 {
				decisions = append(decisions, scoredDecision{value, 0.9})
			}
			break
		}

		// Strategy 2: Keyword matching with confidence scoring.
		for _, kw := range decisionKeywords {
			idx := strings.Index(textLower, kw)
			if idx < 0 {
				continue
			}
			start := strings.LastIndexAny(text[:idx], ".!?\n") + 1
			end := strings.IndexAny(text[idx:], ".!?\n")
			if end < 0 {
				end = min(len(text)-idx, 200)
			}
			sentence := strings.TrimSpace(text[start : idx+end])
			if len(sentence) > 10 && len(sentence) < 300 && !isTrivialDecision(sentence) {
				conf := scoreDecisionConfidence(sentence)
				if conf >= 0.4 {
					decisions = append(decisions, scoredDecision{sentence, conf})
				}
			}
			break // one decision per entry
		}
	}

	// Deduplicate, keeping the highest confidence version.
	seen := make(map[string]int) // key -> index in unique
	var unique []scoredDecision
	for _, d := range decisions {
		key := strings.ToLower(d.text)
		if len(key) > 80 {
			key = key[:80]
		}
		if idx, ok := seen[key]; ok {
			if d.confidence > unique[idx].confidence {
				unique[idx] = d
			}
		} else {
			seen[key] = len(unique)
			unique = append(unique, d)
		}
	}

	// Sort by confidence descending, keep last 5.
	sort.Slice(unique, func(i, j int) bool {
		return unique[i].confidence > unique[j].confidence
	})
	if len(unique) > 5 {
		unique = unique[:5]
	}

	result := make([]string, len(unique))
	for i, d := range unique {
		result[i] = d.text
	}
	return result
}

// ---------------------------------------------------------------------------
// Transcript context extraction
// ---------------------------------------------------------------------------

// extractTranscriptContext reads the tail of a conversation transcript and
// extracts the most valuable context: recent user messages, assistant summaries,
// and tool errors that would otherwise be lost during compaction.
func extractTranscriptContext(transcriptPath string) string {
	// Read last 64KB of transcript (conversation can be huge).
	data, err := readFileTail(transcriptPath, 64*1024)
	if err != nil {
		debugf("PreCompact: read transcript error: %v", err)
		return ""
	}

	lines := strings.Split(string(data), "\n")

	var userMessages []string
	var assistantSummaries []string
	var toolErrors []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line[0] != '{' {
			continue
		}
		var entry transcriptEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		text := extractTextContent(entry)
		if text == "" {
			continue
		}

		switch {
		case entry.Type == "human" || entry.Role == "user" ||
			(entry.Message.Role == "user"):
			// Keep last 5 user messages.
			userMessages = append(userMessages, truncateStr(text, 200))
			if len(userMessages) > 5 {
				userMessages = userMessages[len(userMessages)-5:]
			}
		case entry.Type == "assistant" || entry.Role == "assistant" ||
			(entry.Message.Role == "assistant"):
			// Keep last 3 assistant summaries (first 150 chars only).
			summary := truncateStr(text, 150)
			assistantSummaries = append(assistantSummaries, summary)
			if len(assistantSummaries) > 3 {
				assistantSummaries = assistantSummaries[len(assistantSummaries)-3:]
			}
		case entry.Type == "tool_error" || entry.Type == "error":
			toolErrors = append(toolErrors, truncateStr(text, 150))
			if len(toolErrors) > 3 {
				toolErrors = toolErrors[len(toolErrors)-3:]
			}
		}
	}

	var buf strings.Builder
	if len(userMessages) > 0 {
		buf.WriteString("Recent user requests:\n")
		for _, m := range userMessages {
			buf.WriteString("- " + m + "\n")
		}
	}
	if len(assistantSummaries) > 0 {
		buf.WriteString("Recent assistant actions:\n")
		for _, s := range assistantSummaries {
			buf.WriteString("- " + s + "\n")
		}
	}
	if len(toolErrors) > 0 {
		buf.WriteString("Recent errors (dead ends):\n")
		for _, e := range toolErrors {
			buf.WriteString("- " + e + "\n")
		}
	}
	return buf.String()
}
