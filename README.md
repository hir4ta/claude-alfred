# alfred

Claude Code の完全受動型執事。

主人が呼ぶまで沈黙。呼ばれたら、最高の知識を即座に渡す。
Claude Code の設定・ベストプラクティスを知識ベースから検索し、最適な回答を提供する。

## Alfred ができること

**呼ばれたら** — 知識ベースから最適解を返す。
Claude Code の設定、プロジェクトのレビュー、スキル作成、CLAUDE.md の改善。

**Butler Protocol** — Compact/セッション喪失に強い開発計画。
`.alfred/specs/` に要件・設計・タスク・決定・知見・セッション状態を構造化保存。
PreCompact hook が会話コンテキストを自動保存し、SessionStart で自動復帰する。

**SessionStart** — セッション開始時に CLAUDE.md を自動取り込み。

## 初回セットアップ

### 1. プラグインを追加

Claude Code 内で:

```
/plugin marketplace add hir4ta/claude-alfred
/plugin install alfred@hir4ta/claude-alfred
```

プラグイン（skills, rules, hooks, agents, MCP 設定）が配置される。

### 2. バイナリをインストール

ターミナルで:

```bash
go install github.com/hir4ta/claude-alfred/cmd/alfred@latest
```

MCP サーバーと Hook handler のバイナリをコンパイルする。
初回は依存ライブラリのビルドに 30〜60 秒かかる。

### 3. API キーを設定

```bash
export VOYAGE_API_KEY=your-key  # ~/.zshrc 等に追加
```

セマンティック検索に [Voyage AI](https://voyageai.com/) を使用する。

### 4. 知識ベースを初期化

```bash
alfred setup
```

公式ドキュメント（1,400+ 件）を SQLite に取り込み、Voyage AI で embedding を生成する。
TUI で進捗を表示する。

Claude Code を再起動すれば完了。

### ソースからビルド

```bash
git clone https://github.com/hir4ta/claude-alfred
cd claude-alfred
go build -o alfred ./cmd/alfred
```

## アップデート

### 1. プラグインを更新

Claude Code 内で:

```
/plugin install alfred@hir4ta/claude-alfred
```

### 2. バイナリを更新

Claude Code を終了し、ターミナルで:

```bash
alfred update
```

最新バージョンを確認し、自動で `go install` を実行する。

### 3. Claude Code を再起動

更新完了。

## スキル (5)

Claude Code 内で `/alfred:<スキル名>` で呼び出す。

| スキル | 内容 |
|--------|------|
| `/alfred:configure <種類> [名前]` | 単一の設定ファイルを作成・更新（skill, rule, hook, agent, MCP, CLAUDE.md, memory）+ 独立レビュー |
| `/alfred:setup` | プロジェクト全体のセットアップウィザード — 複数ファイルのスキャン+設定、または Claude Code 機能の解説 |
| `/alfred:brainstorm <テーマ>` | 発散（ブレスト）— 観点・選択肢・仮説・質問を増やす |
| `/alfred:refine <テーマ>` | 壁打ち（収束）— 論点を固定し、選択肢を絞り、決定を出す |
| `/alfred:plan <task-slug>` | Butler Protocol — 対話的に spec を生成し、Compact/セッション喪失に強い開発計画を作成 |

## エージェント (1)

| エージェント | 内容 |
|------------|------|
| `alfred` | Claude Code の設定・ベストプラクティスに関するサポート |

## MCP ツール (7)

スキルとエージェントのバックエンド。
Claude が必要に応じて自動的に呼び出すため、直接呼ぶ必要はない。

| ツール | 内容 |
|--------|------|
| `knowledge` | ハイブリッド vector + FTS5 + Voyage rerank によるドキュメント検索 |
| `review` | プロジェクトの .claude/ 設定を分析（ファイル内容読み込み + KB 照合） |
| `suggest` | git diff を分析して .claude/ 設定の更新を提案 |
| `butler-init` | 新しい開発タスクの spec を初期化（.alfred/specs/ に 6 ファイル生成 + DB 同期） |
| `butler-update` | アクティブ spec のファイルを更新（決定・知見・タスク進捗・セッション状態の記録） |
| `butler-status` | 現在のタスク状態を取得（セッション復帰用） |
| `butler-review` | 3層コードレビュー（spec + 蓄積知見 + ベストプラクティス） |

## コマンド

| コマンド | 内容 |
|----------|------|
| `serve` | MCP サーバー起動（stdio） |
| `setup` | 知識ベース初期化（TUI 進捗表示、seed + embedding 生成） |
| `hook <Event>` | Hook handler（Claude Code から呼ばれる） |
| `update` | 最新バージョンに更新（TUI 進捗表示） |
| `version` | バージョン表示 |

## 仕組み

```
┌─────────────────────────────────────────────┐
│           Claude Code セッション             │
│                                             │
│  Hook ──→ alfred.db                          │
│  SessionStart  (CLAUDE.md 自動取り込み)        │
│  PreCompact    (spec session.md 自動保存)     │
│                                             │
│  あなた: 「認証機能を追加して」               │
│          ↓                                   │
│  butler-init → .alfred/specs/add-auth/       │
│  (6 ファイル生成 + DB 同期)                   │
│          ↓                                   │
│  Compact 発生 → PreCompact が自動保存         │
│          ↓                                   │
│  SessionStart(compact) → 全 spec 自動復帰     │
└─────────────────────────────────────────────┘
```

**Hook** — SessionStart（CLAUDE.md 取り込み + butler-protocol コンテキスト注入）、PreCompact（session.md 自動保存）、PreToolUse / UserPromptSubmit（.claude/ 設定リマインダー）。

**Butler Protocol** — `.alfred/specs/{task-slug}/` に 6 ファイル（requirements, design, tasks, decisions, knowledge, session）を構造化保存。Compact 前にコンテキストスナップショットを session.md に保存し、Compact 後に全 spec ファイルを自動注入して完全復帰する。

**独立レビュー** — `/alfred:configure` は、ファイル生成後に別コンテキストで Explore エージェントを起動する。読み取り専用かつ知識ベース検索が可能で、公式仕様に対する客観的な検証を行う。

## デバッグ

`ALFRED_DEBUG=1` を設定すると `~/.claude-alfred/debug.log` にデバッグログを出力する。

## 依存ライブラリ

| ライブラリ | 用途 |
|-----------|------|
| [mcp-go](https://github.com/mark3labs/mcp-go) | MCP サーバー SDK |
| [go-sqlite3](https://github.com/ncruces/go-sqlite3) | SQLite ドライバ（pure Go, WASM） |
| [bubbletea](https://github.com/charmbracelet/bubbletea) | TUI フレームワーク（setup 画面） |
| [Voyage AI](https://voyageai.com/) | embedding + rerank（voyage-4-large, 2048d） |

## ライセンス

MIT
