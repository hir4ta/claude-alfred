package spec

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidationCheck represents a single validation result.
type ValidationCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "pass" or "fail"
	Message string `json:"message"`
}

// ValidationReport holds the complete validation result.
type ValidationReport struct {
	TaskSlug string            `json:"task_slug"`
	Size     SpecSize          `json:"size"`
	SpecType SpecType          `json:"spec_type"`
	Checks   []ValidationCheck `json:"checks"`
	Summary  string            `json:"summary"`
}

// Validate performs structural completeness checking on a spec.
// It is progressive: checks what exists and reports pass/fail per check.
func Validate(sd *SpecDir, size SpecSize, specType SpecType) (*ValidationReport, error) {
	report := &ValidationReport{
		TaskSlug: sd.TaskSlug,
		Size:     size,
		SpecType: specType,
	}

	// Determine the primary file (requirements.md or bugfix.md).
	primaryFile := FileRequirements
	if specType == TypeBugfix {
		primaryFile = FileBugfix
	}

	// 1. required_sections: ## Goal (or ## Bug Summary for bugfix) present in primary file.
	report.Checks = append(report.Checks, checkRequiredSections(sd, primaryFile, specType))

	// 2. min_fr_count: FR count by size.
	report.Checks = append(report.Checks, checkMinFRCount(sd, primaryFile, size, specType))

	// 3. traceability_fr_to_task: FR-N mapped to T-N.N in design.md.
	if c := checkFRToTask(sd, primaryFile, size); c != nil {
		report.Checks = append(report.Checks, *c)
	}

	// 4. traceability_task_to_fr: T-N.N references valid FR-N in tasks.md.
	if c := checkTaskToFR(sd, primaryFile, size); c != nil {
		report.Checks = append(report.Checks, *c)
	}

	// 5. confidence_annotations: at least one confidence annotation.
	report.Checks = append(report.Checks, checkConfidenceAnnotations(sd, primaryFile))

	// 6. closing_wave: "## Wave: Closing" present in tasks.md.
	report.Checks = append(report.Checks, checkClosingWave(sd))

	// 7. design_fr_references: FR-N in design.md must exist in primary file.
	if c := checkDesignFRReferences(sd, primaryFile, size); c != nil {
		report.Checks = append(report.Checks, *c)
	}

	// 8. testspec_fr_references: FR-N in test-specs.md must exist in primary file.
	if c := checkTestSpecFRReferences(sd, primaryFile); c != nil {
		report.Checks = append(report.Checks, *c)
	}

	// 9. nfr_traceability: NFR-N in primary mapped in design.md (L/XL only).
	if c := checkNFRTraceability(sd, primaryFile, size); c != nil {
		report.Checks = append(report.Checks, *c)
	}

	// 10. gherkin_syntax: ```gherkin blocks must contain Given+When+Then.
	if c := checkGherkinSyntax(sd); c != nil {
		report.Checks = append(report.Checks, *c)
	}

	// 11. orphan_tests: TS-N.N source annotations must reference defined FRs.
	if c := checkOrphanTests(sd, primaryFile); c != nil {
		report.Checks = append(report.Checks, *c)
	}

	// 12. orphan_tasks: task Requirements FR-N must reference defined FRs.
	if c := checkOrphanTasks(sd, primaryFile); c != nil {
		report.Checks = append(report.Checks, *c)
	}

	// Summary.
	passed := 0
	for _, c := range report.Checks {
		if c.Status == "pass" {
			passed++
		}
	}
	report.Summary = fmt.Sprintf("%d/%d checks passed", passed, len(report.Checks))

	return report, nil
}

// checkRequiredSections checks that the primary file has the expected heading.
func checkRequiredSections(sd *SpecDir, primaryFile SpecFile, specType SpecType) ValidationCheck {
	content, err := sd.ReadFile(primaryFile)
	if err != nil {
		return ValidationCheck{
			Name:    "required_sections",
			Status:  "fail",
			Message: fmt.Sprintf("%s not found", primaryFile),
		}
	}

	heading := "## Goal"
	if specType == TypeBugfix {
		heading = "## Bug Summary"
	}

	if strings.Contains(content, heading) {
		return ValidationCheck{
			Name:    "required_sections",
			Status:  "pass",
			Message: fmt.Sprintf("%s contains %s", primaryFile, heading),
		}
	}
	return ValidationCheck{
		Name:    "required_sections",
		Status:  "fail",
		Message: fmt.Sprintf("%s missing %s section", primaryFile, heading),
	}
}

var frPattern = regexp.MustCompile(`### FR-\d+`)

// checkMinFRCount checks the minimum number of FRs based on size.
func checkMinFRCount(sd *SpecDir, primaryFile SpecFile, size SpecSize, specType SpecType) ValidationCheck {
	minFR := 1
	switch size {
	case SizeM:
		minFR = 3
	case SizeL, SizeXL:
		minFR = 5
	}

	// For bugfix specs, check bugfix.md sections instead (they don't use FR-N).
	if specType == TypeBugfix {
		content, err := sd.ReadFile(primaryFile)
		if err != nil {
			return ValidationCheck{
				Name:    "min_fr_count",
				Status:  "fail",
				Message: fmt.Sprintf("%s not found", primaryFile),
			}
		}
		// Bugfix uses sections instead of FR-N identifiers.
		// Count ## sections as requirements (Bug Summary, etc.).
		sections := strings.Count(content, "\n## ")
		if sections >= minFR {
			return ValidationCheck{
				Name:    "min_fr_count",
				Status:  "pass",
				Message: fmt.Sprintf("%d sections found (minimum %d for size %s)", sections, minFR, size),
			}
		}
		return ValidationCheck{
			Name:    "min_fr_count",
			Status:  "fail",
			Message: fmt.Sprintf("%d sections found, minimum %d for size %s", sections, minFR, size),
		}
	}

	content, err := sd.ReadFile(primaryFile)
	if err != nil {
		return ValidationCheck{
			Name:    "min_fr_count",
			Status:  "fail",
			Message: fmt.Sprintf("%s not found", primaryFile),
		}
	}

	frCount := len(frPattern.FindAllString(content, -1))
	if frCount >= minFR {
		return ValidationCheck{
			Name:    "min_fr_count",
			Status:  "pass",
			Message: fmt.Sprintf("%d FRs found (minimum %d for size %s)", frCount, minFR, size),
		}
	}
	return ValidationCheck{
		Name:    "min_fr_count",
		Status:  "fail",
		Message: fmt.Sprintf("%d FRs found, minimum %d for size %s", frCount, minFR, size),
	}
}

var frIDPattern = regexp.MustCompile(`FR-(\d+)`)
var nfrIDPattern = regexp.MustCompile(`NFR-(\d+)`)
var taskIDPattern = regexp.MustCompile(`T-(\d+)\.(\d+)`)
var gherkinBlockPattern = regexp.MustCompile("(?s)```gherkin\\s*\\n(.*?)```")
var gherkinGivenPattern = regexp.MustCompile(`(?m)^\s*Given\s+`)
var gherkinWhenPattern = regexp.MustCompile(`(?m)^\s*When\s+`)
var gherkinThenPattern = regexp.MustCompile(`(?m)^\s*Then\s+`)
var sourceCommentPattern = regexp.MustCompile(`<!--\s*source:\s*FR-(\d+)`)
var taskReqPattern = regexp.MustCompile(`Requirements:\s*(FR-\d+(?:\s*,\s*FR-\d+)*)`)

// checkFRToTask checks that FRs in the primary file are mapped to tasks in design.md.
// Skipped if no design.md exists.
func checkFRToTask(sd *SpecDir, primaryFile SpecFile, size SpecSize) *ValidationCheck {
	if size == SizeS {
		return nil // skip for S-size (no design.md)
	}

	design, err := sd.ReadFile(FileDesign)
	if err != nil {
		return nil // skip if no design.md
	}

	primary, err := sd.ReadFile(primaryFile)
	if err != nil {
		return nil
	}

	// Extract FR-N from primary file.
	frIDs := frIDPattern.FindAllString(primary, -1)
	if len(frIDs) == 0 {
		return nil // no FRs to check
	}

	// Check which FRs appear in design.md traceability.
	unmapped := []string{}
	for _, fr := range frIDs {
		if !strings.Contains(design, fr) {
			unmapped = append(unmapped, fr)
		}
	}

	// Deduplicate.
	seen := map[string]bool{}
	deduped := []string{}
	for _, fr := range unmapped {
		if !seen[fr] {
			seen[fr] = true
			deduped = append(deduped, fr)
		}
	}

	if len(deduped) == 0 {
		return &ValidationCheck{
			Name:    "traceability_fr_to_task",
			Status:  "pass",
			Message: "all FRs mapped in design.md",
		}
	}
	return &ValidationCheck{
		Name:    "traceability_fr_to_task",
		Status:  "fail",
		Message: fmt.Sprintf("unmapped FRs in design.md: %s", strings.Join(deduped, ", ")),
	}
}

// checkTaskToFR checks that tasks reference valid FRs.
// Skipped if no tasks.md exists.
func checkTaskToFR(sd *SpecDir, primaryFile SpecFile, size SpecSize) *ValidationCheck {
	tasks, err := sd.ReadFile(FileTasks)
	if err != nil {
		return nil // skip if no tasks.md
	}

	primary, err := sd.ReadFile(primaryFile)
	if err != nil {
		return nil
	}

	// Extract FR-N from primary file.
	frSet := map[string]bool{}
	for _, m := range frIDPattern.FindAllString(primary, -1) {
		frSet[m] = true
	}
	if len(frSet) == 0 {
		return nil // no FRs to validate against
	}

	// Find FR-N references in tasks.md.
	taskFRs := frIDPattern.FindAllString(tasks, -1)
	invalid := []string{}
	seen := map[string]bool{}
	for _, fr := range taskFRs {
		if !frSet[fr] && !seen[fr] {
			seen[fr] = true
			invalid = append(invalid, fr)
		}
	}

	if len(invalid) == 0 {
		return &ValidationCheck{
			Name:    "traceability_task_to_fr",
			Status:  "pass",
			Message: "all task FR references are valid",
		}
	}
	return &ValidationCheck{
		Name:    "traceability_task_to_fr",
		Status:  "fail",
		Message: fmt.Sprintf("invalid FR references in tasks.md: %s", strings.Join(invalid, ", ")),
	}
}

var confidencePattern = regexp.MustCompile(`<!--\s*confidence:\s*\d`)

// checkConfidenceAnnotations checks that at least one confidence annotation exists.
func checkConfidenceAnnotations(sd *SpecDir, primaryFile SpecFile) ValidationCheck {
	content, err := sd.ReadFile(primaryFile)
	if err != nil {
		return ValidationCheck{
			Name:    "confidence_annotations",
			Status:  "fail",
			Message: fmt.Sprintf("%s not found", primaryFile),
		}
	}

	if confidencePattern.MatchString(content) {
		return ValidationCheck{
			Name:    "confidence_annotations",
			Status:  "pass",
			Message: "confidence annotations found",
		}
	}
	return ValidationCheck{
		Name:    "confidence_annotations",
		Status:  "fail",
		Message: fmt.Sprintf("no confidence annotations in %s", primaryFile),
	}
}

// checkClosingWave checks that tasks.md contains a closing wave.
func checkClosingWave(sd *SpecDir) ValidationCheck {
	content, err := sd.ReadFile(FileTasks)
	if err != nil {
		return ValidationCheck{
			Name:    "closing_wave",
			Status:  "fail",
			Message: "tasks.md not found",
		}
	}

	if strings.Contains(content, "## Wave: Closing") {
		return ValidationCheck{
			Name:    "closing_wave",
			Status:  "pass",
			Message: "closing wave present in tasks.md",
		}
	}
	return ValidationCheck{
		Name:    "closing_wave",
		Status:  "fail",
		Message: "tasks.md missing '## Wave: Closing' section",
	}
}

// extractFRSet extracts unique FR-N identifiers from content and returns them as a set.
func extractFRSet(content string) map[string]bool {
	set := map[string]bool{}
	for _, m := range frIDPattern.FindAllString(content, -1) {
		set[m] = true
	}
	return set
}

// checkDesignFRReferences checks that FR-N references in design.md exist in the primary file.
// Skipped if no design.md or primary file.
func checkDesignFRReferences(sd *SpecDir, primaryFile SpecFile, size SpecSize) *ValidationCheck {
	if size == SizeS {
		return nil
	}

	design, err := sd.ReadFile(FileDesign)
	if err != nil {
		return nil
	}

	primary, err := sd.ReadFile(primaryFile)
	if err != nil {
		return nil
	}

	definedFRs := extractFRSet(primary)
	if len(definedFRs) == 0 {
		return nil
	}

	designFRs := frIDPattern.FindAllString(design, -1)
	seen := map[string]bool{}
	var invalid []string
	for _, fr := range designFRs {
		if !definedFRs[fr] && !seen[fr] {
			seen[fr] = true
			invalid = append(invalid, fr)
		}
	}

	if len(invalid) == 0 {
		return &ValidationCheck{
			Name:    "design_fr_references",
			Status:  "pass",
			Message: "all FR references in design.md are valid",
		}
	}
	return &ValidationCheck{
		Name:    "design_fr_references",
		Status:  "fail",
		Message: fmt.Sprintf("undefined FR references in design.md: %s", strings.Join(invalid, ", ")),
	}
}

// checkTestSpecFRReferences checks that FR-N references in test-specs.md exist in the primary file.
// Skipped if no test-specs.md or primary file.
func checkTestSpecFRReferences(sd *SpecDir, primaryFile SpecFile) *ValidationCheck {
	testSpecs, err := sd.ReadFile(FileTestSpecs)
	if err != nil {
		return nil
	}

	primary, err := sd.ReadFile(primaryFile)
	if err != nil {
		return nil
	}

	definedFRs := extractFRSet(primary)
	if len(definedFRs) == 0 {
		return nil
	}

	testFRs := frIDPattern.FindAllString(testSpecs, -1)
	seen := map[string]bool{}
	var invalid []string
	for _, fr := range testFRs {
		if !definedFRs[fr] && !seen[fr] {
			seen[fr] = true
			invalid = append(invalid, fr)
		}
	}

	if len(invalid) == 0 {
		return &ValidationCheck{
			Name:    "testspec_fr_references",
			Status:  "pass",
			Message: "all FR references in test-specs.md are valid",
		}
	}
	return &ValidationCheck{
		Name:    "testspec_fr_references",
		Status:  "fail",
		Message: fmt.Sprintf("undefined FR references in test-specs.md: %s", strings.Join(invalid, ", ")),
	}
}

// checkNFRTraceability checks that NFR-N in the primary file are mapped in design.md.
// Skipped if size is S/M, no design.md, or no NFR-N in primary.
func checkNFRTraceability(sd *SpecDir, primaryFile SpecFile, size SpecSize) *ValidationCheck {
	if size != SizeL && size != SizeXL {
		return nil
	}

	primary, err := sd.ReadFile(primaryFile)
	if err != nil {
		return nil
	}

	nfrIDs := nfrIDPattern.FindAllString(primary, -1)
	if len(nfrIDs) == 0 {
		return nil // no NFRs defined — skip
	}

	design, err := sd.ReadFile(FileDesign)
	if err != nil {
		return nil
	}

	seen := map[string]bool{}
	var unmapped []string
	for _, nfr := range nfrIDs {
		if seen[nfr] {
			continue
		}
		seen[nfr] = true
		if !strings.Contains(design, nfr) {
			unmapped = append(unmapped, nfr)
		}
	}

	if len(unmapped) == 0 {
		return &ValidationCheck{
			Name:    "nfr_traceability",
			Status:  "pass",
			Message: "all NFRs mapped in design.md",
		}
	}
	return &ValidationCheck{
		Name:    "nfr_traceability",
		Status:  "fail",
		Message: fmt.Sprintf("unmapped NFRs in design.md: %s", strings.Join(unmapped, ", ")),
	}
}

// checkGherkinSyntax checks that ```gherkin blocks in test-specs.md contain Given+When+Then.
// Skipped if no test-specs.md or no gherkin blocks found.
func checkGherkinSyntax(sd *SpecDir) *ValidationCheck {
	content, err := sd.ReadFile(FileTestSpecs)
	if err != nil {
		return nil
	}

	blocks := gherkinBlockPattern.FindAllStringSubmatch(content, -1)
	if len(blocks) == 0 {
		return nil // no gherkin blocks — skip
	}

	var incomplete []int
	for i, block := range blocks {
		body := block[1]
		hasGiven := gherkinGivenPattern.MatchString(body)
		hasWhen := gherkinWhenPattern.MatchString(body)
		hasThen := gherkinThenPattern.MatchString(body)
		if !hasGiven || !hasWhen || !hasThen {
			incomplete = append(incomplete, i+1)
		}
	}

	if len(incomplete) == 0 {
		return &ValidationCheck{
			Name:    "gherkin_syntax",
			Status:  "pass",
			Message: fmt.Sprintf("all %d gherkin blocks have Given/When/Then", len(blocks)),
		}
	}

	nums := make([]string, len(incomplete))
	for i, n := range incomplete {
		nums[i] = fmt.Sprintf("#%d", n)
	}
	return &ValidationCheck{
		Name:    "gherkin_syntax",
		Status:  "fail",
		Message: fmt.Sprintf("gherkin blocks missing Given/When/Then: %s", strings.Join(nums, ", ")),
	}
}

// checkOrphanTests checks that test source annotations (<!-- source: FR-N -->)
// in test-specs.md reference FRs defined in the primary file.
// Skipped if no test-specs.md or no source annotations.
func checkOrphanTests(sd *SpecDir, primaryFile SpecFile) *ValidationCheck {
	testSpecs, err := sd.ReadFile(FileTestSpecs)
	if err != nil {
		return nil
	}

	primary, err := sd.ReadFile(primaryFile)
	if err != nil {
		return nil
	}

	definedFRs := extractFRSet(primary)
	if len(definedFRs) == 0 {
		return nil
	}

	sourceMatches := sourceCommentPattern.FindAllStringSubmatch(testSpecs, -1)
	if len(sourceMatches) == 0 {
		return nil // no source annotations — skip
	}

	seen := map[string]bool{}
	var orphans []string
	for _, m := range sourceMatches {
		frRef := "FR-" + m[1]
		if !definedFRs[frRef] && !seen[frRef] {
			seen[frRef] = true
			orphans = append(orphans, frRef)
		}
	}

	if len(orphans) == 0 {
		return &ValidationCheck{
			Name:    "orphan_tests",
			Status:  "pass",
			Message: "all test source annotations reference valid FRs",
		}
	}
	return &ValidationCheck{
		Name:    "orphan_tests",
		Status:  "fail",
		Message: fmt.Sprintf("orphan test references: %s", strings.Join(orphans, ", ")),
	}
}

// checkOrphanTasks checks that task Requirements FR-N references in tasks.md
// reference FRs defined in the primary file.
// This differs from checkTaskToFR: it checks _Requirements:_ lines specifically,
// while checkTaskToFR checks all FR-N references in the entire tasks.md.
// Skipped if no tasks.md or no primary file.
func checkOrphanTasks(sd *SpecDir, primaryFile SpecFile) *ValidationCheck {
	tasks, err := sd.ReadFile(FileTasks)
	if err != nil {
		return nil
	}

	primary, err := sd.ReadFile(primaryFile)
	if err != nil {
		return nil
	}

	definedFRs := extractFRSet(primary)
	if len(definedFRs) == 0 {
		return nil
	}

	// Extract FR-N from _Requirements: FR-N_ lines.
	reqMatches := taskReqPattern.FindAllStringSubmatch(tasks, -1)
	if len(reqMatches) == 0 {
		return nil
	}

	seen := map[string]bool{}
	var orphans []string
	for _, m := range reqMatches {
		// m[1] contains the FR references (e.g., "FR-1" or "FR-1, FR-2")
		frRefs := frIDPattern.FindAllString(m[1], -1)
		for _, fr := range frRefs {
			if !definedFRs[fr] && !seen[fr] {
				seen[fr] = true
				orphans = append(orphans, fr)
			}
		}
	}

	if len(orphans) == 0 {
		return &ValidationCheck{
			Name:    "orphan_tasks",
			Status:  "pass",
			Message: "all task requirement references are valid",
		}
	}
	return &ValidationCheck{
		Name:    "orphan_tasks",
		Status:  "fail",
		Message: fmt.Sprintf("orphan task FR references: %s", strings.Join(orphans, ", ")),
	}
}
