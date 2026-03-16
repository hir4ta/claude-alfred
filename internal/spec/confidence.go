package spec

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ConfidenceSummary holds parsed confidence statistics for a spec file.
type ConfidenceSummary struct {
	Avg            float64          `json:"avg"`
	Total          int              `json:"total_items"`
	LowCount       int              `json:"low_items"`
	Items          []ConfidenceItem `json:"items,omitempty"`
	Warnings       []string         `json:"low_confidence_warnings,omitempty"`
	GroundingDist  map[string]int   `json:"grounding_distribution,omitempty"`
	GroundingWarns []string         `json:"grounding_warnings,omitempty"`
}

// ConfidenceItem holds a single confidence annotation.
type ConfidenceItem struct {
	Section   string `json:"section"`
	Score     int    `json:"score"`
	Source    string `json:"source,omitempty"`
	Grounding string `json:"grounding,omitempty"`
}

// ConfidenceRe matches confidence annotations in spec files.
var ConfidenceRe = regexp.MustCompile(`<!--\s*confidence:\s*(\d{1,2})(?:\s*\|\s*source:\s*([\w][\w-]*))?(?:\s*\|\s*grounding:\s*([\w]+))?\s*-->`)

// ValidGroundings defines the accepted grounding levels.
var ValidGroundings = map[string]bool{
	"verified": true, "reviewed": true, "inferred": true, "speculative": true,
}

// ParseConfidence extracts confidence annotations from spec file content.
func ParseConfidence(content string) ConfidenceSummary {
	lines := strings.Split(content, "\n")
	var items []ConfidenceItem
	var groundingWarns []string
	currentSection := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			currentSection = strings.TrimPrefix(trimmed, "## ")
			if idx := strings.Index(currentSection, "<!--"); idx > 0 {
				currentSection = strings.TrimSpace(currentSection[:idx])
			}
		}

		matches := ConfidenceRe.FindStringSubmatch(trimmed)
		if len(matches) < 2 {
			continue
		}
		score, err := strconv.Atoi(matches[1])
		if err != nil || score < 1 || score > 10 {
			continue
		}
		section := currentSection
		if section == "" {
			section = "(unnamed)"
		}
		source := ""
		if len(matches) >= 3 {
			source = matches[2]
		}
		grounding := ""
		if len(matches) >= 4 && matches[3] != "" {
			if ValidGroundings[matches[3]] {
				grounding = matches[3]
			} else {
				groundingWarns = append(groundingWarns, fmt.Sprintf("unknown grounding %q in section: %s", matches[3], section))
			}
		}
		items = append(items, ConfidenceItem{Section: section, Score: score, Source: source, Grounding: grounding})
	}

	if len(items) == 0 {
		return ConfidenceSummary{}
	}

	total := 0
	lowCount := 0
	var warnings []string
	groundingDist := map[string]int{}
	for _, item := range items {
		total += item.Score
		if item.Score <= 5 {
			lowCount++
			if item.Source == "assumption" {
				warnings = append(warnings, item.Section)
			}
		}
		if item.Grounding != "" {
			groundingDist[item.Grounding]++
		}
		if item.Grounding == "speculative" && item.Score > 5 {
			groundingWarns = append(groundingWarns, fmt.Sprintf("high confidence (%d) with speculative grounding in section: %s", item.Score, item.Section))
		}
	}

	return ConfidenceSummary{
		Avg:            float64(total) / float64(len(items)),
		Total:          len(items),
		LowCount:       lowCount,
		Items:          items,
		Warnings:       warnings,
		GroundingDist:  groundingDist,
		GroundingWarns: groundingWarns,
	}
}
