<!-- readme-lingo-switcher:start -->
[English](README.md) | [中文](README-zh.md) | [日本語](README-ja.md)
<!-- readme-lingo-switcher:end -->

# readme-lingo

readme-lingo 是一个面向维护者的 Go 命令行工具，用于让多语言 README 文件保持同步。它通过兼容 OpenAI Chat Completions 的 API 翻译源 Markdown 文件，尽量保留 Markdown 结构，写入目标文件，并嵌入隐藏的源文件摘要标记，方便之后检查译文是否已经过期。

本项目使用 MIT 许可证，并可直接通过 GitHub Go module 使用 `go install` 安装。它不会存储 API key，仓库、测试、示例、日志或 issue 报告中也不应包含真实 key。

## 问题与动机

维护者通常会先更新 `README.md`，然后容易忘记同步翻译版 README 的结构和内容。readme-lingo 提供一个可重复执行的命令：通过兼容 OpenAI 的 API 翻译 Markdown，写入目标文件，并嵌入源文件摘要，之后可以用检查命令发现过期译文。

## 功能

- 将一个 README 或 Markdown 文件翻译为任意目标语言标签或语言名称。
- 用一个命令翻译多个目标，并生成可预测的 `README-<target>.md` 输出。
- 通过提示模型保留代码块、链接、表格、front matter 和 HTML 注释，尽量保持 Markdown 结构。
- 可在翻译提示中加入可选术语表指导，使项目术语保持一致。
- 写入包含源摘要、源路径、目标语言、模型和生成时间的隐藏元数据。
- 无需调用 API 即可检查生成的译文是否缺失或过期，并可选择输出 GitHub Actions error annotations。
- 插入或自动管理用于多语言 README 导航的顶部语言切换器。

## 安装

readme-lingo 是托管在 GitHub 上的 Go module，因此不需要单独发布到包注册表。可直接用 Go 安装最新版：

```sh
go install github.com/codecat-ai/readme-lingo/cmd/readme-lingo@latest
```

为了可复现安装，在发布标签后优先使用带版本的命令，例如 `@v0.1.0`。下面的 clone 工作流仍适合开发。

## 配置

readme-lingo 默认使用 OpenRouter 兼容的 base URL 和一个免费模型示例：

```text
Base URL: https://openrouter.ai/api/v1
Model: google/gemma-4-26b-a4b-it:free
API key env: README_LINGO_API_KEY
```

通过环境变量提供 API key：

```sh
export README_LINGO_API_KEY="YOUR_API_KEY_HERE"
```

如果需要使用其他环境变量名：

```sh
export MY_ROUTER_KEY="YOUR_API_KEY_HERE"
go run ./cmd/readme-lingo translate --source README.md --target zh --output README-zh.md --api-key-env MY_ROUTER_KEY
```

不要把真实 API key 放进 shell 历史、示例、测试数据或问题报告。

## 快速开始

从 GitHub 安装并运行：

```sh
go install github.com/codecat-ai/readme-lingo/cmd/readme-lingo@latest
readme-lingo translate --source README.md --target zh --output README-zh.md --dry-run
```

克隆并本地运行：

```sh
git clone https://github.com/codecat-ai/readme-lingo.git
cd readme-lingo
go test ./...
go run ./cmd/readme-lingo translate --source README.md --target zh --output README-zh.md --dry-run
```

构建本地二进制：

```sh
go build -o bin/readme-lingo ./cmd/readme-lingo
./bin/readme-lingo translate --source README.md --target zh --output README-zh.md --dry-run
```

## 示例

翻译单个目标：

```sh
go run ./cmd/readme-lingo translate --source README.md --target zh --output README-zh.md
```

翻译多个目标并使用默认输出文件名：

```sh
go run ./cmd/readme-lingo translate --source README.md --targets zh,ja,fr --output-dir .
```

默认输出名为 `README-<target>.md`。日语使用标准语言代码 `ja`，因此默认日语输出为 `README-ja.md`。也可以在单目标命令中显式指定输出：

```sh
go run ./cmd/readme-lingo translate --source README.md --target ja --output README-ja.md
```

在真实翻译时使用术语表文件提供项目术语指导：

```sh
go run ./cmd/readme-lingo translate --source README.md --targets zh,ja --output-dir . --glossary GLOSSARY.md
```

术语表可以是 UTF-8 文本或 Markdown。它只会在命令执行真实翻译时加入翻译提示；`--dry-run` 和 `--check` 会接受该参数，但不会读取文件。

只显示计划输出，不调用 API：

```sh
go run ./cmd/readme-lingo translate --source README.md --targets zh,ja --output-dir . --dry-run
```

检查期望的译文是否存在，并且是否匹配当前源文件摘要：

```sh
go run ./cmd/readme-lingo translate --source README.md --targets zh,ja --output-dir . --check
```

在检查时为缺失或过期译文输出 GitHub Actions error annotations：

```sh
go run ./cmd/readme-lingo translate --source README.md --targets zh,ja --output-dir . --check --github-annotations
```

在源 README 和生成文件中自动管理顶部语言切换器：

```sh
go run ./cmd/readme-lingo translate \
  --source README.md \
  --targets zh,ja \
  --output-dir . \
  --auto-switcher
```

## 开发

可复用包位于 `pkg/lingo`，覆盖以下能力：

- 面向 OpenAI 兼容 Chat Completions API 的翻译客户端
- Markdown 翻译请求准备
- 用于术语指导的可选术语表传递
- 源文件摘要元数据生成和同步检查
- 单目标和多目标运行的输出规划
- 用于脚本化自动化的 dry-run 与 check 工作流，包括 GitHub Actions annotations

`cmd/readme-lingo` 中的 CLI 有意保持精简，把行为委托给包实现。

## 测试

运行完整本地检查：

```sh
go test ./...
go vet ./...
test -z "$(gofmt -l .)"
```

单元测试使用假的 HTTP transport 和临时文件。测试不会调用真实 API，也不需要 API key，并覆盖不会读取术语表文件的 `--check --github-annotations`。

## 路线图

- 针对大型 README 的更强 Markdown 感知分块
- 用于可复现 `go install ...@vX.Y.Z` 安装的带标签 release
- 定时 README 同步工作流示例

## AI 辅助维护

本项目可能使用 AI 辅助完成维护任务，例如起草文档、审查测试或探索实现方案。维护者负责审查变更、确保仓库中没有秘密信息，并确认生成内容符合项目策略。

## 许可证

MIT。见 [LICENSE](LICENSE)。

<!-- readme-lingo: {"source":"README.md","target":"zh","model":"google/gemma-4-26b-a4b-it:free","digest":"sha256:fe0d70f1e1bfdae6744e948fe1487ccd59c9a103d783c77987a9f13a7dfde081","generated":"2026-05-05T00:00:00Z"} -->
