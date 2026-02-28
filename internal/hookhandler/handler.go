package hookhandler

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// CommonInput is the shared fields all hook events receive on stdin.
type CommonInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	CWD            string `json:"cwd"`
	PermissionMode string `json:"permission_mode"`
	HookEventName  string `json:"hook_event_name"`
}

// HookOutput is the top-level JSON response written to stdout.
type HookOutput struct {
	Continue           *bool          `json:"continue,omitempty"`
	StopReason         string         `json:"stopReason,omitempty"`
	SuppressOutput     *bool          `json:"suppressOutput,omitempty"`
	SystemMessage      string         `json:"systemMessage,omitempty"`
	AdditionalContext  string         `json:"additionalContext,omitempty"`
	Decision           string         `json:"decision,omitempty"`
	Reason             string         `json:"reason,omitempty"`
	HookSpecificOutput map[string]any `json:"hookSpecificOutput,omitempty"`
	Confidence         *float64       `json:"confidence,omitempty"`
	ActionID           *int64         `json:"action_id,omitempty"`
	SuggestedTool      string         `json:"suggested_tool,omitempty"`
}

// Run reads hook JSON from stdin, dispatches to the appropriate handler,
// and writes the response JSON to stdout.
func Run(eventName string) error {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("hookhandler: read stdin: %w", err)
	}

	var output *HookOutput
	switch eventName {
	case "SessionStart":
		output, err = handleSessionStart(input)
	case "PreToolUse":
		output, err = handlePreToolUse(input)
	case "PostToolUse":
		output, err = handlePostToolUse(input)
	case "UserPromptSubmit":
		output, err = handleUserPromptSubmit(input)
	case "PreCompact":
		output, err = handlePreCompact(input)
	case "SessionEnd":
		output, err = handleSessionEnd(input)
	case "PostToolUseFailure":
		output, err = handlePostToolUseFailure(input)
	case "Stop":
		output, err = handleStop(input)
	case "SubagentStart":
		output, err = handleSubagentStart(input)
	case "SubagentStop":
		output, err = handleSubagentStop(input)
	case "Notification":
		output, err = handleNotification(input)
	case "TeammateIdle":
		output, err = handleQualityGate(input, "TeammateIdle")
	case "TaskCompleted":
		output, err = handleQualityGate(input, "TaskCompleted")
	case "PermissionRequest":
		output, err = handlePermissionRequest(input)
	default:
		// Unknown event: no-op.
		return nil
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "[buddy] %s error: %v\n", eventName, err)
		return nil // Don't block Claude on errors.
	}

	if output == nil {
		return nil
	}

	enc := json.NewEncoder(os.Stdout)
	return enc.Encode(output)
}

// makeOutput creates a HookOutput with additionalContext.
func makeOutput(eventName, context string) *HookOutput {
	if context == "" {
		return nil
	}
	return &HookOutput{
		HookSpecificOutput: map[string]any{
			"hookEventName":     eventName,
			"additionalContext": context,
		},
	}
}

// makeDenyOutput creates a PreToolUse deny response.
func makeDenyOutput(reason string) *HookOutput {
	return &HookOutput{
		HookSpecificOutput: map[string]any{
			"hookEventName":          "PreToolUse",
			"permissionDecision":     "deny",
			"permissionDecisionReason": reason,
		},
	}
}

// makeAsyncContextOutput returns output with top-level additionalContext for async hooks.
// The context is delivered to Claude on the next conversation turn.
func makeAsyncContextOutput(context string) *HookOutput {
	if context == "" {
		return nil
	}
	return &HookOutput{AdditionalContext: context}
}

// makeAskOutput creates a PreToolUse "ask" response that prompts the user for confirmation.
func makeAskOutput(reason string) *HookOutput {
	return &HookOutput{
		HookSpecificOutput: map[string]any{
			"hookEventName":            "PreToolUse",
			"permissionDecision":       "ask",
			"permissionDecisionReason": reason,
		},
	}
}

// makeBlockStopOutput returns output that prevents Claude from stopping.
func makeBlockStopOutput(reason string) *HookOutput {
	return &HookOutput{Decision: "block", Reason: reason}
}

// makeUpdatedInputOutput creates a PreToolUse response with updatedInput.
func makeUpdatedInputOutput(updatedInput json.RawMessage, context string) *HookOutput {
	return &HookOutput{
		HookSpecificOutput: map[string]any{
			"hookEventName":     "PreToolUse",
			"updatedInput":      json.RawMessage(updatedInput),
			"additionalContext": context,
		},
	}
}

// makeAllowOutput creates a PreToolUse "allow" response.
func makeAllowOutput(reason string) *HookOutput {
	return &HookOutput{
		HookSpecificOutput: map[string]any{
			"hookEventName":            "PreToolUse",
			"permissionDecision":       "allow",
			"permissionDecisionReason": reason,
		},
	}
}

// formatNudges formats nudges into a compact text string for additionalContext.
func formatNudges(nudges []nudgeEntry) string {
	if len(nudges) == 0 {
		return ""
	}
	var b strings.Builder
	for i, n := range nudges {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "[buddy] %s (%s): %s\n→ %s", n.Pattern, n.Level, n.Observation, n.Suggestion)
	}
	return b.String()
}

type nudgeEntry struct {
	Pattern     string
	Level       string
	Observation string
	Suggestion  string
}

// enrichOutput attaches structured metadata to a HookOutput.
// All fields are omitempty so callers can pass zero values safely.
func enrichOutput(out *HookOutput, confidence float64, actionID int64, tool string) {
	if out == nil {
		return
	}
	if confidence > 0 {
		out.Confidence = &confidence
	}
	if actionID > 0 {
		out.ActionID = &actionID
	}
	if tool != "" {
		out.SuggestedTool = tool
	}
}

// suggestedToolForPattern maps a suggestion pattern to the MCP tool
// most likely to be useful as a follow-up action.
func suggestedToolForPattern(pattern string) string {
	// Strip contextual suffix (e.g., "retry-loop:bugfix:balanced" → "retry-loop").
	base := pattern
	if idx := strings.Index(pattern, ":"); idx > 0 {
		base = pattern[:idx]
	}
	switch base {
	case "retry-loop", "explore-stuck", "edit-fail-spiral":
		return "buddy_diagnose"
	case "knowledge", "solution", "past-fix":
		return "buddy_knowledge"
	case "health-decline", "context-overload", "velocity-wall":
		return "buddy_state"
	case "co-change", "blast-radius", "code-quality":
		return "buddy_analyze"
	default:
		return "buddy_guidance"
	}
}
