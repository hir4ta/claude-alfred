import type { HookEvent } from "../types.ts";

/** PostToolUse: lint/type gate after Edit/Write, test gate after git commit */
export default async function postTool(_ev: HookEvent): Promise<void> {
	// TODO: Phase 1 — lint/type gate + pending-fixes
}
