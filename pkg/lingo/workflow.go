package lingo

import (
	"errors"
	"fmt"
	"strings"
)

const (
	DefaultWorkflowName      = "readme-lingo stale translations"
	DefaultWorkflowSource    = "README.md"
	DefaultWorkflowOutputDir = "."
	DefaultWorkflowGoVersion = "1.24.3"
)

type WorkflowOptions struct {
	Name      string
	Source    string
	Targets   []string
	OutputDir string
	GoVersion string
}

func RenderWorkflow(options WorkflowOptions) (string, error) {
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

	command := fmt.Sprintf(
		"readme-lingo translate --source %s --targets %s --output-dir %s --check --github-annotations",
		shellArg(source),
		shellArg(strings.Join(targets, ",")),
		shellArg(outputDir),
	)

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
	return b.String(), nil
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
