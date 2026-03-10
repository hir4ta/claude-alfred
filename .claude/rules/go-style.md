---
paths:
  - "**/*.go"
---

# Go Style & Naming

## Naming

- Package: lowercase, single-word, no underscores (`tabwriter` not `tab_writer`)
- Avoid meaningless names: `util`, `common`, `helper`, `misc`
- Do not repeat package name in exports: `chubby.File` not `chubby.ChubbyFile`
- MixedCaps for all identifiers, never snake_case or ALL_CAPS
- Constants: `MaxPacketSize` not `MAX_PACKET_SIZE`, name by role not value
- Initialisms: consistent casing (`URL` or `url`, never `Url`; `appID` not `appId`)
- Getters: omit `Get` prefix (`Owner()` not `GetOwner()`), setter is `SetOwner()`
- Receivers: 1-2 letter abbreviation of the type, never `me`/`this`/`self`, consistent across methods
- Interfaces: one-method → method name + `-er` (`Reader`, `Writer`)
- Error vars: prefix with `Err` (`ErrNotFound`)
- Variable name length proportional to scope: `i` for loops, descriptive for wider scope

## Formatting

- Always `gofmt`/`goimports` compliant
- Import groups (blank-line separated): 1. stdlib, 2. third-party/project
- Blank imports (`_ "pkg"`) only in main or test packages

## Receiver Type

- Pointer: mutates receiver, contains sync primitives, large struct, or when in doubt
- Value: map/func/chan, small immutable struct, simple basic type
- Be consistent: do not mix value and pointer receivers on the same type
