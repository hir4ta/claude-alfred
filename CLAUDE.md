# claude-alfred

Claude Code の暴走を止める執事。14 Hooks + Skill + Agent で品質の下限を守る。

## スタック

TypeScript (Bun 1.3+, ESM) / citty (CLI) / vitest (テスト) / Biome (lint)

## アーキテクチャ

```
alfred CLI (init / hook / doctor)
    ├── alfred init → ~/.claude/ に 14 hooks, skill, agent, rules を配置
    └── alfred hook <event> → stdin JSON → 処理 → stdout JSON or exit 2
```

3つの柱 + 2つの防御層:
1. **壁** — PostToolUse (gate) → PreToolUse (DENY)
2. **Plan増幅** — UserPromptSubmit (template) + PermissionRequest (ExitPlanMode) + TaskCompleted (status同期)
3. **実行ループ** — Stop (Plan未完了block + pending-fixes block) + PreCompact/PostCompact/SessionEnd (handoff)
4. **サブエージェント制御** — SubagentStart (品質ルール注入) + SubagentStop
5. **自己防御** — ConfigChange (hook削除防止) + PostToolUseFailure (失敗追跡)

## 構造

```
src/
├── cli.ts                  # citty: init / hook / doctor / reset
├── init.ts                 # セットアップ (14 hooks + skill + agent + rules + gates)
├── doctor.ts               # ヘルスチェック (8項目 + --metrics) + state整合性検証
├── reset.ts                # 状態リセット (--keep-history で履歴保持)
├── hooks/
│   ├── dispatcher.ts       # event → handler ルーティング (14 events)
│   ├── respond.ts          # 共通: respond / deny / block + metrics記録
│   ├── post-tool.ts        # lint/type gate + pending-fixes + pace + batch + test-pass + verify
│   ├── pre-tool.ts         # pending-fixes → DENY + pace red → DENY + commit without test → DENY
│   ├── user-prompt.ts      # Plan テンプレート注入 + 大タスク block (500+) / advisory (200+)
│   ├── permission-request.ts # ExitPlanMode: Review Gates + Success Criteria 検証
│   ├── task-completed.ts   # Plan task status 自動同期
│   ├── session-start.ts    # .alfred作成 + gates自動検出 + handoff復元
│   ├── stop.ts             # pending-fixes block + Plan未完了block + レビュー強制 + pace警告
│   ├── pre-compact.ts      # 構造化ハンドオフ保存
│   ├── post-compact.ts     # コンパクション後ハンドオフ復元
│   ├── session-end.ts      # handoff 保存 + セッション成果記録
│   ├── subagent-start.ts   # サブエージェントに品質ルール注入
│   ├── subagent-stop.ts    # サブエージェント出力検証 + レビュー完了記録
│   ├── post-tool-failure.ts # ツール失敗追跡 + 2回連続→/clear
│   └── config-change.ts    # user_settings 変更 DENY
├── gates/
│   ├── runner.ts           # gate コマンド実行
│   ├── load.ts             # gates.json 読み込み
│   └── detect.ts           # プロジェクト設定 → gates.json 自動検出 (TS/Python/Go/Rust)
├── state/
│   ├── pending-fixes.ts    # 未修正 lint/type エラー
│   ├── session-state.ts    # 統合セッション状態 (pace, test, review, batch, fail, budget)
│   ├── gate-history.ts     # gate 結果トレンド + コミット間隔統計
│   ├── handoff.ts          # 構造化ハンドオフ
│   ├── plan-status.ts      # Plan task status 解析
│   ├── metrics.ts          # DENY/block/respond 発火記録 (50件 cap)
│   └── session-outcomes.ts # セッション成果追跡 (50件 cap)
├── templates/              # init が配置するファイル
│   ├── skill-review.md     # /alfred:review skill
│   ├── agent-reviewer.md   # reviewer agent
│   └── rules-quality.md    # 品質ルール
└── types.ts
```

## コマンド

```bash
task build    # bun build (バンドル)
task check    # tsc --noEmit + Biome lint
task fix      # Biome 自動修正
task test     # vitest run
task clean    # ビルド成果物削除
```

`bun tsc` / `bun vitest` を使う（`npx` 不要）

## 設計原則

1. **リサーチ駆動** — 効果が実証された手法のみ実装 (research-harness-engineering-2026.md)
2. **壁 > 情報提示** — DENY (exit 2) > additionalContext
3. **少ない方が強い** — コンテキスト注入は最小限。指示は20行以内
4. **タスクサイズ制御** — 15 LOC以下・単一ファイル (SWE-bench: 80%+ 成功率)
5. **検証 > 指示** — 「何を検証すべきか」を伝える。HOW ではなく WHAT
6. **fail-open** — 全 hook は try-catch で握りつぶす。alfred の障害で Claude を止めない

## ルール

### ビルド
- `bun build.ts` → `dist/cli.mjs`
- `bun build.ts --compile` → シングルバイナリ
- **dependencies ゼロ** — 全て devDependencies + bun build バンドル

### Hook 設計
- 全 hook は fail-open (try-catch で握りつぶす)
- exit 2 = DENY/block (唯一の強制手段)。stderr にも理由を出力
- additionalContext = advisory (Claude は無視可能)
- PostToolUse 検出 → PreToolUse ブロックの二段構え
- SubagentStart でサブエージェントにも品質ルールを伝搬
- **Hook 出力スキーマ対応表** (https://code.claude.com/docs/en/hooks 準拠):
  - respond() (hookSpecificOutput.additionalContext): SessionStart, PostToolUse, PostToolUseFailure, SubagentStart
  - deny() (hookSpecificOutput.permissionDecision): PreToolUse, PermissionRequest
  - block() (トップレベル decision/reason): Stop, UserPromptSubmit, SubagentStop
  - 出力なし (stderr で advisory): PostCompact, TaskCompleted, PreCompact, SessionEnd, ConfigChange

### 状態ファイル (.alfred/.state/)
- pending-fixes.json — 未修正 lint/type エラー
- session-state.json — 統合セッション状態 (pace, test pass, review, gate batch, fail count, budget, action counters)
- gate-history.json — gate 結果トレンド + コミット間隔 (50件 cap)
- handoff.json — 構造化ハンドオフ (Plan context + gate errors 含む)
- metrics.json — DENY/block/respond 発火記録 (50件 cap, `doctor --metrics` で表示)
- session-outcomes.json — セッション成果追跡 (clean exit率, pending推移, 50件 cap)

### Sprint Contract (Anthropic記事準拠)
- Plan テンプレートに Success Criteria セクションを必須化
- ExitPlanMode 時に Success Criteria の具体性を検証 (コマンド or ファイル参照)
- reviewer は全 findings を報告。Judge (skill) のみが S/A/A フィルタを適用
- reviewer に few-shot 例 + anti-self-persuasion 指示を配置

### Phase Gate (各コミット前に必ず実行)
1. `bun vitest run` — 全テスト pass
2. `bun vitest run src/__tests__/simulation.test.ts` — シミュレーション pass
3. `bun tsc --noEmit && bun biome check src/` — 型 + lint clean
4. `/alfred:review` — 独立レビュー (自己評価は機能しない。必ずサブエージェントで実行)
5. コミット — Phase Gate 通過後にのみコミット

### シミュレーション
- Hook や状態管理の変更後は simulation.test.ts にシナリオを追加する
- シミュレーションは本番フロー (Edit→gate→pending-fixes→DENY) を再現する統合テスト

## 設計ドキュメント

- design-v0.1.md — v0.1.0 全体設計 (v0.2 の詳細設計含む)
- ROADMAP.md — v0.2〜v1.0 の全ロードマップ (詳細設計)
- research-harness-engineering-2026.md — リサーチ結果
- research-claude-code-plugins-2026.md — Plugin 調査結果
