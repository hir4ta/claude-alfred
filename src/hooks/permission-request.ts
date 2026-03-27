import { readFileSync } from "node:fs";
import { getActivePlan } from "../state/plan-status.ts";
import type { HookEvent } from "../types.ts";
import { deny } from "./respond.ts";

// Task header: ### Task N: name [status]
const TASK_HEADER_RE = /^###\s+Task\s+\d+:/m;
// Field patterns (with or without bold)
const FILE_FIELD_RE = /^\s*-\s+\*{0,2}File\*{0,2}:/m;
const VERIFY_FIELD_RE = /^\s*-\s+\*{0,2}Verify\*{0,2}:/m;
// Verify field must contain a specific file path or command, not just generic text
const VERIFY_SPECIFIC_RE =
	/^\s*-\s+\*{0,2}Verify\*{0,2}:\s*\S+.*\.(ts|tsx|js|jsx|py|go|rs|rb|java|kt|swift|c|cpp|h|test|spec|json|toml|yaml|yml|sh)\b/m;
const REVIEW_GATE_RE = /review.*gate/i;
const SUCCESS_CRITERIA_RE = /success\s*criteria/i;
// A concrete criterion references a command (backticks) or file path
const CONCRETE_CRITERION_RE = /`[^`]+`|\.(ts|js|py|go|rs|tsx|jsx|rb|java|sh)\b/;

/** PermissionRequest: Validate plan structure on ExitPlanMode */
export default async function permissionRequest(ev: HookEvent): Promise<void> {
	if (ev.tool?.name !== "ExitPlanMode") return;

	const plan = getActivePlan();
	if (!plan) return; // fail-open

	const content = readFileSync(plan.path, "utf-8");
	const problems = validatePlanStructure(content);

	if (problems.length > 0) {
		deny(`Plan structure issues:\n${problems.join("\n")}`);
	}
}

function validatePlanStructure(content: string): string[] {
	const problems: string[] = [];

	// Check Success Criteria
	if (!SUCCESS_CRITERIA_RE.test(content)) {
		problems.push("- Missing Success Criteria section");
	} else {
		// Extract criteria lines (checkboxes after Success Criteria heading)
		const match = SUCCESS_CRITERIA_RE.exec(content);
		const criteriaSection = match ? content.slice(match.index + match[0].length) : "";
		const criteriaEnd = criteriaSection.search(/^##\s/m);
		const criteriaBlock =
			criteriaEnd >= 0 ? criteriaSection.slice(0, criteriaEnd) : criteriaSection;
		const criteriaLines = criteriaBlock.split("\n").filter((l) => /^\s*-\s+\[/.test(l));
		if (criteriaLines.length === 0 || !criteriaLines.some((l) => CONCRETE_CRITERION_RE.test(l))) {
			problems.push(
				"- Success Criteria must include concrete, testable conditions (commands in backticks or specific file references)",
			);
		}
	}

	// Check Review Gates
	if (!REVIEW_GATE_RE.test(content)) {
		problems.push("- Missing Review Gates section");
	}

	// Check each task has File and Verify fields
	const taskSections = splitTaskSections(content);
	for (const section of taskSections) {
		if (!FILE_FIELD_RE.test(section.body)) {
			problems.push(`- Task "${section.name}": missing File field`);
		}
		if (!VERIFY_FIELD_RE.test(section.body)) {
			problems.push(`- Task "${section.name}": missing Verify field`);
		} else if (!VERIFY_SPECIFIC_RE.test(section.body)) {
			problems.push(
				`- Task "${section.name}": Verify field must reference a specific file or command (e.g., "Verify: bun vitest run src/__tests__/foo.test.ts")`,
			);
		}
	}

	return problems;
}

interface TaskSection {
	name: string;
	body: string;
}

function splitTaskSections(content: string): TaskSection[] {
	const sections: TaskSection[] = [];
	const lines = content.split("\n");
	let current: TaskSection | null = null;

	for (const line of lines) {
		const match = line.match(TASK_HEADER_RE);
		if (match) {
			if (current) sections.push(current);
			// Extract task name from "### Task N: name [status]"
			const nameMatch = line.match(/^###\s+Task\s+\d+:\s*(.+?)(?:\s*\[.+\])?\s*$/);
			current = { name: nameMatch?.[1]?.trim() ?? "unknown", body: "" };
		} else if (current) {
			// Stop at next ## heading
			if (/^##\s/.test(line)) {
				sections.push(current);
				current = null;
			} else {
				current.body += `${line}\n`;
			}
		}
	}
	if (current) sections.push(current);

	return sections;
}
