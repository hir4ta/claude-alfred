# Requirements: Hook LLM Migration

## Goal

既存の command hook にLLMベースの判定を追加するハイブリッドアーキテクチャを構築し、インテント分類とタスク完了判定の精度を大幅に向上させる。

## Success Criteria

- インテント分類: 曖昧な日本語/英語プロンプトで正しいスキルが推薦される
- タスク完了判定: agent hook が候補を提案し、Claude 本体が最終判断（自動チェックの false positive を排除）
- 意思決定抽出: キーワードマッチでは拾えない暗黙的決定も検出
- 既存の command hook 機能に退行なし

## Out of Scope

- SessionStart / PreToolUse / Stop hook の変更
- prompt/agent hook の新規フレームワーク構築（Claude Code 組み込み機能を利用）
- Voyage AI 統合の変更（既存のまま）
- Web ダッシュボード/TUI の変更
- Nudge impression tracking (DEC-6 で廃止。LLM 分類精度向上により不要)
- Knowledge gap recording (recordKnowledgeGap 削除。API は空配列を返す)

---

## Functional Requirements

### FR-1: UserPromptSubmit prompt hook によるインテント分類
<!-- confidence: 9 | source: design-doc | grounding: verified -->

WHEN ユーザーがプロンプトを送信した時, the system SHALL prompt hook (Haiku) を用いてインテントを7種 (research/plan/implement/bugfix/review/tdd/save-knowledge) に分類し、対応するスキル推薦を additionalContext として出力する。

**AC-1.1**: Given 「このバグ直して」というプロンプト, When prompt hook が実行される, Then intent=bugfix, skill=/alfred:mend が additionalContext に含まれる
**AC-1.2**: Given 「パフォーマンス改善したいんだけど、まず現状を調べて」という曖昧なプロンプト, When prompt hook が実行される, Then intent=research が正しく分類される
**AC-1.3**: Given 「テスト書いてからリファクタして」という複合プロンプト, When prompt hook が実行される, Then 主要インテント (tdd) が選択される

### FR-2: UserPromptSubmit command hook のスリム化
<!-- confidence: 8 | source: inference | grounding: inferred -->

WHEN UserPromptSubmit command hook が実行される時, the system SHALL インテント分類ロジック（classifyIntent, classifyIntentSemantic, buildSkillNudge）と nudge impression tracking を削除し、knowledge search + spec 提案ガード（簡易キーワードチェック + Stage 1.5 並行開発ガード）のみを実行する。

**AC-2.1**: Given command hook 実行時, When インテント分類が削除されている, Then knowledge search 結果のみが additionalContext に含まれる（スキル推薦は含まれない）
**AC-2.2**: Given アクティブ spec がなく spec-prompt guard が未発火の時, When 実装系キーワードを含むプロンプトが送信される, Then spec 提案 DIRECTIVE が出力される
**AC-2.3**: Given アクティブ spec があり worked-slugs に含まれないスラッグの時, When 実装系キーワードを含むプロンプトが送信される, Then 並行開発 WARNING が出力される（Stage 1.5 維持）
**AC-2.4**: Given recordKnowledgeGap が削除されている, When knowledge gap API (`GET /api/knowledge/gaps`) が呼ばれる, Then 空配列を返す（エラーにならない）

### FR-3: PostToolUse agent hook によるタスク完了候補提案
<!-- confidence: 8 | source: design-doc | grounding: reviewed -->

WHEN Edit/Write/Bash ツールが実行完了した時, the system SHALL agent hook を用いて tasks.json を Read し、ファイル変更がどのタスクを完了させたかを意味的に評価し、完了候補を additionalContext として提案する。

**AC-3.1**: Given T-1.3 が「intent classification を prompt hook に移行」というタスクで, When `src/hooks/user-prompt.ts` が編集された, Then T-1.3 が完了候補として提案される
**AC-3.2**: Given README.md のみが編集された, When タスクに README 関連がない, Then agent hook は完了候補を提案しない
**AC-3.3**: Given agent hook がタイムアウトした, When 結果が返らない, Then Claude の動作に影響なし（additionalContext が空になるだけ）

### FR-4: PostToolUse command hook からの auto-check 削除
<!-- confidence: 8 | source: inference | grounding: inferred -->

WHEN PostToolUse command hook が実行される時, the system SHALL autoCheckTasks ロジック（ファイルパスマッチング + backtick 抽出）を削除し、git commit 検出・Living Spec・wave completion・DB 操作のみを実行する。

**AC-4.1**: Given Edit ツールでファイルが保存された, When command hook が実行される, Then tasks.json は自動的に更新されない
**AC-4.2**: Given git commit が検出された, When command hook が実行される, Then Living Spec auto-append と wave completion 検出は従来通り動作する

### FR-5: PreCompact agent hook による意思決定抽出
<!-- confidence: 6 | source: inference | grounding: inferred -->

WHEN PreCompact が実行される時, the system SHALL agent hook を用いて transcript を Read し、LLM が意思決定を識別・構造化して、Bash ツール経由で alfred CLI の内部コマンド (`alfred hook-internal save-decision`) を呼び出し DB に保存する。NOTE: PreCompact は additionalContext 非対応のため、agent hook は Bash 経由で直接 DB 保存する（additionalContext injection ではない）。

**AC-5.1**: Given transcript に「JWTではなくセッションCookieを採用する」という文脈がある, When agent hook が実行される, Then decision エントリが `.alfred/knowledge/decisions/` に保存され DB にインデックスされる
**AC-5.2**: Given transcript に意思決定が含まれない, When agent hook が実行される, Then 何も保存されない
**AC-5.3**: Given agent hook が 60s 以内にタイムアウトした, When 結果が返らない, Then 意思決定は抽出されない（command hook のキーワード抽出は V1 で削除済み。agent hook のみが意思決定抽出を担当）

### FR-6: PreCompact command hook のスリム化
<!-- confidence: 8 | source: inference | grounding: inferred -->

WHEN PreCompact command hook が実行される時, the system SHALL 意思決定抽出ロジック（extractDecisions, DECISION_KEYWORDS 等）を完全削除し、chapter memory snapshot・auto-complete・session breadcrumb のみを実行する。

**AC-6.1**: Given extractDecisions が完全削除されている, When command hook が実行される, Then chapter memory snapshot が保存される
**AC-6.2**: Given extractDecisions が完全削除されている, When command hook が実行される, Then session breadcrumb (`.pending-compact.json`) が書き出される

### FR-7: hooks.json のハイブリッド構成
<!-- confidence: 9 | source: design-doc | grounding: verified -->

The system SHALL plugin/hooks/hooks.json にハイブリッド構成（command + prompt/agent 並列）を定義し、3つのイベントに LLM hook を追加する。

**AC-7.1**: UserPromptSubmit に command hook と prompt hook の2つが登録されている
**AC-7.2**: PostToolUse に command hook と agent hook の2つが登録されている（agent は matcher: Edit|Write|Bash）
**AC-7.3**: PreCompact に command hook と agent hook の2つが登録されている
**AC-7.4**: SessionStart / PreToolUse / Stop は command hook のみ（変更なし）

### FR-8: PostToolUse agent hook のコスト制御
<!-- confidence: 7 | source: inference | grounding: inferred -->

WHEN PostToolUse agent hook が発火条件を満たした時, the system SHALL アクティブ spec が存在する場合のみ agent hook を実行し、コスト制御のためのスロットル機構を備える。

**AC-8.1**: Given アクティブ spec が存在しない, When Edit ツールが実行された, Then agent hook は発火しない（matcher 通過後に prompt 内で早期リターン）
**AC-8.2**: Given 各 prompt/agent hook のプロンプトが設計されている, When トークン数を計測する, Then UserPromptSubmit prompt は 500 トークン以下、agent prompts は 1000 トークン以下

### FR-9: レビューループ強制（gate clear に再レビュー検証）
<!-- confidence: 9 | source: user | grounding: verified -->

WHEN fix_mode で修正が行われた後に `dossier gate clear` が呼ばれた時, the system SHALL 修正後にレビューが再実行されたことを検証し、再レビューが未実行なら gate clear を拒否する。

**AC-9.1**: Given fix_mode が有効で re_reviewed=false の時, When `dossier gate clear` が呼ばれる, Then エラーを返す（「修正後にレビューを再実行してください」）
**AC-9.2**: Given fix_mode が有効の時, When code-reviewer agent がレスポンスを返した, Then review-gate.json の re_reviewed が true にセットされる
**AC-9.3**: Given re_reviewed=true の時, When `dossier gate clear` が呼ばれる, Then 通常通りクリアされる（30文字 reason 必須は維持）
**AC-9.4**: Given fix_mode でない通常の gate の時, When `dossier gate clear` が呼ばれる, Then 既存動作と同じ（re_reviewed チェックなし）

### FR-10: dossier update の JSON ファイル自動 replace
<!-- confidence: 9 | source: code | grounding: verified -->

WHEN `dossier update` で `.json` 拡張子のファイルが更新される時, the system SHALL mode パラメータに関わらず自動的に `replace` モードで書き込む。JSON ファイルへの `append` は構造的に不正な JSON を生成するため。

**AC-10.1**: Given `dossier update file=tasks.json mode=append` が呼ばれた, When ファイルが .json 拡張子, Then 実際には replace モードで書き込まれる
**AC-10.2**: Given `dossier update file=tasks.json` が呼ばれた（mode 省略）, When ファイルが .json 拡張子, Then replace モードで書き込まれる（append デフォルトが上書きされる）
**AC-10.3**: Given `dossier update file=requirements.md mode=append` が呼ばれた, When ファイルが .md 拡張子, Then 従来通り append モードで動作する

---

## Non-Functional Requirements

### NFR-1: レイテンシ影響なし
<!-- confidence: 8 | source: design-doc | grounding: reviewed -->

prompt/agent hook は command hook と並列実行されるため、全体のフック実行時間を増加させない。prompt hook のタイムアウトは 30s、agent hook は 60s とし、タイムアウト時は graceful に失敗する。

### NFR-2: コスト効率
<!-- confidence: 7 | source: inference | grounding: inferred -->

prompt hook は Haiku モデルを使用し、プロンプトは 500 トークン以下に抑える。PostToolUse agent hook は matcher で Edit|Write|Bash に限定し、不要な呼び出しを防ぐ。

### NFR-3: フォールバック保証
<!-- confidence: 8 | source: inference | grounding: inferred -->

prompt/agent hook が失敗（タイムアウト、エラー）した場合、command hook の出力のみで動作が継続する。LLM 判定は「あれば精度向上」の additive であり、なくても基本機能は動作する。

### NFR-4: 既存テスト互換
<!-- confidence: 8 | source: code | grounding: reviewed -->

command hook のスリム化後も、既存の `src/hooks/__tests__/` テストのうち、維持対象の機能（knowledge search, Living Spec, wave completion, gate enforcement）のテストは全てパスする。削除対象の機能（intent classification keyword, autoCheckTasks）のテストは削除または prompt/agent hook 用に書き換える。

### NFR-5: 出力マージ検証
<!-- confidence: 5 | source: assumption | grounding: speculative -->

同一イベントで command hook と prompt/agent hook の両方が additionalContext を返した場合の動作を検証し、両方が Claude に注入されることを確認する。競合時の挙動をテストで担保する。
