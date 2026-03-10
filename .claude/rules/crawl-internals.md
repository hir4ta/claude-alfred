---
paths:
  - "cmd/alfred/hooks_session.go"
  - "internal/install/**"
---

# Crawl & Seed Patterns

- Auto-crawl lock: parent creates with O_EXCL + writes child PID; child only cleans up on exit (no overwrite)
- Crawl() accepts context.Context for cancellation propagation to HTTP requests
- ApplySeedData: embedder is optional (nil -> FTS-only, no vector embeddings)
- Crawl() accepts custom sources (`[]CustomSource`) for user-defined documentation URLs
- Auto-crawl: diff-based with HTTP conditional requests (ETag/If-Modified-Since) via crawl_meta table
