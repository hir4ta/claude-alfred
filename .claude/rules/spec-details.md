---
paths:
  - "src/spec/**"
  - ".alfred/specs/**"
---

# Spec Details

## Slug & Lifecycle
- task_slug: `^[a-z0-9][a-z0-9\-]{0,63}$`; spec.ValidSlug exported regex
- Task lifecycle: active → complete (preserves spec files, sets completed_at) or delete (removes files)
- ActiveTask fields: slug, started_at, status (active/completed), completed_at, size (S/M/L), spec_type (feature/bugfix)
- Spec file locking: advisory flock on `.lock` file (exponential backoff 100/200/400/800ms ~1.5s total, context-aware cancellation, graceful fallback + stderr warning)

## Size & Type System
- SpecSize: S (3 files), M (4 files), L (5 files). XL and D are removed.
- Auto-detected from description length (< 100 → S, < 300 → M, else L)
- SpecType: feature (default, uses requirements.md), bugfix (uses bugfix.md)
- FilesForSize(size, specType): returns file list for any (size, type) combination
- Init functional options: WithSize(SpecSize), WithSpecType(SpecType); InitWithResult returns SpecDir + Size + SpecType + Files
- State files: _active.json (active), _complete.json (completed), _cancel.json (cancelled) — JSON format, auto-migrated from legacy _active.md
- Backward compat: legacy _active.md auto-migrated to _active.json on first read; legacy XL/D sizes are treated as malformed (hard error)

## Spec Files
- Spec v4: 5 files (requirements, design, tasks, test-specs, research); session.md removed — progress tracked via tasks.md; decisions saved via `ledger save sub_type=decision` directly
- Spec cross-references: `@spec:task-slug/file.md` format parsed by `spec.ParseRefs()`, resolved against filesystem
- Spec complete auto-extracts: design.md patterns → permanent knowledge (sub_type=pattern)
- Wave: Closing required in all tasks.md: self-review, CLAUDE.md update, test verification, knowledge save

## Templates
- Spec templates: `src/spec/templates.ts` — inline EN/JA templates rendered via `renderForSize()` (TemplateData: TaskSlug, Description, Date, SpecType)
- Supported file templates: requirements.md, bugfix.md, delta.md, design.md, tasks.md, test-specs.md, research.md
- Bugfix template: Bug Summary, Severity & Impact P0-P3, Reproduction Steps, Root Cause Analysis with 5 Whys, Fix Strategy, Regression Prevention
- Delta template: Change Summary, Files Affected with CHG-N IDs, Before/After per CHG-N, Rationale, Impact Scope, Test Plan, Rollback Strategy
- Template 2-layer resolution (planned): `.alfred/templates/specs/` (user override) > embedded defaults

## Traceability
- EARS notation: requirements use 6 patterns (Ubiquitous, WHEN, WHILE, WHERE, IF-THEN, Complex)
- Traceability IDs: FR-N (functional), NFR-N (non-functional), DEC-N (decisions), T-N.N (tasks wave.task), TS-N.N (tests)
- Traceability matrix: design.md maps Req ID → Component → Task ID → Test ID
- CHG-N: delta spec change identifiers (logical change unit, scoped per change not per file)

## Confidence & Grounding
- Spec confidence scoring: `<!-- confidence: N | source: TYPE | grounding: LEVEL -->` annotations
- Source: user/design-doc/code/inference/assumption; Grounding: verified/reviewed/inferred/speculative (optional, backward compatible)
- Status returns avg + low_items + low_confidence_warnings (score <= 5 + assumption) + grounding_distribution + grounding_warnings
- Grounding levels: verified (code/test proven) > reviewed (design-reviewed/user-confirmed) > inferred (reasoned from evidence) > speculative (hypothesis)

## Validation (dossier validate)
- 15 checks: required_sections, min_fr_count (S:1+, M:3+, L:5+; bugfix uses substantive content check), traceability (fr_to_task, task_to_fr — supports both `### T-N.N` header and `- [ ] T-N.N` checkbox formats), confidence_annotations, closing_wave, design_fr_references, testspec_fr_references, nfr_traceability (L only), gherkin_syntax, orphan_tests, orphan_tasks, content_placeholder, research_completeness (L only), grounding_coverage (opt-in: L, >30% speculative fails). XL-only checks (xl_wave_count, xl_nfr_required, confidence_coverage) and delta checks removed.

