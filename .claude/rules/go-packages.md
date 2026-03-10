---
paths:
  - "**/*.go"
---

# Go Package Organization

## Structure

- One package == one directory; all .go files share the same `package` declaration
- Use `internal/` for non-public packages (Go toolchain enforces this)
- Large packages (5-20 files, thousands of lines) are normal in Go; do NOT create tiny packages

## File Splitting

- Split when a file has multiple unrelated responsibilities, not just because it's long
- All files in the same directory share the same package; splitting does not require a new package
- Group by logical concern: one file per major type/subsystem or functional area
- Do NOT create one file per type (not Go convention)
- Do NOT create a new package just to reduce file size

## When to Use Subdirectories

- When code represents a genuinely distinct concern or abstraction boundary
- When the package is independently reusable
- Do NOT create a subdirectory just because a file got long

## Interface Design

- Define interfaces in the consuming package, not the implementing package
- Return concrete types from constructors; let consumers define needed interfaces
- Verify interface compliance at compile time: `var _ io.Reader = (*MyType)(nil)`
