import { existsSync } from "node:fs";
import { join } from "node:path";
import type { HookEvent } from "./dispatcher.js";
import { isGateActive } from "./review-gate.js";
import { denyTool, isSpecFilePath, tryReadActiveSpec } from "./spec-guard.js";
import { readLastIntent } from "./state.js";

const BLOCKABLE_TOOLS = new Set(["Edit", "Write"]);
const IMPLEMENT_INTENTS = new Set(["implement", "bugfix", "tdd"]);

/**
 * PreToolUse handler: block Edit/Write on review-gate, intent guard, or unapproved spec.
 * Enforcement order: .alfred/ exempt → review-gate → intent guard → approval gate.
 * Fail-open: any error results in allowing the tool (NFR-2).
 */
export async function preToolUse(ev: HookEvent): Promise<void> {
	const toolName = ev.tool_name ?? "";

	// Only block Edit/Write. Everything else passes through.
	if (!BLOCKABLE_TOOLS.has(toolName)) return;

	// .alfred/ edits are always allowed (spec creation/update).
	const toolInput = (ev.tool_input ?? {}) as Record<string, unknown>;
	const filePath = typeof toolInput.file_path === "string" ? toolInput.file_path : "";
	if (filePath && isSpecFilePath(ev.cwd, filePath)) return;

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

	// Intent guard: blocks source edits when implement intent detected but no active spec.
	const spec = tryReadActiveSpec(ev.cwd);
	if (!spec && ev.cwd && existsSync(join(ev.cwd, ".alfred"))) {
		const intent = readLastIntent(ev.cwd);
		if (intent && IMPLEMENT_INTENTS.has(intent)) {
			const reason = [
				"No active spec. Create a spec before implementing: /alfred:brief or dossier action=init",
				'- "This change is too small for a spec" → Use dossier init size=S (adds <2min)',
				'- "I\'ll create the spec after" → The Stop hook will block you from finishing without a spec',
			].join("\n");
			denyTool(reason);
			return;
		}
		return; // No spec, no implement intent → free coding
	}
	if (!spec) return;

	// M/L/XL with unapproved review → deny.
	if (["M", "L", "XL"].includes(spec.size) && spec.reviewStatus !== "approved") {
		const reason = [
			`Spec '${spec.slug}' (size ${spec.size}) is not approved. Submit review via \`alfred dashboard\` or run self-review before implementation.`,
			'- "I\'ll get the review after implementation" → The Stop hook will block you from finishing anyway',
			'- "This edit is trivial" → All M/L/XL edits are gated. Use dossier init size=S for trivial changes',
		].join("\n");
		denyTool(reason);
	}
}
