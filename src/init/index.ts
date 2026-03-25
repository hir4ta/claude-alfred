/**
 * `alfred init` — Setup alfred in ~/.claude/ and project .alfred/
 *
 * Installs: MCP server, hooks, rules, skills, agents, gates, DB
 */
import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { homedir } from "node:os";
import { join } from "node:path";
import { detectGates } from "../gates/index.js";
import { detectProjectProfile } from "../profile/detect.js";
import { Store } from "../store/index.js";
import { resolveOrRegisterProject } from "../store/project.js";

interface InitOptions {
	scan?: boolean;
	force?: boolean;
}

export async function alfredInit(cwd: string, opts: InitOptions = {}): Promise<void> {
	const home = homedir();
	const claudeDir = join(home, ".claude");

	console.log("alfred init\n");

	// 1. MCP server registration
	installMcp(claudeDir, opts.force);

	// 2. Hooks
	installHooks(claudeDir, opts.force);

	// 3. Rules
	installRules(claudeDir);

	// 4. Skills
	installSkills(claudeDir);

	// 5. Agents
	installAgents(claudeDir);

	// 6. Project setup
	initProject(cwd);

	// 7. DB
	initDb();

	console.log("\nalfred initialized.");
}

function installMcp(claudeDir: string, force?: boolean): void {
	const mcpPath = join(claudeDir, ".mcp.json");
	let mcp: Record<string, unknown> = {};

	if (existsSync(mcpPath)) {
		try { mcp = JSON.parse(readFileSync(mcpPath, "utf-8")); } catch { /* new file */ }
	}

	const servers = (mcp.mcpServers ?? {}) as Record<string, unknown>;
	if (servers.alfred && !force) {
		console.log("  ✓ MCP: alfred already registered");
		return;
	}

	servers.alfred = {
		type: "stdio",
		command: "alfred",
		args: ["serve"],
		env: { VOYAGE_API_KEY: "${VOYAGE_API_KEY}" },
	};
	mcp.mcpServers = servers;

	mkdirSync(claudeDir, { recursive: true });
	writeFileSync(mcpPath, JSON.stringify(mcp, null, 2) + "\n");
	console.log("  ✓ MCP: alfred registered → ~/.claude/.mcp.json");
}

function installHooks(claudeDir: string, force?: boolean): void {
	const settingsPath = join(claudeDir, "settings.json");
	let settings: Record<string, unknown> = {};

	if (existsSync(settingsPath)) {
		try { settings = JSON.parse(readFileSync(settingsPath, "utf-8")); } catch { /* new file */ }
	}

	const hooks = (settings.hooks ?? {}) as Record<string, unknown>;

	// Only install if not already present (or force)
	const alfredHooks = {
		PreToolUse: [{ matcher: "Edit|Write", hooks: [{ type: "command", command: "alfred hook pre-tool-use", timeout: 3 }] }],
		PostToolUse: [{ matcher: "Bash|Edit|Write", hooks: [{ type: "command", command: "alfred hook post-tool-use", timeout: 5 }] }],
		UserPromptSubmit: [{ hooks: [{ type: "command", command: "alfred hook user-prompt-submit", timeout: 10 }] }],
		SessionStart: [{ hooks: [{ type: "command", command: "alfred hook session-start", timeout: 5 }] }],
		PreCompact: [
			{ hooks: [
				{ type: "command", command: "alfred hook pre-compact", timeout: 10 },
				{ type: "agent", prompt: "Read the transcript and extract error resolutions (error → fix patterns). For each, run: alfred hook-internal save-decision --title '...' --error_signature '...' --resolution '...'", timeout: 60 },
			] },
		],
		Stop: [{ hooks: [{ type: "command", command: "alfred hook stop", timeout: 3 }] }],
	};

	let installed = 0;
	for (const [event, config] of Object.entries(alfredHooks)) {
		if (!hooks[event] || force) {
			hooks[event] = config;
			installed++;
		}
	}

	settings.hooks = hooks;
	writeFileSync(settingsPath, JSON.stringify(settings, null, 2) + "\n");
	console.log(`  ✓ Hooks: ${installed} events installed → ~/.claude/settings.json`);
}

function installRules(claudeDir: string): void {
	const rulesDir = join(claudeDir, "rules");
	mkdirSync(rulesDir, { recursive: true });

	const qualityRules = `---
description: alfred quality enforcement rules — applied to all projects
---

# Quality Rules

## Test First
- When implementing a new function or module, write the test file FIRST
- Test file must have at least 2 meaningful assertions per test case
- Do not mark implementation as complete until tests pass

## Error Handling
- Check function return values explicitly — do not silently ignore errors
- Prefer early return over deeply nested if/else
- Never catch errors just to log them — either handle or re-throw

## Code Changes
- Keep each logical change under 200 lines of diff
- If a change exceeds 200 lines, split into smaller commits
- Run the project's lint and type check commands after each file edit

## Self-Check Before Completion
- Before marking any task as done, verify:
  1. Are there edge cases that need tests?
  2. Could this fail silently (produce wrong output without crashing)?
  3. Is there a simpler approach?
  4. Does this follow the project's existing patterns?

## When Stuck
- If the same approach fails 3 times, stop and research:
  1. Check official documentation
  2. Search for similar issues on GitHub/StackOverflow
  3. Try a fundamentally different approach
`;

	writeFileSync(join(rulesDir, "alfred-quality.md"), qualityRules);
	console.log("  ✓ Rules: alfred-quality.md → ~/.claude/rules/");
}

function installSkills(claudeDir: string): void {
	// /alfred:review
	const reviewDir = join(claudeDir, "skills", "alfred-review");
	mkdirSync(reviewDir, { recursive: true });

	writeFileSync(join(reviewDir, "SKILL.md"), `---
name: alfred-review
description: >
  Deep multi-agent code review with Judge filtering. Use when wanting
  thorough review before a major commit, after a milestone, or when
  wanting a second opinion. Spawns 3 parallel sub-reviewers (security,
  logic, design), then filters findings for actionability.
  NOT for everyday small edits (hooks handle that).
user-invocable: true
argument-hint: "[--staged | --commit SHA | --range BASE..HEAD]"
allowed-tools: Read, Glob, Grep, Agent, Bash(git diff *, git log *, git show *, git status *)
context: fork
---

# /alfred:review — Judge-Filtered Code Review

## Phase 1: Gather Context

1. Parse \`$ARGUMENTS\` for scope:
   - \`--staged\` (default): \`git diff --cached\`
   - \`--commit SHA\`: \`git show SHA\`
   - \`--range BASE..HEAD\`: \`git diff BASE..HEAD\`
2. If no args: use \`git diff\` (unstaged changes)
3. Extract changed file paths and languages

## Phase 2: Parallel Review (spawn 3 agents simultaneously)

Launch all 3 agents in a single message with the diff:

**Agent 1: security** — Injection, auth, secrets, input validation, TOCTOU
**Agent 2: logic** — Correctness, edge cases, error handling, silent failures
**Agent 3: design** — Naming, structure, duplication, complexity, conventions

Each agent returns findings as structured text with severity and file:line.

## Phase 3: Judge Filtering

For each finding, evaluate:
1. **Actionable?** — Can the developer fix this without ambiguity?
2. **In scope?** — Is this in the current diff, not a pre-existing issue?
3. **Real problem?** — Is this a genuine issue, not a style preference?

Discard findings that fail any criterion.

## Phase 4: Output

Present findings sorted by severity (Critical → High → Medium).
If 0 critical and 0 high: "Ready to commit."
If any critical: "Fix critical issues before committing."
`);

	// /alfred:conventions
	const convDir = join(claudeDir, "skills", "alfred-conventions");
	mkdirSync(convDir, { recursive: true });

	writeFileSync(join(convDir, "SKILL.md"), `---
name: alfred-conventions
description: >
  Scan the codebase and discover implicit coding conventions.
  Use on first setup of a new project, after major refactors,
  or when wanting to document existing patterns. Saves confirmed
  conventions and generates rules for Claude Code.
user-invocable: true
allowed-tools: Read, Glob, Grep, Bash(wc *, head *, git log --oneline *)
---

# /alfred:conventions — Convention Discovery

## Step 1: Analyze

Scan the codebase for patterns:
1. **Import ordering** — Read 10 representative source files
2. **Naming conventions** — Files, functions, types, constants
3. **Error handling** — try/catch style, Result types, early returns
4. **Test structure** — Co-located or separate? .test. or .spec.?
5. **Directory structure** — Feature-based or layer-based?

## Step 2: Present

For each convention found, show:
- Pattern description + 3 example files
- Confidence: high (>80%) / medium (50-80%) / low (<50%)

Ask user to confirm/reject each.

## Step 3: Save

For confirmed conventions:
1. Call \`alfred save type=convention\` for each
2. Report: "Saved N conventions."
`);

	console.log("  ✓ Skills: alfred-review, alfred-conventions → ~/.claude/skills/");
}

function installAgents(claudeDir: string): void {
	const agentsDir = join(claudeDir, "agents");
	mkdirSync(agentsDir, { recursive: true });

	writeFileSync(join(agentsDir, "alfred-reviewer.md"), `---
name: alfred-reviewer
description: >
  Single-perspective code reviewer. Used as a sub-agent by /alfred:review.
  Focuses on one review dimension (security, logic, or design).
  Returns structured findings. Never spawns sub-agents itself.
tools: Read, Glob, Grep, Bash(git diff *, git show *)
disallowedTools: Write, Edit, Agent
maxTurns: 15
---

You are a focused code reviewer. You receive a diff and a checklist.
Review ONLY the diff — do not flag pre-existing issues.

Output each finding with severity (critical/high/medium/low), file path, line number,
issue description, and suggested fix.

If no issues found, state: "No issues found in this review dimension."
`);

	console.log("  ✓ Agent: alfred-reviewer → ~/.claude/agents/");
}

function initProject(cwd: string): void {
	const alfredDir = join(cwd, ".alfred");
	const stateDir = join(alfredDir, ".state");
	mkdirSync(stateDir, { recursive: true });

	// gates.json
	const gatesPath = join(alfredDir, "gates.json");
	if (!existsSync(gatesPath)) {
		const gates = detectGates(cwd);
		writeFileSync(gatesPath, JSON.stringify(gates, null, 2) + "\n");
		console.log("  ✓ Gates: auto-detected → .alfred/gates.json");
	} else {
		console.log("  ✓ Gates: .alfred/gates.json exists");
	}

	// Project profile
	const profilePath = join(stateDir, "project-profile.json");
	if (!existsSync(profilePath)) {
		const profile = detectProjectProfile(cwd);
		writeFileSync(profilePath, JSON.stringify(profile, null, 2) + "\n");
		console.log("  ✓ Profile: auto-detected → .alfred/.state/project-profile.json");
	} else {
		console.log("  ✓ Profile: .alfred/.state/project-profile.json exists");
	}

	// Knowledge directories
	for (const dir of ["error_resolutions", "exemplars", "conventions"]) {
		mkdirSync(join(alfredDir, "knowledge", dir), { recursive: true });
	}
	console.log("  ✓ Knowledge: .alfred/knowledge/ directories created");
}

function initDb(): void {
	const store = Store.openDefault();
	store.close();
	console.log("  ✓ DB: ~/.alfred/alfred.db (Schema V1)");
}
