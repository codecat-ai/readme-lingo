package lingo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type TranslateOptions struct {
	SourcePath string
	Targets    []string
	OutputPath string
	OutputDir  string
	DryRun     bool
	Switcher   string
	Model      string
	Now        func() time.Time
}

type TranslateResult struct {
	Plans []OutputPlan
}

func RunTranslate(ctx context.Context, opts TranslateOptions, translator Translator, log io.Writer) (TranslateResult, error) {
	plans, err := PlanOutputs(opts.SourcePath, opts.Targets, opts.OutputPath, opts.OutputDir)
	if err != nil {
		return TranslateResult{}, err
	}
	if _, err := os.Stat(opts.SourcePath); err != nil {
		return TranslateResult{}, err
	}
	if opts.DryRun {
		for _, plan := range plans {
			fmt.Fprintf(logOrDiscard(log), "plan: %s -> %s (%s)\n", plan.SourcePath, plan.OutputPath, plan.Target)
		}
		return TranslateResult{Plans: plans}, nil
	}
	if translator == nil {
		return TranslateResult{}, errors.New("translator is required")
	}
	source, err := os.ReadFile(opts.SourcePath)
	if err != nil {
		return TranslateResult{}, err
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	model := opts.Model
	if model == "" {
		model = DefaultModel
	}
	var switcher string
	if opts.Switcher != "" {
		switcher, err = BuildSwitcher(opts.Switcher)
		if err != nil {
			return TranslateResult{}, err
		}
	}
	for _, plan := range plans {
		translated, err := translator.Translate(ctx, TranslateRequest{
			SourcePath: opts.SourcePath,
			Target:     plan.Target,
			Markdown:   string(source),
		})
		if err != nil {
			return TranslateResult{}, err
		}
		if switcher != "" {
			translated = InsertSwitcher(translated, switcher)
		}
		translated = AddMetadata(translated, Metadata{
			SourcePath:  opts.SourcePath,
			Target:      plan.Target,
			Model:       model,
			SourceHash:  SourceDigest(source),
			GeneratedAt: now().UTC(),
		})
		if err := os.MkdirAll(filepath.Dir(plan.OutputPath), 0o755); err != nil {
			return TranslateResult{}, err
		}
		if err := os.WriteFile(plan.OutputPath, []byte(translated), 0o644); err != nil {
			return TranslateResult{}, err
		}
		fmt.Fprintf(logOrDiscard(log), "wrote %s\n", plan.OutputPath)
	}
	return TranslateResult{Plans: plans}, nil
}

type CheckOptions struct {
	SourcePath string
	Targets    []string
	OutputPath string
	OutputDir  string
}

type CheckResult struct {
	Plans   []OutputPlan
	Missing []OutputPlan
	Stale   []OutputPlan
}

func RunCheck(opts CheckOptions) (CheckResult, error) {
	plans, err := PlanOutputs(opts.SourcePath, opts.Targets, opts.OutputPath, opts.OutputDir)
	if err != nil {
		return CheckResult{}, err
	}
	source, err := os.ReadFile(opts.SourcePath)
	if err != nil {
		return CheckResult{}, err
	}
	result := CheckResult{Plans: plans}
	for _, plan := range plans {
		translated, err := os.ReadFile(plan.OutputPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				result.Missing = append(result.Missing, plan)
				continue
			}
			return result, err
		}
		if !IsSynchronized(source, translated) {
			result.Stale = append(result.Stale, plan)
		}
	}
	if len(result.Missing) > 0 || len(result.Stale) > 0 {
		return result, errors.New("translations are missing or stale")
	}
	return result, nil
}

func logOrDiscard(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
}
