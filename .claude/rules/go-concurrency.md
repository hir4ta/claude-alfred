---
paths:
  - "**/*.go"
---

# Go Concurrency

- Share memory by communicating (channels), not communicate by sharing memory (mutexes)
- Default to unbuffered channels or size 1; larger buffers require justification
- Prefer synchronous functions; let callers add concurrency
- Make goroutine lifetimes obvious; document when/why they exit
- Never launch fire-and-forget goroutines without coordination
- Never spawn goroutines in `init()`
- Specify channel direction: `<-chan` for receive, `chan<-` for send
- Zero-value `sync.Mutex` is valid; do not use pointer to mutex
- Never copy a struct containing `sync.Mutex`
- `context.Context` is always the first parameter; never store in structs
