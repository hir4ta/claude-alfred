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
- DB schema V6: enabled column for memory governance (V5→V6 additive migration)
- Tables: records (memories/specs/project), embeddings (vector BLOBs), schema_version
- `enabled` column: INTEGER NOT NULL DEFAULT 1; all search queries filter by `enabled = 1`
- SetEnabled scoped to source_type=memory (prevents accidental spec/project disabling)
- Store.DB() is test-only; production code uses Store methods (no raw SQL outside internal/store)
