---
paths:
  - "src/store/**"
  - "src/mcp/**"
  - ".alfred/knowledge/**"
---

# Knowledge & Search Internals

- Knowledge persistence: `.alfred/knowledge/{decisions,patterns,rules}/*.json` = source of truth; DB `knowledge_index` = derived search index
- Knowledge file format: JSON (mneme-compatible schemas: DecisionEntry, PatternEntry, RuleEntry)
- Sub-type classification: decision/pattern/rule (general abolished); boost: rule=2.0x, decision=1.5x, pattern=1.3x
- Knowledge maturity: hit_count tracks search appearances, last_accessed for staleness
- Knowledge promotion: pattern→rule (15+ hits); manual confirmation via ledger promote
- Ledger tool actions: search, save, promote, candidates, reflect, audit-conventions
- Search pipeline: Voyage vector search → rerank → recency signal → hit_count tracking → FTS5 fallback → keyword fallback. Returns ScoredDoc[] with per-doc score + matchReason
- FTS5: knowledge_fts virtual table with bm25 ranking, auto-synced via triggers (title weighted 3x)
- Tag alias expansion: auth→authentication/login/認証, 16 categories bilingual (EN/JP)
- Knowledge governance: `enabled` column in knowledge_index; disabled entries excluded from search
- Knowledge tab: toggle enabled/disabled via API (PATCH /api/knowledge/{id}/enabled)
- Knowledge files are git-friendly: sharing via repository, diff-reviewable in PRs
- Quality gate (`src/mcp/quality-gate.ts`): ledger save 時に実行。セマンティック重複 (0.90 near_duplicate / 0.85 similar_existing)、アクショナビリティ (ACTIONABILITY_PATTERNS EN/JA)、矛盾検出 (classifyConflict + cosine >= 0.85)。WARNING のみ、BLOCK しない。embedding は await + 3秒タイムアウト、成功時は insertEmbedding に再利用
- Review calibration: review findings は `review-finding` タグ + `enabled=0` + `status=draft` で保存 (通常検索から除外)。`ledger verify outcome=confirmed` → enabled=1 + status=approved、`outcome=rejected` → status=rejected + enabled=0 維持
