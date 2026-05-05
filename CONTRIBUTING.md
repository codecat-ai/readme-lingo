# Contributing

Thanks for helping improve readme-lingo. Keep contributions focused, tested, and free of secrets.

## Development Setup

```sh
git clone https://github.com/codecat-ai/readme-lingo.git
cd readme-lingo
go test ./...
```

## TDD Expectations

For behavior changes, add or update a failing test first, run it, then implement the minimal code needed to pass. Include the failing and passing commands in your pull request notes when practical.

## Secret Handling

Do not commit real API keys, tokens, screenshots containing credentials, logs with authorization headers, or examples with live secrets. Use placeholders such as `YOUR_API_KEY_HERE`.

## Checks

Before opening a pull request, run:

```sh
go test ./...
go vet ./...
test -z "$(gofmt -l .)"
```

## Commit Messages

Use concise English commit messages. Do not add `Co-authored-by` trailers for AI tools.
