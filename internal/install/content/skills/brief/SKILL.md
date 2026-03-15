---
name: brief
description: >
  Structured spec generation with multi-perspective deliberation (no sub-agents).
  Creates requirements, design, decisions, and session files in .alfred/specs/.
  Use when starting a new task, organizing a design, planning before implementation,
  or wanting a structured development plan. NOT for divergent brainstorming
  (use /alfred:salon). NOT for autonomous implementation (use /alfred:attend).
user-invocable: true
argument-hint: "task-slug [description]"
allowed-tools: Read, Edit, Glob, Grep, AskUserQuestion, WebSearch, WebFetch, mcp__plugin_alfred_alfred__knowledge, mcp__plugin_alfred_alfred__dossier
model: sonnet
context: current
---

# /alfred:brief — Spec Generator with Multi-Perspective Deliberation

Generate a structured spec through systematic deliberation from 3 perspectives
(Architect, Devil's Advocate, Researcher), creating a development plan resilient
to Compact/session loss.

**No sub-agents are spawned.** All deliberation happens inline to avoid rate limits.

## Core Principle
**What Compact loses most: reasoning process, rationale for design decisions, dead-end
explorations, implicit agreements.** By explicitly writing these to files with
multi-perspective analysis, we create specs that are both robust and well-reasoned.

## Steps

### 1. [WHAT] Parse $ARGUMENTS
- task-slug (required): URL-safe identifier
- description (optional): brief summary
- If no arguments, confirm via AskUserQuestion

### 2. [CHECK] Call `dossier` with action=status
- If active spec exists for this slug -> resume mode (skip to Step 7)
- If no spec -> creation mode (continue)

### 3. [REQUIREMENTS] Interactive gathering (max 3 questions)
- What is the goal? (one sentence)
- What does success look like? (measurable criteria)
- What is explicitly out of scope?

### 4. [RESEARCH] Knowledge base + codebase exploration
- Call `knowledge` to search for relevant best practices and patterns
- Read key source files relevant to the task
- Note any relevant prior art or conventions

### 5. [DELIBERATION] Multi-perspective analysis (inline, single pass)

Analyze the requirements from 3 perspectives in a single structured response.
Do NOT spawn sub-agents. Think through each perspective yourself:

#### Perspective 1: Architect
- Propose a concrete architecture (components, data flow, interfaces)
- Name specific files, functions, data structures — no hand-waving
- List 2-3 alternative approaches considered and why rejected
- Define key technical decisions with rationale
- Propose task breakdown ordered by dependency

#### Perspective 2: Devil's Advocate
- List 5-7 things that could go wrong with the proposed architecture
- Identify hidden complexity and underestimated effort
- Surface edge cases the requirements don't mention
- Challenge assumptions: is the scope right? Are success criteria measurable?
- For each concern, state WHY it's a problem and how to mitigate

#### Perspective 3: Researcher
- Search the codebase for existing patterns that apply
- Identify reusable code, libraries, or infrastructure
- Find applicable design patterns and their trade-offs
- Recommend proven approaches to adopt vs build custom

#### Synthesis
After considering all 3 perspectives:
1. List points of AGREEMENT (settled decisions)
2. List points of CONFLICT and for each:
   - State both sides clearly
   - Resolve with rationale, or flag for user decision
3. Produce a unified design incorporating:
   - The Architect's structure
   - Mitigations for the Devil's Advocate's concerns
   - The Researcher's proven patterns

### 6. [CREATE SPEC] Save to .alfred/specs/

1. Call `dossier` with action=init to create the spec directory (skip if already exists)
2. Call `dossier` with action=update for each file:
   - **requirements.md**: Goals, success criteria, out of scope (from Step 3)
   - **design.md**: Unified design (from Step 5 synthesis), alternatives considered
   - **decisions.md**: All decisions with rationale + alternatives + which perspective
     proposed each. Flag unresolved conflicts.
   - **session.md**: Current position + task breakdown as Next Steps

3. **Assign confidence scores** (1-10) to each section using HTML comments:
   ```markdown
   ## API設計 <!-- confidence: 8 -->
   Structure approach (Architect + Researcher aligned, prior art confirms)

   ## 認証方式 <!-- confidence: 3 -->
   OAuth2 or API Key — unresolved (needs user decision)
   ```
   Scale: 1-3 low (speculation), 4-6 medium (inference), 7-9 high (evidence), 10 certain.
   Items scoring ≤ 5 are flagged in Step 7 output for user attention.

### 7. [OUTPUT] Confirm to user

```
Spec created for '{task-slug}'.

Deliberation: 3 perspectives analyzed (Architect, Devil's Advocate, Researcher)
- Settled decisions: N
- Conflicts resolved: N (by evidence)
- Escalated to you: N (need your input)

Confidence: requirements avg X.X (N items ≤ 5), design avg X.X

Spec files: .alfred/specs/{task-slug}/
- requirements.md ✓
- design.md ✓
- decisions.md ✓
- session.md ✓

[If escalated conflicts exist:]
Before starting, please decide on these open questions:
1. <conflict description> — Option A vs Option B
2. ...
```

### 8. [APPROVAL GATE] Wait for user review

After spec creation, set the task to pending review and wait for user approval:

1. Call `dossier` with action=update, file=session.md to set review_status:
   - Add `## Review Status\npending` to session.md
2. Tell the user:
   ```
   Spec ready for review. Open `alfred dashboard` → Specs tab → press 'r' to review.
   Or review the files directly and tell me "approved" or your feedback.
   ```
3. **STOP and wait** — do not proceed until the user confirms.
4. When the user says they've reviewed, call `dossier` with action=review to check:
   - If `review_status: approved` → done, present completion summary
   - If `review_status: changes_requested` → read comments, apply fixes, then go back to step 8
   - If `review_status: pending` → remind the user to review

## Resume Mode (from Step 2)

If an active spec already exists:
1. Call `dossier` with action=status to get current session state
2. Read spec files in recovery order:
   - session.md (where am I?)
   - requirements.md (what am I building?)
   - design.md (how?)
   - decisions.md (why these choices?)
3. Present summary: "Resuming task '{slug}'. Last position: {current_position}."
4. Ask: "Continue from here, or update the plan?"

## Troubleshooting

- **Spec init fails**: Check if `.alfred/specs/{slug}` already exists. Use `dossier` action=status first.
- **User doesn't answer requirements questions**: Proceed with reasonable defaults, flag assumptions with low confidence scores (1-3).

## Guardrails

- Do NOT skip requirements gathering — even for "obvious" tasks
- Do NOT spawn sub-agents — all deliberation is inline (rate limit prevention)
- Do NOT leave decisions.md empty — record ALL deliberation outcomes
- Do NOT create tasks without success criteria
- ALWAYS record alternatives considered with rationale
- ALWAYS record which perspective proposed each decision
- ALWAYS update session.md with current position after plan completion
- Maximum 20 turns total — force convergence if analysis is not settling
