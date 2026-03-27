# alfred

Claude Code の暴走を止める執事。品質の下限を機械的に守る **evaluator harness**。

> Claude は優秀だが、lint エラーを放置して次のファイルに行く。テストなしでコミットする。自分のコードを褒めてレビューを終える。
> alfred はそれを **物理的に止める**。お願い (advisory) ではなく、exit 2 (DENY) で。

## なぜ evaluator harness か

Anthropic の [Harness Design](https://www.anthropic.com/engineering/harness-design-long-running-apps) 記事が示した核心:

- **自己評価は機能しない** — Claude は自分の仕事の問題を見つけても「大したことない」と自分を説得する
- **独立 evaluator が必須** — generator と evaluator を分離することで品質が跳ねる
- **全コンポーネントは仮定** — 「モデルが単独でできないこと」を encode し、陳腐化したら捨てる

alfred は Claude Code の 14 hooks として動作し、**外部プロセスを evaluator として**、Claude の行動を機械的にゲートする。TypeScript, Python, Go, Rust を自動検出。世の SDD ツールの大半は「お願い」。alfred は「壁」。

## 何を防ぐか

```
Edit → biome check 失敗 → pending-fixes 記録
  → 別ファイルを Edit しようとする → DENY (exit 2)
  → 同じファイルを修正 → biome check 通過 → 解除
```

| 状況 | alfred の行動 |
|---|---|
| lint/type エラーを放置して別ファイルへ | **DENY** — 修正するまでブロック |
| テスト未実行で git commit | **DENY** — テスト pass を要求 |
| レビュー未実行で完了宣言 | **block** — /alfred:review を要求 |
| Plan に Verify フィールドがない | **DENY** — 検証基準なしに実装させない |
| Plan に Success Criteria がない | **DENY** — Sprint contract なしに実装させない |
| 大タスクを Plan なしで直接実行 | **block** — Plan mode を強制 |
| 35分以上コミットなし + 5ファイル変更 | **DENY** — スコープ肥大を阻止 |
| hook 設定を削除しようとする | **DENY** — 自己防衛 |

## 14 Hooks

**壁** — 壊れたコードを通さない
- **PostToolUse**: 編集後に gate 実行 (lint/type/test)。失敗 → pending-fixes
- **PreToolUse**: pending-fixes → DENY。Pace red → DENY。commit gates → DENY

**Plan 増幅** — 設計の質を底上げ
- **UserPromptSubmit**: Plan テンプレート動的注入 + 大タスク block (500+) / advisory (200+)
- **PermissionRequest**: ExitPlanMode 時に Plan 構造 + Success Criteria を検証
- **TaskCompleted**: Plan 完了ステータスの自動同期

**実行ループ** — 中途半端に終わらせない
- **Stop**: 未修正エラー / Plan 未完了 / レビュー未実行 → block
- **PreCompact / PostCompact**: コンテキスト圧縮前後の状態保存・復元
- **SessionStart / SessionEnd**: 自動セットアップ + セッション成果記録 (clean exit率追跡)

**サブエージェント制御** — 品質ルールを伝搬
- **SubagentStart**: サブエージェントに品質ルール注入
- **SubagentStop**: reviewer 出力の検証 + レビュー完了記録

**自己防衛** — harness 自体を守る
- **PostToolUseFailure**: 2回連続失敗 → /clear 提案
- **ConfigChange**: user_settings 変更 → DENY

## 設計原則

1. **壁 > 情報提示** — DENY (exit 2) で止める。additionalContext は無視される前提
2. **リサーチ駆動** — 全設計判断に SWE-bench / Anthropic 記事 / Self-Refine 論文の裏付け
3. **fail-open** — 全 hook は try-catch。alfred の障害で Claude を止めない
4. **少ない方が強い** — コンテキスト注入は 2000 token 予算。指示は 20 行以内
5. **成果追跡** — session outcomes で clean exit 率・pending 推移を計測。`doctor --metrics` で表示
6. **dependencies ゼロ** — 全て devDependencies + bun build バンドル

## インストール

```bash
bun install
bun build.ts
bun link

alfred init       # ~/.claude/ に 14 hooks + skill + agent + rules を配置
alfred doctor     # セットアップの健全性を確認 (8項目 + state整合性検証)
```

## コマンド

```bash
alfred init              # セットアップ (14 hooks + skill + agent + rules + gates 自動検出)
alfred hook <event>      # Hook イベント処理 (Claude Code が呼び出す)
alfred doctor            # ヘルスチェック (bun, hooks, skill, agent, rules, gates, state, path)
alfred doctor --metrics  # DENY/block/respond 発火統計を表示
alfred reset             # .alfred/.state/ を初期化 (--keep-history で履歴保持)
```

`alfred init` が配置するもの:
- `~/.claude/settings.json` — 14 hooks を登録 (既存の hook は保持)
- `~/.claude/skills/alfred-review/SKILL.md` — /alfred:review skill
- `~/.claude/agents/alfred-reviewer.md` — 独立レビュー agent
- `~/.claude/rules/alfred-quality.md` — 品質ルール
- `.alfred/gates.json` — プロジェクトの lint/type/test gate を自動検出

hooks は Claude Code が各イベントで自動呼び出し。手動操作は不要。

## Skills

- `/alfred:review` — 独立コードレビュー。HubSpot 2-stage pattern: Reviewer (独立サブエージェント) が全 findings を報告 → Judge が S/A/A でフィルタ → critical/high は修正後に再レビュー (最大2サイクル)。reviewer は gate 実行 (test/lint/type) + few-shot 例 + anti-self-persuasion 指示で calibration

## Gate 自動検出

`alfred init` がプロジェクトの設定ファイルから lint/type/test gate を自動検出:

| 言語 | 検出元 | on_write (lint/type) | on_commit (test) |
|---|---|---|---|
| **TypeScript** | biome.json / .eslintrc* / tsconfig.json | `biome check {file}` / `eslint {file}` / `tsc --noEmit` | — |
| **TypeScript** | vitest / jest (devDeps) | — | `vitest --changed` / `jest --changedSince` |
| **Python** | pyproject.toml / uv.lock / ruff.toml | `[uv run] ruff check {file}` | `[uv run] pytest` |
| **Python** | pyrightconfig.json / mypy.ini | `[uv run] pyright` / `[uv run] mypy .` | — |
| **Go** | go.mod | `go vet ./...` | `go test ./...` |
| **Rust** | Cargo.toml | `cargo clippy -- -D warnings` | `cargo test` |

`uv.lock` 検出時は自動で `uv run` プレフィクスが付与されます。

## スタック

TypeScript (Bun 1.3+, ESM) / citty (CLI) / vitest (テスト) / Biome (lint)
