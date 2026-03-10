---
paths:
  - "**/*_test.go"
---

# Go Testing

## Table-Driven Tests

- Store cases in a slice of structs; iterate with `t.Run()` for named subtests
- Use field names in struct literals; omit zero-valued fields for clarity
- Each test case must be independent

## Error Messages

- Always include: function called, input given, actual result, expected result
- Format: `Func(input) = actual, want expected`
- Use `t.Errorf` for non-fatal; `t.Fatalf` only when subsequent checks are meaningless

## Helpers

- Helpers that do setup/cleanup must call `t.Helper()` for accurate error reporting
- Never call `t.Fatal` from a goroutine other than the test function itself

## General

- Test files pair with source files: `auth.go` / `auth_test.go`
- Prefer real implementations over hand-written mocks
- Mark independent tests with `t.Parallel()` at both test and subtest level
