package lingo

import (
	"errors"
	"fmt"
	"regexp"
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
	Schedule  string
	Branches  []string
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
	schedule := strings.TrimSpace(options.Schedule)
	if schedule == "" {
		schedule = "0 0 * * 1"
	}
	branches := normalizeWorkflowBranches(options.Branches)
	targets, err := normalizeWorkflowTargets(options.Targets)
	if err != nil {
		return "", err
	}

	switch platform {
	case "github":
		return renderGitHubWorkflow(name, source, targets, outputDir, goVersion, schedule, branches), nil
	case "gitlab":
		if name == DefaultWorkflowName {
			name = DefaultGitLabJobName
		}
		return renderGitLabWorkflow(name, source, targets, outputDir, goVersion, schedule, branches), nil
	default:
		return "", fmt.Errorf("invalid workflow platform %q: use github or gitlab", platform)
	}
}

func renderGitHubWorkflow(name string, source string, targets []string, outputDir string, goVersion string, schedule string, branches []string) string {
	command := workflowCheckCommand(source, targets, outputDir, true)
	var b strings.Builder
	fmt.Fprintf(&b, "name: %s\n\n", yamlScalar(name))
	b.WriteString("on:\n")
	b.WriteString("  pull_request:\n")
	if len(branches) > 0 {
		b.WriteString("    branches:\n")
		for _, branch := range branches {
			fmt.Fprintf(&b, "      - %s\n", yamlScalar(branch))
		}
	}
	b.WriteString("  schedule:\n")
	if len(branches) > 0 {
		b.WriteString("    # Scheduled workflows run on the default branch; this job guard limits scheduled checks.\n")
	}
	fmt.Fprintf(&b, "    - cron: \"%s\"\n\n", escapeDoubleQuoted(schedule))
	b.WriteString("jobs:\n")
	b.WriteString("  readme-lingo:\n")
	if len(branches) > 0 {
		fmt.Fprintf(&b, "    if: github.event_name != 'schedule' || %s\n", githubBranchExpression(branches))
	}
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

func renderGitLabWorkflow(name string, source string, targets []string, outputDir string, goVersion string, schedule string, branches []string) string {
	command := workflowCheckCommand(source, targets, outputDir, false)
	var b strings.Builder
	fmt.Fprintf(&b, "%s:\n", yamlScalar(name))
	fmt.Fprintf(&b, "  image: %s\n", gitLabGoImage(goVersion))
	b.WriteString("  rules:\n")
	if len(branches) > 0 {
		fmt.Fprintf(&b, "    - if: '$CI_PIPELINE_SOURCE == \"merge_request_event\" && (%s)'\n", gitLabBranchExpression("$CI_MERGE_REQUEST_TARGET_BRANCH_NAME", branches))
		if schedule != "0 0 * * 1" {
			fmt.Fprintf(&b, "    # Configure the GitLab pipeline schedule cron as: %s\n", yamlCommentText(schedule))
		}
		fmt.Fprintf(&b, "    - if: '$CI_PIPELINE_SOURCE == \"schedule\" && (%s)'\n", gitLabBranchExpression("$CI_COMMIT_BRANCH", branches))
	} else {
		b.WriteString("    - if: '$CI_PIPELINE_SOURCE == \"merge_request_event\"'\n")
		if schedule != "0 0 * * 1" {
			fmt.Fprintf(&b, "    # Configure the GitLab pipeline schedule cron as: %s\n", yamlCommentText(schedule))
		}
		b.WriteString("    - if: '$CI_PIPELINE_SOURCE == \"schedule\"'\n")
	}
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

func normalizeWorkflowBranches(branches []string) []string {
	normalized := make([]string, 0, len(branches))
	for _, branch := range branches {
		branch = strings.TrimSpace(branch)
		if branch != "" {
			normalized = append(normalized, branch)
		}
	}
	return normalized
}

func githubBranchExpression(branches []string) string {
	parts := make([]string, 0, len(branches))
	for _, branch := range branches {
		parts = append(parts, githubBranchPatternExpression(branch))
	}
	return strings.Join(parts, " || ")
}

func githubBranchPatternExpression(pattern string) string {
	ref := "github.ref_name"
	if !strings.Contains(pattern, "*") {
		return fmt.Sprintf("%s == %s", ref, githubExpressionString(pattern))
	}
	if pattern == "*" {
		return "true"
	}
	first := strings.Index(pattern, "*")
	last := strings.LastIndex(pattern, "*")
	prefix := pattern[:first]
	suffix := pattern[last+1:]
	var parts []string
	if prefix != "" {
		parts = append(parts, fmt.Sprintf("startsWith(%s, %s)", ref, githubExpressionString(prefix)))
	}
	if suffix != "" {
		parts = append(parts, fmt.Sprintf("endsWith(%s, %s)", ref, githubExpressionString(suffix)))
	}
	if len(parts) == 0 {
		return "true"
	}
	return strings.Join(parts, " && ")
}

func githubExpressionString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func gitLabBranchExpression(variable string, branches []string) string {
	parts := make([]string, 0, len(branches))
	for _, branch := range branches {
		if strings.Contains(branch, "*") {
			parts = append(parts, fmt.Sprintf("%s =~ /^%s$/", variable, gitLabRegexFromGlob(branch)))
		} else {
			parts = append(parts, fmt.Sprintf("%s == \"%s\"", variable, escapeDoubleQuoted(branch)))
		}
	}
	return strings.Join(parts, " || ")
}

func gitLabRegexFromGlob(pattern string) string {
	quoted := regexp.QuoteMeta(pattern)
	quoted = strings.ReplaceAll(quoted, `\*`, ".*")
	return strings.ReplaceAll(quoted, `/`, `\/`)
}

func gitLabGoImage(goVersion string) string {
	image := "golang:" + goVersion
	if strings.ContainsAny(image, " #[]{}&,*?|<>=!%@`\"'\n\r\t") {
		return yamlScalar(image)
	}
	return image
}

func yamlCommentText(value string) string {
	return strings.Join(strings.Fields(value), " ")
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
	value = strings.ReplaceAll(value, "\n", `\n`)
	value = strings.ReplaceAll(value, "\r", `\r`)
	value = strings.ReplaceAll(value, "\t", `\t`)
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
