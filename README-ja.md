<!-- readme-lingo-switcher:start -->
[English](README.md) | [中文](README-zh.md) | [日本語](README-ja.md)
<!-- readme-lingo-switcher:end -->

# readme-lingo

readme-lingo は、多言語 README を同期して保つための Go 製コマンドラインユーティリティです。OpenAI 互換の Chat Completions API を使ってソース Markdown ファイルを翻訳し、Markdown の構造をできるだけ維持しながらターゲットファイルを書き出します。また、後続のチェックで翻訳が古くなったか判断できるように、非表示のソース digest メタデータを埋め込みます。

このプロジェクトは MIT ライセンスで、現時点では GitHub からクローンしてローカルでビルドする利用を想定しています。API キーを保存せず、リポジトリ、テスト、例、ログ、Issue 報告に本物のキーを入れる必要もありません。

## 問題と動機

メンテナーは多くの場合、まず `README.md` を更新し、その後で翻訳版 README の構造や内容を同期することを忘れがちです。readme-lingo は、OpenAI 互換 API で Markdown を翻訳し、対象ファイルを書き込み、後から古い翻訳を検出できるソースダイジェストを埋め込む、繰り返し可能なコマンドを提供します。

## 機能

- 1 つの README または Markdown ファイルを任意の言語タグや言語名に翻訳します。
- 複数ターゲットを 1 つのコマンドで翻訳し、予測しやすい `README-<target>.md` 出力を生成します。
- コードフェンス、リンク、表、front matter、HTML コメントを保つようモデルに指示し、Markdown 構造をできるだけ維持します。
- ソースダイジェスト、ソースパス、対象言語、モデル、生成時刻を含む非表示メタデータを追加します。
- API を呼び出さずに、生成済み翻訳が欠落または古くなっていないか確認します。
- 多言語 README ナビゲーション用の上部言語スイッチャーを挿入します。

## インストール

readme-lingo はまだパッケージレジストリに公開されていません。タグ付きリリースが利用可能になるまでは、以下の GitHub ローカルワークフローを使ってください。

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

このプロジェクトはタグ付き Go module release としてまだ公開されていないため、`go install github.com/codecat-ai/readme-lingo@latest` はまだ案内していません。

## 例

単一ターゲットを翻訳:

```sh
go run ./cmd/readme-lingo translate --source README.md --target zh --output README-zh.md
```

複数ターゲットをデフォルト名で翻訳:

```sh
go run ./cmd/readme-lingo translate --source README.md --targets zh,ja,fr --output-dir .
```

デフォルトの出力名は `README-<target>.md` です。リポジトリ規約として `README-ja.md` を使いたい場合は、日本語ターゲットに対して単一ターゲットコマンドで明示的に出力を指定します。

```sh
go run ./cmd/readme-lingo translate --source README.md --target ja --output README-ja.md
```

API を呼び出さずに予定される出力だけを表示:

```sh
go run ./cmd/readme-lingo translate --source README.md --targets zh,ja --output-dir . --dry-run
```

期待される翻訳ファイルが存在し、現在のソース digest と一致するか確認:

```sh
go run ./cmd/readme-lingo translate --source README.md --targets zh,jp --output-dir . --check
```

生成ファイルの先頭に言語スイッチャーを挿入:

```sh
go run ./cmd/readme-lingo translate \
  --source README.md \
  --targets zh,jp \
  --output-dir . \
  --switcher "en:README.md,zh:README-zh.md,ja:README-ja.md"
```

## 開発

再利用可能なパッケージは `pkg/lingo` にあり、次の機能を扱います。

- OpenAI 互換 Chat Completions API 用の翻訳クライアント
- Markdown 翻訳リクエストの準備
- ソース digest メタデータの生成と同期チェック
- 単一ターゲットおよび複数ターゲット実行の出力計画
- スクリプト化された自動化に向けた dry-run と check ワークフロー

`cmd/readme-lingo` の CLI は意図的に薄くし、挙動をパッケージに委譲しています。

## テスト

ローカルチェックをすべて実行:

```sh
go test ./...
go vet ./...
test -z "$(gofmt -l .)"
```

ユニットテストは fake HTTP transport と一時ファイルを使います。実際の API は呼び出さず、API key も不要です。

## ロードマップ

- 大きな README に向けた、より Markdown を意識した分割
- オプションの用語集サポート
- CI annotation 向けの stale-check レポート改善
- リリースタグと公開 module 向けインストール手順
- 定期的な README 同期ワークフロー例

## AI 支援メンテナンス

このプロジェクトでは、ドキュメント草案、テストレビュー、実装案の調査などのメンテナンス作業に AI 支援を使う場合があります。変更のレビュー、秘密情報をリポジトリに入れないこと、生成出力がプロジェクト方針に合っていることの確認はメンテナの責任です。

## ライセンス

MIT。詳しくは [LICENSE](LICENSE) を参照してください。

<!-- readme-lingo: {"source":"README.md","target":"jp","model":"google/gemma-4-26b-a4b-it:free","digest":"sha256:559d286f5bef64b232d7e3a4b998084c3a60cee0270b3977a0cc4cbb3621ad1f","generated":"2026-05-05T00:00:00Z"} -->
