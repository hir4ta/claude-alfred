---
paths:
  - "internal/store/**"
---

# Store Implementation Patterns

## Vector Search
- BLOB storage + Go native cosine similarity (no sqlite-vec)
- Dimension validation on insert
- `embeddings.source` always "docs" (source_type filtered via docs table JOIN after RRF fusion)

## SQL Safety
- FTS query sanitization: individual terms sanitized via `JoinFTS5Terms()` before OR-joining
- LIKE prefix queries: use `escapeLIKEPrefix()` + `ESCAPE '\'` clause to prevent wildcard injection
- RecordFeedback uses UPSERT (INSERT...ON CONFLICT) to handle docs without prior RecordInjection

## Schema
- DB schema V6+: incremental migration (V3+ preserves data, legacy schemas rebuilt)
- Store.DB() is test-only; production code uses Store methods (no raw SQL outside internal/store)
