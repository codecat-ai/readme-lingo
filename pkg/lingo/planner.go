package lingo

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

const DefaultMaxChunkChars = 12000

type OutputPlan struct {
	SourcePath string
	Target     string
	OutputPath string
}

type MarkdownChunk struct {
	Markdown string
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

func PlanMarkdownChunks(markdown string, maxChars int) ([]MarkdownChunk, error) {
	if maxChars <= 0 {
		return nil, errors.New("max chars must be positive")
	}
	sections := markdownSections(markdown)
	if len(sections) == 0 {
		return []MarkdownChunk{{Markdown: ""}}, nil
	}
	chunks := make([]MarkdownChunk, 0, len(sections))
	var current string
	for _, section := range sections {
		section = strings.TrimRight(section, "\n")
		if section == "" {
			continue
		}
		if current == "" {
			current = section
			continue
		}
		candidate := current + "\n\n" + section
		if len(candidate) <= maxChars {
			current = candidate
			continue
		}
		chunks = append(chunks, MarkdownChunk{Markdown: current})
		current = section
	}
	if current != "" {
		chunks = append(chunks, MarkdownChunk{Markdown: current})
	}
	if len(chunks) == 0 {
		chunks = append(chunks, MarkdownChunk{Markdown: strings.TrimRight(markdown, "\n")})
	}
	return chunks, nil
}

func markdownSections(markdown string) []string {
	lines := strings.SplitAfter(markdown, "\n")
	sections := make([]string, 0)
	var current strings.Builder
	inFence := false
	var fenceMarker string

	flush := func() {
		if current.Len() == 0 {
			return
		}
		sections = append(sections, current.String())
		current.Reset()
	}

	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if fenceStart, marker := markdownFenceMarker(trimmed); fenceStart {
			if !inFence {
				inFence = true
				fenceMarker = marker
			} else if marker == fenceMarker {
				inFence = false
				fenceMarker = ""
			}
		}
		if !inFence && isMarkdownHeading(trimmed) {
			if current.Len() > 0 {
				flush()
			}
		}
		current.WriteString(line)
	}
	flush()
	if len(sections) > 1 && !strings.HasPrefix(strings.TrimLeft(sections[0], "\n \t"), "#") {
		sections[1] = sections[0] + sections[1]
		sections = sections[1:]
	}
	return sections
}

func markdownFenceMarker(trimmedLine string) (bool, string) {
	if strings.HasPrefix(trimmedLine, "```") {
		return true, "```"
	}
	if strings.HasPrefix(trimmedLine, "~~~") {
		return true, "~~~"
	}
	return false, ""
}

func isMarkdownHeading(trimmedLine string) bool {
	count := 0
	for count < len(trimmedLine) && trimmedLine[count] == '#' {
		count++
	}
	if count == 0 || count > 6 || count >= len(trimmedLine) {
		return false
	}
	return trimmedLine[count] == ' ' || trimmedLine[count] == '\t'
}
