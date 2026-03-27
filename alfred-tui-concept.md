# alfred-tui: TUI Dashboard Concept

alfred の state データをリアルタイムで可視化する別パッケージ。

## モチベーション

- alfred の「効果の実証」が現状の最大の弱点 (評価 3/5)
- `doctor --metrics` は数値の羅列。トレンドやパターンが見えない
- リッチな可視化で alfred の価値を目に見える形にする

## 技術スタック

- OpenTUI (@opentui/core) — Zig ネイティブコアの TUI ライブラリ
- Bun — ランタイム
- 別パッケージ (alfred 本体の zero-dependency を維持)

## プロジェクト横断ダッシュボード

`alfred init` 実行時に `~/.alfred/registry.json` へプロジェクトパスを自動登録（v0.9.8+ で実装済み）。

```json
// ~/.alfred/registry.json
[
  { "path": "/Users/user/Projects/claude-alfred", "registered_at": "2026-03-27T..." },
  { "path": "/Users/user/Projects/webapp", "registered_at": "2026-03-27T..." }
]
```

### UX

```bash
# どのパスからでも実行可能
alfred-dash

# デフォルト: 全プロジェクトの集約ビュー
# Tab: プロジェクト単位で切り替え
# 各プロジェクトの .alfred/.state/ を読み取り
```

### 画面構成

```
┌─────────────────────────────────────────────────────┐
│ [All] [claude-alfred] [webapp] [api-server]    Tab  │
├─────────────────┬───────────────────────────────────┤
│ Session Status  │ DENY/Block Activity               │
│  pace: 12min    │  ██████░░ resolution: 75%         │
│  files: 3       │  last: pending-fixes (2min ago)   │
│  test: passed   │                                   │
│  review: done   │ Gate Health                       │
├─────────────────┤  lint:  ████████░░ 85% pass       │
│ Pending Fixes   │  tsc:   ██████████ 100% pass      │
│  (none)         │                                   │
│                 │ First-pass Clean                  │
│                 │  ██████░░░░ 62%                   │
│                 ├───────────────────────────────────┤
│                 │ Review History                    │
│                 │  3 reviews, 2 PASS, 1 FAIL        │
│                 │  findings: 5 (1H, 2M, 2L)         │
├─────────────────┴───────────────────────────────────┤
│ Commit Pace: ·····•···•··•····•··•··•               │
└─────────────────────────────────────────────────────┘
```

## 可視化対象

| パネル | データソース | 表示内容 |
|--------|-------------|----------|
| Session Status | session-state.json | pace, pending fixes, test/review clearance |
| Pending Fixes | pending-fixes.json | 現在のエラーリスト (file, gate, error) |
| DENY/Block Activity | metrics.json | DENY resolution rate, 直近の発火イベント |
| Gate Health | gate-history.json | pass/fail 率, エラーパターン top N |
| First-pass Clean | metrics.json | 初回 gate 通過率のトレンド |
| Review History | metrics.json | findings 数, severity 分布, PASS/FAIL 率 |
| Commit Pace | gate-history.json | コミット間隔のトレンド |

## アーキテクチャ

```
alfred-dash (別リポ)
├── ~/.alfred/registry.json からプロジェクト一覧を取得
├── 各プロジェクトの .alfred/.state/ を polling (1s) で読み取り
├── read-only — state ファイルへの書き込みは一切しない
├── alfred 本体がなくても起動可能 (state ファイルがあれば動く)
└── bun install -g && alfred-dash で起動
```

## インストール

```bash
# alfred 本体とは別にインストール
bun install -g alfred-dash
# どのディレクトリからでも起動
alfred-dash
```

## 未決事項

- ~~プロジェクト横断ダッシュボードの是非~~ → registry.json で解決済み
- 別リポ vs claude-alfred monorepo 化
- OpenTUI の Bun バンドル互換性の検証
- リアルタイム polling vs ファイル監視 (fs.watch)
- 集約ビューの集計ロジック（プロジェクト間の metrics をどうマージするか）
