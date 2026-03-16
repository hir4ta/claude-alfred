package spec

import (
	"context"
	"os"
	"strings"
	"testing"
)

// assertCheckStatus verifies a named check exists in the report and has the expected status.
func assertCheckStatus(t *testing.T, report *ValidationReport, name, wantStatus string) {
	t.Helper()
	for _, c := range report.Checks {
		if c.Name == name {
			if c.Status != wantStatus {
				t.Errorf("%s = %s, want %s (%s)", name, c.Status, wantStatus, c.Message)
			}
			return
		}
	}
	t.Errorf("%s check not found in report (expected %s)", name, wantStatus)
}

// assertCheckAbsent verifies a named check is NOT in the report (was skipped).
func assertCheckAbsent(t *testing.T, report *ValidationReport, name string) {
	t.Helper()
	for _, c := range report.Checks {
		if c.Name == name {
			t.Errorf("%s should be skipped, but found with status %s", name, c.Status)
			return
		}
	}
}

// assertCheckMessageContains verifies a named check exists, has the expected status,
// and its message contains the substring.
func assertCheckMessageContains(t *testing.T, report *ValidationReport, name, wantStatus, substr string) {
	t.Helper()
	for _, c := range report.Checks {
		if c.Name == name {
			if c.Status != wantStatus {
				t.Errorf("%s = %s, want %s (%s)", name, c.Status, wantStatus, c.Message)
			}
			if !strings.Contains(c.Message, substr) {
				t.Errorf("%s message should contain %q, got %q", name, substr, c.Message)
			}
			return
		}
	}
	t.Errorf("%s check not found in report", name)
}

func setupValidateSpec(t *testing.T, size SpecSize, specType SpecType) *SpecDir {
	t.Helper()
	tmp := t.TempDir()
	sd, err := Init(tmp, "validate-test", "Test validation", WithSize(size), WithSpecType(specType))
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	return sd
}

func TestValidateWellFormedSpec(t *testing.T) {
	t.Parallel()
	sd := setupValidateSpec(t, SizeL, TypeFeature)

	// Write well-formed requirements.md with FRs, NFRs, and confidence.
	reqs := `# Requirements: validate-test

## Goal
<!-- confidence: 8 | source: user -->
Test validation

## Functional Requirements

### FR-1: First requirement
<!-- confidence: 7 | source: code -->
WHEN something, the system SHALL do something.

### FR-2: Second
### FR-3: Third
### FR-4: Fourth
### FR-5: Fifth

## Non-Functional Requirements

### NFR-1: Performance
The system SHALL respond within 200ms.
`
	sd.WriteFile(context.Background(), FileRequirements, reqs)

	// Write design.md with traceability including NFR.
	design := `# Design: validate-test

## Requirements Traceability

| Req ID | Component |
|--------|-----------|
| FR-1 | ComponentA |
| FR-2 | ComponentB |
| FR-3 | ComponentC |
| FR-4 | ComponentD |
| FR-5 | ComponentE |
| NFR-1 | ComponentA |
`
	sd.WriteFile(context.Background(), FileDesign, design)

	// Write tasks.md with closing wave and FR references.
	tasks := `# Tasks: validate-test

## Wave 1
- [ ] T-1.1 [S] First task
  _Requirements: FR-1_

## Wave: Closing
- [ ] T-C.1 Self-review
`
	sd.WriteFile(context.Background(), FileTasks, tasks)

	// Write test-specs.md with gherkin blocks and source annotations.
	testSpecs := `# Test Specifications: validate-test

## Coverage Matrix
| Req ID | Test IDs | Type | Priority | Status |
|--------|----------|------|----------|--------|
| FR-1   | TS-1.1   | Unit | P0       | Pending |

## Test Cases

### TS-1.1: First test (FR-1, Happy Path)
<!-- source: FR-1 -->
` + "```gherkin\nGiven a precondition\nWhen an action occurs\nThen the expected result happens\n```\n"
	sd.WriteFile(context.Background(), FileTestSpecs, testSpecs)

	report, err := Validate(sd, SizeL, TypeFeature)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	for _, c := range report.Checks {
		if c.Status != "pass" {
			t.Errorf("check %s: %s (%s)", c.Name, c.Status, c.Message)
		}
	}
	// 6 original + 6 new checks = 12 total.
	if report.Summary != "12/12 checks passed" {
		t.Errorf("Summary = %q, want '12/12 checks passed'", report.Summary)
	}
}

func TestValidateEmptySpec(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	sd := &SpecDir{ProjectPath: tmp, TaskSlug: "empty-spec"}
	os.MkdirAll(sd.Dir(), 0o755)

	report, err := Validate(sd, SizeS, TypeFeature)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	// required_sections should fail (no requirements.md).
	for _, c := range report.Checks {
		if c.Name == "required_sections" && c.Status != "fail" {
			t.Errorf("required_sections should fail on empty spec")
		}
	}
}

func TestValidateFRCountBySize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		size     SpecSize
		frCount  int
		wantPass bool
	}{
		{"S_0_FRs", SizeS, 0, false},
		{"S_1_FR", SizeS, 1, true},
		{"M_2_FRs", SizeM, 2, false},
		{"M_3_FRs", SizeM, 3, true},
		{"L_4_FRs", SizeL, 4, false},
		{"L_5_FRs", SizeL, 5, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tmp := t.TempDir()
			sd := &SpecDir{ProjectPath: tmp, TaskSlug: "fr-count"}
			os.MkdirAll(sd.Dir(), 0o755)

			// Build requirements.md with N FRs.
			content := "# Requirements\n\n## Goal\n<!-- confidence: 7 | source: user -->\nTest\n\n"
			for i := 1; i <= tc.frCount; i++ {
				content += "### FR-" + string(rune('0'+i)) + ": Requirement\n"
			}
			os.WriteFile(sd.FilePath(FileRequirements), []byte(content), 0o644)

			report, _ := Validate(sd, tc.size, TypeFeature)
			for _, c := range report.Checks {
				if c.Name == "min_fr_count" {
					if tc.wantPass && c.Status != "pass" {
						t.Errorf("min_fr_count = %s, want pass (%s)", c.Status, c.Message)
					}
					if !tc.wantPass && c.Status != "fail" {
						t.Errorf("min_fr_count = %s, want fail (%s)", c.Status, c.Message)
					}
				}
			}
		})
	}
}

func TestValidateTraceabilityFail(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	sd := &SpecDir{ProjectPath: tmp, TaskSlug: "trace-test"}
	os.MkdirAll(sd.Dir(), 0o755)

	reqs := "# Requirements\n\n## Goal\n<!-- confidence: 7 | source: user -->\nTest\n\n### FR-1: Req\n### FR-2: Req2\n"
	os.WriteFile(sd.FilePath(FileRequirements), []byte(reqs), 0o644)

	// design.md only mentions FR-1, not FR-2.
	design := "# Design\n\n## Traceability\n| FR-1 | ComponentA |\n"
	os.WriteFile(sd.FilePath(FileDesign), []byte(design), 0o644)

	report, _ := Validate(sd, SizeL, TypeFeature)
	for _, c := range report.Checks {
		if c.Name == "traceability_fr_to_task" && c.Status != "fail" {
			t.Errorf("traceability should fail for unmapped FR-2, got %s: %s", c.Status, c.Message)
		}
	}
}

func TestValidateMissingFiles(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	sd := &SpecDir{ProjectPath: tmp, TaskSlug: "partial"}
	os.MkdirAll(sd.Dir(), 0o755)

	// Only create requirements.md.
	reqs := "# Requirements\n\n## Goal\n<!-- confidence: 7 | source: user -->\nTest\n\n### FR-1: Req\n"
	os.WriteFile(sd.FilePath(FileRequirements), []byte(reqs), 0o644)

	report, err := Validate(sd, SizeS, TypeFeature)
	if err != nil {
		t.Fatalf("Validate should not error on missing files: %v", err)
	}
	// Should still have checks — some pass, some fail gracefully.
	if len(report.Checks) == 0 {
		t.Error("Validate should return checks even with missing files")
	}
}

func TestValidateDesignFRReferences(t *testing.T) {
	t.Parallel()

	t.Run("valid_references_pass", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()
		sd := &SpecDir{ProjectPath: tmp, TaskSlug: "design-fr"}
		os.MkdirAll(sd.Dir(), 0o755)

		reqs := "# Requirements\n\n## Goal\n<!-- confidence: 7 | source: user -->\nTest\n\n### FR-1: Req\n### FR-2: Req2\n### FR-3: Req3\n"
		os.WriteFile(sd.FilePath(FileRequirements), []byte(reqs), 0o644)
		design := "# Design\n\n## Traceability\n| FR-1 | A |\n| FR-2 | B |\n| FR-3 | C |\n"
		os.WriteFile(sd.FilePath(FileDesign), []byte(design), 0o644)

		report, _ := Validate(sd, SizeL, TypeFeature)
		assertCheckStatus(t, report, "design_fr_references", "pass")
	})

	t.Run("invalid_references_fail", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()
		sd := &SpecDir{ProjectPath: tmp, TaskSlug: "design-fr-fail"}
		os.MkdirAll(sd.Dir(), 0o755)

		reqs := "# Requirements\n\n## Goal\n<!-- confidence: 7 | source: user -->\nTest\n\n### FR-1: Req\n### FR-2: Req2\n"
		os.WriteFile(sd.FilePath(FileRequirements), []byte(reqs), 0o644)
		design := "# Design\n\n## Traceability\n| FR-1 | A |\n| FR-5 | B |\n"
		os.WriteFile(sd.FilePath(FileDesign), []byte(design), 0o644)

		report, _ := Validate(sd, SizeL, TypeFeature)
		assertCheckMessageContains(t, report, "design_fr_references", "fail", "FR-5")
	})

	t.Run("s_size_skips", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()
		sd := &SpecDir{ProjectPath: tmp, TaskSlug: "design-fr-skip"}
		os.MkdirAll(sd.Dir(), 0o755)

		reqs := "# Requirements\n\n## Goal\n<!-- confidence: 7 | source: user -->\nTest\n\n### FR-1: Req\n"
		os.WriteFile(sd.FilePath(FileRequirements), []byte(reqs), 0o644)

		report, _ := Validate(sd, SizeS, TypeFeature)
		assertCheckAbsent(t, report, "design_fr_references")
	})
}

func TestValidateTestSpecFRReferences(t *testing.T) {
	t.Parallel()

	t.Run("valid_pass", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()
		sd := &SpecDir{ProjectPath: tmp, TaskSlug: "ts-fr"}
		os.MkdirAll(sd.Dir(), 0o755)

		reqs := "# Requirements\n\n## Goal\n<!-- confidence: 7 | source: user -->\nTest\n\n### FR-1: Req\n### FR-2: Req2\n"
		os.WriteFile(sd.FilePath(FileRequirements), []byte(reqs), 0o644)
		testSpecs := "# Test Specs\n\n| FR-1 | TS-1.1 |\n| FR-2 | TS-2.1 |\n"
		os.WriteFile(sd.FilePath(FileTestSpecs), []byte(testSpecs), 0o644)

		report, _ := Validate(sd, SizeL, TypeFeature)
		assertCheckStatus(t, report, "testspec_fr_references", "pass")
	})

	t.Run("invalid_fail", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()
		sd := &SpecDir{ProjectPath: tmp, TaskSlug: "ts-fr-fail"}
		os.MkdirAll(sd.Dir(), 0o755)

		reqs := "# Requirements\n\n## Goal\n<!-- confidence: 7 | source: user -->\nTest\n\n### FR-1: Req\n"
		os.WriteFile(sd.FilePath(FileRequirements), []byte(reqs), 0o644)
		testSpecs := "# Test Specs\n\n| FR-1 | TS-1.1 |\n| FR-99 | TS-99.1 |\n"
		os.WriteFile(sd.FilePath(FileTestSpecs), []byte(testSpecs), 0o644)

		report, _ := Validate(sd, SizeL, TypeFeature)
		assertCheckMessageContains(t, report, "testspec_fr_references", "fail", "FR-99")
	})
}

func TestValidateNFRTraceability(t *testing.T) {
	t.Parallel()

	t.Run("mapped_pass", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()
		sd := &SpecDir{ProjectPath: tmp, TaskSlug: "nfr-pass"}
		os.MkdirAll(sd.Dir(), 0o755)

		reqs := "# Requirements\n\n## Goal\n<!-- confidence: 7 | source: user -->\nTest\n\n### FR-1: Req\n\n### NFR-1: Performance\n"
		os.WriteFile(sd.FilePath(FileRequirements), []byte(reqs), 0o644)
		design := "# Design\n\n## Traceability\n| FR-1 | A |\n| NFR-1 | A |\n"
		os.WriteFile(sd.FilePath(FileDesign), []byte(design), 0o644)

		report, _ := Validate(sd, SizeL, TypeFeature)
		assertCheckStatus(t, report, "nfr_traceability", "pass")
	})

	t.Run("unmapped_fail", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()
		sd := &SpecDir{ProjectPath: tmp, TaskSlug: "nfr-fail"}
		os.MkdirAll(sd.Dir(), 0o755)

		reqs := "# Requirements\n\n## Goal\n<!-- confidence: 7 | source: user -->\nTest\n\n### FR-1: Req\n\n### NFR-1: Performance\n"
		os.WriteFile(sd.FilePath(FileRequirements), []byte(reqs), 0o644)
		design := "# Design\n\n## Traceability\n| FR-1 | A |\n"
		os.WriteFile(sd.FilePath(FileDesign), []byte(design), 0o644)

		report, _ := Validate(sd, SizeL, TypeFeature)
		assertCheckStatus(t, report, "nfr_traceability", "fail")
	})

	t.Run("no_nfr_skips", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()
		sd := &SpecDir{ProjectPath: tmp, TaskSlug: "nfr-skip"}
		os.MkdirAll(sd.Dir(), 0o755)

		reqs := "# Requirements\n\n## Goal\n<!-- confidence: 7 | source: user -->\nTest\n\n### FR-1: Req\n"
		os.WriteFile(sd.FilePath(FileRequirements), []byte(reqs), 0o644)

		report, _ := Validate(sd, SizeL, TypeFeature)
		assertCheckAbsent(t, report, "nfr_traceability")
	})

	t.Run("m_size_skips", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()
		sd := &SpecDir{ProjectPath: tmp, TaskSlug: "nfr-m"}
		os.MkdirAll(sd.Dir(), 0o755)

		reqs := "# Requirements\n\n## Goal\n<!-- confidence: 7 | source: user -->\nTest\n\n### FR-1: Req\n\n### NFR-1: Perf\n"
		os.WriteFile(sd.FilePath(FileRequirements), []byte(reqs), 0o644)

		report, _ := Validate(sd, SizeM, TypeFeature)
		assertCheckAbsent(t, report, "nfr_traceability")
	})
}

func TestValidateGherkinSyntax(t *testing.T) {
	t.Parallel()

	t.Run("valid_pass", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()
		sd := &SpecDir{ProjectPath: tmp, TaskSlug: "gherkin-pass"}
		os.MkdirAll(sd.Dir(), 0o755)

		content := "# Test Specs\n\n" +
			"```gherkin\nGiven a precondition\nWhen an action\nThen a result\n```\n"
		os.WriteFile(sd.FilePath(FileTestSpecs), []byte(content), 0o644)

		report, _ := Validate(sd, SizeL, TypeFeature)
		assertCheckStatus(t, report, "gherkin_syntax", "pass")
	})

	t.Run("missing_then_fail", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()
		sd := &SpecDir{ProjectPath: tmp, TaskSlug: "gherkin-fail"}
		os.MkdirAll(sd.Dir(), 0o755)

		content := "# Test Specs\n\n" +
			"```gherkin\nGiven a precondition\nWhen an action\n```\n"
		os.WriteFile(sd.FilePath(FileTestSpecs), []byte(content), 0o644)

		report, _ := Validate(sd, SizeL, TypeFeature)
		assertCheckStatus(t, report, "gherkin_syntax", "fail")
	})

	t.Run("no_blocks_skips", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()
		sd := &SpecDir{ProjectPath: tmp, TaskSlug: "gherkin-skip"}
		os.MkdirAll(sd.Dir(), 0o755)

		content := "# Test Specs\n\nNo gherkin blocks here.\n"
		os.WriteFile(sd.FilePath(FileTestSpecs), []byte(content), 0o644)

		report, _ := Validate(sd, SizeL, TypeFeature)
		assertCheckAbsent(t, report, "gherkin_syntax")
	})
}

func TestValidateOrphanTests(t *testing.T) {
	t.Parallel()

	t.Run("linked_pass", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()
		sd := &SpecDir{ProjectPath: tmp, TaskSlug: "orphan-test-pass"}
		os.MkdirAll(sd.Dir(), 0o755)

		reqs := "# Requirements\n\n## Goal\n<!-- confidence: 7 | source: user -->\nTest\n\n### FR-1: Req\n"
		os.WriteFile(sd.FilePath(FileRequirements), []byte(reqs), 0o644)
		testSpecs := "# Test Specs\n\n### TS-1.1: Test\n<!-- source: FR-1 -->\n"
		os.WriteFile(sd.FilePath(FileTestSpecs), []byte(testSpecs), 0o644)

		report, _ := Validate(sd, SizeL, TypeFeature)
		assertCheckStatus(t, report, "orphan_tests", "pass")
	})

	t.Run("orphan_fail", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()
		sd := &SpecDir{ProjectPath: tmp, TaskSlug: "orphan-test-fail"}
		os.MkdirAll(sd.Dir(), 0o755)

		reqs := "# Requirements\n\n## Goal\n<!-- confidence: 7 | source: user -->\nTest\n\n### FR-1: Req\n"
		os.WriteFile(sd.FilePath(FileRequirements), []byte(reqs), 0o644)
		testSpecs := "# Test Specs\n\n### TS-1.1: Test\n<!-- source: FR-99 -->\n"
		os.WriteFile(sd.FilePath(FileTestSpecs), []byte(testSpecs), 0o644)

		report, _ := Validate(sd, SizeL, TypeFeature)
		assertCheckMessageContains(t, report, "orphan_tests", "fail", "FR-99")
	})
}

func TestValidateOrphanTasks(t *testing.T) {
	t.Parallel()

	t.Run("linked_pass", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()
		sd := &SpecDir{ProjectPath: tmp, TaskSlug: "orphan-task-pass"}
		os.MkdirAll(sd.Dir(), 0o755)

		reqs := "# Requirements\n\n## Goal\n<!-- confidence: 7 | source: user -->\nTest\n\n### FR-1: Req\n"
		os.WriteFile(sd.FilePath(FileRequirements), []byte(reqs), 0o644)
		tasks := "# Tasks\n\n## Wave 1\n- [ ] T-1.1 [S] Task\n  _Requirements: FR-1_\n\n## Wave: Closing\n- [ ] T-C.1 Review\n"
		os.WriteFile(sd.FilePath(FileTasks), []byte(tasks), 0o644)

		report, _ := Validate(sd, SizeL, TypeFeature)
		assertCheckStatus(t, report, "orphan_tasks", "pass")
	})

	t.Run("orphan_fail", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()
		sd := &SpecDir{ProjectPath: tmp, TaskSlug: "orphan-task-fail"}
		os.MkdirAll(sd.Dir(), 0o755)

		reqs := "# Requirements\n\n## Goal\n<!-- confidence: 7 | source: user -->\nTest\n\n### FR-1: Req\n"
		os.WriteFile(sd.FilePath(FileRequirements), []byte(reqs), 0o644)
		tasks := "# Tasks\n\n## Wave 1\n- [ ] T-2.1 [S] Task\n  _Requirements: FR-99_\n\n## Wave: Closing\n- [ ] T-C.1 Review\n"
		os.WriteFile(sd.FilePath(FileTasks), []byte(tasks), 0o644)

		report, _ := Validate(sd, SizeL, TypeFeature)
		assertCheckMessageContains(t, report, "orphan_tasks", "fail", "FR-99")
	})
}

func TestValidateBugfix(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	sd, err := Init(tmp, "bugfix-validate", "fix bug", WithSize(SizeS), WithSpecType(TypeBugfix))
	if err != nil {
		t.Fatalf("Init: %v", err)
	}

	report, err := Validate(sd, SizeS, TypeBugfix)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	// required_sections should check bugfix.md for "## Bug Summary".
	for _, c := range report.Checks {
		if c.Name == "required_sections" && c.Status != "pass" {
			t.Errorf("required_sections should pass for bugfix template: %s", c.Message)
		}
	}
}
