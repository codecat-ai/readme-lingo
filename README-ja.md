<!-- readme-lingo-switcher:start -->
[English](README.md) | [中文](README-zh.md) | [日本語](README-ja.md)
<!-- readme-lingo-switcher:end -->

# readme-lingo

readme-lingo は、多言語 README を同期して保つための Go 製コマンドラインユーティリティです。OpenAI 互換の Chat Completions API を使ってソース Markdown ファイルを翻訳し、Markdown の構造をできるだけ維持しながらターゲットファイルを書き出します。また、後続のチェックで翻訳が古くなったか判断できるように、非表示のソース digest メタデータを埋め込みます。

このプロジェクトは MIT ライセンスで、GitHub の Go module から `go install` で直接インストールできます。API キーを保存せず、リポジトリ、テスト、例、ログ、Issue 報告に本物のキーを入れる必要もありません。

## 問題と動機

メンテナーは多くの場合、まず `README.md` を更新し、その後で翻訳版 README の構造や内容を同期することを忘れがちです。readme-lingo は、OpenAI 互換 API で Markdown を翻訳し、対象ファイルを書き込み、後から古い翻訳を検出できるソースダイジェストを埋め込む、繰り返し可能なコマンドを提供します。

## 機能

- 1 つの README または Markdown ファイルを任意の言語タグや言語名に翻訳します。
- 複数ターゲットを 1 つのコマンドで翻訳し、予測しやすい `README-<target>.md` 出力を生成します。リポジトリ固有のレイアウト向けに安全なファイル名パターンも選べます。
- コードフェンス、リンク、表、front matter、HTML コメントを保つようモデルに指示し、Markdown 構造をできるだけ維持します。
- 大きな Markdown ソースを見出し単位のチャンクに分けることで、各翻訳リクエストを小さく保ちながら、出力順と 1 つのメタデータ footer を維持できます。
- 翻訳プロンプトに任意の用語集ガイダンスを含め、プロジェクト用語の一貫性を保てます。
- ソースダイジェスト、ソースパス、対象言語、モデル、生成時刻を含む非表示メタデータを追加します。
- API を呼び出さずに、生成済み翻訳が欠落または古くなっていないか確認し、必要に応じて GitHub Actions error annotations を出力します。
- スケジュール実行と pull request または merge request で古い翻訳をチェックする GitHub Actions または GitLab CI テンプレートを生成します。
- 多言語 README ナビゲーション用の上部言語スイッチャーを挿入または自動管理します。

## インストール

readme-lingo は GitHub 上の Go module なので、別途パッケージレジストリへ公開する必要はありません。最新版は Go で直接インストールできます:

```sh
go install github.com/codecat-ai/readme-lingo/cmd/readme-lingo@latest
```

再現可能なインストールには、タグ付きリリース後に `@v0.1.0` のようなバージョン指定を推奨します。以下の clone ワークフローは開発用として引き続き有用です。

## 設定

readme-lingo は OpenRouter 互換の base URL と無料モデル例をデフォルトにしています。

```text
Base URL: https://openrouter.ai/api/v1
Model: google/gemma-4-26b-a4b-it:free
API key env: README_LINGO_API_KEY
```

API key は環境変数で渡します。

```sh
export README_LINGO_API_KEY="YOUR_API_KEY_HERE"
```

別の環境変数名を使う場合:

```sh
export MY_ROUTER_KEY="YOUR_API_KEY_HERE"
go run ./cmd/readme-lingo translate --source README.md --target zh --output README-zh.md --api-key-env MY_ROUTER_KEY
```

実際の API key をコマンド履歴、例、テストデータ、バグ報告に含めないでください。

## クイックスタート

GitHub からインストールして実行します。

```sh
go install github.com/codecat-ai/readme-lingo/cmd/readme-lingo@latest
readme-lingo translate --source README.md --target zh --output README-zh.md --dry-run
```

クローンしてローカルで実行します。

```sh
git clone https://github.com/codecat-ai/readme-lingo.git
cd readme-lingo
go test ./...
go run ./cmd/readme-lingo translate --source README.md --target zh --output README-zh.md --dry-run
```

ローカルバイナリをビルドします。

```sh
go build -o bin/readme-lingo ./cmd/readme-lingo
./bin/readme-lingo translate --source README.md --target zh --output README-zh.md --dry-run
```

## 例

単一ターゲットを翻訳:

```sh
go run ./cmd/readme-lingo translate --source README.md --target zh --output README-zh.md
```

複数ターゲットをデフォルト名で翻訳:

```sh
go run ./cmd/readme-lingo translate --source README.md --targets zh,ja,fr --output-dir .
```

デフォルトの出力名は `README-<target>.md` です。日本語では標準の言語コード `ja` を使うため、デフォルトの日本語出力は `README-ja.md` です。単一ターゲットコマンドで明示的に出力を指定することもできます。

```sh
go run ./cmd/readme-lingo translate --source README.md --target ja --output README-ja.md
```

複数ターゲットのレイアウトには、カスタム出力ファイル名パターンを使えます。

```sh
go run ./cmd/readme-lingo translate \
  --source docs/README.md \
  --targets zh,ja \
  --output-dir docs/i18n \
  --output-pattern "{sourceBase}.{target}{sourceExt}"
```

パターンでは `{target}`、`{sourceBase}`、`{sourceExt}` を使えます。この例では `docs/i18n/README.zh.md` と `docs/i18n/README.ja.md` を計画します。パターンには `{target}` が必須で、生成されるファイル名にパス区切り文字を含めることはできません。

実際に翻訳するときに、プロジェクト用語のガイダンスとして用語集ファイルを使います。

```sh
go run ./cmd/readme-lingo translate --source README.md --targets zh,ja --output-dir . --glossary GLOSSARY.md
```

用語集は UTF-8 のテキストまたは Markdown にできます。実際の翻訳時だけ翻訳プロンプトに含まれ、`--dry-run` と `--check` はこのフラグを受け付けますがファイルは読みません。

大きな README ファイルでは、見出し単位のチャンク分割を有効にできます。

```sh
go run ./cmd/readme-lingo translate --source README.md --targets zh,ja --output-dir . --chunk-headings --max-chars 12000
```

`--chunk-headings` は fenced code block の外側にある Markdown 見出し境界で分割します。`--max-chars` を超える section は section 内で分割せず、そのまま 1 チャンクとして扱います。生成される対象ファイルには readme-lingo メタデータ footer が 1 つだけ付きます。`--dry-run` と `--check` はこれらのチャンク分割フラグを受け付けますが、API は呼び出しません。

API を呼び出さずに予定される出力だけを表示:

```sh
go run ./cmd/readme-lingo translate --source README.md --targets zh,ja --output-dir . --dry-run
```

期待される翻訳ファイルが存在し、現在のソース digest と一致するか確認:

```sh
go run ./cmd/readme-lingo translate --source README.md --targets zh,ja --output-dir . --check
```

チェック時に欠落または古い翻訳の GitHub Actions error annotations を出力:

```sh
go run ./cmd/readme-lingo translate --source README.md --targets zh,ja --output-dir . --check --github-annotations
```

スケジュール実行と pull request で古い翻訳をチェックする GitHub Actions workflow を生成します。デフォルトの platform は GitHub Actions です。

```sh
go run ./cmd/readme-lingo workflow --targets zh,ja > .github/workflows/readme-lingo.yml
```

merge request とスケジュール実行で古い翻訳をチェックする GitLab CI job テンプレートを生成します。

```sh
go run ./cmd/readme-lingo workflow --platform gitlab --targets zh,ja > .gitlab-ci.yml
```

README の場所、Go の設定、workflow/job 名が異なる場合は、生成される CI テンプレートをカスタマイズできます。

```sh
go run ./cmd/readme-lingo workflow \
  --platform github \
  --source docs/README.md \
  --targets zh,ja \
  --output-dir docs \
  --go-version 1.22.x \
  --name "Docs translation check" \
  --schedule "30 2 * * 1" \
  --branches main,release
```

`--schedule` は GitHub Actions の cron をデフォルトの `0 0 * * 1` から指定値へ変更します。GitLab CI では pipeline schedule が job YAML ではなく GitLab 側で設定されるため、生成 YAML に注意コメントを含めます。`--branches` は GitHub に pull request ブランチフィルターを追加し、GitLab にはブランチ対応 rules を追加します。GitHub のスケジュール workflow は引き続きデフォルトブランチで実行されるため、生成される job にはスケジュールチェック用の `if:` guard が入ります。

ソース README と生成ファイルの上部言語スイッチャーを自動管理:

```sh
go run ./cmd/readme-lingo translate \
  --source README.md \
  --targets zh,ja \
  --output-dir . \
  --auto-switcher
```

## 開発

再利用可能なパッケージは `pkg/lingo` にあり、次の機能を扱います。

- OpenAI 互換 Chat Completions API 用の翻訳クライアント
- Markdown 翻訳リクエストの準備
- 大きな Markdown 翻訳リクエスト向けの見出し単位チャンク計画
- 用語ガイダンスのための任意の用語集伝播
- ソース digest メタデータの生成と同期チェック
- 単一ターゲットおよび複数ターゲット実行の出力計画。安全な設定可能命名パターンも含みます
- スクリプト化された自動化に向けた dry-run と check ワークフロー。GitHub Actions annotations も含みます
- 任意のブランチフィルターに対応した、スケジュール済み古い翻訳チェック向けの GitHub Actions と GitLab CI テンプレート生成

`cmd/readme-lingo` の CLI は意図的に薄くし、挙動をパッケージに委譲しています。

## テスト

ローカルチェックをすべて実行:

```sh
go test ./...
go vet ./...
test -z "$(gofmt -l .)"
```

ユニットテストは fake HTTP transport と一時ファイルを使います。実際の API は呼び出さず、API key も不要で、出力命名パターンと、用語集ファイルを読まない `--check --github-annotations` もカバーします。

## ロードマップ

- 再現可能な `go install ...@vX.Y.Z` インストール用のタグ付きリリース
- ダウンロード用バイナリの first-class release artifacts と checksums
- CI 連携向けの機械可読 JSON plan 出力

## AI 支援メンテナンス

このプロジェクトでは、ドキュメント草案、テストレビュー、実装案の調査などのメンテナンス作業に AI 支援を使う場合があります。変更のレビュー、秘密情報をリポジトリに入れないこと、生成出力がプロジェクト方針に合っていることの確認はメンテナの責任です。

## ライセンス

MIT。詳しくは [LICENSE](LICENSE) を参照してください。

<!-- readme-lingo: {"source":"README.md","target":"ja","model":"google/gemma-4-26b-a4b-it:free","digest":"sha256:4b58ff671674d6d2729c6947842eb30369579317221d89438970ffc6a4df7086","generated":"2026-05-05T00:00:00Z"} -->
