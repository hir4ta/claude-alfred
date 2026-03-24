# Research: Hook LLM Migration

## 1. Claude Code Hook Handler Types

公式ドキュメントから判明した全4ハンドラタイプ:

| Type | Claude利用 | ツールアクセス | デフォルトTimeout | 用途 |
|------|-----------|-------------|-----------------|------|
| `command` | なし | なし（シェル実行） | 600s | DB操作、ファイルI/O、状態管理 |
| `prompt` | **単一ターン** | なし | **30s** | LLM判定（allow/deny + additionalContext） |
| `agent` | **マルチターン** | Read/Grep/Glob/Bash等 | **60s** | 複雑な検証（ファイル読み、テスト実行） |
| `http` | 外部依存 | なし | 30s | リモートサーバー |

### 重要: prompt/agent は ANTHROPIC_API_KEY 不要

Claude Code 自身のセッション内で動作。追加認証なし。

### $ARGUMENTS の内容（イベント別）

- **UserPromptSubmit**: `{session_id, cwd, prompt, permission_mode, ...}`
- **PostToolUse**: `{session_id, cwd, tool_name, tool_input, tool_response, ...}`
- **PreToolUse**: `{session_id, cwd, tool_name, tool_input, tool_use_id, ...}`
- **PreCompact**: `{session_id, cwd, transcript_path, trigger, ...}`
- **Stop**: `{session_id, cwd, stop_hook_active, last_assistant_message, ...}`
- **SessionStart**: `{session_id, cwd, source, model, ...}`

### 応答フォーマット（イベント別）

| Event | additionalContext | permissionDecision | decision:block |
|-------|:-:|:-:|:-:|
| SessionStart | ✅ (hookSpecificOutput) | — | — |
| UserPromptSubmit | ✅ (top-level) | — | ✅ |
| PreToolUse | ✅ (hookSpecificOutput) | ✅ allow/deny/ask | — |
| PostToolUse | ✅ (hookSpecificOutput) | — | ✅ |
| PreCompact | — | — | — |
| Stop | — | — | ✅ |

<!-- confidence: 9 | source: design-doc | grounding: verified -->

### 同一イベントに複数ハンドラ登録可能

```json
{
  "hooks": {
    "UserPromptSubmit": [
      { "hooks": [{ "type": "command", "command": "alfred hook UserPromptSubmit" }] },
      { "hooks": [{ "type": "prompt", "prompt": "...", "model": "claude-haiku-4-5-20251001" }] }
    ]
  }
}
```

**並列実行**: 同一イベントの複数 hook は並列実行される。
<!-- confidence: 8 | source: design-doc | grounding: reviewed -->

## 2. 現行アーキテクチャの制約分析

### 全6 hook の操作分類

| Hook | DB操作 | ファイルI/O | 状態管理 | LLM化候補の判定 |
|------|--------|-----------|---------|--------------|
| **SessionStart** | upsert knowledge, register project | .alfred/knowledge/ walk | — | ❌ DB必須 |
| **UserPromptSubmit** | knowledge search, hit_count | nudge/spec-prompt state | ✅ intent分類 | ⚠️ 部分的 |
| **PreToolUse** | — | review-gate.json read | — | ⚠️ 単純すぎてLLM不要 |
| **PostToolUse** | knowledge FTS, promotion | tasks.json write, design.md | wave-progress, worked-slugs | ⚠️ 部分的 |
| **PreCompact** | upsert decisions/snapshots | pending-compact.json | — | ⚠️ 部分的 |
| **Stop** | — | worked-slugs read | — | ❌ 単純すぎ |

<!-- confidence: 9 | source: code | grounding: verified -->

### 根本的制約: prompt/agent hook は DB 書き込み不可

- `prompt` hook: LLM判定のみ（ツールアクセスなし）
- `agent` hook: Read/Grep/Glob/Bash は使えるが、**alfred の SQLite DB への直接書き込みは不可能**
- Knowledge sync、hit_count tracking、decision upsert 等は command hook でしか実行できない

## 3. ハイブリッドアーキテクチャ（提案）

### 設計原則

**「command が基盤、prompt/agent が頭脳」**

同一イベントに2つの hook を並列登録:
1. **command hook** (slimmed): DB書き込み、ファイルI/O、状態管理のみ
2. **prompt/agent hook** (new): LLM判定、インテント分類、タスク評価

両者は独立並列実行。prompt hook の `additionalContext` が Claude 本体のコンテキストに注入される。

### Hook 別移行方針

#### UserPromptSubmit — prompt hook 追加（最大効果）

**現状の問題**: キーワードマッチによるインテント分類が精度不足
**移行方針**:
- command hook: knowledge search + 状態管理（spec-prompt guard, nudge impressions）のみに縮小
- **prompt hook (NEW)**: Haiku がインテント分類 + skill 推薦を生成
- prompt の additionalContext → `[CONTEXT] Skill suggestion: /alfred:attend — intent=implement`

```
UserPromptSubmit prompt hook のプロンプト例:
"以下のユーザー入力のインテントを分類し、適切なスキルを推薦してください。
インテント: research/plan/implement/bugfix/review/tdd/save-knowledge
スキル: brief(計画)/attend(実装)/inspect(レビュー)/tdd(TDD)/mend(バグ修正)
ユーザー入力: $ARGUMENTS"
```

#### PostToolUse — prompt hook 追加（タスク判定改善）

**現状の問題**: ファイルパスの文字列マッチで auto-check が脆い
**移行方針**:
- command hook: git commit 検出、Living Spec、wave completion、DB 操作のみに縮小
- **auto-check を command から削除** → prompt hook に移行
- **prompt hook (NEW)**: ファイル変更 + タスク一覧を見て、完了候補を提案
- additionalContext → `[CONTEXT] Task completion candidates: T-1.3 (src/hooks/intent.ts matches task description)`
- **Claude 本体が最終判断** → `dossier check` を呼ぶかどうかは Claude が決める

```
PostToolUse prompt hook のプロンプト例:
"以下のツール実行結果がどのタスクを完了させたか評価してください。
確信度が高い場合のみ候補として挙げてください。
ツール: $ARGUMENTS
タスク一覧: [tasks.json content from command hook output]"
```

**問題点**: prompt hook は tasks.json を直接読めない。$ARGUMENTS には tool_input/tool_response しか含まれない。
**解決策**: command hook が tasks.json の未完了タスク一覧を一時ファイルに書き出す → prompt hook のプロンプトに埋め込む... は不可能（並列実行のため）。
**代替案**: prompt hook を `agent` タイプにし、自力で tasks.json を Read する。

#### PreCompact — agent hook 追加（意思決定抽出改善）

**現状の問題**: キーワードスコアリング（base 0.35 + bonuses）で意思決定を抽出、false positive/negative あり
**移行方針**:
- command hook: DB 書き込み（snapshot、session breadcrumb）のみに縮小
- **agent hook (NEW)**: transcript を Read → LLM が意思決定を識別・構造化
- agent の additionalContext → `[DIRECTIVE] 以下の意思決定を ledger save してください: [structured decisions]`
- **Claude 本体が ledger save を呼ぶ**（DB 書き込みは Claude 経由）

**注意**: PreCompact は additionalContext を返せない（上記テーブル参照）。
→ agent hook の出力が直接 Claude に注入されない。
→ **意思決定抽出の改善は PreCompact では実現困難。別のアプローチが必要。**

#### PreToolUse — command hook 維持

**理由**: レビューゲート強制は binary allow/deny。review-gate.json の読み取り + パス判定のみ。
LLM を挟むと遅延が増え、確定的な強制が揺らぐリスク。
**判断**: 移行不要。現行 command hook が最適。

#### SessionStart — command hook 維持

**理由**: Knowledge sync（FS walk → DB upsert）、プロジェクト登録、spec コンテキスト注入。
全て DB/FS 操作。LLM 判定の余地なし。
**判断**: 移行不要。

#### Stop — command hook 維持

**理由**: review-gate 確認 + 未完了タスクカウント。65行の単純ロジック。
**判断**: 移行不要。

<!-- confidence: 7 | source: inference | grounding: inferred -->

## 4. 技術的リスクと未知数

### R-1: prompt hook と command hook の出力マージ

同一イベントで両方が additionalContext を返した場合、両方が Claude に注入されるのか？
一方が上書きするのか？ドキュメントに明記なし。

**検証方法**: 実際に2つの hook を登録して動作確認が必要。
<!-- confidence: 5 | source: assumption | grounding: speculative -->

### R-2: prompt hook の $ARGUMENTS にタスク情報が含まれない

PostToolUse の $ARGUMENTS は `{tool_name, tool_input, tool_response}` のみ。
tasks.json の内容は含まれない。prompt hook 単独ではタスク判定不可能。

**解決策**:
- (a) `agent` タイプにして tasks.json を自力 Read
- (b) command hook → 一時ファイル → prompt hook が参照（並列実行なので不可）
- (c) prompt hook のプロンプトにタスク情報を静的埋め込み（更新されない）

→ **(a) が唯一の現実的解決策**

### R-3: PreCompact で additionalContext が使えない

公式ドキュメントで PreCompact は additionalContext 非対応。
agent hook で意思決定を抽出しても、結果を Claude に渡す手段がない。

**解決策**:
- (a) 抽出結果を `.alfred/.state/extracted-decisions.json` に書き出し、SessionStart で読み込む
- (b) PreCompact の改善は諦め、意思決定抽出を UserPromptSubmit や PostToolUse に移す
- (c) agent hook 内で Bash ツール経由で alfred CLI を呼び DB 書き込み

→ **(c) が最も直接的**: `agent` hook 内で `bash alfred hook-internal save-decisions '{json}'` のようなコマンドを実行

### R-4: prompt hook のレイテンシ

Haiku でも 1-3s のレイテンシ。command hook と並列実行されるため全体の遅延にはならないが、
prompt hook の結果が遅れて Claude に到達する可能性。

**影響**: 低（additionalContext は次のターンで反映されるため、数秒の遅延は許容範囲）

### R-5: コスト増

prompt/agent hook は Claude API のトークンを消費する。
Haiku は安価だが、PostToolUse は高頻度（ファイル保存ごと）で呼ばれるため累積コストに注意。

**緩和策**: matcher でフィルタ（Edit|Write|Bash のみ）、短いプロンプト設計

## 5. 調査結論

### 移行対象（3 hook + 新規1）

| Hook | 現行 | 移行先 | 追加 hook type | 期待効果 |
|------|------|--------|--------------|---------|
| **UserPromptSubmit** | command | command + **prompt** | prompt (Haiku) | インテント分類精度向上 |
| **PostToolUse** | command | command + **agent** | agent (Haiku) | タスク完了判定の精度向上 |
| **PreCompact** | command | command + **agent** | agent (Haiku) | 意思決定抽出の精度向上 |

### 維持対象（3 hook）

| Hook | 理由 |
|------|------|
| **SessionStart** | 全て DB/FS 操作。LLM 不要 |
| **PreToolUse** | binary allow/deny。確定的強制が必要 |
| **Stop** | 65行の単純ロジック。LLM 不要 |

### 「全面移行」の再定義

当初の「6つ全てを prompt/agent に移行」は技術的に不可能:
- DB 書き込みは command hook でしか実行できない
- PreCompact は additionalContext 非対応
- PreToolUse/Stop は確定的強制が必要

**現実的な「全面移行」= ハイブリッドアーキテクチャ**:
- LLM が価値を出せる3つの hook に prompt/agent を追加
- command hook は DB/FS の基盤として残す
- 既存の command hook を大幅スリム化（LLM 判定ロジックを削除）
