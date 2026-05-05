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
		return usageError("expected subcommand: translate")
	}
	switch args[0] {
	case "translate":
		return runTranslate(args[1:])
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		return usageError("unknown subcommand: " + args[0])
	}
}

func runTranslate(args []string) error {
	fs := flag.NewFlagSet("translate", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	source := fs.String("source", "README.md", "source Markdown file")
	target := fs.String("target", "", "target language tag or name")
	targetsValue := fs.String("targets", "", "comma-separated target language tags or names")
	output := fs.String("output", "", "output file for a single target")
	outputDir := fs.String("output-dir", "", "output directory for default target filenames")
	dryRun := fs.Bool("dry-run", false, "validate inputs and print planned outputs without calling the API")
	check := fs.Bool("check", false, "verify translated files exist and match the source digest")
	switcher := fs.String("switcher", "", "comma-separated target:path pairs for the top language switcher")
	baseURL := fs.String("base-url", lingo.DefaultBaseURL, "OpenAI-compatible API base URL")
	model := fs.String("model", lingo.DefaultModel, "chat completions model")
	apiKeyEnv := fs.String("api-key-env", lingo.DefaultKeyEnv, "environment variable containing the API key")
	if err := fs.Parse(args); err != nil {
		return err
	}

	targets, err := collectTargets(*target, *targetsValue)
	if err != nil {
		return err
	}
	if *check {
		result, err := lingo.RunCheck(lingo.CheckOptions{
			SourcePath: *source,
			Targets:    targets,
			OutputPath: *output,
			OutputDir:  *outputDir,
		})
		for _, plan := range result.Missing {
			fmt.Fprintf(os.Stdout, "missing: %s\n", plan.OutputPath)
		}
		for _, plan := range result.Stale {
			fmt.Fprintf(os.Stdout, "stale: %s\n", plan.OutputPath)
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
		SourcePath: *source,
		Targets:    targets,
		OutputPath: *output,
		OutputDir:  *outputDir,
		DryRun:     *dryRun,
		Switcher:   *switcher,
		Model:      *model,
	}, client, os.Stdout)
	return err
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
}
