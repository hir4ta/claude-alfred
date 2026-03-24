# Design: bun-compile-onestep

## 方針

<!-- confidence: 9 | source: code | grounding: verified -->

二段階: build.ts (dev bundle) + patch-opentui.sh → bun build --compile (binary)

### build.ts (FR-1)
tsdown.config.ts を置換。Bun JS API で dev バンドル生成。@opentui は external (dev 時は node_modules から解決)。

### patch-opentui.sh (FR-2)
`@opentui/core` の動的テンプレートリテラル import を対象プラットフォーム用の静的 import に sed 置換。grep -c で検証。sed -i.bak でクロスプラットフォーム対応。

### CI / build-binaries.sh (FR-3)
release.yml: patch → `bun build --compile src/cli.ts`（--external 不要）。build-binaries.sh: プラットフォーム別に patch → compile → restore。

### バイナリ E2E (FR-4)
bun が静的 import を解決 → .dylib を $bunfs に埋め込み。@opentui/core の isBunfsPath() が $bunfs パスを処理。

### package.json (FR-5, NFR-1)
build スクリプト変更。tsdown/rolldown 除去。

## トレーサビリティ

| Req | Task |
|-----|------|
| FR-1 | T-1.1 |
| FR-2 | T-1.2 |
| FR-3 | T-1.3 |
| FR-4, FR-5 | T-1.4 |
| NFR-1 | T-1.4 |
