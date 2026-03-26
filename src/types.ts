/** Hook event from Claude Code (stdin JSON) */
export interface HookEvent {
	hook_type: string;
	session_id?: string;
	permission_mode?: string;
	stop_hook_active?: boolean;
	tool_name?: string;
	tool_input?: Record<string, unknown>;
	tool_output?: string;
	// UserPromptSubmit
	prompt?: string;
	// PermissionRequest
	tool?: { name: string };
}

/** Hook response written to stdout */
export interface HookResponse {
	hookSpecificOutput?: {
		additionalContext?: string;
		permissionDecision?: "allow" | "deny" | "ask";
		permissionDecisionReason?: string;
		decision?: "block";
		reason?: string;
	};
}

/** Pending fix entry stored in .alfred/.state/pending-fixes.json */
export interface PendingFix {
	file: string;
	errors: string[];
	gate: string;
}

/** Gate configuration in .alfred/gates.json */
export interface GatesConfig {
	on_write?: Record<string, GateDefinition>;
	on_commit?: Record<string, GateDefinition>;
}

export interface GateDefinition {
	command: string;
	timeout?: number;
	run_once_per_batch?: boolean;
}

/** Project profile in .alfred/.state/project-profile.json */
export interface ProjectProfile {
	language: string[];
	runtime?: string;
	test_framework?: string;
	test_pattern?: string;
	linter?: string;
	detected_at: string;
}

/** Handoff state saved by PreCompact */
export interface HandoffState {
	summary: string;
	changed_files: string[];
	pending_fixes: boolean;
	next_steps: string;
	saved_at: string;
}
