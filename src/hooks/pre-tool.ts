import { effectiveStatus } from "../spec/types.js";
import type { HookEvent } from "./dispatcher.js";
import { isGateActive } from "./review-gate.js";
import {
	allowTool,
	denyTool,
	isActiveSpecMalformed,
	isSpecFilePath,
	tryReadActiveSpec,
} from "./spec-guard.js";

const BLOCKABLE_TOOLS = new Set(["Edit", "Write"]);

/**
 * PreToolUse handler: block Edit/Write on review-gate or unapproved spec.
 * Enforcement order: .alfred/ exempt → malformed check → review-gate → approval gate.
 * Only uses allowTool() for definitively exempt edits (.alfred/ files, deferred/cancelled).
 * For all other cases, returns silently so the prompt hook (LLM judge) can evaluate.
 */
export async function preToolUse(ev: HookEvent): Promise<void> {
	const toolName = ev.tool_name ?? "";

	// Only block Edit/Write. Everything else passes through.
	if (!BLOCKABLE_TOOLS.has(toolName)) return;

	// .alfred/ edits are always allowed (spec creation/update).
	const toolInput = (ev.tool_input ?? {}) as Record<string, unknown>;
	const filePath = typeof toolInput.file_path === "string" ? toolInput.file_path : "";
	if (filePath && isSpecFilePath(ev.cwd, filePath)) {
		allowTool("Spec file edit");
		return;
	}

	// Fail-closed: if _active.md exists but can't be parsed, deny rather than silently allowing.
	if (isActiveSpecMalformed(ev.cwd)) {
		denyTool(
			"Failed to read spec state (_active.md exists but could not be parsed). Fix or delete .alfred/specs/_active.md before editing source files.",
		);
		return;
	}

	const spec = tryReadActiveSpec(ev.cwd);

	// FR-20: Exempt deferred/cancelled tasks from all gates.
	if (spec) {
		const status = effectiveStatus(spec.status);
		if (status === "deferred" || status === "cancelled") {
			allowTool("Deferred/cancelled spec");
			return;
		}
	}

	// Review gate: blocks source edits until spec/wave review is completed.
	const gate = isGateActive(ev.cwd);
	if (gate) {
		const gateLabel =
			gate.gate === "wave-review" ? `Wave ${gate.wave ?? "?"} review` : "Spec self-review";
		const reason = [
			`${gateLabel} required for spec '${gate.slug}'. Complete review, then run: dossier action=gate sub_action=clear reason="<review summary>"`,
			`- Gate reason: ${gate.reason}`,
			'- "I already reviewed mentally" → Run actual review (3-agent or /alfred:inspect), then clear the gate',
		].join("\n");
		denyTool(reason);
		return;
	}

	// No active spec → let the prompt hook (LLM judge) decide if a spec is needed.
	if (!spec) return;

	// M/L/XL with unapproved review → deny.
	if (["M", "L", "XL"].includes(spec.size) && spec.reviewStatus !== "approved") {
		const reason = [
			`Spec '${spec.slug}' (size ${spec.size}) is not approved. Submit review via \`alfred dashboard\` or run self-review before implementation.`,
			'- "I\'ll get the review after implementation" → The Stop hook will block you from finishing anyway',
			'- "This edit is trivial" → All M/L/XL edits are gated. Use dossier init size=S for trivial changes',
		].join("\n");
		denyTool(reason);
		return;
	}

	// Active spec + all gates passed → silent return.
	// The prompt hook (LLM judge) will see the spec context and allow quickly.
}
