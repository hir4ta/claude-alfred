---
paths:
  - "**/*.go"
---

# Go Error Handling

- Always return `error` as the last return value
- Never discard errors with `_` without an explicit justification comment
- Handle errors immediately with early return; keep happy path at minimal indentation
- Error strings: lowercase, no trailing punctuation (unless proper noun/acronym)
- Use `%w` in `fmt.Errorf` when callers may need `errors.Is()`/`errors.As()`
- Use `%v` when the underlying error is an implementation detail
- Do not log an error AND return it; choose one
- Do not match errors by string content; use `errors.Is()`/`errors.As()`
- Do not panic for normal error conditions
- `os.Exit()` only in `main()`; libraries return errors
