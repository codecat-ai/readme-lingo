package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/codecat-ai/readme-lingo/pkg/lingo"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return usageError("expected subcommand: translate or workflow")
	}
	switch args[0] {
	case "translate":
		return runTranslate(args[1:])
	case "workflow":
		return runWorkflow(args[1:])
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		return usageError("unknown subcommand: " + args[0])
	}
}

func runWorkflow(args []string) error {
	fs := flag.NewFlagSet("workflow", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	source := fs.String("source", lingo.DefaultWorkflowSource, "source Markdown file")
	targetsValue := fs.String("targets", "", "comma-separated target language tags or names")
	outputDir := fs.String("output-dir", lingo.DefaultWorkflowOutputDir, "output directory for default target filenames")
	goVersion := fs.String("go-version", lingo.DefaultWorkflowGoVersion, "Go version for generated CI setup")
	name := fs.String("name", lingo.DefaultWorkflowName, "workflow name for GitHub Actions or job name for GitLab CI")
	platform := fs.String("platform", lingo.DefaultWorkflowPlatform, "workflow platform: github or gitlab")
	schedule := fs.String("schedule", "", "cron schedule for generated workflow checks")
	branchesValue := fs.String("branches", "", "comma-separated branch filters for generated workflow checks")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*targetsValue) == "" {
		return errors.New("provide --targets")
	}
	targets, err := lingo.SplitTargets(*targetsValue)
	if err != nil {
		return fmt.Errorf("invalid --targets: %w", err)
	}
	workflow, err := lingo.RenderWorkflow(lingo.WorkflowOptions{
		Name:      *name,
		Platform:  *platform,
		Source:    *source,
		Targets:   targets,
		OutputDir: *outputDir,
		GoVersion: *goVersion,
		Schedule:  *schedule,
		Branches:  splitOptionalList(*branchesValue),
	})
	if err != nil {
		return err
	}
	fmt.Fprint(os.Stdout, workflow)
	return nil
}

func splitOptionalList(value string) []string {
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			items = append(items, part)
		}
	}
	return items
}

func runTranslate(args []string) error {
	fs := flag.NewFlagSet("translate", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	source := fs.String("source", "README.md", "source Markdown file")
	target := fs.String("target", "", "target language tag or name")
	targetsValue := fs.String("targets", "", "comma-separated target language tags or names")
	output := fs.String("output", "", "output file for a single target")
	outputDir := fs.String("output-dir", "", "output directory for default target filenames")
	outputPattern := fs.String("output-pattern", "", "filename pattern for multi-target outputs; supports {target}, {sourceBase}, and {sourceExt}")
	glossary := fs.String("glossary", "", "UTF-8 text or Markdown file with project terminology guidance")
	dryRun := fs.Bool("dry-run", false, "validate inputs and print planned outputs without calling the API")
	check := fs.Bool("check", false, "verify translated files exist and match the source digest")
	githubAnnotations := fs.Bool("github-annotations", false, "emit GitHub Actions annotations for --check failures")
	switcher := fs.String("switcher", "", "comma-separated target:path pairs for the top language switcher")
	autoSwitcher := fs.Bool("auto-switcher", false, "automatically manage a top language switcher from the source and target outputs")
	chunkHeadings := fs.Bool("chunk-headings", false, "split Markdown into heading-aware translation chunks")
	maxChars := fs.Int("max-chars", lingo.DefaultMaxChunkChars, "maximum characters per heading chunk")
	baseURL := fs.String("base-url", lingo.DefaultBaseURL, "OpenAI-compatible API base URL")
	model := fs.String("model", lingo.DefaultModel, "chat completions model")
	apiKeyEnv := fs.String("api-key-env", lingo.DefaultKeyEnv, "environment variable containing the API key")
	if err := fs.Parse(args); err != nil {
		return err
	}
	outputPatternSet := flagWasProvided(fs, "output-pattern")
	if outputPatternSet && strings.TrimSpace(*outputPattern) == "" {
		return errors.New("--output-pattern cannot be empty")
	}

	targets, err := collectTargets(*target, *targetsValue)
	if err != nil {
		return err
	}
	if *maxChars <= 0 {
		return errors.New("max chars must be positive")
	}
	if *check {
		result, err := lingo.RunCheck(lingo.CheckOptions{
			SourcePath:    *source,
			Targets:       targets,
			OutputPath:    *output,
			OutputDir:     *outputDir,
			OutputPattern: selectedOutputPattern(*outputPattern, outputPatternSet),
		})
		for _, plan := range result.Missing {
			fmt.Fprintf(os.Stdout, "missing: %s\n", plan.OutputPath)
			if *githubAnnotations {
				printGitHubAnnotation(plan, "missing translation", "Missing translation output "+plan.OutputPath)
			}
		}
		for _, plan := range result.Stale {
			fmt.Fprintf(os.Stdout, "stale: %s\n", plan.OutputPath)
			if *githubAnnotations {
				printGitHubAnnotation(plan, "stale translation", "Stale translation output "+plan.OutputPath)
			}
		}
		if err == nil {
			fmt.Fprintln(os.Stdout, "ok: translations are synchronized")
		}
		return err
	}

	apiKey := os.Getenv(*apiKeyEnv)
	if apiKey == "" && !*dryRun {
		return fmt.Errorf("API key is required; set %s or pass --api-key-env with a different environment variable name", *apiKeyEnv)
	}
	client := lingo.NewClient(lingo.ClientConfig{
		BaseURL: *baseURL,
		Model:   *model,
		APIKey:  apiKey,
	})
	_, err = lingo.RunTranslate(context.Background(), lingo.TranslateOptions{
		SourcePath:    *source,
		Targets:       targets,
		OutputPath:    *output,
		OutputDir:     *outputDir,
		OutputPattern: selectedOutputPattern(*outputPattern, outputPatternSet),
		GlossaryPath:  *glossary,
		DryRun:        *dryRun,
		ChunkHeadings: *chunkHeadings,
		MaxChunkChars: *maxChars,
		Switcher:      *switcher,
		AutoSwitcher:  *autoSwitcher,
		Model:         *model,
	}, client, os.Stdout)
	return err
}

func flagWasProvided(fs *flag.FlagSet, name string) bool {
	found := false
	fs.Visit(func(flag *flag.Flag) {
		if flag.Name == name {
			found = true
		}
	})
	return found
}

func selectedOutputPattern(pattern string, wasProvided bool) string {
	if !wasProvided {
		return ""
	}
	return pattern
}

func printGitHubAnnotation(plan lingo.OutputPlan, title string, message string) {
	fmt.Fprintf(
		os.Stdout,
		"::error file=%s,title=%s::%s\n",
		escapeGitHubAnnotationData(plan.SourcePath),
		escapeGitHubAnnotationData(title),
		escapeGitHubAnnotationData(message),
	)
}

func escapeGitHubAnnotationData(value string) string {
	value = strings.ReplaceAll(value, "%", "%25")
	value = strings.ReplaceAll(value, "\r", "%0D")
	value = strings.ReplaceAll(value, "\n", "%0A")
	return value
}

func collectTargets(target string, targetsValue string) ([]string, error) {
	var targets []string
	if strings.TrimSpace(target) != "" {
		targets = append(targets, strings.TrimSpace(target))
	}
	if strings.TrimSpace(targetsValue) != "" {
		more, err := lingo.SplitTargets(targetsValue)
		if err != nil {
			return nil, err
		}
		targets = append(targets, more...)
	}
	if len(targets) == 0 {
		return nil, errors.New("provide --target or --targets")
	}
	return targets, nil
}

func usageError(message string) error {
	printUsage()
	return errors.New(message)
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  readme-lingo translate --source README.md --target zh --output README-zh.md")
	fmt.Fprintln(os.Stderr, "  readme-lingo translate --source README.md --targets zh,ja,fr --output-dir .")
	fmt.Fprintln(os.Stderr, "  readme-lingo workflow --targets zh,ja")
	fmt.Fprintln(os.Stderr, "  readme-lingo workflow --platform gitlab --targets zh,ja --schedule '30 2 * * 1' --branches main,release")
}
