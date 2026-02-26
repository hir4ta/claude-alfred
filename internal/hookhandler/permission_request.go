package hookhandler

import (
	"encoding/json"
	"fmt"
	"strings"
)

type permissionRequestInput struct {
	CommonInput
	ToolName   string `json:"tool_name"`
	ServerName string `json:"server_name,omitempty"`
}

// handlePermissionRequest auto-allows buddy MCP tools to avoid user interruption.
// Matches tools prefixed with "buddy_" or server named "claude-buddy".
func handlePermissionRequest(input []byte) (*HookOutput, error) {
	var in permissionRequestInput
	if err := json.Unmarshal(input, &in); err != nil {
		return nil, fmt.Errorf("parse input: %w", err)
	}

	// Auto-allow buddy MCP tools.
	if strings.HasPrefix(in.ToolName, "buddy_") ||
		strings.HasPrefix(in.ToolName, "mcp__claude-buddy__") ||
		in.ServerName == "claude-buddy" {
		return makeAllowOutput("[buddy] Auto-allowing buddy MCP tool"), nil
	}

	// Safe read-only tools: auto-allow without user interruption.
	switch in.ToolName {
	case "Read", "Glob", "Grep", "WebSearch", "WebFetch":
		return makeAllowOutput("[buddy] Auto-allowing read-only tool"), nil
	}

	return nil, nil
}
