# Requirements: bun-compile-onestep

## 概要

tsdown を廃止し `bun build --compile` でソースから直接バイナリ生成する一本化。OpenTUI は維持。

<!-- confidence: 9 | source: code | grounding: verified -->

## 機能要件

- FR-1: tsdown.config.ts を廃止し build.ts (Bun JS API) で dev バンドルを生成する
- FR-2: `bun build --compile` でソースから直接バイナリを生成する（patch-opentui.sh で動的 import を静的化）
- FR-3: CI (release.yml) と build-binaries.sh を一本化パイプラインに更新する
- FR-4: コンパイル済みバイナリが任意ディレクトリから `alfred specs` / `alfred dashboard` を実行できる
- FR-5: ローカル開発（`bun src/cli.ts`、`task build`）が引き続き動作する

## 非機能要件

- NFR-1: tsdown / rolldown を devDependencies から除去
