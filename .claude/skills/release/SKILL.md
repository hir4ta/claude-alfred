---
name: release
description: Release claude-alfred. Version can be specified or auto-detected
allowed-tools: Bash(go *, git *, gh *), Read, Edit, Glob, Grep, Write
---

Release claude-alfred.

## Version Detection

Arguments: `$ARGUMENTS`

- If a version is specified (e.g., `0.13.14`, `v0.14.0`) → use it (strip `v` prefix)
- If empty → auto-detect:
  1. Get latest tag via `git describe --tags --abbrev=0`
  2. Get commit list via `git log <latest-tag>..HEAD --oneline`
  3. Analyze commit messages for semver bump:
     - **minor bump** (0.13.x → 0.14.0): commits contain feature keywords (Add, feat, new feature, implement, etc.)
     - **patch bump** (0.13.13 → 0.13.14): everything else (fix, refactor, improve, optimize, update, chore, schema, test, etc.)
  4. Show detected version and reasoning to user, get confirmation before proceeding

## Pre-checks

1. Check working tree with `git status`
2. If uncommitted changes exist → ask user whether to include in release commit
3. If no changes and no commits since last tag → abort (nothing to release)

## Validation Gate

Run in order; **abort release if any fails**:

```
go build -o /dev/null ./cmd/alfred
go test ./... -count=1
go vet ./...
```

## Plugin Bundle + Marketplace Update

1. **Check plugin/ source of truth**: `internal/install/content/` (rules, skills, agents) is the source.
   `plugin/` is generated — if plugin/ was edited directly, sync `internal/install/content/` first.
2. Regenerate plugin/ directory (ldflags version injection is **required**):
   ```
   go run -ldflags "-X main.version=<VERSION>" ./cmd/alfred plugin-bundle ./plugin
   ```
3. Update `plugins[0].version` in `.claude-plugin/marketplace.json` to `<VERSION>`

Both plugin/ and marketplace.json are included in the same release commit.

## README Badges

shields.io dynamic badges auto-update. Only update Go version badge if `go.mod` Go version changes.

## Commit & Tag

1. Stage all release files: `git add -f plugin/ .claude-plugin/marketplace.json`
   - Include other uncommitted files if agreed with user
2. Commit message: `v<VERSION>: <one-line summary of commits>` (in English)
   - Generate summary from `git log <prev-tag>..HEAD --oneline`
   - **NEVER add Co-Authored-By** (public repository)
3. `git tag v<VERSION>`

## Local Binary Update

```
go install -ldflags "-X main.version=<VERSION> -X main.commit=$(git rev-parse --short HEAD) -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" ./cmd/alfred
```

## Push & CI

**NEVER use `--tags`** (pushes all local tags)

```
git push origin main
git push origin v<VERSION>
```

### CI Monitoring

1. `gh run list --limit 1` — check Release workflow started
2. `gh run watch <run-id>` — watch until completion
3. If CI fails → fix the issue, delete the tag (`git tag -d v<VERSION> && git push origin :refs/tags/v<VERSION>`), and re-release

## Verify Release

After CI succeeds, verify assets:
```
gh release view v<VERSION> --json assets --jq '.assets[].name'
```
Must include: `alfred_darwin_arm64.tar.gz`, `alfred_darwin_amd64.tar.gz`, `alfred_linux_amd64.tar.gz`, `alfred_linux_arm64.tar.gz`, `checksums.txt`

## Homebrew

Auto-updated via GoReleaser `brews` section. No manual action needed.

## Completion Report

| Item | Value |
|------|-------|
| Version | v<VERSION> |
| Commit | <hash> |
| CI | success/failure (duration) |
| Release URL | https://github.com/hir4ta/claude-alfred/releases/tag/v<VERSION> |
| Homebrew | auto-updated via GoReleaser |
