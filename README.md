# alfred

Claude Code の暴走を止める執事。品質の下限を機械的に守る **evaluator harness**。

> Claude は優秀だが、lint エラーを放置して次のファイルに行く。テストなしでコミットする。自分のコードを褒めてレビューを終える。
> alfred はそれを **物理的に止める**。お願い (advisory) ではなく、exit 2 (DENY) で。

## なぜ evaluator harness か

Anthropic の [Harness Design](https://www.anthropic.com/engineering/harness-design-long-running-apps) 記事が示した核心:

- **自己評価は機能しない** — Claude は自分の仕事の問題を見つけても「大したことない」と自分を説得する
- **独立 evaluator が必須** — generator と evaluator を分離することで品質が跳ねる
- **全コンポーネントは仮定** — 「モデルが単独でできないこと」を encode し、陳腐化したら捨てる
- **simplest solution possible** — 必要な時だけ複雑性を追加。不要なものは削除

alfred は Claude Code の 13 hooks として動作し、**外部プロセスを evaluator として**、Claude の行動を機械的にゲートする。TypeScript, Python, Go, Rust を自動検出。世の SDD ツールの大半は「お願い」。alfred は「壁」。

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
| Plan (4+ tasks) に Success Criteria がない | **DENY** — 検証基準なしに実装させない |
| Plan (4+ tasks) に具体的な Verify がない | **DENY** — テスト名/コマンドを要求 |
| 大タスクを Plan なしで直接実行 | **advisory** — Plan mode を提案 (block ではない) |
| 60分以上コミットなし + 8ファイル変更 | **DENY** — スコープ肥大を阻止 (Plan ありは 90分/12ファイルまで猶予) |
| hook 設定を変更しようとする | **DENY** — 自己防衛 (非 hook 設定は許可) |

## 13 Hooks (6 enforcement + 7 advisory)

**壁 (enforcement)** — 壊れたコードを通さない
- **PostToolUse**: 編集後に gate 実行 (lint/type/test)。失敗 → pending-fixes + gate outcome 記録
- **PreToolUse**: pending-fixes → DENY。Pace red → DENY (Plan 考慮)。commit gates → DENY

**Plan 増幅 (enforcement)** — 設計の質を底上げ
- **UserPromptSubmit**: Plan テンプレート動的注入 + 大タスク advisory (800+ / 400+)
- **PermissionRequest**: ExitPlanMode 時に Plan 検証 (大Plan: Success Criteria + Verify 具体性必須)

**実行ループ (enforcement + advisory)** — 中途半端に終わらせない
- **Stop**: 未修正エラー → block / 大Plan 未完了 → block / 小Plan 未完了 → warn / レビュー未実行 → block
- **PreCompact / PostCompact / SessionEnd**: pending-fixes reminder (stderr)
- **SessionStart**: 自動セットアップ + エラートレンド注入

**サブエージェント制御 (enforcement + advisory)** — 品質ルールを伝搬
- **SubagentStart**: サブエージェントに品質ルール注入
- **SubagentStop**: reviewer PASS/FAIL + Score 検証 + レビュー完了記録

**自己防衛 (enforcement + advisory)** — harness 自体を守る
- **PostToolUseFailure**: 2回連続失敗 → /clear 提案
- **ConfigChange**: hook 設定変更 → DENY (非 hook 設定は許可)

## 設計原則

1. **壁 > 情報提示** — DENY (exit 2) で止める。additionalContext は無視される前提
2. **リサーチ駆動** — 全設計判断に SWE-bench / Anthropic 記事 / Self-Refine 論文の裏付け
3. **fail-open** — 全 hook は try-catch。alfred の障害で Claude を止めない
4. **タスクスコープ適応** — 計画なし: 1-2ファイル集中。計画あり: 計画の境界に従う
5. **simplest solution** — 全コンポーネントは load-bearing 仮定を持つ。仮定が崩れたら捨てる
6. **効果測定** — DENY resolution rate + gate pass rate + advisory skip 率で実効性を計測
7. **dependencies ゼロ** — 全て devDependencies + bun build バンドル

## インストール

```bash
bun install
bun build.ts
bun link

alfred init       # ~/.claude/ に 13 hooks + skill + agent + rules を配置
alfred doctor     # セットアップの健全性を確認 (8項目 + state整合性検証)
alfred doctor --fix  # 壊れた state ファイルを自動修復
```

## コマンド

```bash
alfred init              # セットアップ (13 hooks + skill + agent + rules + gates 自動検出)
alfred hook <event>      # Hook イベント処理 (Claude Code が呼び出す)
alfred doctor            # ヘルスチェック (bun, hooks, skill, agent, rules, gates, state, path)
alfred doctor --metrics  # DENY/block/respond + gate pass rate + advisory skip 統計を表示
alfred doctor --fix      # 壊れた state ファイルを自動修復 (既知ファイルのみ)
alfred reset             # .alfred/.state/ を初期化 (--keep-history で履歴保持)
```

`alfred init` が配置するもの:
- `~/.claude/settings.json` — 13 hooks を登録 (既存の hook は保持)
- `~/.claude/skills/alfred-review/SKILL.md` — /alfred:review skill
- `~/.claude/agents/alfred-reviewer.md` — 独立レビュー agent (PASS/FAIL + Score threshold)
- `~/.claude/rules/alfred-quality.md` — 品質ルール (適応型タスクスコープ)
- `.alfred/gates.json` — プロジェクトの lint/type/test gate を自動検出

hooks は Claude Code が各イベントで自動呼び出し。手動操作は不要。

## Skills

- `/alfred:review` — 独立コードレビュー。HubSpot 2-stage pattern: Reviewer (独立サブエージェント) が全 findings を報告 (Review: PASS/FAIL + Score: Correctness/Design/Security 各1-5) → Judge が S/A/A でフィルタ → critical/high は修正後に再レビュー (最大2サイクル)

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
