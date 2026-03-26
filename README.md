# alfred

[Claude Code](https://docs.anthropic.com/en/docs/claude-code) の品質バトラー。Claude Code の動作を監視し、品質ゲートを強制し、過去のセッションから学習する。

**不可視。機械的。容赦なし。**

## alfred が行うこと

alfred は Claude Code 内部で hooks + MCP サーバーとして動作する。すべてのファイル編集、bash コマンド、コミットを監視し、提案ではなく「壁」で品質を強制する。

- **Lint/型ゲート**: PostToolUse がファイル書き込み後に lint と型チェックを実行。エラーは DIRECTIVE として注入され、Claude は修正するまで先に進めない
- **編集ブロック**: PreToolUse が未修正の lint/型エラーがある状態での編集をブロック (DENY)
- **テスト先行強制**: UserPromptSubmit が実装系プロンプトを検出し、テスト先行 DIRECTIVE を注入
- **エラー解決キャッシュ**: Bash エラーを Voyage AI ベクトル検索で過去の解決策とマッチし、コンテキストとして注入
- **Exemplar 注入**: 実装プロンプトに対して関連する before/after コード例を注入 (Few-shot、研究に基づく)
- **Convention 強制**: プロジェクト固有のコーディング規約をセッション開始時に注入
- **品質スコアリング**: すべてのゲート pass/fail、エラー hit/miss を追跡しスコアリング (0-100)
- **セルフリフレクション**: コミットゲートで4項目の検証チェックリストを注入 (エッジケース、サイレント障害、シンプルさ、規約)

## アーキテクチャ

```
User → Claude Code → (alfred hooks: 監視 + コンテキスト注入 + ゲート)
              ↓ 必要な時だけ
           alfred MCP (知識DB)
```

| コンポーネント | 役割 | 比重 |
|---|---|---|
| Hooks (6 events) | 監視、コンテキスト注入、品質ゲート | 70% |
| DB + Voyage AI | 知識蓄積、ベクトル検索 | 20% |
| MCP tool | Claude Code → 知識DBインターフェース | 10% |

## インストール

```bash
# ビルド
bun install
bun build.ts
bun link          # 'alfred' コマンドをグローバルに利用可能にする

# セットアップ (~/.claude/ に設定を配置)
alfred init
```

必須: `VOYAGE_API_KEY` 環境変数 (https://dash.voyageai.com/ で取得)

## コマンド

```bash
alfred init          # セットアップ: MCP, hooks, rules, skills, agents, gates
alfred serve         # MCP サーバー起動 (stdio, Claude Code が呼び出す)
alfred hook <event>  # Hook イベント処理 (Claude Code が呼び出す)
alfred status        # プロジェクトの品質状態を表示
alfred tui           # ターミナル品質ダッシュボード
alfred scan          # フル品質スキャン (lint/型/テスト + スコア)
alfred doctor        # インストール健全性チェック
alfred uninstall     # alfred をシステムから削除
alfred version       # バージョン表示
```

## Skills

- `/alfred:review` — Judge フィルタリング付きマルチエージェントコードレビュー (HubSpot 3基準パターン)
- `/alfred:conventions` — コードベースのコーディング規約を検出し、採用率を表示

## TUI ダッシュボード

```bash
task tui   # or: bun src/tui/main.tsx
```

表示内容: 品質スコア、ゲート pass/fail、知識ヒット、直近イベントストリーム、セッション情報。`?` でヘルプ (Tab で EN/JA 切替)。

## スタック

TypeScript (Bun 1.3+, ESM) / SQLite (bun:sqlite) / Voyage AI (voyage-4-large + rerank-2.5) / MCP SDK / TUI (OpenTUI)

## 設計ドキュメント

`design/` にアーキテクチャ、詳細設計、リサーチ参考文献を配置。
