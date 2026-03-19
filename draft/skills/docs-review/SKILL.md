---
name: docs-review
description: >
  Multi-agent technical documentation review. Spawns 5 parallel review agents,
  each evaluating docs from a different perspective (structure, content/accuracy,
  writing quality, audience fitness, maintainability), then aggregates findings
  into a scored report. Use when reviewing technical documentation, evaluating
  doc quality, auditing docs before release, or wanting a comprehensive doc
  review. NOT for code review (use a code review tool). NOT for writing docs
  (just ask directly).
user-invocable: true
argument-hint: "[docs-directory-path]"
---

# /docs-review — Multi-Agent Documentation Review

You are the **documentation review orchestrator**. You coordinate 5 specialized
review agents that evaluate documentation from different perspectives, then
aggregate their findings into a unified scored report.

## Evaluation Criteria

All agents use the shared evaluation rubric:
- **[evaluation-criteria.md](evaluation-criteria.md)** — 29 criteria across 6 categories (A-F), 0-3 scoring scale

Read this file first to understand the full rubric before dispatching agents.

## Phase 0: Setup

1. Parse `$ARGUMENTS` to determine the docs directory to review
   - Path given → use that path
   - No arguments → ask the user which directory to review
2. Read [evaluation-criteria.md](evaluation-criteria.md) to load the rubric
3. Verify the target directory exists and list its files

## Phase 1: Dispatch Review Agents (parallel)

Spawn **5 agents in parallel** using the Agent tool. Each agent receives:
- The evaluation criteria (relevant categories only)
- The target docs directory path
- The list of files to review
- Instructions to output results in the structured format below

### Agent Assignments

| Agent | Categories | Focus |
|-------|-----------|-------|
| **structure-reviewer** | A (A1-A5) | Diataxis separation, navigation, info architecture, progressive disclosure, cross-references |
| **content-reviewer** | B (B1-B5) + E (E1-E4) | Accuracy vs codebase, completeness, freshness, code examples, error docs, API surface, data models, type refs, structural consistency |
| **writing-reviewer** | C (C1-C6) | Clarity, style consistency, actionable instructions, formatting, terminology, accessibility |
| **audience-reviewer** | D (D1-D4) | New joiner experience, use cases, onboarding effectiveness, flow visualization (Mermaid) |
| **maintainability-reviewer** | F (F1-F5) | Code proximity, single source of truth, templates, staleness detection, feedback mechanisms |

### Agent Prompt Template

Each agent prompt MUST include:

1. The agent's assigned categories and their full criteria from evaluation-criteria.md
2. The "チェック方法" (check method) for each criterion
3. The scoring rubric (0-3 scale with definitions)
4. The target directory path and file list
5. Instructions to use Read, Glob, Grep to verify claims
6. The output format below

### Agent Output Format

Each agent must return results in this exact format:

```
## Review: {Agent Name}

### Scores

| Criterion | Score | Evidence (1 sentence) |
|-----------|-------|----------------------|
| A1 | 2 | ... |
| ... | ... | ... |

### Issues Found

1. **[Critical/High/Medium/Low]** Description — file:line
2. ...

### Strengths

1. ...
```

## Phase 2: Aggregation

After all 5 agents return:

1. **Collect** all scores into a unified table
2. **Check blockers**: B1=0, A3=0, or D1=0 → force "要改善" regardless of total
3. **Calculate** category subtotals and grand total (out of 87)
4. **Deduplicate** issues that appear in multiple agents' findings
5. **Sort** issues by severity: Critical > High > Medium > Low
6. **Cap** at 20 issues maximum (prioritize higher severity)
7. **Determine** overall rating from the scoring table in evaluation-criteria.md

## Phase 3: Output

Present the aggregated report to the user:

```markdown
## ドキュメントレビュー結果

### スコアサマリー

| 基準 | スコア | 根拠（1文） |
|------|--------|------------|
| A1 | 2 | ... |
| A2 | 3 | ... |
| ... | ... | ... |

### カテゴリ別スコア

| カテゴリ | スコア | 満点 |
|----------|--------|------|
| A. 構造と構成 | X | 15 |
| B. コンテンツ品質 | X | 15 |
| C. 文章品質 | X | 18 |
| D. 対象読者への適切さ | X | 12 |
| E. リファレンス品質 | X | 12 |
| F. 保守性 | X | 15 |
| **合計** | **XX** | **87** |

**総合評価: {不十分/要改善/合格/良好/優秀}**
{ブロッカー条件に該当する場合はここに記載}

### 検出された問題（重要度順）

1. **[Critical]** 問題の説明 — 該当ファイル:行
2. **[High]** ...
3. ...

### 良い点

1. ...

### 改善提案（優先度順）

1. 提案内容 — 期待される効果
2. ...
```

## Guardrails

- ALWAYS read evaluation-criteria.md before dispatching agents
- ALWAYS spawn all 5 agents in a single message (parallel execution)
- Each agent MUST use Read/Glob/Grep to verify — never score from assumptions
- content-reviewer MUST cross-reference docs against actual code (package.json, prisma schema, source files)
- Prefer false negatives over false positives — noise erodes trust
- If a criterion is not applicable (e.g., no Mermaid diagrams expected), score N/A and exclude from totals
- NEVER modify documentation files — this skill is read-only
- Severity definitions: Critical = blocker criteria at 0, High = 必須 category at 1 or factual errors, Medium = 重要 category at 1 or inconsistency, Low = 推奨 category improvements

## Troubleshooting

- **Agent returns incomplete results**: Re-send with explicit missing criteria listed
- **Too many issues (>20)**: Keep top 20 by severity, note "N additional findings omitted"
- **Docs directory not found**: Ask the user for the correct path
- **No docs to review**: Report "No documentation files found in the specified directory"
