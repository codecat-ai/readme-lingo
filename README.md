<!-- readme-lingo-switcher:start -->
[English](README.md) | [中文](README-zh.md) | [日本語](README-jp.md)
<!-- readme-lingo-switcher:end -->

# readme-lingo

readme-lingo is a Go command-line utility for maintainers who keep multilingual README files synchronized. It translates a source Markdown file with an OpenAI-compatible Chat Completions API, preserves Markdown-oriented structure, writes target files, and embeds a hidden source digest marker so later checks can detect stale translations.

The project is MIT licensed and currently intended for local builds from GitHub. It does not store API keys and it never needs a real key in the repository, tests, examples, logs, or issue reports.

## Problem and Motivation

Maintainers often update `README.md` first and then forget that translated README files need the same structural changes. readme-lingo gives that workflow a repeatable command: translate Markdown through an OpenAI-compatible API, write target files, and embed a source digest that later checks can use to flag stale translations.

## Features

- Translate one README or Markdown file into any target language tag or language name.
- Translate multiple targets in one command with predictable `README-<target>.md` outputs.
- Preserve Markdown-oriented structure by prompting the model to keep code fences, links, tables, front matter, and HTML comments intact.
- Add hidden metadata with the source digest, source path, target, model, and generation time.
- Check whether generated translations are missing or stale without calling the API.
- Insert a top language switcher for multilingual README navigation.

## Installation

readme-lingo is not published to a package registry yet. Use the local GitHub workflow below until tagged releases are available.

## Configuration

readme-lingo defaults to the OpenRouter-compatible base URL and a free model example:

```text
Base URL: https://openrouter.ai/api/v1
Model: google/gemma-4-26b-a4b-it:free
API key env: README_LINGO_API_KEY
```

Provide an API key through the environment:

```sh
export README_LINGO_API_KEY="YOUR_API_KEY_HERE"
```

Use a different variable name when needed:

```sh
export MY_ROUTER_KEY="YOUR_API_KEY_HERE"
go run ./cmd/readme-lingo translate --source README.md --target zh --output README-zh.md --api-key-env MY_ROUTER_KEY
```

Do not pass real API keys in command history, examples, test data, or bug reports.

## Quick Start

Clone and run locally:

```sh
git clone https://github.com/codecat-ai/readme-lingo.git
cd readme-lingo
go test ./...
go run ./cmd/readme-lingo translate --source README.md --target zh --output README-zh.md --dry-run
```

Build a local binary:

```sh
go build -o bin/readme-lingo ./cmd/readme-lingo
./bin/readme-lingo translate --source README.md --target zh --output README-zh.md --dry-run
```

No `go install github.com/codecat-ai/readme-lingo@latest` command is documented yet because this project has not been published as a tagged Go module release.

## Examples

Translate one target:

```sh
go run ./cmd/readme-lingo translate --source README.md --target zh --output README-zh.md
```

Translate multiple targets using default output names:

```sh
go run ./cmd/readme-lingo translate --source README.md --targets zh,ja,fr --output-dir .
```

The default output name is `README-<target>.md`. If you want `README-jp.md` for repository convention while translating into Japanese, pass an explicit output for the single-target command:

```sh
go run ./cmd/readme-lingo translate --source README.md --target ja --output README-jp.md
```

Show planned outputs without calling the API:

```sh
go run ./cmd/readme-lingo translate --source README.md --targets zh,ja --output-dir . --dry-run
```

Check whether expected translations exist and match the current source digest:

```sh
go run ./cmd/readme-lingo translate --source README.md --targets zh,jp --output-dir . --check
```

Insert a top language switcher in generated files:

```sh
go run ./cmd/readme-lingo translate \
  --source README.md \
  --targets zh,jp \
  --output-dir . \
  --switcher "en:README.md,zh:README-zh.md,jp:README-jp.md"
```

## Development

The reusable package lives in `pkg/lingo` and covers:

- translation client for OpenAI-compatible Chat Completions APIs
- Markdown translation request preparation
- source digest metadata generation and synchronization checks
- output planning for single-target and multi-target runs
- dry-run and check workflows for scriptable automation

The CLI in `cmd/readme-lingo` is intentionally thin and delegates behavior to the package.

## Testing

Run the full local checks:

```sh
go test ./...
go vet ./...
test -z "$(gofmt -l .)"
```

Unit tests use fake HTTP transports and temporary files. They do not call real APIs and do not require API keys.

## Roadmap

- More Markdown-aware chunking for large README files
- Optional glossary support
- Better stale-check reporting for CI annotations
- Release tags and published module installation instructions
- Examples for scheduled README synchronization workflows

## AI-Assisted Maintenance

This project may use AI assistance for maintenance tasks such as drafting documentation, reviewing tests, or exploring implementation options. Maintainers are responsible for reviewing changes, keeping secrets out of the repository, and ensuring generated output matches project policy.

## License

MIT. See [LICENSE](LICENSE).
