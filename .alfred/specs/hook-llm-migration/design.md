# Design: Hook LLM Migration

## Architecture Overview

**ハイブリッドアーキテクチャ**: 同一イベントに command hook（基盤）と prompt/agent hook（頭脳）を並列登録。

```
┌─────────────────────────────────────────────────────┐
│ Claude Code Hook Event                               │
│                                                       │
│  ┌──────────────────┐  ┌──────────────────────────┐  │
│  │ command hook      │  │ prompt/agent hook         │  │
│  │ (alfred CLI)      │  │ (Claude Haiku)            │  │
│  │                   │  │                            │  │
│  │ • DB writes       │  │ • Intent classification   │  │
│  │ • State files     │  │ • Task evaluation         │  │
│  │ • Knowledge search│  │ • Decision extraction     │  │
│  │ • Git/FS ops      │  │                            │  │
│  └────────┬─────────┘  └────────────┬───────────────┘  │
│           │                          │                   │
│           ▼                          ▼                   │
│  additionalContext           additionalContext            │
│  (knowledge results,         (skill suggestion,           │
│   spec DIRECTIVE)             task candidates)            │
│           │                          │                   │
│           └──────────┬───────────────┘                   │
│                      ▼                                   │
│            Claude Main Session                           │
│            (両方の context を受け取る)                    │
└─────────────────────────────────────────────────────────┘
```

## Component Design

### C-1: UserPromptSubmit Prompt Hook

**目的**: LLM ベースのインテント分類 + スキル推薦
**Handler type**: `prompt` (単一ターン、ツール不要)
**Model**: `claude-haiku-4-5-20251001`
**Timeout**: 30s (default)

**プロンプト設計** (FR-1, FR-8):
```
あなたはClaude Code開発支援ツール alfred のインテント分類器です。
ユーザーのプロンプトを以下の7種類に分類してください。

## インテント → スキル対応表
| Intent | 説明 | 推薦スキル |
|--------|------|-----------|
| research | 調査・情報収集・学習 | /alfred:brief |
| plan | 設計・計画・アーキテクチャ | /alfred:attend |
| implement | 実装・機能追加・リファクタ | /alfred:attend |
| bugfix | バグ修正・エラー対応 | /alfred:mend |
| review | コードレビュー・品質確認 | /alfred:inspect |
| tdd | テスト駆動開発・テスト作成 | /alfred:tdd |
| save-knowledge | ナレッジ保存・メモ | (なし) |

## ルール
- 複合的なプロンプトは主要インテントを1つ選択
- 雑談・質問・不明な場合は「none」
- save-knowledge と research が両方該当する場合は save-knowledge を優先

## 出力形式
additionalContext として以下を出力:
[CONTEXT] Skill suggestion: /alfred:{skill} — {skill名} ({intent}の理由を10文字以内で)

ユーザー入力:
$ARGUMENTS
```

**応答フォーマット**:
```json
{
  "additionalContext": "[CONTEXT] Skill suggestion: /alfred:attend — 自律実装 (implement: 機能追加の意図)"
}
```

<!-- confidence: 8 | source: design-doc | grounding: reviewed -->

### C-2: UserPromptSubmit Command Hook (Slimmed)

**変更内容** (FR-2):

**削除する関数**:
- `classifyIntent()` — キーワードマッチ分類
- `classifyIntentSemantic()` — Voyage ベクトル分類
- `buildSkillNudge()` — スキル推薦メッセージ構築
- `intentDescription()` — インテント説明文
- `getNudgeImpressions()` / `recordNudge()` / `resetNudgeCount()` — 推薦追跡
- `buildRelevanceExplanation()` — 関連性説明
- `recordKnowledgeGap()` — Knowledge gap 記録（JSONL）。NOTE: `GET /api/knowledge/gaps` API は空配列を返すように修正（エンドポイント自体は残す）

**残す機能**:
- `searchPipeline()` — Knowledge 検索 (Voyage → FTS5 → keyword)
- `checkSpecRequired()` — Spec 提案ガード（**簡略化**: 後述）
- `emitDirectives()` — ディレクティブ出力

**Spec 提案ガードの簡略化**:
現行の `checkSpecRequired()` はインテント分類結果に依存。インテント分類が prompt hook に移行するため、command hook 内では**簡易キーワードチェック**に置換:

```typescript
// 簡易実装チェック: implement/bugfix/tdd インテントの可能性を判定
const IMPL_KEYWORDS = [
  "implement", "add", "create", "build", "refactor", "fix", "bug", "test", "tdd",
  "実装", "追加", "作成", "修正", "バグ", "テスト", "リファクタ"
];
function looksLikeImplementation(prompt: string): boolean {
  const lower = prompt.toLowerCase();
  return IMPL_KEYWORDS.some(kw => lower.includes(kw));
}
```

この簡易チェックは「spec 提案すべきか」と「並行開発ガード」の判定のみに使用。精度の高いインテント分類は prompt hook が担当。

**Stage 1.5 並行開発ガードの維持**:
`checkSpecRequired()` の2段階ロジックを維持:
- **Stage 1**: `!activeSpec && looksLikeImplementation(prompt)` → DIRECTIVE (spec 提案)
- **Stage 1.5**: `activeSpec && looksLikeImplementation(prompt) && !workedSlugs.includes(slug)` → WARNING (並行開発ガード)

これにより既存の安全機構を失わない。

**Interface** (`src/hooks/user-prompt.ts`):
```typescript
export async function userPromptSubmit(
  ev: HookEvent,
  signal: AbortSignal,
): Promise<void> {
  // 1. Knowledge search (existing pipeline)
  const results = await searchPipeline(ev.prompt, ev.cwd, signal);

  // 2. Spec proposal guard (simplified check)
  const specDirective = await checkSpecRequired(ev.cwd, ev.prompt);

  // 3. Emit directives (knowledge results + spec proposal)
  const items: DirectiveItem[] = [];
  if (specDirective) items.push(specDirective);
  if (results.length > 0) items.push(buildKnowledgeContext(results));
  emitDirectives("UserPromptSubmit", items);
}
```

<!-- confidence: 8 | source: code | grounding: reviewed -->

### C-3: PostToolUse Agent Hook

**目的**: LLM ベースのタスク完了候補提案
**Handler type**: `agent` (tasks.json を Read する必要あり)
**Model**: `claude-haiku-4-5-20251001`
**Timeout**: 60s (default)
**Matcher**: `Edit|Write|Bash`

**コスト制御** (FR-8):
- プロンプト冒頭でアクティブ spec の有無をチェック指示。`_active.json` が見つからない場合は即座に終了（空レスポンス）
- これにより spec なしの通常開発セッションでは agent が早期リターンし、コスト発生なし

**CLI PATH 解決**:
- agent hook の Bash 環境では `alfred` が PATH に存在しない可能性がある
- プロンプト内で `$ARGUMENTS` の `cwd` フィールドを活用し、必要に応じて絶対パスを使用する指示を含める

**プロンプト設計** (FR-3, FR-8):
```
あなたはタスク完了判定エージェントです。

## 前提条件チェック
まず .alfred/specs/ 配下に _active.json が存在するか確認してください。
存在しない場合は何もせず終了してください。

## 手順
1. _active.json からスラッグを特定
2. そのスラッグの tasks.json を Read
3. 未完了タスク（checked: false）のリストを確認
4. 以下のツール実行結果と照合し、完了した可能性のあるタスクを判定

## ツール実行結果
$ARGUMENTS

## 判定基準
- タスクの title/files/subtasks と変更内容が意味的に一致するか
- 部分的な一致は候補にしない（確信度が高い場合のみ）
- false positive は絶対に避ける: 疑わしい場合は候補にしない

## 出力形式
additionalContext として以下を出力:
[CONTEXT] 以下のタスクが完了した可能性があります。確認して dossier check を呼んでください:
- T-X.Y: タスク名 (理由: 変更ファイルがタスクの対象ファイルと一致)
```

**応答フォーマット**:
```json
{
  "hookSpecificOutput": {
    "hookEventName": "PostToolUse",
    "additionalContext": "[CONTEXT] 以下のタスクが完了した可能性があります..."
  }
}
```

**候補なしの場合**: 空の additionalContext または応答なし（Claude に不要な情報を渡さない）

<!-- confidence: 7 | source: inference | grounding: inferred -->

### C-4: PostToolUse Command Hook (Slimmed)

**変更内容** (FR-4):

**削除する関数**:
- `autoCheckTasks()` — タスク自動チェック
- `matchTaskDescription()` — backtick パスマッチ
- `matchTaskFiles()` — files 配列マッチ

**削除する呼び出し箇所**:
- Edit/Write ハンドラ内の `autoCheckTasks()` 呼び出し
- Bash success ハンドラ内の `autoCheckTasks()` 呼び出し

**残す機能**:
- git commit 検出（regex）
- `handleLivingSpec()` — design.md 自動追記
- `detectDrift()` — spec/実装乖離検出
- `detectWaveCompletion()` — Wave 完了検出（tasks.json を読むが、チェックは変更しない）
- `extractReviewKnowledge()` — Agent レスポンスからの知見抽出
- `saveKnowledgeOnCommit()` — commit 時の昇格チェック
- `checkSpecCompletion()` — Spec 完了候補検出
- Edit/Write → worked-slug 追跡、pending → in-progress 自動遷移

**Wave completion への影響**:
`detectWaveCompletion()` は tasks.json の `task.checked` を読んで Wave 完了を判定。
auto-check が削除されるため、タスクのチェックは Claude 本体が `dossier check` を呼ぶタイミングに依存。
→ Wave completion 検出タイミングが遅延する可能性があるが、正確性は向上。

<!-- confidence: 8 | source: code | grounding: reviewed -->

### C-5: PreCompact Agent Hook

**目的**: LLM ベースの意思決定抽出
**Handler type**: `agent` (transcript を Read + Bash で DB 保存)
**Model**: `claude-haiku-4-5-20251001`
**Timeout**: 60s (default)

**プロンプト設計** (FR-5, FR-8):
```
あなたは意思決定抽出エージェントです。

## 手順
1. 以下の transcript パスからファイルを Read
2. 技術的な意思決定を識別（例: 技術選定、設計方針、トレードオフの判断）
3. 各意思決定を構造化
4. Bash で保存コマンドを実行

## Transcript 情報
$ARGUMENTS

## 意思決定の識別基準
- 明示的な選択: 「AではなくBを採用」「Xに決定」
- 暗黙的な選択: 設計上の判断、トレードオフの解決
- 除外: 単なる実装手順、自明な選択、ユーザーの直接指示の繰り返し

## 保存コマンド
各意思決定を以下のコマンドで保存（cwdは$ARGUMENTSのcwdフィールドを使用）:
cd {cwd} && alfred hook-internal save-decision --title "タイトル" --decision "決定内容" --reasoning "理由" --alternatives "却下案"
NOTE: alfredコマンドがPATHにない場合は、node_modules/.bin/alfred やnpx alfredを試してください。

## 出力
保存した意思決定の数を報告。意思決定がない場合は「意思決定なし」と報告。
```

<!-- confidence: 6 | source: inference | grounding: inferred -->

### C-6: PreCompact Command Hook (Slimmed)

**変更内容** (FR-6):

**削除する関数/定数**:
- `extractDecisions()` — キーワードスコアリング
- `DECISION_KEYWORDS`, `RATIONALE_SIGNALS`, `ALTERNATIVE_SIGNALS`, `ARCH_TERMS`
- `Decision` interface
- `preCompact()` 内の意思決定抽出ループ

**残す機能**:
- Chapter memory snapshot（tasks.json → snapshot knowledge）
- Auto-complete task（全タスク checked → complete）
- Session breadcrumb（`.pending-compact.json`）
- Orphan embedding cleanup

**設計方針** (FR-6, DEC-5):
command hook の意思決定抽出を完全削除。agent hook のみが意思決定抽出を担当。
並列実行でのフォールバックや重複防止の複雑さを排除し、シンプルな責務分離を実現。
agent hook がタイムアウトした場合、そのセッションでは意思決定が抽出されないが、これは許容範囲（手動で `ledger save` 可能）。

<!-- confidence: 6 | source: inference | grounding: inferred -->

### C-7: Internal CLI Command

**目的**: Agent hook から DB 書き込みを可能にするブリッジ
**コマンド**: `alfred hook-internal save-decision`

**Interface** (`src/cli.ts` に追加):
```typescript
// alfred hook-internal save-decision
//   --title "タイトル"
//   --decision "決定内容"
//   --reasoning "理由"
//   --alternatives "却下案" (optional)
//   --cwd "/project/path" (optional, defaults to process.cwd())
```

**処理フロー**:
1. CLI 引数をパース
2. `resolveOrRegisterProject(cwd)` でプロジェクト ID 取得
3. `getGitUserName(cwd)` で著者取得
4. DecisionEntry JSON を生成
5. `.alfred/knowledge/decisions/` にファイル書き込み
6. `upsertKnowledge()` で DB にインデックス

**セキュリティ**: hook-internal は外部から直接呼ばれることを想定しない。引数のサニタイズは行うが、認証は不要。

<!-- confidence: 7 | source: inference | grounding: inferred -->

### C-8: hooks.json ハイブリッド構成

**変更内容** (FR-7):

```json
{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "alfred hook SessionStart",
            "timeout": 5
          }
        ]
      }
    ],
    "PreCompact": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "alfred hook PreCompact",
            "timeout": 10
          }
        ]
      },
      {
        "hooks": [
          {
            "type": "agent",
            "prompt": "<PreCompact agent prompt>",
            "model": "claude-haiku-4-5-20251001",
            "timeout": 60
          }
        ]
      }
    ],
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "alfred hook UserPromptSubmit",
            "timeout": 10
          }
        ]
      },
      {
        "hooks": [
          {
            "type": "prompt",
            "prompt": "<UserPromptSubmit prompt>",
            "model": "claude-haiku-4-5-20251001",
            "timeout": 30
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "alfred hook PostToolUse",
            "timeout": 5
          }
        ]
      },
      {
        "matcher": "Edit|Write|Bash",
        "hooks": [
          {
            "type": "agent",
            "prompt": "<PostToolUse agent prompt>",
            "model": "claude-haiku-4-5-20251001",
            "timeout": 60
          }
        ]
      }
    ],
    "PreToolUse": [
      {
        "matcher": "Edit|Write",
        "hooks": [
          {
            "type": "command",
            "command": "alfred hook PreToolUse",
            "timeout": 5
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "alfred hook Stop",
            "timeout": 5
          }
        ]
      }
    ]
  }
}
```

<!-- confidence: 8 | source: design-doc | grounding: reviewed -->

### C-9: レビューループ強制

**目的**: fix_mode 後の gate clear に再レビュー実行を必須化
**変更ファイル**: `src/hooks/review-gate.ts`, `src/mcp/dossier/lifecycle.ts`, `src/hooks/post-tool.ts`

**ReviewGate インターフェース拡張**:
```typescript
interface ReviewGate {
  gate: "spec-review" | "wave-review";
  slug: string;
  wave?: number;
  set_at: string;
  reason: string;
  fix_mode?: boolean;
  fix_mode_at?: string;
  re_reviewed?: boolean;    // NEW: fix_mode 中にレビューが再実行されたか
  re_reviewed_at?: string;  // NEW: 再レビュー実行時刻
}
```

**gate clear 検証ロジック** (`src/mcp/dossier/lifecycle.ts`):
```typescript
case "clear": {
  // 既存: reason 30文字以上チェック
  if (!params.reason || params.reason.trim().length < 30) { ... }

  const gate = readReviewGate(projectPath);
  if (!gate) { ... }

  // NEW: fix_mode 後は再レビュー必須
  if (gate.fix_mode && !gate.re_reviewed) {
    return errorResult(
      "fix_mode で修正後、レビューを再実行してから gate clear してください。" +
      "alfred:code-reviewer agent または /alfred:inspect でレビューを実行してください。"
    );
  }

  clearReviewGate(projectPath);
  // ...
}
```

**再レビュー検出** (`src/hooks/post-tool.ts`):
PostToolUse の Agent レスポンスハンドラ（既存の `extractReviewKnowledge` 付近）で、
レビューエージェントのレスポンスを検出したら `re_reviewed: true` をセット:

```typescript
// Agent tool_response にレビュー結果が含まれるか判定
if (tool_name === "Agent" && isReviewResponse(tool_response)) {
  const gate = readReviewGate(projectPath);
  if (gate?.fix_mode) {
    writeReviewGate(projectPath, { ...gate, re_reviewed: true, re_reviewed_at: new Date().toISOString() });
  }
}
```

**isReviewResponse 判定**: tool_response に "Critical"/"High"/"Medium"/"Low" + "finding" 等のレビュー結果パターンが含まれるか。既存の `extractReviewFindings()` のパターンマッチを再利用。

<!-- confidence: 8 | source: user | grounding: reviewed -->

### C-10: dossier update JSON 自動 replace

**目的**: JSON ファイルへの append を防止し、構造的に不正な JSON の生成を防ぐ
**変更ファイル**: `src/mcp/dossier/crud.ts`

**変更内容** (FR-10):
`dossierUpdate()` の冒頭で、ファイル拡張子が `.json` の場合に mode を強制的に `replace` に切り替える:

```typescript
export function dossierUpdate(projectPath: string, store: Store, params: DossierParams) {
  if (!params.file) return errorResult("file is required for update");
  if (!params.content) return errorResult("content is required for update");

  // JSON files MUST use replace mode — append creates invalid JSON
  const isJson = params.file.endsWith(".json");
  const mode = isJson ? "replace" : (params.mode ?? "append");

  // ... rest unchanged
}
```

1行の変更。既存の append ロジック（stripTemplate + 追記）は .md ファイルでのみ動作する。

<!-- confidence: 9 | source: code | grounding: verified -->

## Data Models

### 新規状態ファイル

なし。既存の `.alfred/.state/` 構造を維持。`review-gate.json` に `re_reviewed` / `re_reviewed_at` フィールドを追加（後方互換: 既存ファイルにフィールドがなければ `false` 扱い）。

### Internal CLI 引数スキーマ

```typescript
interface SaveDecisionArgs {
  title: string;      // max 200 chars
  decision: string;   // max 1000 chars
  reasoning: string;  // max 1000 chars
  alternatives?: string; // max 1000 chars, newline-separated
  cwd?: string;       // defaults to process.cwd()
}
```

## Requirements Traceability Matrix

| Req ID | Component | Task ID | Test ID |
|--------|-----------|---------|---------|
| FR-1 | C-1 (UserPromptSubmit prompt hook) | T-1.1, T-1.2 | TS-1.1, TS-1.2, TS-1.3 |
| FR-2 | C-2 (UserPromptSubmit command hook slimmed) | T-1.3, T-1.4 | TS-2.1, TS-2.2 |
| FR-3 | C-3 (PostToolUse agent hook) | T-2.1, T-2.2 | TS-3.1, TS-3.2, TS-3.3 |
| FR-4 | C-4 (PostToolUse command hook slimmed) | T-2.3 | TS-4.1, TS-4.2 |
| FR-5 | C-5 (PreCompact agent hook), C-7 (Internal CLI) | T-3.1, T-3.2, T-3.3 | TS-5.1, TS-5.2, TS-5.3 |
| FR-6 | C-6 (PreCompact command hook slimmed) | T-3.4 | TS-6.1 |
| FR-7 | C-8 (hooks.json) | T-4.1 | TS-7.1 |
| FR-8 | C-1, C-3, C-5 (prompt design) | T-1.1, T-2.1, T-3.1 | TS-8.1 |
| NFR-1 | C-8 (parallel execution) | T-4.2 | TS-9.1 |
| NFR-2 | C-1, C-3, C-5 (Haiku model) | T-1.1, T-2.1, T-3.1 | — |
| NFR-3 | C-2, C-4, C-6 (fallback) | T-1.4, T-2.3, T-3.4 | TS-9.2 |
| NFR-4 | All | T-4.3 | TS-9.3 |
| NFR-5 | C-8 (output merge) | T-4.2 | TS-9.4 |
| FR-9 | C-9 (review loop enforcement) | T-4.1, T-4.2 | TS-10.1, TS-10.2, TS-10.3 |
| FR-10 | C-10 (dossier update JSON replace) | T-4.6 | TS-11.1, TS-11.2 |

## Tech Decisions

- DEC-1: prompt hook に Haiku を指定（コスト効率 + 十分な分類精度）
- DEC-2: PostToolUse は `agent` タイプ（tasks.json の Read が必要）
- DEC-3: PreCompact agent は Bash 経由で DB 保存（additionalContext 非対応のため）
- DEC-4: Spec 提案ガードは command hook に残す（状態ファイル読み取りが必要）
- DEC-5: command hook のフォールバック意思決定抽出は V1 で完全削除（重複回避の複雑さ > フォールバック価値）
- DEC-6: nudge impression tracking は廃止（LLM 分類の精度向上で不要に）
- DEC-7: fix_mode 後の gate clear に re_reviewed フラグ検証を追加（レビュー→修正→再レビューループの機械的強制）
