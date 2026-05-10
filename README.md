<!-- readme-lingo-switcher:start -->
[English](README.md) | [中文](README-zh.md) | [日本語](README-ja.md)
<!-- readme-lingo-switcher:end -->

# readme-lingo

readme-lingo is a Go command-line utility for maintainers who keep multilingual README files synchronized. It translates a source Markdown file with an OpenAI-compatible Chat Completions API, preserves Markdown-oriented structure, writes target files, and embeds a hidden source digest marker so later checks can detect stale translations.

The project is MIT licensed and installable directly from its GitHub Go module with `go install`. It does not store API keys and it never needs a real key in the repository, tests, examples, logs, or issue reports.

## Problem and Motivation

Maintainers often update `README.md` first and then forget that translated README files need the same structural changes. readme-lingo gives that workflow a repeatable command: translate Markdown through an OpenAI-compatible API, write target files, and embed a source digest that later checks can use to flag stale translations.

## Features

- Translate one README or Markdown file into any target language tag or language name.
- Translate multiple targets in one command with predictable `README-<target>.md` outputs.
- Preserve Markdown-oriented structure by prompting the model to keep code fences, links, tables, front matter, and HTML comments intact.
- Optionally split large Markdown sources into heading-aware chunks so each translation request stays smaller while output order and one metadata footer are preserved.
- Include optional glossary guidance in translation prompts so project terminology stays consistent.
- Add hidden metadata with the source digest, source path, target, model, and generation time.
- Check whether generated translations are missing or stale without calling the API, with optional GitHub Actions error annotations.
- Generate a GitHub Actions workflow that runs scheduled and pull request stale-translation checks.
- Insert or automatically manage a top language switcher for multilingual README navigation.

## Installation

readme-lingo is a Go module hosted on GitHub, so no separate package registry publication is required. Install the latest version directly with Go:

```sh
go install github.com/codecat-ai/readme-lingo/cmd/readme-lingo@latest
```

For reproducible installs, prefer a tagged version once releases are tagged, for example `@v0.1.0`. The clone workflow below remains useful for development.

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

Install and run from GitHub:

```sh
go install github.com/codecat-ai/readme-lingo/cmd/readme-lingo@latest
readme-lingo translate --source README.md --target zh --output README-zh.md --dry-run
```

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

## Examples

Translate one target:

```sh
go run ./cmd/readme-lingo translate --source README.md --target zh --output README-zh.md
```

Translate multiple targets using default output names:

```sh
go run ./cmd/readme-lingo translate --source README.md --targets zh,ja,fr --output-dir .
```

The default output name is `README-<target>.md`. Japanese uses the standard `ja` language code, so the default Japanese output is `README-ja.md`. You can still pass an explicit output for single-target commands:

```sh
go run ./cmd/readme-lingo translate --source README.md --target ja --output README-ja.md
```

Use a glossary file for project terminology guidance during real translations:

```sh
go run ./cmd/readme-lingo translate --source README.md --targets zh,ja --output-dir . --glossary GLOSSARY.md
```

The glossary can be UTF-8 text or Markdown. It is included in translation prompts only when the command performs real translation; `--dry-run` and `--check` accept the flag without reading the file.

Opt in to heading-aware chunking for large README files:

```sh
go run ./cmd/readme-lingo translate --source README.md --targets zh,ja --output-dir . --chunk-headings --max-chars 12000
```

`--chunk-headings` splits on Markdown heading boundaries outside fenced code blocks. Sections larger than `--max-chars` stay intact instead of being split mid-section, and the generated target still receives one readme-lingo metadata footer. `--dry-run` and `--check` accept the chunking flags without calling the API.

Show planned outputs without calling the API:

```sh
go run ./cmd/readme-lingo translate --source README.md --targets zh,ja --output-dir . --dry-run
```

Check whether expected translations exist and match the current source digest:

```sh
go run ./cmd/readme-lingo translate --source README.md --targets zh,ja --output-dir . --check
```

Emit GitHub Actions error annotations for missing or stale translations during checks:

```sh
go run ./cmd/readme-lingo translate --source README.md --targets zh,ja --output-dir . --check --github-annotations
```

Generate a GitHub Actions workflow for scheduled and pull request stale-translation checks:

```sh
go run ./cmd/readme-lingo workflow --targets zh,ja > .github/workflows/readme-lingo.yml
```

Customize the generated workflow when your README or Go setup differs:

```sh
go run ./cmd/readme-lingo workflow \
  --source docs/README.md \
  --targets zh,ja \
  --output-dir docs \
  --go-version 1.22.x \
  --name "Docs translation check"
```

Automatically manage the top language switcher in the source and generated README files:

```sh
go run ./cmd/readme-lingo translate \
  --source README.md \
  --targets zh,ja \
  --output-dir . \
  --auto-switcher
```

## Development

The reusable package lives in `pkg/lingo` and covers:

- translation client for OpenAI-compatible Chat Completions APIs
- Markdown translation request preparation
- heading-aware chunk planning for large Markdown translation requests
- optional glossary propagation for terminology guidance
- source digest metadata generation and synchronization checks
- output planning for single-target and multi-target runs
- dry-run and check workflows for scriptable automation, including GitHub Actions annotations
- GitHub Actions workflow template generation for scheduled stale-translation checks

The CLI in `cmd/readme-lingo` is intentionally thin and delegates behavior to the package.

## Testing

Run the full local checks:

```sh
go test ./...
go vet ./...
test -z "$(gofmt -l .)"
```

Unit tests use fake HTTP transports and temporary files. They do not call real APIs, do not require API keys, and cover `--check --github-annotations` without reading glossary files.

## Roadmap

- Tagged releases for reproducible `go install ...@vX.Y.Z` installs
- Additional CI workflow templates beyond GitHub Actions

## AI-Assisted Maintenance

This project may use AI assistance for maintenance tasks such as drafting documentation, reviewing tests, or exploring implementation options. Maintainers are responsible for reviewing changes, keeping secrets out of the repository, and ensuring generated output matches project policy.

## License

MIT. See [LICENSE](LICENSE).
