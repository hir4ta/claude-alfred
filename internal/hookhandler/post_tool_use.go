package hookhandler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"time"

	"github.com/hir4ta/claude-buddy/internal/sessiondb"
	"github.com/hir4ta/claude-buddy/internal/store"
)

type postToolUseInput struct {
	CommonInput
	ToolName     string          `json:"tool_name"`
	ToolInput    json.RawMessage `json:"tool_input"`
	ToolResponse json.RawMessage `json:"tool_response"`
	ToolUseID    string          `json:"tool_use_id"`
}

// Write tools that indicate file modification.
var writeTools = map[string]bool{
	"Write": true, "Edit": true, "NotebookEdit": true,
}

func handlePostToolUse(input []byte) (*HookOutput, error) {
	var in postToolUseInput
	if err := json.Unmarshal(input, &in); err != nil {
		return nil, fmt.Errorf("parse input: %w", err)
	}

	sdb, err := sessiondb.Open(in.SessionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[buddy] PostToolUse: open session db: %v\n", err)
		return nil, nil
	}
	defer sdb.Close()

	isWrite := writeTools[in.ToolName]
	inputHash := hashInput(in.ToolName, in.ToolInput)

	if err := sdb.RecordEvent(in.ToolName, inputHash, isWrite); err != nil {
		fmt.Fprintf(os.Stderr, "[buddy] PostToolUse: record event: %v\n", err)
		return nil, nil
	}

	// Track file reads for Read, Grep, Glob.
	// Also record file last read sequence for stale-read detection.
	switch in.ToolName {
	case "Read":
		var ri struct {
			FilePath string `json:"file_path"`
		}
		if json.Unmarshal(in.ToolInput, &ri) == nil && ri.FilePath != "" {
			_ = sdb.IncrementFileRead(ri.FilePath)
			if seq, err := sdb.CurrentEventSeq(); err == nil {
				_ = sdb.RecordFileLastRead(ri.FilePath, seq)
			}
		}
	case "Grep":
		var gi struct {
			Path string `json:"path"`
		}
		if json.Unmarshal(in.ToolInput, &gi) == nil && gi.Path != "" {
			_ = sdb.IncrementFileRead(gi.Path)
		}
	}

	// Update session context for mode tracking.
	switch in.ToolName {
	case "EnterPlanMode":
		_ = sdb.SetContext("plan_mode", "active")
	case "ExitPlanMode":
		_ = sdb.SetContext("plan_mode", "")
	case "Task":
		_ = sdb.SetContext("subagent_active", "true")
	}

	// Track test command execution for workflow guidance.
	if in.ToolName == "Bash" {
		var bi struct {
			Command string `json:"command"`
		}
		if json.Unmarshal(in.ToolInput, &bi) == nil && testCmdPattern.MatchString(bi.Command) {
			_ = sdb.SetContext("has_test_run", "true")

			// Positive signal: test-first recognition for bugfix/refactor.
			taskTypeStr, _ := sdb.GetContext("task_type")
			tc, hasWrite, _, _ := sdb.BurstState()
			if (taskTypeStr == "bugfix" || taskTypeStr == "refactor") && !hasWrite && tc <= 3 {
				set, _ := sdb.TrySetCooldown("test_first_ack", 30*time.Minute)
				if set {
					_ = sdb.EnqueueNudge("test-first", "info",
						"Good practice: running tests before editing",
						"Test-first approach established. This gives a baseline to verify changes against.",
					)
				}
			}
		}
	}

	// Workflow order check — enqueue nudge if write doesn't match expected workflow.
	if isWrite {
		if nudge := checkWorkflowForCurrentTask(sdb); nudge != "" {
			_ = sdb.EnqueueNudge("workflow", "info", "Workflow suggestion", nudge)
		}
	}

	// Run lightweight signal detection → deliver via additionalContext.
	det := &HookDetector{sdb: sdb}
	if signal := det.Detect(); signal != "" {
		return makeAsyncContextOutput(signal), nil
	}

	// Search for past error solutions when Bash fails.
	// Also record the failure for PreToolUse past-failure warning.
	if in.ToolName == "Bash" && len(in.ToolResponse) > 0 {
		resp := string(in.ToolResponse)
		if containsError(resp) {
			var bi struct {
				Command string `json:"command"`
			}
			if json.Unmarshal(in.ToolInput, &bi) == nil && bi.Command != "" {
				sig := extractCmdSignature(bi.Command)
				errSummary := extractErrorSignature(resp)
				if sig != "" {
					_ = sdb.RecordBashFailure(sig, errSummary)
				}
			}
		}
		matchPastErrorSolutions(sdb, resp)
	}

	// File-context knowledge: after Read/Edit/Write, search for related patterns.
	if in.ToolName == "Read" || isWrite {
		var fi struct {
			FilePath string `json:"file_path"`
		}
		if json.Unmarshal(in.ToolInput, &fi) == nil && fi.FilePath != "" {
			matchFileContextKnowledge(sdb, fi.FilePath)
		}
	}

	return nil, nil
}

// checkWorkflowForCurrentTask checks workflow order based on stored task type.
func checkWorkflowForCurrentTask(sdb *sessiondb.SessionDB) string {
	taskTypeStr, _ := sdb.GetContext("task_type")
	if taskTypeStr == "" {
		return ""
	}

	on, _ := sdb.IsOnCooldown("workflow_nudge")
	if on {
		return ""
	}

	_, hasWrite, _, _ := sdb.BurstState()
	hasTestRun, _ := sdb.GetContext("has_test_run")
	planMode, _ := sdb.GetContext("plan_mode")

	suggestion := checkWorkflowOrder(
		TaskType(taskTypeStr), hasWrite,
		hasTestRun == "true",
		planMode == "active",
	)
	if suggestion == "" {
		return ""
	}

	_ = sdb.SetCooldown("workflow_nudge", 10*time.Minute)
	return suggestion
}

// matchFileContextKnowledge searches for patterns related to a file path.
func matchFileContextKnowledge(sdb *sessiondb.SessionDB, filePath string) {
	key := "file_knowledge:" + filepath.Base(filePath)
	on, _ := sdb.IsOnCooldown(key)
	if on {
		return
	}

	st, err := store.OpenDefault()
	if err != nil {
		return
	}
	defer st.Close()

	patterns, _ := st.SearchPatternsByFile(filePath, 2)
	if len(patterns) == 0 {
		return
	}

	_ = sdb.SetCooldown(key, 10*time.Minute)

	msg := "Related knowledge for this file:"
	for _, p := range patterns {
		content := p.Content
		if len([]rune(content)) > 100 {
			content = string([]rune(content)[:100]) + "..."
		}
		msg += fmt.Sprintf("\n  [%s] %s", p.PatternType, content)
	}

	_ = sdb.EnqueueNudge("file-knowledge", "info",
		fmt.Sprintf("Past knowledge found for %s", filepath.Base(filePath)),
		msg,
	)
}

// matchPastErrorSolutions checks Bash output for errors and searches past solutions.
func matchPastErrorSolutions(sdb *sessiondb.SessionDB, response string) {
	if !containsError(response) {
		return
	}

	on, _ := sdb.IsOnCooldown("past_solution")
	if on {
		return
	}

	sig := extractErrorSignature(response)
	solutions := searchErrorSolutions(sdb, sig)
	if len(solutions) == 0 {
		return
	}

	_ = sdb.EnqueueNudge(
		"past-solution", "info",
		"Similar error found in past sessions",
		formatSolution(solutions[0]),
	)
	_ = sdb.SetCooldown("past_solution", 5*time.Minute)
}

func hashInput(toolName string, toolInput json.RawMessage) uint64 {
	h := fnv.New64a()
	h.Write([]byte(toolName))
	h.Write([]byte(":"))
	var buf bytes.Buffer
	if err := json.Compact(&buf, toolInput); err == nil {
		h.Write(buf.Bytes())
	} else {
		h.Write(toolInput)
	}
	return h.Sum64()
}
