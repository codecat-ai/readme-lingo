package lingo

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

type OutputPlan struct {
	SourcePath string
	Target     string
	OutputPath string
}

func SplitTargets(value string) ([]string, error) {
	parts := strings.Split(value, ",")
	targets := make([]string, 0, len(parts))
	for _, part := range parts {
		target := strings.TrimSpace(part)
		if target == "" {
			continue
		}
		targets = append(targets, target)
	}
	if len(targets) == 0 {
		return nil, errors.New("at least one target is required")
	}
	return targets, nil
}

func PlanOutputs(source string, targets []string, output string, outputDir string) ([]OutputPlan, error) {
	if strings.TrimSpace(source) == "" {
		return nil, errors.New("source path is required")
	}
	if len(targets) == 0 {
		return nil, errors.New("at least one target is required")
	}
	if len(targets) > 1 && output != "" {
		return nil, errors.New("--output cannot be used with multiple targets; use --output-dir")
	}

	source = filepath.Clean(source)
	plans := make([]OutputPlan, 0, len(targets))
	for _, target := range targets {
		target = strings.TrimSpace(target)
		if target == "" {
			return nil, errors.New("target cannot be empty")
		}
		outputPath := filepath.Clean(output)
		if outputPath == "." || outputPath == "" {
			outputPath = defaultOutputPath(source, target, outputDir)
		}
		plans = append(plans, OutputPlan{
			SourcePath: source,
			Target:     target,
			OutputPath: outputPath,
		})
	}
	return plans, nil
}

func defaultOutputPath(source string, target string, outputDir string) string {
	dir := filepath.Dir(source)
	if outputDir != "" {
		dir = outputDir
	}
	base := filepath.Base(source)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	if ext == "" {
		ext = ".md"
	}
	return filepath.Clean(filepath.Join(dir, fmt.Sprintf("%s-%s%s", stem, slugTarget(target), ext)))
}

func slugTarget(target string) string {
	target = strings.TrimSpace(target)
	var b strings.Builder
	lastDash := false
	for _, r := range target {
		ok := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
		if ok {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
