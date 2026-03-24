# Design — fix-spec-api-project-path

## 概要

Spec関連API 4エンドポイントに `?project=` パラメータ対応を追加する。

## 修正対象

<!-- confidence: 9 | source: code | grounding: verified -->

| # | エンドポイント | ファイル | 行 | 問題 |
|---|---|---|---|---|
| 1 | `GET /api/tasks/:slug/specs` | `src/api/server.ts` | 298-305 | projectPath固定 |
| 2 | `GET /api/tasks/:slug/specs/:file/history` | `src/api/server.ts` | 256-278 | projectPath固定 |
| 3 | `GET /api/tasks/:slug/specs/:file/versions/:version` | `src/api/server.ts` | 280-296 | projectPath固定 |
| 4 | `GET /api/tasks/:slug/validation` | `src/api/server.ts` | 307-328 | projectPath固定 + readActiveState |

## 修正パターン

既存の正しい実装（`/api/tasks/:slug/specs/:file`, L232-254）と同一パターンを適用:

```typescript
const filterProjectId = getProjectFilter(c.req.query("project"));
let targetPath = projectPath;
if (filterProjectId) {
    const filterProj = getProject(store, filterProjectId);
    if (!filterProj) return c.json({ error: "project not found" }, 404);
    targetPath = filterProj.path;
}
const sd = new SpecDir(targetPath, slug);
```

### validation エンドポイント補足

`readActiveState(projectPath)` も `readActiveState(targetPath)` に変更が必要。

## トレーサビリティ

| Req | Component | Task | Test |
|---|---|---|---|
| bugfix.json | src/api/server.ts | T-1.1 | 手動確認 |
