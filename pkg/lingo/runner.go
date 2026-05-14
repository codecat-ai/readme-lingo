package lingo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type TranslateOptions struct {
	SourcePath    string
	Targets       []string
	OutputPath    string
	OutputDir     string
	OutputPattern string
	GlossaryPath  string
	DryRun        bool
	ChunkHeadings bool
	MaxChunkChars int
	Switcher      string
	AutoSwitcher  bool
	Model         string
	Now           func() time.Time
}

type TranslateResult struct {
	Plans []OutputPlan
}

func RunTranslate(ctx context.Context, opts TranslateOptions, translator Translator, log io.Writer) (TranslateResult, error) {
	plans, err := planTranslateOutputs(opts.SourcePath, opts.Targets, opts.OutputPath, opts.OutputDir, opts.OutputPattern)
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
	if opts.ChunkHeadings && opts.MaxChunkChars <= 0 {
		return TranslateResult{}, errors.New("max chars must be positive")
	}
	source, err := os.ReadFile(opts.SourcePath)
	if err != nil {
		return TranslateResult{}, err
	}
	var glossary string
	if opts.GlossaryPath != "" {
		data, err := os.ReadFile(opts.GlossaryPath)
		if err != nil {
			return TranslateResult{}, fmt.Errorf("read glossary %q: %w", opts.GlossaryPath, err)
		}
		glossary = string(data)
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
	if opts.Switcher != "" && opts.AutoSwitcher {
		return TranslateResult{}, errors.New("--switcher and --auto-switcher cannot be used together")
	}
	if opts.Switcher != "" {
		switcher, err = BuildSwitcher(opts.Switcher)
		if err != nil {
			return TranslateResult{}, err
		}
	}
	if opts.AutoSwitcher {
		switcher, err = BuildAutoSwitcher(opts.SourcePath, plans)
		if err != nil {
			return TranslateResult{}, err
		}
		sourceWithSwitcher := InsertSwitcher(string(source), switcher)
		if err := os.WriteFile(opts.SourcePath, []byte(sourceWithSwitcher), 0o644); err != nil {
			return TranslateResult{}, err
		}
		source = []byte(sourceWithSwitcher)
		fmt.Fprintf(logOrDiscard(log), "updated %s\n", opts.SourcePath)
	}
	for _, plan := range plans {
		translated, err := translateMarkdown(ctx, translator, TranslateRequest{
			SourcePath: opts.SourcePath,
			Target:     plan.Target,
			Markdown:   string(source),
			Glossary:   glossary,
		}, opts.ChunkHeadings, opts.MaxChunkChars)
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

func translateMarkdown(ctx context.Context, translator Translator, req TranslateRequest, chunkHeadings bool, maxChunkChars int) (string, error) {
	if !chunkHeadings {
		return translator.Translate(ctx, req)
	}
	chunks, err := PlanMarkdownChunks(req.Markdown, maxChunkChars)
	if err != nil {
		return "", err
	}
	translatedChunks := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		chunkReq := req
		chunkReq.Markdown = chunk.Markdown
		translated, err := translator.Translate(ctx, chunkReq)
		if err != nil {
			return "", err
		}
		translatedChunks = append(translatedChunks, strings.Trim(translated, "\n"))
	}
	return strings.Join(translatedChunks, "\n\n"), nil
}

type CheckOptions struct {
	SourcePath    string
	Targets       []string
	OutputPath    string
	OutputDir     string
	OutputPattern string
}

type CheckResult struct {
	Plans   []OutputPlan
	Missing []OutputPlan
	Stale   []OutputPlan
}

func RunCheck(opts CheckOptions) (CheckResult, error) {
	plans, err := planTranslateOutputs(opts.SourcePath, opts.Targets, opts.OutputPath, opts.OutputDir, opts.OutputPattern)
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

func planTranslateOutputs(source string, targets []string, output string, outputDir string, outputPattern string) ([]OutputPlan, error) {
	if outputPattern != "" {
		return PlanOutputsWithPattern(source, targets, output, outputDir, outputPattern)
	}
	return PlanOutputs(source, targets, output, outputDir)
}

func logOrDiscard(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
}
