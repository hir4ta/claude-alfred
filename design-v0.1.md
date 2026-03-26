# alfred v0.1.0 設計書

Claude Code の性能を倍増させる執事。Hooks + Skills + Agents のみ。

## 思想

**リサーチが示した3つの変数だけに集中する:**

1. **タスクサイズ制御** — 15 LOC以下・単一ファイルに保つ (80%+ 成功率)
2. **コンテキストの質** — 必要最小限の情報だけ注入 (少ない方が強い)
3. **検証ループ** — 「何を検証すべきか」を伝える (HOWではなくWHAT)

**やらないこと:** 知識DB, ベクトル検索, 品質スコア, TUI, MCP, Convention/Security/Layer チェック

## アーキテクチャ

```
alfred CLI (init / hook / doctor)
    │
    ├── alfred init
    │   └── ~/.claude/ に hooks, skills, agents, rules を配置
    │
    └── alfred hook <event>
        └── stdin JSON → 処理 → stdout JSON or exit 2
```

### 3つの柱

| 柱 | Hook | 強制レベル | リサーチ根拠 |
|---|---|---|---|
| 壁 | PostToolUse → PreToolUse | DENY (exit 2) | 検証は最もレバレッジが高い (Anthropic公式) |
| Plan増幅 | UserPromptSubmit + PermissionRequest | additionalContext + DENY | タスク分解の質が成功率を決める (SWE-bench) |
| 実行ループ | Stop + PreCompact | block + additionalContext | 2回失敗→/clear (Anthropic公式), 構造化ハンドオフ (Sourcegraph) |

## Hook 設計

### PostToolUse — 品質壁 (検出)

**トリガー**: Edit/Write/Bash 完了後
**タイムアウト**: 5s

1. **Edit/Write 後**: gates.json の on_write を実行 (lint, typecheck)
   - 失敗 → pending-fixes.json に書込 → additionalContext で「修正してください」
   - 成功 → pending-fixes.json から該当ファイルを削除

2. **Bash (git commit) 後**: gates.json の on_commit を実行 (test)
   - 失敗 → additionalContext で「テストが失敗しています」+ 失敗テスト名

3. **Bash (test失敗) 後**: 失敗回数をカウント
   - 2回連続同じ失敗 → additionalContext で「/clear して新しいアプローチを試してください」

### PreToolUse — 品質壁 (ブロック)

**トリガー**: Edit/Write 実行前
**タイムアウト**: 3s

1. pending-fixes.json を読む
2. 修正対象ファイルへの Edit は許可
3. 他ファイルへの Edit は **DENY** (permissionDecision: "deny")
4. Pace チェック: 35分+コミットなし+多ファイル変更 → **DENY**

### UserPromptSubmit — Plan 増幅

**トリガー**: ユーザーがプロンプト送信時
**タイムアウト**: 10s

1. **Plan mode 検出** (`permission_mode === "plan"`):
   - Plan テンプレートを additionalContext で注入
   - テンプレートにレビューゲートを含める

2. **通常モード**:
   - 大タスク検出 (長いプロンプト or 複数ファイル言及) → 「Plan mode で設計してから実装してください」

### PermissionRequest (ExitPlanMode) — Plan 検証

**トリガー**: Claude が Plan mode を終了しようとした時
**matcher**: "ExitPlanMode"

1. Plan ファイルを読む
2. レビューゲートが含まれているか検証
3. 含まれていなければ **DENY** + 「レビューゲートを追加してください」

### SessionStart — 初期コンテキスト

**トリガー**: セッション開始時
**タイムアウト**: 5s
**注意**: settings.json に書く (Plugin hooks では additionalContext が動かないバグ #16538)

1. .alfred/ ディレクトリがなければ自動作成 (gates.json 自動検出)
2. 前回セッションのハンドオフ情報があれば注入
3. プロジェクトプロファイル (言語, テストFW, リンター) を注入

### Stop — レビュー強制 + Pace

**トリガー**: Claude が応答を終了しようとした時
**タイムアウト**: 5s
**注意**: settings.json に書く (Plugin hooks では exit 2 が動かないバグ #10412)

1. `stop_hook_active` チェック (無限ループ防止)
2. pending-fixes が残っていれば **block** + 「修正してから完了してください」
3. Pace 警告: 35分超 → additionalContext で警告

### PreCompact — 構造化ハンドオフ

**トリガー**: コンテキストコンパクション前
**タイムアウト**: 10s

1. 現在の作業状態を .alfred/.state/handoff.json に保存:
   - 何をやっているか (タスクの要約)
   - どこまで進んだか (完了/未完了タスク)
   - 変更したファイル
   - pending-fixes の状態
   - 次にやるべきこと

## Plan テンプレート

UserPromptSubmit で注入するテンプレート:

```
## Context
<なぜこの変更が必要か>

## Tasks
各タスクは以下を満たすこと:
- 1ファイルのみ変更
- 15行以下の差分
- 検証テスト (具体的なテストファイル名と関数名) を明記

### Task 1: <name>
- File: <変更するファイル>
- Change: <何をするか (振る舞いベース)>
- Verify: <テストファイル:テスト関数名>
- Boundary: <してはいけないこと>

(タスクごとに繰り返し)

## Review Gates
- [ ] Design Review: 実装開始前に /alfred:review で設計をレビュー
- [ ] Phase Review: 3タスクごとに /alfred:review で実装をレビュー
- [ ] Final Review: 全タスク完了後に /alfred:review で全変更をレビュー
```

## /alfred:review Skill

マルチエージェントレビュー。3つの視点から独立してレビューし、Judge がフィルタリング。

### 視点
1. **correctness**: ロジック・エッジケース・テスト漏れ
2. **design**: シンプルさ・凝集度・依存方向
3. **security**: 入力検証・機密情報・インジェクション

### Judge 基準 (HubSpot パターン)
- **Succinctness**: 簡潔で要点を突いているか
- **Accuracy**: 技術的に正しいか (コードベースのコンテキストで)
- **Actionability**: 具体的な修正提案があるか

### 起動タイミング
- ユーザーが `/alfred:review` で手動起動
- Plan テンプレートに組み込まれた Review Gate として Claude が自発的に起動

## Gates (lint/type 実行)

### .alfred/gates.json

```json
{
  "on_write": {
    "lint": { "command": "...", "timeout": 3000 },
    "typecheck": { "command": "...", "timeout": 10000, "run_once_per_batch": true }
  },
  "on_commit": {
    "test": { "command": "...", "timeout": 30000 }
  }
}
```

`alfred init` が package.json/tsconfig.json/biome.json 等から自動検出して生成。

## 状態ファイル

```
.alfred/
├── gates.json                  # CI風ゲート設定 (git管理)
└── .state/                     # 一時状態 (gitignore)
    ├── pending-fixes.json      # 未修正 lint/type エラー
    ├── session-pace.json       # Pace 追跡 (最終コミット時刻, 変更ファイル数)
    ├── handoff.json            # 構造化ハンドオフ (PreCompact で保存)
    └── project-profile.json    # 言語/テストFW/リンター
```

## CLI コマンド

```bash
alfred init     # ~/.claude/ にセットアップ + .alfred/ 作成
alfred doctor   # ヘルスチェック (hooks登録確認, gates.json存在確認)
alfred hook <event>  # Hook ハンドラー実行 (stdin JSON)
```

## src/ 構造

```
src/
├── cli.ts              # citty: init / hook / doctor
├── init.ts             # セットアップロジック
├── doctor.ts           # ヘルスチェック
├── hooks/
│   ├── dispatcher.ts   # event → handler ルーティング
│   ├── post-tool.ts    # lint/type gate + pending-fixes 書込
│   ├── pre-tool.ts     # pending-fixes → DENY + pace
│   ├── user-prompt.ts  # Plan テンプレート注入
│   ├── session-start.ts # プロファイル + ハンドオフ復元
│   ├── stop.ts         # レビュー強制 + pace
│   └── pre-compact.ts  # 構造化ハンドオフ保存
├── gates/
│   ├── runner.ts       # gate コマンド実行
│   └── detect.ts       # package.json → gates.json 自動検出
├── state/
│   ├── pending-fixes.ts # read/write pending-fixes.json
│   ├── pace.ts          # pace 追跡
│   └── handoff.ts       # ハンドオフ read/write
└── types.ts            # 共通型定義
```

## 依存関係

### devDependencies のみ (bun build でバンドル)
- citty — CLI フレームワーク
- typescript — 型チェック
- vitest — テスト
- @biomejs/biome — lint/format

### 削除
- @modelcontextprotocol/sdk (MCP 不要)
- @opentui/core, @opentui/react, react (TUI 不要)
- zod (バリデーション不要 — 型で十分)
- @vitest/coverage-v8 (カバレッジ不要 — テストが通ればよい)

## init が配置するファイル

### ~/.claude/settings.json (hooks追加)
```json
{
  "hooks": {
    "PostToolUse": [{"type": "command", "command": "alfred hook post-tool", "timeout": 5000}],
    "PreToolUse": [{"type": "command", "command": "alfred hook pre-tool", "timeout": 3000}],
    "UserPromptSubmit": [{"type": "command", "command": "alfred hook user-prompt", "timeout": 10000}],
    "SessionStart": [{"type": "command", "command": "alfred hook session-start", "timeout": 5000}],
    "Stop": [{"type": "command", "command": "alfred hook stop", "timeout": 5000}],
    "PreCompact": [{"type": "command", "command": "alfred hook pre-compact", "timeout": 10000}],
    "PermissionRequest": [{"type": "command", "command": "alfred hook permission-request", "timeout": 5000, "matcher": "ExitPlanMode"}]
  }
}
```

### ~/.claude/skills/alfred-review/SKILL.md
/alfred:review スキル定義

### ~/.claude/agents/alfred-reviewer.md
レビューサブエージェント定義

### ~/.claude/rules/alfred-quality.md
品質ルール (20行以内)

## Phase 0 の実施手順

1. `src/` 全削除
2. 不要な devDependencies 削除
3. package.json version → 0.1.0
4. 空の src/cli.ts + src/hooks/dispatcher.ts 作成
5. `task check` が通る状態にする
6. コミット: "chore: v0.1.0 skeleton — full rewrite"
