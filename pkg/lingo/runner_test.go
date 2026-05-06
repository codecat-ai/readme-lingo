package lingo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type recordingTranslator struct {
	calls []TranslateRequest
}

func (r *recordingTranslator) Translate(_ context.Context, req TranslateRequest) (string, error) {
	r.calls = append(r.calls, req)
	return "# Translated " + req.Target + "\n", nil
}

func TestDryRunPlansOutputsWithoutNetwork(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "README.md")
	if err := os.WriteFile(source, []byte("# Demo\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	translator := &recordingTranslator{}
	var log strings.Builder

	result, err := RunTranslate(context.Background(), TranslateOptions{
		SourcePath: source,
		Targets:    []string{"zh", "ja"},
		OutputDir:  dir,
		DryRun:     true,
		Now:        fixedNow,
	}, translator, &log)
	if err != nil {
		t.Fatalf("RunTranslate returned error: %v", err)
	}
	if len(translator.calls) != 0 {
		t.Fatalf("dry-run called translator %d times", len(translator.calls))
	}
	if len(result.Plans) != 2 || !strings.Contains(log.String(), "README-zh.md") || !strings.Contains(log.String(), "README-ja.md") {
		t.Fatalf("dry-run plan output missing: result=%+v log=%q", result, log.String())
	}
}

func TestRunTranslateMultipleTargetsWritesMetadata(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "README.md")
	if err := os.WriteFile(source, []byte("# Demo\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	translator := &recordingTranslator{}

	_, err := RunTranslate(context.Background(), TranslateOptions{
		SourcePath: source,
		Targets:    []string{"zh", "ja"},
		OutputDir:  dir,
		Model:      "test-model",
		Now:        fixedNow,
	}, translator, nil)
	if err != nil {
		t.Fatalf("RunTranslate returned error: %v", err)
	}
	if len(translator.calls) != 2 {
		t.Fatalf("translator calls = %d, want 2", len(translator.calls))
	}
	for _, target := range []string{"zh", "ja"} {
		output := filepath.Join(dir, "README-"+target+".md")
		data, err := os.ReadFile(output)
		if err != nil {
			t.Fatalf("read output %s: %v", output, err)
		}
		meta, ok := ParseMetadata(data)
		if !ok {
			t.Fatalf("metadata missing in %s: %s", output, data)
		}
		if meta.Target != target || meta.Model != "test-model" {
			t.Fatalf("unexpected metadata for %s: %+v", target, meta)
		}
		if !IsSynchronized([]byte("# Demo\n"), data) {
			t.Fatalf("%s is not synchronized", output)
		}
	}
}

func TestRunTranslateIncludesGlossaryInEachRequest(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "README.md")
	if err := os.WriteFile(source, []byte("# Demo\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	glossary := filepath.Join(dir, "GLOSSARY.md")
	glossaryText := "Keep CLI flag names unchanged.\nUse readme-lingo as the product name."
	if err := os.WriteFile(glossary, []byte(glossaryText), 0o644); err != nil {
		t.Fatalf("write glossary: %v", err)
	}
	translator := &recordingTranslator{}

	_, err := RunTranslate(context.Background(), TranslateOptions{
		SourcePath:   source,
		Targets:      []string{"zh", "ja"},
		OutputDir:    dir,
		GlossaryPath: glossary,
		Now:          fixedNow,
	}, translator, nil)
	if err != nil {
		t.Fatalf("RunTranslate returned error: %v", err)
	}
	if len(translator.calls) != 2 {
		t.Fatalf("translator calls = %d, want 2", len(translator.calls))
	}
	for _, call := range translator.calls {
		if call.Glossary != glossaryText {
			t.Fatalf("glossary for %s = %q, want %q", call.Target, call.Glossary, glossaryText)
		}
	}
}

func TestRunTranslateReturnsUsefulGlossaryReadError(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "README.md")
	if err := os.WriteFile(source, []byte("# Demo\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	glossary := filepath.Join(dir, "missing-glossary.md")

	_, err := RunTranslate(context.Background(), TranslateOptions{
		SourcePath:   source,
		Targets:      []string{"zh"},
		OutputDir:    dir,
		GlossaryPath: glossary,
		Now:          fixedNow,
	}, &recordingTranslator{}, nil)
	if err == nil {
		t.Fatal("expected glossary read error")
	}
	if !strings.Contains(err.Error(), glossary) {
		t.Fatalf("error %q does not mention glossary path %q", err, glossary)
	}
}

func TestDryRunDoesNotReadGlossary(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "README.md")
	if err := os.WriteFile(source, []byte("# Demo\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	translator := &recordingTranslator{}

	_, err := RunTranslate(context.Background(), TranslateOptions{
		SourcePath:   source,
		Targets:      []string{"zh"},
		OutputDir:    dir,
		GlossaryPath: filepath.Join(dir, "missing-glossary.md"),
		DryRun:       true,
		Now:          fixedNow,
	}, translator, nil)
	if err != nil {
		t.Fatalf("dry-run should not read glossary: %v", err)
	}
	if len(translator.calls) != 0 {
		t.Fatalf("dry-run called translator %d times", len(translator.calls))
	}
}

func TestRunTranslateAutoSwitcherUpdatesSourceAndTargets(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "README.md")
	if err := os.WriteFile(source, []byte("# Demo\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	translator := &recordingTranslator{}

	_, err := RunTranslate(context.Background(), TranslateOptions{
		SourcePath:   source,
		Targets:      []string{"zh", "ja"},
		OutputDir:    dir,
		AutoSwitcher: true,
		Model:        "test-model",
		Now:          fixedNow,
	}, translator, nil)
	if err != nil {
		t.Fatalf("RunTranslate returned error: %v", err)
	}
	sourceData, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	for _, want := range []string{"[English](README.md)", "[中文](README-zh.md)", "[日本語](README-ja.md)"} {
		if !strings.Contains(string(sourceData), want) {
			t.Fatalf("source switcher missing %s: %s", want, sourceData)
		}
	}
	for _, target := range []string{"zh", "ja"} {
		output := filepath.Join(dir, "README-"+target+".md")
		data, err := os.ReadFile(output)
		if err != nil {
			t.Fatalf("read output %s: %v", output, err)
		}
		if !strings.Contains(string(data), "[日本語](README-ja.md)") {
			t.Fatalf("target switcher missing Japanese link: %s", data)
		}
		if !IsSynchronized(sourceData, data) {
			t.Fatalf("%s metadata should match source after switcher update", output)
		}
	}
}

func TestRunCheckReportsMissingAndStaleFiles(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "README.md")
	if err := os.WriteFile(source, []byte("# Demo\nChanged\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	stale := AddMetadata("# Old\n", Metadata{
		SourcePath:  source,
		Target:      "zh",
		Model:       "test-model",
		SourceHash:  SourceDigest([]byte("# Demo\n")),
		GeneratedAt: fixedNow(),
	})
	if err := os.WriteFile(filepath.Join(dir, "README-zh.md"), []byte(stale), 0o644); err != nil {
		t.Fatalf("write stale translation: %v", err)
	}

	result, err := RunCheck(CheckOptions{
		SourcePath: source,
		Targets:    []string{"zh", "ja"},
		OutputDir:  dir,
	})
	if err == nil {
		t.Fatal("expected stale/missing check error")
	}
	if len(result.Stale) != 1 || len(result.Missing) != 1 {
		t.Fatalf("check result = %+v", result)
	}
}

func fixedNow() time.Time {
	return time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
}
