import type { HookEvent } from "../types.ts";

/** PreToolUse: DENY if pending-fixes exist on other files, pace check */
export default async function preTool(_ev: HookEvent): Promise<void> {
	// TODO: Phase 1 — pending-fixes → DENY + pace
}
