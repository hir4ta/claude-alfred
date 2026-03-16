---
paths:
  - "internal/store/**"
---

# Store Implementation Patterns

## Vector Search
- BLOB storage + Go native cosine similarity (no sqlite-vec)
- Dimension validation on insert
- `embeddings.source` always "records" (source_type filtered via records table JOIN)

## SQL Safety
- LIKE queries: use `escapeLIKEPrefix()` / `escapeLIKEContains()` + `ESCAPE '\'` clause to prevent wildcard injection
- SearchMemoriesKeyword: LIKE-based fallback for no-Voyage-key mode

## Schema
- DB schema V7: validity windows + memory versioning (V6→V7 additive migration)
- Tables: records (memories/specs/project), embeddings (vector BLOBs), schema_version, records_fts (FTS5), tag_aliases, session_links
- `enabled` column: INTEGER NOT NULL DEFAULT 1; all search queries filter by `enabled = 1`
- `valid_until` TEXT (DATETIME): memories past expiry excluded from search (same as enabled=0)
- `review_by` TEXT (DATETIME): past-due memories trigger SessionStart warning
- `superseded_by` INTEGER REFERENCES records(id) ON DELETE SET NULL: memory version chain (max 5)
- All search queries also filter: `(valid_until IS NULL OR valid_until > datetime('now')) AND superseded_by IS NULL`
- SetEnabled scoped to source_type=memory (prevents accidental spec/project disabling)
- Store.DB() is test-only; production code uses Store methods (no raw SQL outside internal/store)

## Vitality
- SubTypeHalfLife(subType) in fts.go: assumption=30d, inference=45d, general=60d, pattern=90d, decision=90d, rule=120d
- ComputeVitality/ListLowVitality in docs.go: on-demand composite score (0-100)
- DetectConflicts threshold 0.70 with classifyConflict keyword polarity (contradiction vs duplicate)
