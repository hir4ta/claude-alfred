---
name: review
description: >
  Knowledge-powered code review with multi-agent architecture. Spawns 3 specialized
  sub-reviewers (security, logic, design) in parallel for thorough coverage, then
  aggregates findings. Use when: (1) before committing, (2) after a milestone,
  (3) want a second opinion on changes.
user-invocable: true
argument-hint: "[focus area]"
allowed-tools: Read, Glob, Grep, Agent, Bash(git diff:*, git log:*, git show:*, git status:*), mcp__plugin_alfred_alfred__knowledge, mcp__plugin_alfred_alfred__spec
context: fork
---

# /alfred:review — Multi-Agent Code Review

Review changes with 3 specialized sub-reviewers running in parallel, then aggregate findings.

## Key Principles
- Surface **actionable** findings, not noise. Every finding should help the developer.
- Prioritize critical issues (security, bugs, scope violations) over style.
- Reference sources so the developer can verify and learn.
- Prefer false negatives over false positives — noise erodes trust.

## Steps

### 1. [CONTEXT] Gather review context

- Call `spec` with action=status to check if an active spec exists
- Run `git diff --cached` (or `git diff` / `git diff HEAD~3..HEAD` as appropriate)
- Run `git log --oneline -5` for recent commit context
- If a focus area is provided in $ARGUMENTS, note it for sub-reviewers

### 2. [PARALLEL REVIEW] Spawn 3 specialized agents simultaneously

Launch **all 3 agents in a single message** using the Agent tool. Pass each the
diff content and any relevant context (spec, focus area).

**Agent 1: Security Review**
- TOCTOU vulnerabilities, race conditions in auth checks
- IDOR: parameters used in queries without ownership verification
- Missing input validation at trust boundaries
- Hardcoded secrets, credentials in code or tests
- SSRF, injection (SQL/command/XSS), missing CSRF protection
- Session/auth weaknesses, deprecated crypto, sensitive data in logs
- Missing rate limiting on auth endpoints

**Agent 2: Logic & Correctness Review**
- Off-by-one, nil dereference, empty collection handling
- Division by zero, integer overflow, float precision
- Error swallowing, partial failure cleanup, resource leaks
- defer symmetry (open/close pairs), context cancellation propagation
- Race conditions, goroutine leaks, missing synchronization
- Boundary values, unit mismatches, exhaustive case handling

**Agent 3: Design & Architecture Review**
- Scope violations against spec requirements
- Contradictions with recorded decisions
- Breaking API contracts, removing safeguards without reason
- N+1 queries, unbounded collection growth, missing LIMIT
- Implicit coupling, inconsistent patterns, over-engineering
- Reintroduced anti-patterns that were previously fixed

### 3. [AGGREGATE] Collect and unify findings

- Deduplicate findings that describe the same issue from different angles
- Discard clear false positives (intentional design choices, documented exceptions)
- Sort by severity: Critical > Warning > Info
- Cap at 15 findings total

### 4. [SPEC CHECK] Cross-reference with spec (if active)

- Read decisions.md — verify no contradictions with recorded decisions
- Read requirements.md — verify changes are within defined scope
- Elevate spec violations to Critical severity

### 5. [KNOWLEDGE CHECK] Search for relevant best practices

- Call `knowledge` with queries derived from the diff (changed patterns, file types)
- Cross-reference sub-reviewer findings with knowledge base
- Add best practice references as Info findings only if highly relevant

### 6. [OUTPUT] Present unified report

```
## Review Summary

Reviewed N files, M lines changed.
Sub-reviewers: security ✓, logic ✓, design ✓

### Critical (must fix)
[CATEGORY] file:line — description
  → suggestion (source: spec decision / KB entry / convention)

### Warning (should review)
...

### Info (good to know)
...

## Verdict
[PASS | PASS WITH WARNINGS | NEEDS FIXES]
N critical, N warnings, N info findings.
```

## Exit Criteria
- All 3 sub-reviewers completed
- Findings deduplicated and prioritized
- Spec checked (if active)
- Knowledge base consulted for relevant best practices
- Clear verdict provided
- If no findings: "No issues found. Changes look good."
