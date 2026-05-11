package lingo

import (
	"errors"
	"fmt"
	"strings"
)

const (
	DefaultWorkflowName      = "readme-lingo stale translations"
	DefaultWorkflowPlatform  = "github"
	DefaultGitLabJobName     = "readme_lingo"
	DefaultWorkflowSource    = "README.md"
	DefaultWorkflowOutputDir = "."
	DefaultWorkflowGoVersion = "1.24.3"
)

type WorkflowOptions struct {
	Name      string
	Platform  string
	Source    string
	Targets   []string
	OutputDir string
	GoVersion string
}

func RenderWorkflow(options WorkflowOptions) (string, error) {
	platform := strings.TrimSpace(options.Platform)
	if platform == "" {
		platform = DefaultWorkflowPlatform
	}
	name := strings.TrimSpace(options.Name)
	if name == "" {
		name = DefaultWorkflowName
	}
	source := strings.TrimSpace(options.Source)
	if source == "" {
		source = DefaultWorkflowSource
	}
	outputDir := strings.TrimSpace(options.OutputDir)
	if outputDir == "" {
		outputDir = DefaultWorkflowOutputDir
	}
	goVersion := strings.TrimSpace(options.GoVersion)
	if goVersion == "" {
		goVersion = DefaultWorkflowGoVersion
	}
	targets, err := normalizeWorkflowTargets(options.Targets)
	if err != nil {
		return "", err
	}

	switch platform {
	case "github":
		return renderGitHubWorkflow(name, source, targets, outputDir, goVersion), nil
	case "gitlab":
		if name == DefaultWorkflowName {
			name = DefaultGitLabJobName
		}
		return renderGitLabWorkflow(name, source, targets, outputDir, goVersion), nil
	default:
		return "", fmt.Errorf("invalid workflow platform %q: use github or gitlab", platform)
	}
}

func renderGitHubWorkflow(name string, source string, targets []string, outputDir string, goVersion string) string {
	command := workflowCheckCommand(source, targets, outputDir, true)
	var b strings.Builder
	fmt.Fprintf(&b, "name: %s\n\n", yamlScalar(name))
	b.WriteString("on:\n")
	b.WriteString("  pull_request:\n")
	b.WriteString("  schedule:\n")
	b.WriteString("    - cron: \"0 0 * * 1\"\n\n")
	b.WriteString("jobs:\n")
	b.WriteString("  readme-lingo:\n")
	b.WriteString("    runs-on: ubuntu-latest\n")
	b.WriteString("    steps:\n")
	b.WriteString("      - uses: actions/checkout@v4\n")
	b.WriteString("      - uses: actions/setup-go@v5\n")
	b.WriteString("        with:\n")
	fmt.Fprintf(&b, "          go-version: \"%s\"\n", escapeDoubleQuoted(goVersion))
	b.WriteString("      - name: Install readme-lingo\n")
	b.WriteString("        run: go install github.com/codecat-ai/readme-lingo/cmd/readme-lingo@latest\n")
	b.WriteString("      - name: Check README translations\n")
	fmt.Fprintf(&b, "        run: %s\n", command)
	return b.String()
}

func renderGitLabWorkflow(name string, source string, targets []string, outputDir string, goVersion string) string {
	command := workflowCheckCommand(source, targets, outputDir, false)
	var b strings.Builder
	fmt.Fprintf(&b, "%s:\n", yamlScalar(name))
	fmt.Fprintf(&b, "  image: golang:%s\n", goVersion)
	b.WriteString("  rules:\n")
	b.WriteString("    - if: '$CI_PIPELINE_SOURCE == \"merge_request_event\"'\n")
	b.WriteString("    - if: '$CI_PIPELINE_SOURCE == \"schedule\"'\n")
	b.WriteString("  script:\n")
	b.WriteString("    - go install github.com/codecat-ai/readme-lingo/cmd/readme-lingo@latest\n")
	fmt.Fprintf(&b, "    - %s\n", command)
	return b.String()
}

func workflowCheckCommand(source string, targets []string, outputDir string, githubAnnotations bool) string {
	command := fmt.Sprintf(
		"readme-lingo translate --source %s --targets %s --output-dir %s --check",
		shellArg(source),
		shellArg(strings.Join(targets, ",")),
		shellArg(outputDir),
	)
	if githubAnnotations {
		command += " --github-annotations"
	}
	return command
}

func normalizeWorkflowTargets(targets []string) ([]string, error) {
	normalized := make([]string, 0, len(targets))
	for _, target := range targets {
		target = strings.TrimSpace(target)
		if target != "" {
			normalized = append(normalized, target)
		}
	}
	if len(normalized) == 0 {
		return nil, errors.New("provide --targets")
	}
	return normalized, nil
}

func yamlScalar(value string) string {
	if value == "" || strings.ContainsAny(value, ":#[]{}&,*?|<>=!%@`\"'\n\r\t") || strings.HasPrefix(value, " ") || strings.HasSuffix(value, " ") {
		return `"` + escapeDoubleQuoted(value) + `"`
	}
	return value
}

func escapeDoubleQuoted(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return value
}

func shellArg(value string) string {
	if value != "" && strings.IndexFunc(value, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("./_:-,@%", r))
	}) == -1 {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
